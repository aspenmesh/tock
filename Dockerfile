FROM golang:1.10
WORKDIR /go/src/github.com/aspenmesh/tock

COPY . .

RUN go test ./...
