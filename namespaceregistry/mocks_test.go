package namespaceregistry

import (
	"fmt"
	"sort"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─────────────────────────────────────────────────────────────────────────
// Schlanke, handgeschriebene Test-Doubles für die Fabric-Interfaces.
//
// fabric-contract-api-go liefert kein eigenes Mock-Paket; das ältere
// fabric-chaincode-go/shimtest.MockStub deckt zwar ChaincodeStubInterface ab,
// aber nicht TransactionContextInterface (inkl. GetClientIdentity()), das
// contractapi zusätzlich verlangt. Daher hier ein minimaler, aber
// vollständiger Mock für genau die Methoden, die dieser Chaincode nutzt.
// ─────────────────────────────────────────────────────────────────────────

// mockClientIdentity implementiert cid.ClientIdentity minimal für Tests.
type mockClientIdentity struct {
	mspID string
	id    string
	err   error
}

func (m *mockClientIdentity) GetID() (string, error)      { return m.id, m.err }
func (m *mockClientIdentity) GetMSPID() (string, error)   { return m.mspID, m.err }
func (m *mockClientIdentity) GetAttributeValue(attrName string) (value string, found bool, err error) {
	return "", false, nil
}
func (m *mockClientIdentity) AssertAttributeValue(attrName, attrValue string) error { return nil }
func (m *mockClientIdentity) GetX509Certificate() (*x509CertPlaceholder, error)     { return nil, nil }

// x509CertPlaceholder ersetzt *x509.Certificate nur für die Signatur;
// der Chaincode ruft GetX509Certificate() nirgends auf, daher genügt dies.
type x509CertPlaceholder struct{}

// mockStub implementiert contractapi.TransactionContextInterface + die
// Teilmenge von shim.ChaincodeStubInterface, die der Contract tatsächlich
// verwendet: GetState, PutState, GetStateByRange, GetHistoryForKey,
// GetTxID, GetTxTimestamp, SetEvent.
type mockStub struct {
	shim.ChaincodeStubInterface // eingebettet, damit nicht implementierte Methoden nicht separat gestubbt werden müssen (panicken bei Aufruf)

	state    map[string][]byte
	history  map[string][]historyRecord
	txID     string
	txTime   time.Time
	identity *mockClientIdentity
	lastEvent     string
	lastEventData []byte
}

type historyRecord struct {
	txID     string
	ts       time.Time
	isDelete bool
	value    []byte
}

func newMockStub(mspID string, txID string) *mockStub {
	return &mockStub{
		state:    map[string][]byte{},
		history:  map[string][]historyRecord{},
		txID:     txID,
		txTime:   time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC),
		identity: &mockClientIdentity{mspID: mspID, id: "x509::" + mspID},
	}
}

func (s *mockStub) GetState(key string) ([]byte, error) {
	return s.state[key], nil
}

func (s *mockStub) PutState(key string, value []byte) error {
	s.state[key] = value
	s.history[key] = append(s.history[key], historyRecord{txID: s.txID, ts: s.txTime, value: append([]byte{}, value...)})
	return nil
}

func (s *mockStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	keys := make([]string, 0, len(s.state))
	for k := range s.state {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return &mockRangeIterator{keys: keys, values: s.state, pos: 0}, nil
}

func (s *mockStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	return &mockHistoryIterator{records: s.history[key], pos: 0}, nil
}

func (s *mockStub) GetTxID() string { return s.txID }

func (s *mockStub) GetTxTimestamp() (*timestamppb.Timestamp, error) {
	return timestamppb.New(s.txTime), nil
}

func (s *mockStub) SetEvent(name string, payload []byte) error {
	s.lastEvent = name
	s.lastEventData = payload
	return nil
}

// TransactionContextInterface-Teil
func (s *mockStub) GetStub() shim.ChaincodeStubInterface { return s }

func (s *mockStub) GetClientIdentity() clientIdentityInterface { return s.identity }

// clientIdentityInterface entkoppelt uns vom exakten cid.ClientIdentity
// Interface-Typ, damit der Mock ohne den echten cid-Import kompiliert.
type clientIdentityInterface interface {
	GetID() (string, error)
	GetMSPID() (string, error)
}

// ─── Range Iterator Mock ──────────────────────────────────────────────────

type mockRangeIterator struct {
	keys   []string
	values map[string][]byte
	pos    int
}

func (it *mockRangeIterator) HasNext() bool { return it.pos < len(it.keys) }

func (it *mockRangeIterator) Next() (*queryResultKV, error) {
	if !it.HasNext() {
		return nil, fmt.Errorf("no more items")
	}
	k := it.keys[it.pos]
	it.pos++
	return &queryResultKV{Key: k, Value: it.values[k]}, nil
}

func (it *mockRangeIterator) Close() error { return nil }

// queryResultKV spiegelt die Felder von queryresult.KV, die der Contract
// tatsächlich liest (Key, Value).
type queryResultKV struct {
	Key   string
	Value []byte
}

// ─── History Iterator Mock ────────────────────────────────────────────────

type mockHistoryIterator struct {
	records []historyRecord
	pos     int
}

func (it *mockHistoryIterator) HasNext() bool { return it.pos < len(it.records) }

func (it *mockHistoryIterator) Close() error { return nil }
