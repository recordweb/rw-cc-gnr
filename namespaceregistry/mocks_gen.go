package namespaceregistry

import (
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ─────────────────────────────────────────────────────────────────────────
// Mock-Generierung für Unit-Tests.
//
// Dieses Projekt folgt demselben Testmuster wie die offiziellen
// hyperledger/fabric-samples (z.B. asset-transfer-basic/chaincode-go):
// Mocks werden NICHT von Hand geschrieben, sondern per `counterfeiter`
// aus den echten Interfaces generiert. Das garantiert, dass die Mocks
// exakt zur tatsächlich installierten Version von fabric-chaincode-go /
// fabric-contract-api-go passen, statt aus dem Gedächtnis nachgebaute
// Interface-Signaturen zu riskieren.
//
// Einmalig nach `go mod tidy` ausführen:
//
//	go install github.com/maxbrunsfeld/counterfeiter/v6@latest
//	go generate ./...
//
// Das erzeugt die Dateien im Unterverzeichnis mocks/:
//   mocks/chaincode_stub.go   (Fake-Implementierung von ChaincodeStubInterface)
//   mocks/transaction_context.go (Fake von contractapi.TransactionContextInterface)
//   mocks/client_identity.go  (Fake von cid.ClientIdentity)
//   mocks/state_query_iterator.go
//   mocks/history_query_iterator.go
//
// Diese generierten Dateien werden NICHT händisch editiert und sollten
// (wie bei fabric-samples üblich) mit ins Repo committet werden, damit
// `go test ./...` ohne vorherige counterfeiter-Installation läuft.
// ─────────────────────────────────────────────────────────────────────────

//go:generate counterfeiter -o mocks/chaincode_stub.go -fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

//go:generate counterfeiter -o mocks/transaction_context.go -fake-name TransactionContext . transactionContext
type transactionContext interface {
	contractapi.TransactionContextInterface
}

//go:generate counterfeiter -o mocks/client_identity.go -fake-name ClientIdentity . clientIdentity
type clientIdentity interface {
	cid.ClientIdentity
}

//go:generate counterfeiter -o mocks/state_query_iterator.go -fake-name StateQueryIterator . stateQueryIterator
type stateQueryIterator interface {
	shim.StateQueryIteratorInterface
}

//go:generate counterfeiter -o mocks/history_query_iterator.go -fake-name HistoryQueryIterator . historyQueryIterator
type historyQueryIterator interface {
	shim.HistoryQueryIteratorInterface
}
