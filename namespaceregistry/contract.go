// Package namespaceregistry implementiert den RecordWeb Global Namespace
// Registry (RW-GNR) Chaincode für Hyperledger Fabric.
//
// Zweck (gemäss RWP #25): Verwaltung von Routing-Metadaten für globale
// RecordWeb-Namespaces (canonical UUIDv4, RWP #23). Der Chaincode speichert
// AUSSCHLIESSLICH: namespace, resolverEndpoint, registeredBy, registeredAt,
// txId (sowie sekundäre Metadaten wie updatedAt, endorsedBy, schemaVersion).
//
// Er speichert NIEMALS: DID-Dokumente, Records, Record-Inhalte oder
// Access-Control-Entscheidungen. Diese Trennung ist eine bewusste,
// governance-relevante Grenze (RWP #25) und darf durch künftige Erweiterungen
// dieses Chaincodes nicht aufgeweicht werden.
package namespaceregistry

import "github.com/hyperledger/fabric-contract-api-go/contractapi"

// NamespaceContract implementiert den Hyperledger Fabric Smart Contract
// für die RecordWeb Global Namespace Registry (RW-GNR).
//
// Öffentliche Transaktionsfunktionen:
//   - RegisterNamespace(namespace, resolverEndpoint)       [Submit]
//   - UpdateResolverEndpoint(namespace, newResolverEndpoint) [Submit]
//   - ResolveNamespace(namespace)                           [Evaluate]
//   - GetMyNamespaces()                                     [Evaluate]
//   - GetNamespaceHistory(namespace)                        [Evaluate]
//
// Die Registrar-Identität (registeredBy) wird in allen Fällen aus der
// authentifizierten Fabric-Client-Identität abgeleitet, niemals aus einem
// Transaktionsargument entgegengenommen (RWP #25, Kommentar).
type NamespaceContract struct {
	contractapi.Contract
}
