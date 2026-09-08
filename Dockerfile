# Using a newer Go toolchain than go.mod's "go 1.21" minimum on purpose:
# counterfeiter (used below to generate test mocks) requires Go >= 1.22
# (v6.11.2+) or even >= 1.25 (latest v6.12.x). go.mod's "go 1.21" is a
# minimum-compatibility declaration, not a pin — building with a newer
# toolchain is safe and does not change the module's declared minimum.
FROM golang:1.23 AS build
WORKDIR /chaincode

# All source is copied BEFORE running `go mod tidy`. Running `go mod tidy`
# against only go.mod (no .go files present yet) finds no imports to
# resolve at all ("warning: all matched no packages") and produces an
# empty/near-empty go.sum, which then fails at the build step further down.
COPY . .

# namespaceregistry/mocks/ is intentionally NOT committed to this repo yet
# — it is generated code (counterfeiter fakes of the Fabric interfaces).
# Pinned to v6.11.2 explicitly (not @latest): v6.12.x requires Go >= 1.25,
# and pinning avoids silently picking up a future version with a new,
# higher minimum-Go requirement again.
RUN go install github.com/maxbrunsfeld/counterfeiter/v6@v6.11.2

# `go generate` must run BEFORE `go mod tidy`. mocks_gen.go and the *_test.go
# files already import ".../namespaceregistry/mocks", a package that only
# exists once counterfeiter has actually generated it below. If `go mod
# tidy` runs first, it sees that import, finds no such local package yet,
# and tries (and fails) to resolve it as an external module instead
# ("no matching versions for query \"latest\"").
#
# `go generate` itself needs enough of the module graph resolved to
# typecheck mocks_gen.go's source interfaces (shim.ChaincodeStubInterface,
# cid.ClientIdentity, contractapi.TransactionContextInterface) — `go mod
# download` (not `tidy`) fetches exactly the modules already listed in
# go.mod without trying to resolve anything unresolvable yet.
RUN go mod download
RUN go generate ./...

# Only now, with namespaceregistry/mocks/ physically present, does `go mod
# tidy` see a complete, resolvable import graph.
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode-server .

FROM debian:bookworm-slim
COPY --from=build /chaincode-server /chaincode-server
EXPOSE 9999
CMD ["/chaincode-server"]