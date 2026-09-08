FROM golang:1.21 AS build
WORKDIR /chaincode

# All source is copied BEFORE running `go mod tidy`. Running `go mod tidy`
# against only go.mod (no .go files present yet) finds no imports to
# resolve at all ("warning: all matched no packages") and produces an
# empty/near-empty go.sum, which then fails at the build step further down.
# go.sum is intentionally not committed/relied upon in this repo yet: it
# was hand-written offline and could not be verified against the real Go
# module proxy, so resolving it fresh here (with real internet access)
# is more reliable than trusting it.
COPY . .
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode-server .

FROM debian:bookworm-slim
COPY --from=build /chaincode-server /chaincode-server
EXPOSE 9999
CMD ["/chaincode-server"]