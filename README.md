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

nats request catalog.create '{"name": "hat"}'
nats request catalog.list ""
HAT_ID={{hat_good_id_here}}
nats request "warehouse.add_stock.41" '[{"good_id": "'"$HAT_ID"'", "amount": 10}]'
curl localhost:80/warehouses
curl localhost:80/stock/41
nats request "order.create" '{"items":[{"good_id": "'"$HAT_ID"'", "amount": 5}]}'
curl localhost:80/stock/41
```
