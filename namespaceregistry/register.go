package namespaceregistry

import (
	"encoding/json"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// eventNamespaceRegistered / eventNamespaceUpdated are the names of the
// chaincode events emitted on successful mutations. External consumers
// (e.g. a resolver cache or an audit listener) can subscribe to these
// event names instead of polling the ledger.
const (
	eventNamespaceRegistered = "NamespaceRegistered"
	eventNamespaceUpdated    = "NamespaceUpdated"
)

// RegisterNamespace registers a new Global Namespace Identifier in the
// registry and returns the resulting record, including the namespace that
// was assigned.
//
// The namespace itself is NOT a caller-supplied argument. It is derived
// deterministically from the current transaction ID (see
// deriveNamespaceFromTxID in namespace_id.go). This removes the caller's
// ability to pick, predict, or collide a namespace value, and guarantees
// exactly one freshly derived namespace per RegisterNamespace call.
//
// Business rule (deliberate, confirmed): a namespace may only ever be
// registered once and therefore have exactly one resolverEndpoint. The
// same resolverEndpoint may be reused across many separate
// RegisterNamespace calls/namespaces — this is intentional. Each call
// yields a new, distinct namespace pointing at that endpoint; it is up to
// the calling organisation to track which of its namespaces is used for
// what purpose.
//
// registeredBy is NOT a parameter either: the registrar identity is
// derived exclusively from the authenticated Fabric client identity (see
// identity.go), never overridable via transaction arguments.
func (c *NamespaceContract) RegisterNamespace(
	ctx contractapi.TransactionContextInterface,
	resolverEndpoint string,
) (*NamespaceRecord, error) {
	const fn = "RegisterNamespace"
	txID := ctx.GetStub().GetTxID()

	mspID, cerr := callerMSPID(ctx)
	if cerr != nil {
		logger.Error(fn, txID, "", "", "identity could not be determined", fieldsWithError(cerr))
		return nil, cerr
	}

	if cerr := validateResolverEndpoint(resolverEndpoint); cerr != nil {
		logger.Warn(fn, txID, mspID, "", "invalid resolverEndpoint", fieldsWithError(cerr))
		return nil, cerr
	}

	namespace := deriveNamespaceFromTxID(ctx)

	// Defensive check: a collision would only be possible if the same
	// TxID were ever reused (which Fabric itself prevents at the ledger
	// level), or if this function were called more than once within the
	// same transaction. Checked anyway, since PutState would otherwise
	// silently overwrite an existing record.
	existing, err := ctx.GetStub().GetState(namespace)
	if err != nil {
		wrapped := wrapError(ErrLedgerRead, err, "failed to read world state for namespace %q", namespace)
		logger.Error(fn, txID, mspID, namespace, "ledger read error", fieldsWithError(err))
		return nil, wrapped
	}
	if existing != nil {
		cerr := newError(ErrNamespaceExists, "derived namespace %q already exists (unexpected TxID collision or duplicate derivation within the same transaction)", namespace)
		logger.Error(fn, txID, mspID, namespace, "unexpected namespace collision on registration", nil)
		return nil, cerr
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "failed to determine transaction timestamp")
		logger.Error(fn, txID, mspID, namespace, "timestamp error", fieldsWithError(err))
		return nil, wrapped
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
		wrapped := wrapError(ErrSerialization, err, "failed to serialize NamespaceRecord")
		logger.Error(fn, txID, mspID, namespace, "serialization error", fieldsWithError(err))
		return nil, wrapped
	}

	if err := ctx.GetStub().PutState(namespace, data); err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "failed to persist NamespaceRecord for namespace %q", namespace)
		logger.Error(fn, txID, mspID, namespace, "ledger write error", fieldsWithError(err))
		return nil, wrapped
	}

	if err := ctx.GetStub().SetEvent(eventNamespaceRegistered, data); err != nil {
		// A failed event emission must not roll back the state change
		// that already happened; we log it because downstream consumers
		// (e.g. resolver caches) might miss the update as a result.
		logger.Error(fn, txID, mspID, namespace, "failed to set chaincode event", fieldsWithError(err))
	}

	logger.Info(fn, txID, mspID, namespace, "namespace successfully registered", map[string]interface{}{
		"resolverEndpoint": resolverEndpoint,
	})
	return &record, nil
}

// UpdateResolverEndpoint updates the resolverEndpoint of an already
// registered namespace. Only the organisation that originally registered
// the namespace may perform this operation; the check is made against the
// authenticated caller identity, never against an argument.
func (c *NamespaceContract) UpdateResolverEndpoint(
	ctx contractapi.TransactionContextInterface,
	namespace string,
	newResolverEndpoint string,
) error {
	const fn = "UpdateResolverEndpoint"
	txID := ctx.GetStub().GetTxID()

	mspID, cerr := callerMSPID(ctx)
	if cerr != nil {
		logger.Error(fn, txID, "", namespace, "identity could not be determined", fieldsWithError(cerr))
		return cerr
	}

	if cerr := validateNamespace(namespace); cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "invalid namespace format", fieldsWithError(cerr))
		return cerr
	}
	if cerr := validateResolverEndpoint(newResolverEndpoint); cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "invalid resolverEndpoint", fieldsWithError(cerr))
		return cerr
	}

	record, cerr := c.getNamespaceRecord(ctx, namespace)
	if cerr != nil {
		logger.Warn(fn, txID, mspID, namespace, "namespace not found for update", fieldsWithError(cerr))
		return cerr
	}

	if record.RegisteredBy != mspID {
		cerr := newError(ErrUnauthorized, "calling organisation %q is not the registrar of namespace %q (registered by %q)", mspID, namespace, record.RegisteredBy)
		logger.Warn(fn, txID, mspID, namespace, "unauthorized update attempt", map[string]interface{}{
			"registeredBy": record.RegisteredBy,
		})
		return cerr
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "failed to determine transaction timestamp")
		logger.Error(fn, txID, mspID, namespace, "timestamp error", fieldsWithError(err))
		return wrapped
	}

	record.ResolverEndpoint = newResolverEndpoint
	record.UpdatedAt = formatTxTimestamp(txTimestamp)
	record.TxID = txID

	data, err := json.Marshal(record)
	if err != nil {
		wrapped := wrapError(ErrSerialization, err, "failed to serialize updated NamespaceRecord")
		logger.Error(fn, txID, mspID, namespace, "serialization error", fieldsWithError(err))
		return wrapped
	}

	if err := ctx.GetStub().PutState(namespace, data); err != nil {
		wrapped := wrapError(ErrLedgerWrite, err, "failed to persist updated NamespaceRecord for namespace %q", namespace)
		logger.Error(fn, txID, mspID, namespace, "ledger write error", fieldsWithError(err))
		return wrapped
	}

	if err := ctx.GetStub().SetEvent(eventNamespaceUpdated, data); err != nil {
		logger.Error(fn, txID, mspID, namespace, "failed to set chaincode event", fieldsWithError(err))
	}

	logger.Info(fn, txID, mspID, namespace, "resolverEndpoint successfully updated", map[string]interface{}{
		"newResolverEndpoint": newResolverEndpoint,
	})
	return nil
}
