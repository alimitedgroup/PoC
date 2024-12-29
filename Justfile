set export

OTLP_URL := "localhost:4317"
DB_URL := "postgres://catalog:catalog@localhost:5432/catalog"

up:
    docker compose up -d --build

down:
    docker compose down

reset:
    docker compose down -v

catalog: up
    go run github.com/alimitedgroup/palestra_poc/srv/catalog

warehouse: up
    go run github.com/alimitedgroup/palestra_poc/srv/warehouse