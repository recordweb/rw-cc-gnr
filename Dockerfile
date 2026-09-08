FROM golang:1.21 AS build
WORKDIR /chaincode
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode-server .

FROM debian:bookworm-slim
COPY --from=build /chaincode-server /chaincode-server
EXPOSE 9999
CMD ["/chaincode-server"]