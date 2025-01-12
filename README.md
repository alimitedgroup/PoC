# Distributed warehouse management system

The champagne of reliable distributed systems.

## Services

-   `catalog`: manages the list of items of this warehouse

# Prerequisites

install:

-   https://github.com/casey/just
-   golang
-   docker
-   docker compose
-   natscli

## Commands

### Run

`just up`

### Clean environment

`just down`

### Listen messages

`nats subscribe warehouse_events.public_merce_stock_update_event`
`nats subscribe warehouse_events.public_create_order_event`
