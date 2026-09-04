package namespaceregistry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	ns "github.com/recordweb/rw-cc-gnr/namespaceregistry"
	"github.com/recordweb/rw-cc-gnr/namespaceregistry/mocks"
)

const (
	validNamespace = "a3f9e21c-1234-4abc-8def-1234567890ab" // gültiger, kleingeschriebener UUIDv4
	validEndpoint  = "https://resolver.example.org/rwp/1.0/identifiers"
	testMSPID      = "SwissGovOrgMSP"
	otherMSPID     = "RecordWebOrgMSP"
)

// setup baut einen einsatzbereiten mocks.TransactionContext + mocks.ChaincodeStub
// zusammen, inklusive einer Standard-MSP-Identität und einem festen
// deterministischen Tx-Timestamp — analog zum etablierten Testmuster in
// hyperledger/fabric-samples/asset-transfer-basic/chaincode-go.
func setup(t *testing.T, mspID string) (*mocks.TransactionContext, *mocks.ChaincodeStub) {
	t.Helper()

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	clientIdentity := &mocks.ClientIdentity{}
	clientIdentity.GetMSPIDReturns(mspID, nil)
	transactionContext.GetClientIdentityReturns(clientIdentity)

	fixedTime := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	chaincodeStub.GetTxIDReturns("tx-0001")
	chaincodeStub.GetTxTimestampReturns(timestamppb.New(fixedTime), nil)

	return transactionContext, chaincodeStub
}

func TestRegisterNamespace_Success(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	stub.GetStateReturns(nil, nil) // noch nicht registriert

	contract := ns.NamespaceContract{}
	err := contract.RegisterNamespace(ctx, validNamespace, validEndpoint)
	require.NoError(t, err)
	require.Equal(t, 1, stub.PutStateCallCount())

	key, value := stub.PutStateArgsForCall(0)
	require.Equal(t, validNamespace, key)
	require.Contains(t, string(value), testMSPID)
	require.Contains(t, string(value), validEndpoint)
}

func TestRegisterNamespace_RejectsInvalidUUID(t *testing.T) {
	ctx, stub := setup(t, testMSPID)

	contract := ns.NamespaceContract{}
	err := contract.RegisterNamespace(ctx, "not-a-uuid", validEndpoint)
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_NAMESPACE_FORMAT")
	require.Equal(t, 0, stub.PutStateCallCount())
}

func TestRegisterNamespace_RejectsUppercaseUUID(t *testing.T) {
	ctx, _ := setup(t, testMSPID)

	contract := ns.NamespaceContract{}
	upper := "A3F9E21C-1234-4ABC-8DEF-1234567890AB"
	err := contract.RegisterNamespace(ctx, upper, validEndpoint)
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_NAMESPACE_FORMAT")
}

func TestRegisterNamespace_RejectsNonHttpsEndpoint(t *testing.T) {
	ctx, _ := setup(t, testMSPID)

	contract := ns.NamespaceContract{}
	err := contract.RegisterNamespace(ctx, validNamespace, "http://insecure.example.org")
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_RESOLVER_ENDPOINT")
}

func TestRegisterNamespace_RejectsDuplicate(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	stub.GetStateReturns([]byte(`{"namespace":"`+validNamespace+`"}`), nil)

	contract := ns.NamespaceContract{}
	err := contract.RegisterNamespace(ctx, validNamespace, validEndpoint)
	require.Error(t, err)
	require.Contains(t, err.Error(), "NAMESPACE_ALREADY_EXISTS")
}

func TestRegisterNamespace_PropagatesLedgerReadError(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	stub.GetStateReturns(nil, errors.New("ledger unavailable"))

	contract := ns.NamespaceContract{}
	err := contract.RegisterNamespace(ctx, validNamespace, validEndpoint)
	require.Error(t, err)
	require.Contains(t, err.Error(), "LEDGER_READ_FAILED")
}

func TestRegisterNamespace_SetsChaincodeEvent(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	stub.GetStateReturns(nil, nil)

	contract := ns.NamespaceContract{}
	err := contract.RegisterNamespace(ctx, validNamespace, validEndpoint)
	require.NoError(t, err)

	require.Equal(t, 1, stub.SetEventCallCount())
	name, _ := stub.SetEventArgsForCall(0)
	require.Equal(t, "NamespaceRegistered", name)
}

func TestUpdateResolverEndpoint_OnlyOriginalRegistrarAllowed(t *testing.T) {
	ctx, stub := setup(t, otherMSPID) // anderer Aufrufer als der ursprüngliche Registrar
	existing := `{"namespace":"` + validNamespace + `","resolverEndpoint":"` + validEndpoint + `","registeredBy":"` + testMSPID + `"}`
	stub.GetStateReturns([]byte(existing), nil)

	contract := ns.NamespaceContract{}
	err := contract.UpdateResolverEndpoint(ctx, validNamespace, "https://new.example.org/resolver")
	require.Error(t, err)
	require.Contains(t, err.Error(), "UNAUTHORIZED_REGISTRAR")
	require.Equal(t, 0, stub.PutStateCallCount())
}

func TestUpdateResolverEndpoint_SucceedsForOriginalRegistrar(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	existing := `{"namespace":"` + validNamespace + `","resolverEndpoint":"` + validEndpoint + `","registeredBy":"` + testMSPID + `"}`
	stub.GetStateReturns([]byte(existing), nil)

	contract := ns.NamespaceContract{}
	newEndpoint := "https://new.example.org/resolver"
	err := contract.UpdateResolverEndpoint(ctx, validNamespace, newEndpoint)
	require.NoError(t, err)
	require.Equal(t, 1, stub.PutStateCallCount())

	_, value := stub.PutStateArgsForCall(0)
	require.Contains(t, string(value), newEndpoint)
}

func TestUpdateResolverEndpoint_NotFound(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	stub.GetStateReturns(nil, nil)

	contract := ns.NamespaceContract{}
	err := contract.UpdateResolverEndpoint(ctx, validNamespace, validEndpoint)
	require.Error(t, err)
	require.Contains(t, err.Error(), "NAMESPACE_NOT_FOUND")
}
