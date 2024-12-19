FROM golang:1.23-alpine

WORKDIR /app

COPY srv/local/go.mod srv/local/go.mod
COPY srv/local/go.sum srv/local/go.sum
COPY common/ common/

WORKDIR /app/srv/local

RUN go mod download

COPY srv/local .

RUN go build -o main

ENTRYPOINT ["./main"]