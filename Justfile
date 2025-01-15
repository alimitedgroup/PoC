set export := true

up:
    docker compose up -d --build

down:
    docker compose down

reset:
    docker compose down -v
    just up

doc:
    go run golang.org/x/pkgsite/cmd/pkgsite@latest -dev

catalog $OTLP_URL="localhost:4317" $DB_URL="postgres://catalog:catalog@localhost:5432/catalog": up
    go run github.com/alimitedgroup/PoC/srv/catalog

orders $OTLP_URL="localhost:4317" $DB_URL="postgres://orders:orders@localhost:5432/orders": up
    go run github.com/alimitedgroup/PoC/srv/orders

warehouse $OTLP_URL="localhost:4317" $DB_URL="postgres://warehouse:warehouse@localhost:5432/warehouse" $WAREHOUSE_ID="42": up
    go run github.com/alimitedgroup/PoC/srv/warehouse

api $OTLP_URL="localhost:4317": up
    go run github.com/alimitedgroup/PoC/srv/api_gateway
