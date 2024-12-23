FROM golang:alpine AS builder

ARG SERVICE

WORKDIR /src

RUN apk add --no-cache git
RUN go env -w GOMODCACHE=/root/.cache/go-build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    go mod download

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    --mount=target=. \
    go build -buildvcs=true -ldflags="-s -w" -o /out/service ./srv/$SERVICE

FROM alpine

COPY --from=builder /out/service /service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /data

ENTRYPOINT ["/service"]