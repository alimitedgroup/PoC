## Run

`docker compose up -d --build`

## Clean environment

`docker compose down --volumes`

## Listen messages

`nats subscribe warehouse_events.public_merce_stock_update_event`
`nats subscribe warehouse_events.public_create_order_event`
