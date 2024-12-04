## Run
`docker compose up -d --build`

## Listen messages
`nats subscribe warehouse_events.public_merce_event`
`nats subscribe warehouse_events.public_order_event`
