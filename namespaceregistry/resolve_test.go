package namespaceregistry_test

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/require"

	ns "github.com/recordweb/rw-cc-gnr/namespaceregistry"
	"github.com/recordweb/rw-cc-gnr/namespaceregistry/mocks"
)

const otherNamespace = "b4c8f32d-5678-4bcd-9ef0-2345678901bc"

func TestResolveNamespace_Success(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	existing := `{"namespace":"` + validNamespace + `","resolverEndpoint":"` + validEndpoint + `","registeredBy":"` + testMSPID + `"}`
	stub.GetStateReturns([]byte(existing), nil)

	contract := ns.NamespaceContract{}
	record, err := contract.ResolveNamespace(ctx, validNamespace)
	require.NoError(t, err)
	require.Equal(t, validNamespace, record.Namespace)
	require.Equal(t, validEndpoint, record.ResolverEndpoint)
}

func TestResolveNamespace_NotFound(t *testing.T) {
	ctx, stub := setup(t, testMSPID)
	stub.GetStateReturns(nil, nil)

	contract := ns.NamespaceContract{}
	_, err := contract.ResolveNamespace(ctx, validNamespace)
	require.Error(t, err)
	require.Contains(t, err.Error(), "NAMESPACE_NOT_FOUND")
}

func TestResolveNamespace_RejectsInvalidFormat(t *testing.T) {
	ctx, _ := setup(t, testMSPID)

	contract := ns.NamespaceContract{}
	_, err := contract.ResolveNamespace(ctx, "invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_NAMESPACE_FORMAT")
}

// TestGetMyNamespaces_FiltersByCallerMSP steuert den generierten
// mocks.StateQueryIterator über HasNextReturnsOnCall / NextReturnsOnCall,
// exakt nach dem Muster, das counterfeiter für Iterator-Interfaces erzeugt
// (siehe hyperledger/fabric-samples für GetAllAssets-artige Tests).
func TestGetMyNamespaces_FiltersByCallerMSP(t *testing.T) {
	ctx, stub := setup(t, testMSPID)

	iterator := &mocks.StateQueryIterator{}
	stub.GetStateByRangeReturns(iterator, nil)

	ownRecord := `{"namespace":"` + validNamespace + `","resolverEndpoint":"` + validEndpoint + `","registeredBy":"` + testMSPID + `"}`
	foreignRecord := `{"namespace":"` + otherNamespace + `","resolverEndpoint":"` + validEndpoint + `","registeredBy":"` + otherMSPID + `"}`

	iterator.HasNextReturnsOnCall(0, true)
	iterator.HasNextReturnsOnCall(1, true)
	iterator.HasNextReturnsOnCall(2, false)
	iterator.NextReturnsOnCall(0, &queryresult.KV{Key: validNamespace, Value: []byte(ownRecord)}, nil)
	iterator.NextReturnsOnCall(1, &queryresult.KV{Key: otherNamespace, Value: []byte(foreignRecord)}, nil)

	contract := ns.NamespaceContract{}
	records, err := contract.GetMyNamespaces(ctx)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, validNamespace, records[0].Namespace)
	require.Equal(t, testMSPID, records[0].RegisteredBy)
}

func TestGetMyNamespaces_EmptyRegistryReturnsEmptySlice(t *testing.T) {
	ctx, stub := setup(t, testMSPID)

	iterator := &mocks.StateQueryIterator{}
	iterator.HasNextReturns(false)
	stub.GetStateByRangeReturns(iterator, nil)

	contract := ns.NamespaceContract{}
	records, err := contract.GetMyNamespaces(ctx)
	require.NoError(t, err)
	require.Empty(t, records)
}
