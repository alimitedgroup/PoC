# Distributed warehouse management system

The champagne of reliable distributed systems.

## Services

- `warehouse`: local service
- `catalog`: manages the list of items
- `order`: manage order from multiple warehouses
- `api-gateway`: HTTP API layer to comunicate with other services

# Prerequisites

install:

- [just](https://github.com/casey/just)
- golang
- docker
- docker compose
- natscli

## Commands

### Run

`just up`

### Clean environment

`just down`

### Examples

```sh
just reset

curl -X POST localhost:80/catalog -H "Content-Type: application/json" -d '{"name": "hat"}'
HAT_ID=
curl localhost:80/catalog
curl -X POST localhost:80/stock/41 -H "Content-Type: application/json" -d '[{"good_id": "'$HAT_ID'", "amount": 20}]'
curl localhost:80/warehouses
curl localhost:80/stock/41
curl -X POST localhost:80/orders -H "Content-Type: application/json" -d '{"items":[{"good_id": "'$HAT_ID'", "amount": 5}]}'
curl localhost:80/stock/41
curl localhost:80/orders
```
