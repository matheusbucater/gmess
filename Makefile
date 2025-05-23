MIGRATE =
MIGRATION =

build:
	go build -o bin/gry cmd/main.go

migration:
	~/go/bin/migrate create -ext sql -dir internal/db/migrations -seq $(MIGRATION)

migrate:
	sqlite3 data/messages.db < $(MIGRATE)

generate:
	~/go/bin/sqlc generate
