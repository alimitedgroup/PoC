FROM golang:1.23-alpine

WORKDIR /app

COPY srv/central/go.mod srv/central/go.mod
COPY srv/central/go.sum srv/central/go.sum
COPY common/ common/

WORKDIR /app/srv/central

RUN go mod download

COPY srv/central/ .

RUN go build -o main main.go

ENTRYPOINT ["./main"]