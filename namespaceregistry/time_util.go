package namespaceregistry

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// rfc3339Format ist das einheitliche Zeitformat für alle im NamespaceRecord
// und in HistoryEntry gespeicherten Zeitangaben.
const rfc3339Format = time.RFC3339

// formatTxTimestamp wandelt den deterministischen Transaktions-Zeitstempel
// (von ctx.GetStub().GetTxTimestamp()) in einen RFC3339-String um.
//
// WICHTIG: Es wird bewusst NICHT time.Now() verwendet. time.Now() liefert
// auf jedem endorsierenden Peer einen leicht unterschiedlichen Wert, was in
// deterministischem Chaincode zu unterschiedlichen Schreib-Ergebnissen pro
// Peer und damit zu einem Endorsement-Mismatch führen würde. Der
// Transaktions-Zeitstempel wird stattdessen vom Client beim Einreichen der
// Transaktion gesetzt und ist für alle Endorser identisch.
func formatTxTimestamp(ts *timestamppb.Timestamp) string {
	return ts.AsTime().UTC().Format(rfc3339Format)
}
