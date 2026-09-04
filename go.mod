module github.com/recordweb/rw-cc-gnr

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20240124143825-7dd60bcf7e07
	github.com/hyperledger/fabric-contract-api-go v1.2.1
	github.com/hyperledger/fabric-protos-go v0.3.2
	github.com/stretchr/testify v1.9.0
	google.golang.org/protobuf v1.31.0
)

// ─────────────────────────────────────────────────────────────────────────
// Setup nach dem ersten Checkout:
//
//   go mod tidy
//
// Das ergänzt go.sum sowie alle weiteren indirekten Abhängigkeiten
// automatisch (u.a. go-openapi/*, xeipuuv/gojsonschema für die
// JSON-Schema-Validierung der Contract-Metadaten durch fabric-contract-api-go).
//
// Für die Mock-Generierung (siehe namespaceregistry/mocks_gen.go) wird
// zusätzlich counterfeiter als Dev-Tool benötigt, aber NICHT als
// Modul-Dependency, da es nur zur Codegenerierung läuft:
//
//   go install github.com/maxbrunsfeld/counterfeiter/v6@latest
//   go generate ./...
//
// Hinweis zu GetTxTimestamp(): gibt *timestamppb.Timestamp
// (google.golang.org/protobuf/types/known/timestamppb) zurück, nicht das
// ältere github.com/golang/protobuf/ptypes/timestamp — daher die
// explizite Angabe von google.golang.org/protobuf hier.
