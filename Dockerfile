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
# — it is generated code (counterfeiter fakes of the Fabric interfaces),
# and is generated here, in the one environment in this whole pipeline that
# actually has real internet access and permission to install a Go tool.
# Pinned to v6.11.2 explicitly (not @latest): v6.12.x requires Go >= 1.25,
# and pinning avoids silently picking up a future version with a new,
# higher minimum-Go requirement again.
RUN go install github.com/maxbrunsfeld/counterfeiter/v6@v6.11.2
RUN go mod tidy
RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode-server .

FROM debian:bookworm-slim
COPY --from=build /chaincode-server /chaincode-server
EXPOSE 9999
CMD ["/chaincode-server"]