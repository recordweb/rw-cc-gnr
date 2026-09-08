module github.com/recordweb/rw-cc-gnr

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20230228194215-b84622ba6a7a
	github.com/hyperledger/fabric-contract-api-go v1.2.1
	github.com/hyperledger/fabric-protos-go v0.3.0
	github.com/stretchr/testify v1.9.0
	google.golang.org/protobuf v1.31.0
)

// ─────────────────────────────────────────────────────────────────────────
// IMPORTANT — do this once, with real internet access, BEFORE the first
// VPS/CI build:
//
//   go mod tidy
//
// The previous go.mod pinned an invalid/non-existent fabric-chaincode-go
// pseudo-version, which only surfaces as an error during `go mod download`
// on a machine with real network access (the development sandbox that
// produced this file could not reach the Go module proxy to verify it —
// that was a mistake, not a deliberate choice). The versions above are
// confirmed-real pseudo-versions taken from working reference projects
// pinning fabric-contract-api-go v1.2.1, but running `go mod tidy` locally
// once is still the correct way to let Go resolve go.sum and every
// indirect dependency (go-openapi/*, xeipuuv/gojsonschema, etc.) itself,
// rather than trusting any hand-maintained version list, including this one.
//
// For mock generation (see namespaceregistry/mocks_gen.go), counterfeiter
// is a dev-tool, not a module dependency:
//
//   go install github.com/maxbrunsfeld/counterfeiter/v6@latest
//   go generate ./...
//
// Note on GetTxTimestamp(): returns *timestamppb.Timestamp
// (google.golang.org/protobuf/types/known/timestamppb), not the older
// github.com/golang/protobuf/ptypes/timestamp — hence the explicit
// google.golang.org/protobuf requirement above.
