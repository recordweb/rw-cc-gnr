package namespaceregistry

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// deriveNamespaceFromTxID generates a canonical, lowercase UUIDv4-formatted
// string deterministically from the current transaction ID.
//
// Why not a "real" random UUIDv4 (uuid.New())? Chaincode must produce
// byte-identical results on every endorsing peer for a transaction to pass
// endorsement — true randomness (crypto/rand under the hood of uuid.New())
// would yield a different value on each peer and make the transaction fail
// endorsement comparison as soon as more than one organisation endorses,
// which is always the case on this network (multi-org endorsement policy).
//
// Fabric's transaction ID is itself a SHA-256 hash over the transaction's
// nonce, creator identity, and payload, computed once by the submitting
// client and included in the transaction envelope — so it is identical on
// every endorsing peer. Hashing it again here only reshapes it into the
// UUIDv4 byte layout; it does not reduce its uniqueness guarantee.
//
// Uniqueness guarantee: exactly one namespace may be derived per
// transaction, because deriveNamespaceFromTxID must be called at most once
// within RegisterNamespace. If two organisations register a namespace in
// the same block, each does so in its own transaction with its own TxID,
// so no collision can occur. Submitting the *same* resolverEndpoint in two
// separate transactions is explicitly allowed and intentional: it yields
// two distinct namespaces pointing at the same endpoint (one endpoint may
// serve many namespaces; one namespace may only ever have one endpoint).
//
// The resulting string is RFC 4122 compliant (version 4, variant 10) and
// therefore indistinguishable in syntax from a randomly generated UUIDv4 —
// it just is not drawn from a random source, which is a deliberate,
// accepted trade-off for chaincode determinism.
func deriveNamespaceFromTxID(ctx contractapi.TransactionContextInterface) string {
	txID := ctx.GetStub().GetTxID()
	digest := sha256.Sum256([]byte(txID))

	b := make([]byte, 16)
	copy(b, digest[:16])
	b[6] = (b[6] & 0x0F) | 0x40 // set version 4
	b[8] = (b[8] & 0x3F) | 0x80 // set variant 10 (RFC 4122)

	hexStr := hex.EncodeToString(b)
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]
}
