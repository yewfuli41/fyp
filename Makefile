include .env.local
export

dev:
	go run .

generate:
	go tool gqlgen generate

migrate:
	PGPASSWORD=$(DB_PASSWORD) psql -U $(DB_USER) \
	-h $(DB_HOST) \
	-d $(DB_NAME) \
	-p $(DB_PORT) \
	-f database/schema.sql
	PGPASSWORD=$(DB_PASSWORD) psql -U $(DB_USER) \
	-h $(DB_HOST) \
	-d $(DB_NAME) \
	-p $(DB_PORT) \
	-f database/db_user.sql

test:
	go test -coverprofile=coverage.out -v ./...
	go tool cover -html=coverage.out -o coverage.html
	@if command -v open >/dev/null 2>&1; then \
		open coverage.html; \
	elif command -v explorer.exe >/dev/null 2>&1; then \
		explorer.exe coverage.html; \
	fi

mock:
	mockery
.PHONY: database
database:
	psql -U $(DB_USER) -d fyp -p 5433

database-wsl:
	sudo -u postgres psql -d fyp -p 5433