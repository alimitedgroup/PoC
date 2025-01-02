set export

up:
    docker compose up -d --build

down:
    docker compose down

reset:
    docker compose down -v
    just up

doc:
    go run golang.org/x/pkgsite/cmd/pkgsite@latest -dev

catalog \
    $OTLP_URL="localhost:4317" \
    $DB_URL="postgres://catalog:catalog@localhost:5432/catalog": up
    go run github.com/alimitedgroup/palestra_poc/srv/catalog

warehouse \
    $OTLP_URL="localhost:4317" \
    $DB_URL="postgres://warehouse:warehouse@localhost:5432/warehouse" \
    $WAREHOUSE_ID="42": up
    go run github.com/alimitedgroup/palestra_poc/srv/warehouse