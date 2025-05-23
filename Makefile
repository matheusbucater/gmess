build:
	go build -o bin/gry cmd/main.go

generate:
	~/go/bin/sqlc generate
