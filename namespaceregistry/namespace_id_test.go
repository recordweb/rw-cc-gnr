package namespaceregistry

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/recordweb/rw-cc-gnr/namespaceregistry/mocks"
)

// TestDeriveNamespaceFromTxID_PassesOwnValidation ensures the internally
// derived namespace always satisfies the same UUIDv4 rules enforced on
// caller-supplied namespaces (Update/Resolve/History) — i.e. the contract
// never generates a value it would itself reject.
func TestDeriveNamespaceFromTxID_PassesOwnValidation(t *testing.T) {
	stub := &mocks.ChaincodeStub{}
	stub.GetTxIDReturns("some-arbitrary-tx-id-1234")

	ctx := &mocks.TransactionContext{}
	ctx.GetStubReturns(stub)

	namespace := deriveNamespaceFromTxID(ctx)
	err := validateNamespace(namespace)
	require.Nil(t, err, "derived namespace must pass validateNamespace: %v", err)
}

func TestDeriveNamespaceFromTxID_DeterministicForSameTxID(t *testing.T) {
	stub1 := &mocks.ChaincodeStub{}
	stub1.GetTxIDReturns("identical-tx-id")
	ctx1 := &mocks.TransactionContext{}
	ctx1.GetStubReturns(stub1)

	stub2 := &mocks.ChaincodeStub{}
	stub2.GetTxIDReturns("identical-tx-id")
	ctx2 := &mocks.TransactionContext{}
	ctx2.GetStubReturns(stub2)

	require.Equal(t, deriveNamespaceFromTxID(ctx1), deriveNamespaceFromTxID(ctx2))
}

func TestDeriveNamespaceFromTxID_DifferentForDifferentTxID(t *testing.T) {
	stub1 := &mocks.ChaincodeStub{}
	stub1.GetTxIDReturns("tx-id-one")
	ctx1 := &mocks.TransactionContext{}
	ctx1.GetStubReturns(stub1)

	stub2 := &mocks.ChaincodeStub{}
	stub2.GetTxIDReturns("tx-id-two")
	ctx2 := &mocks.TransactionContext{}
	ctx2.GetStubReturns(stub2)

	require.NotEqual(t, deriveNamespaceFromTxID(ctx1), deriveNamespaceFromTxID(ctx2))
}
