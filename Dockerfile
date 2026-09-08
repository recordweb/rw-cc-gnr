FROM golang:1.21 AS build
WORKDIR /chaincode

# All source is copied BEFORE running `go mod tidy`. Running `go mod tidy`
# against only go.mod (no .go files present yet) finds no imports to
# resolve at all ("warning: all matched no packages") and produces an
# empty/near-empty go.sum, which then fails at the build step further down.
COPY . .

# namespaceregistry/mocks/ is intentionally NOT committed to this repo yet
# — it is generated code (counterfeiter fakes of the Fabric interfaces),
# and is generated here, in the one environment in this whole pipeline that
# actually has real internet access and permission to install a Go tool
# (neither the VPS shell user nor the sandbox that authored these files
# has both). Without this step, `go mod tidy` below tries to resolve
# ".../namespaceregistry/mocks" as if it were a real external module and
# fails with "no matching versions for query \"latest\"".
RUN go install github.com/maxbrunsfeld/counterfeiter/v6@latest
RUN go mod tidy
RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode-server .

FROM debian:bookworm-slim
COPY --from=build /chaincode-server /chaincode-server
EXPOSE 9999
CMD ["/chaincode-server"]