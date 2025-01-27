set export := true

build:
    docker compose pull
    docker compose build

up:
    docker compose up -d --build

down:
    docker compose down

reset:
    docker compose down -v --remove-orphans
    just up

doc:
    go run golang.org/x/pkgsite/cmd/pkgsite@latest -dev

catalog $OTLP_URL="localhost:4317" $DB_URL="postgres://catalog:catalog@catalog-postgres:5432/catalog" $NATS_URL="nats://nats:4222": up
    go run github.com/alimitedgroup/PoC/srv/catalog

order $OTLP_URL="localhost:4317" $NATS_URL="nats://nats:4222": up
    go run github.com/alimitedgroup/PoC/srv/order

warehouse $OTLP_URL="localhost:4317" $NATS_URL="nats://nats:4222" $WAREHOUSE_ID="42": up
    go run github.com/alimitedgroup/PoC/srv/warehouse

api $OTLP_URL="localhost:4317": up
    go run github.com/alimitedgroup/PoC/srv/api_gateway

cli *args:
    go run github.com/alimitedgroup/PoC/cli {{args}}