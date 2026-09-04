package namespaceregistry

import (
	"encoding/json"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// getNamespaceRecord ist der interne, gemeinsam genutzte Lese-Helfer für
// einen einzelnen Namespace. Öffentliche Contract-Methoden (ResolveNamespace,
// UpdateResolverEndpoint) bauen darauf auf, um Duplikation zu vermeiden.
func (c *NamespaceContract) getNamespaceRecord(
	ctx contractapi.TransactionContextInterface,
	namespace string,
) (*NamespaceRecord, *ContractError) {
	data, err := ctx.GetStub().GetState(namespace)
	if err != nil {
		return nil, wrapError(ErrLedgerRead, err, "World State konnte nicht gelesen werden für namespace %q", namespace)
	}
	if data == nil {
		return nil, newError(ErrNamespaceNotFound, "namespace %q ist nicht registriert", namespace)
	}

	var record NamespaceRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, wrapError(ErrSerialization, err, "NamespaceRecord konnte nicht deserialisiert werden für namespace %q", namespace)
	}
	return &record, nil
}

// ResolveNamespace liefert den vollständigen NamespaceRecord für einen
// gegebenen Namespace. Dies ist eine öffentliche, ungeschützte Lesefunktion:
// jede Organisation im Channel darf jeden Namespace auflösen (die Registry
// ist bewusst als öffentlich lesbares Routing-Verzeichnis konzipiert,
// RWP #25).
func (c *NamespaceContract) ResolveNamespace(
	ctx contractapi.TransactionContextInterface,
	namespace string,
) (*NamespaceRecord, error) {
	const fn = "ResolveNamespace"
	txID := ctx.GetStub().GetTxID()

	if cerr := validateNamespace(namespace); cerr != nil {
		logger.Warn(fn, txID, "", namespace, "ungültiges Namespace-Format bei Abfrage", fieldsWithError(cerr))
		return nil, cerr
	}

	record, cerr := c.getNamespaceRecord(ctx, namespace)
	if cerr != nil {
		logger.Warn(fn, txID, "", namespace, "Namespace nicht gefunden", fieldsWithError(cerr))
		return nil, cerr
	}

	logger.Info(fn, txID, "", namespace, "Namespace erfolgreich aufgelöst", nil)
	return record, nil
}

// GetMyNamespaces liefert alle NamespaceRecords, deren RegisteredBy der
// MSP-ID der aufrufenden Organisation entspricht. Die aufrufende Identität
// wird ausschliesslich aus dem authentifizierten Client-Kontext abgeleitet
// (kein Parameter) — eine Organisation kann so niemals die Namespace-Liste
// einer anderen Organisation über diese Funktion abfragen.
//
// Implementierungshinweis: Mit LevelDB als State-Database ist nur ein
// unselektiver Key-Range-Scan über den gesamten World State möglich; die
// Filterung nach RegisteredBy erfolgt danach in-memory im Chaincode. Bei
// CouchDB als State-Database (siehe Netzwerk-Migration) kann dies künftig
// durch eine indizierte Rich Query (Mango-Query auf "registeredBy")
// ersetzt werden, was bei wachsender Registry-Grösse deutlich günstiger ist.
func (c *NamespaceContract) GetMyNamespaces(
	ctx contractapi.TransactionContextInterface,
) ([]*NamespaceRecord, error) {
	const fn = "GetMyNamespaces"
	txID := ctx.GetStub().GetTxID()

	mspID, cerr := callerMSPID(ctx)
	if cerr != nil {
		logger.Error(fn, txID, "", "", "Identität konnte nicht ermittelt werden", fieldsWithError(cerr))
		return nil, cerr
	}

	iter, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		wrapped := wrapError(ErrLedgerRead, err, "State-Range konnte nicht gelesen werden")
		logger.Error(fn, txID, mspID, "", "Ledger-Lesefehler", fieldsWithError(err))
		return nil, wrapped
	}
	defer iter.Close()

	records := []*NamespaceRecord{}
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			wrapped := wrapError(ErrLedgerRead, err, "Fehler beim Iterieren über den World State")
			logger.Error(fn, txID, mspID, "", "Iterationsfehler", fieldsWithError(err))
			return nil, wrapped
		}

		var record NamespaceRecord
		if err := json.Unmarshal(kv.Value, &record); err != nil {
			// Ein einzelner beschädigter/fremder Eintrag darf die gesamte
			// Abfrage nicht zum Absturz bringen; wir loggen und überspringen.
			logger.Error(fn, txID, mspID, kv.Key, "Eintrag konnte nicht deserialisiert werden, wird übersprungen", fieldsWithError(err))
			continue
		}
		if record.RegisteredBy == mspID {
			records = append(records, &record)
		}
	}

	logger.Info(fn, txID, mspID, "", "eigene Namespaces abgefragt", map[string]interface{}{
		"count": len(records),
	})
	return records, nil
}
