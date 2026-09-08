# Using a newer Go toolchain than go.mod's "go 1.21" minimum on purpose:
# counterfeiter (used below to generate test mocks) requires Go >= 1.22
# (v6.11.2+) or even >= 1.25 (latest v6.12.x). go.mod's "go 1.21" is a
# minimum-compatibility declaration, not a pin — building with a newer
# toolchain is safe and does not change the module's declared minimum.
FROM golang:1.23 AS build
WORKDIR /chaincode

COPY . .

# namespaceregistry/mocks/ is intentionally NOT committed to this repo yet
# — it is generated code (counterfeiter fakes of the Fabric interfaces).
# Pinned to v6.11.2 explicitly (not @latest): v6.12.x requires Go >= 1.25.
RUN go install github.com/maxbrunsfeld/counterfeiter/v6@v6.11.2

# Chicken-and-egg problem: mocks_gen.go and the *_test.go files already
# import ".../namespaceregistry/mocks", a package that only exists once
# counterfeiter generates it below — but counterfeiter itself works like a
# type-checking `go build` and needs a fully resolved go.sum (with real
# checksums, not just downloaded modules) to even parse contractapi's
# types. Plain `go mod download` only populates the module cache; it does
# NOT write the full go.sum needed for a consistent build (a known Go
# behaviour, see golang/go#41341). `go mod tidy -e` tolerates the one
# currently-unresolvable import (the not-yet-generated mocks package) and
# still writes correct checksums for everything else, which is enough for
# counterfeiter to successfully typecheck the real Fabric interfaces.
RUN go mod tidy -e
RUN go generate ./...

# Only now, with namespaceregistry/mocks/ physically present, does a full
# `go mod tidy` (no -e needed) see a complete, resolvable import graph.
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode-server .

FROM debian:bookworm-slim
COPY --from=build /chaincode-server /chaincode-server
EXPOSE 9999
CMD ["/chaincode-server"]