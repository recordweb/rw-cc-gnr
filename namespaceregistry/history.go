package namespaceregistry

import (
	"encoding/json"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// GetNamespaceHistory liefert die vollständige, unveränderliche
// Änderungshistorie eines Namespace, inklusive Löschungen. Dies ist eine
// der beiden von RWP #25 zwingend vorgeschriebenen Lesefunktionen
// (neben ResolveNamespace) und dient der Nachvollziehbarkeit/Auditierung.
func (c *NamespaceContract) GetNamespaceHistory(
	ctx contractapi.TransactionContextInterface,
	namespace string,
) ([]*HistoryEntry, error) {
	const fn = "GetNamespaceHistory"
	txID := ctx.GetStub().GetTxID()

	if cerr := validateNamespace(namespace); cerr != nil {
		logger.Warn(fn, txID, "", namespace, "ungültiges Namespace-Format bei History-Abfrage", fieldsWithError(cerr))
		return nil, cerr
	}

	iter, err := ctx.GetStub().GetHistoryForKey(namespace)
	if err != nil {
		wrapped := wrapError(ErrHistoryUnavailable, err, "Historie konnte nicht gelesen werden für namespace %q", namespace)
		logger.Error(fn, txID, "", namespace, "Historie-Lesefehler", fieldsWithError(err))
		return nil, wrapped
	}
	defer iter.Close()

	history := []*HistoryEntry{}
	for iter.HasNext() {
		mod, err := iter.Next()
		if err != nil {
			wrapped := wrapError(ErrHistoryUnavailable, err, "Fehler beim Iterieren über die Historie von namespace %q", namespace)
			logger.Error(fn, txID, "", namespace, "Historie-Iterationsfehler", fieldsWithError(err))
			return nil, wrapped
		}

		entry := &HistoryEntry{
			TxID:     mod.TxId,
			IsDelete: mod.IsDelete,
		}
		if mod.Timestamp != nil {
			entry.Timestamp = mod.Timestamp.AsTime().UTC().Format(rfc3339Format)
		}
		if !mod.IsDelete && len(mod.Value) > 0 {
			var rec NamespaceRecord
			if err := json.Unmarshal(mod.Value, &rec); err != nil {
				// Ein nicht deserialisierbarer historischer Eintrag wird
				// geloggt, aber die restliche Historie bleibt nutzbar.
				logger.Error(fn, txID, "", namespace, "historischer Eintrag konnte nicht deserialisiert werden", fieldsWithError(err))
			} else {
				entry.Record = &rec
			}
		}
		history = append(history, entry)
	}

	logger.Info(fn, txID, "", namespace, "Historie erfolgreich abgefragt", map[string]interface{}{
		"entries": len(history),
	})
	return history, nil
}
