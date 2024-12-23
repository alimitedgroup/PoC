set export

OTLP_URL := "localhost:4317"
DB_URL := "postgres://catalog:catalog@localhost:5432/catalog"

compose:
    docker compose up -d --build

catalog: compose
    go run github.com/alimitedgroup/palestra_poc/srv/catalog