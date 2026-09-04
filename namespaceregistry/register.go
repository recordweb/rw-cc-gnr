package namespaceregistry

import (
	"encoding/json"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// eventNamespaceRegistered / eventNamespaceUpdated sind die Namen der
// Chaincode-Events, die bei erfolgreicher Mutation emittiert werden.
// Externe Konsumenten (z.B. ein Resolver-Cache oder ein Audit-Listener)
// können sich auf diese Event-Namen abonnieren, statt den Ledger zu pollen.
const (
	eventNamespaceRegistered = "NamespaceRegistered"
	eventNamespaceUpdated    = "NamespaceUpdated"
)

// RegisterNamespace registriert einen neuen Global Namespace Identifier
// in der Registry. Schlägt fehl, wenn der Namespace kein gültiger,
// kanonischer UUIDv4 ist, wenn resolverEndpoint keine gültige HTTPS-URL
// ist, oder wenn der Namespace bereits existiert.
//
// registeredBy wird NICHT als Parameter entgegengenommen: die Registrar-
// Identität wird ausschliesslich aus der authentifizierten Fabric-Client-
// Identität abgeleitet (siehe identity.go), um Identity-Spoofing
// strukturell auszuschliessen.
func (c *NamespaceContract) RegisterNamespace(
	ctx contractapi.TransactionContextInterface,
	namespace string,
	resolverEndpoint string,
) error {
	const fn = "RegisterNamespace"
	txID := ctx.GetStub().GetTxID()

	mspID, cerr := callerMSPID(ctx)
	if cerr != nil {
		logger.Error(fn, txID, "", namespace, "Identität konnte nicht ermittelt werden", fieldsWithError(cerr))
		return cerr
	}

	if cerr := validateNamespace(namespace); cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "ungültiges Namespace-Format", fieldsWithError(cerr))
		return cerr
	}
	if cerr := validateResolverEndpoint(resolverEndpoint); cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "ungültiger resolverEndpoint", fieldsWithError(cerr))
		return cerr
	}

	existing, err := ctx.GetStub().GetState(namespace)
	if err != nil {
		wrapped := wrapError(ErrLedgerRead, err, "World State konnte nicht gelesen werden für namespace %q", namespace)
		logger.Error(fn, txID, mspID, namespace, "Ledger-Lesefehler", fieldsWithError(err))
		return wrapped
	}
	if existing != nil {
		cerr := newError(ErrNamespaceExists, "namespace %q ist bereits registriert", namespace)
		logger.Warn(fn, txID, mspID, namespace, "Registrierungsversuch für existierenden Namespace", nil)
		return cerr
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "Transaktions-Zeitstempel konnte nicht ermittelt werden")
		logger.Error(fn, txID, mspID, namespace, "Zeitstempel-Fehler", fieldsWithError(err))
		return wrapped
	}
	timestamp := formatTxTimestamp(txTimestamp)

	record := NamespaceRecord{
		DocType:          docTypeNamespaceRecord,
		Namespace:        namespace,
		ResolverEndpoint: resolverEndpoint,
		RegisteredBy:     mspID,
		RegisteredAt:     timestamp,
		UpdatedAt:        timestamp,
		TxID:             txID,
		EndorsedBy:       []string{mspID},
		SchemaVersion:    CurrentSchemaVersion,
	}

	data, err := json.Marshal(record)
	if err != nil {
		wrapped := wrapError(ErrSerialization, err, "NamespaceRecord konnte nicht serialisiert werden")
		logger.Error(fn, txID, mspID, namespace, "Serialisierungsfehler", fieldsWithError(err))
		return wrapped
	}

	if err := ctx.GetStub().PutState(namespace, data); err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "NamespaceRecord konnte nicht persistiert werden für namespace %q", namespace)
		logger.Error(fn, txID, mspID, namespace, "Ledger-Schreibfehler", fieldsWithError(err))
		return wrapped
	}

	if err := ctx.GetStub().SetEvent(eventNamespaceRegistered, data); err != nil {
		// Ein fehlgeschlagenes Event-Setzen darf die bereits erfolgte
		// Zustandsänderung nicht zurückrollen; wir loggen es aber, weil
		// nachgeschaltete Konsumenten (z.B. Resolver-Caches) dadurch
		// die Aktualisierung verpassen könnten.
		logger.Error(fn, txID, mspID, namespace, "Chaincode-Event konnte nicht gesetzt werden", fieldsWithError(err))
	}

	logger.Info(fn, txID, mspID, namespace, "Namespace erfolgreich registriert", map[string]interface{}{
		"resolverEndpoint": resolverEndpoint,
	})
	return nil
}

// UpdateResolverEndpoint aktualisiert den resolverEndpoint eines bereits
// registrierten Namespace. Nur die MSP, die den Namespace ursprünglich
// registriert hat, darf diese Operation ausführen; die Prüfung erfolgt
// gegen die authentifizierte Aufrufer-Identität, nicht gegen ein Argument.
func (c *NamespaceContract) UpdateResolverEndpoint(
	ctx contractapi.TransactionContextInterface,
	namespace string,
	newResolverEndpoint string,
) error {
	const fn = "UpdateResolverEndpoint"
	txID := ctx.GetStub().GetTxID()

	mspID, cerr := callerMSPID(ctx)
	if cerr != nil {
		logger.Error(fn, txID, "", namespace, "Identität konnte nicht ermittelt werden", fieldsWithError(cerr))
		return cerr
	}

	if cerr := validateNamespace(namespace); cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "ungültiges Namespace-Format", fieldsWithError(cerr))
		return cerr
	}
	if cerr := validateResolverEndpoint(newResolverEndpoint); cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "ungültiger resolverEndpoint", fieldsWithError(cerr))
		return cerr
	}

	record, cerr := c.getNamespaceRecord(ctx, namespace)
	if cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "Namespace für Update nicht gefunden", fieldsWithError(cerr))
		return cerr
	}

	if record.RegisteredBy != mspID {
		cerr := newError(ErrUnauthorized, "aufrufende Organisation %q ist nicht Registrar von namespace %q (registriert durch %q)", mspID, namespace, record.RegisteredBy)
		logger.Warn(fn, txID, mspID, namespace, "nicht autorisierter Update-Versuch", map[string]interface{}{
			"registeredBy": record.RegisteredBy,
		})
		return cerr
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "Transaktions-Zeitstempel konnte nicht ermittelt werden")
		logger.Error(fn, txID, mspID, namespace, "Zeitstempel-Fehler", fieldsWithError(err))
		return wrapped
	}

	record.ResolverEndpoint = newResolverEndpoint
	record.UpdatedAt = formatTxTimestamp(txTimestamp)
	record.TxID = txID

	data, err := json.Marshal(record)
	if err != nil {
		wrapped := wrapError(ErrSerialization, err, "aktualisierter NamespaceRecord konnte nicht serialisiert werden")
		logger.Error(fn, txID, mspID, namespace, "Serialisierungsfehler", fieldsWithError(err))
		return wrapped
	}

	if err := ctx.GetStub().PutState(namespace, data); err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "aktualisierter NamespaceRecord konnte nicht persistiert werden für namespace %q", namespace)
		logger.Error(fn, txID, mspID, namespace, "Ledger-Schreibfehler", fieldsWithError(err))
		return wrapped
	}

	if err := ctx.GetStub().SetEvent(eventNamespaceUpdated, data); err != nil {
		logger.Error(fn, txID, mspID, namespace, "Chaincode-Event konnte nicht gesetzt werden", fieldsWithError(err))
	}

	logger.Info(fn, txID, mspID, namespace, "resolverEndpoint erfolgreich aktualisiert", map[string]interface{}{
		"newResolverEndpoint": newResolverEndpoint,
	})
	return nil
}
