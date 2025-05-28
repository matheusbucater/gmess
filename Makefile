MIGRATE=
MIGRATION=

build:
	go build -o bin/gmess cmd/main.go

migration $(MIGRATION):
	~/go/bin/migrate create -ext sql -dir internal/db/migrations -seq $(MIGRATION)

migrate $(MIGRATE):
	sqlite3 data/messages.db < $(MIGRATE)

generate:
	~/go/bin/sqlc generate
