package namespaceregistry_test

import (
	"testing"
	"time"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	ns "github.com/recordweb/rw-cc-gnr/namespaceregistry"
	"github.com/recordweb/rw-cc-gnr/namespaceregistry/mocks"
)

func TestGetNamespaceHistory_ReturnsChronologicalEntries(t *testing.T) {
	ctx, stub := setup(t, testMSPID)

	iterator := &mocks.HistoryQueryIterator{}
	stub.GetHistoryForKeyReturns(iterator, nil)

	firstValue := `{"namespace":"` + validNamespace + `","resolverEndpoint":"` + validEndpoint + `","registeredBy":"` + testMSPID + `"}`
	secondValue := `{"namespace":"` + validNamespace + `","resolverEndpoint":"https://updated.example.org","registeredBy":"` + testMSPID + `"}`

	t1 := timestamppb.New(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))
	t2 := timestamppb.New(time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC))

	iterator.HasNextReturnsOnCall(0, true)
	iterator.HasNextReturnsOnCall(1, true)
	iterator.HasNextReturnsOnCall(2, false)
	iterator.NextReturnsOnCall(0, &queryresult.KeyModification{
		TxId: "tx-0001", Timestamp: t1, IsDelete: false, Value: []byte(firstValue),
	}, nil)
	iterator.NextReturnsOnCall(1, &queryresult.KeyModification{
		TxId: "tx-0002", Timestamp: t2, IsDelete: false, Value: []byte(secondValue),
	}, nil)

	contract := ns.NamespaceContract{}
	history, err := contract.GetNamespaceHistory(ctx, validNamespace)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "tx-0001", history[0].TxID)
	require.False(t, history[0].IsDelete)
	require.Equal(t, validEndpoint, history[0].Record.ResolverEndpoint)
	require.Equal(t, "tx-0002", history[1].TxID)
	require.Equal(t, "https://updated.example.org", history[1].Record.ResolverEndpoint)
}

func TestGetNamespaceHistory_RejectsInvalidNamespace(t *testing.T) {
	ctx, _ := setup(t, testMSPID)

	contract := ns.NamespaceContract{}
	_, err := contract.GetNamespaceHistory(ctx, "not-a-uuid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_NAMESPACE_FORMAT")
}

func TestGetNamespaceHistory_EmptyHistory(t *testing.T) {
	ctx, stub := setup(t, testMSPID)

	iterator := &mocks.HistoryQueryIterator{}
	iterator.HasNextReturns(false)
	stub.GetHistoryForKeyReturns(iterator, nil)

	contract := ns.NamespaceContract{}
	history, err := contract.GetNamespaceHistory(ctx, validNamespace)
	require.NoError(t, err)
	require.Empty(t, history)
}
