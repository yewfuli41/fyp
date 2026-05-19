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