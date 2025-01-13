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

order $OTLP_URL="localhost:4317" $DB_URL="postgres://order:order@localhost:5432/order": up
    go run github.com/alimitedgroup/PoC/srv/order

warehouse $OTLP_URL="localhost:4317" $DB_URL="postgres://warehouse:warehouse@localhost:5432/warehouse" $WAREHOUSE_ID="42": up
    go run github.com/alimitedgroup/PoC/srv/warehouse
