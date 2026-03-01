APP_IMAGE ?= ghcr.io/your-user/gridlogger-backend:latest
FRONTEND_IMAGE ?= ghcr.io/your-user/gridlogger-frontend:latest
NAMESPACE ?= gridlogger

.PHONY: test

test:
	go test ./...

.PHONY: run
run:
	docker compose up --build

.PHONY: loop
loop:
	while true; do \
		curl --location --request POST 'http://localhost:8080/api/projects/1/ping'; \
		sleep 10; \
	done

.PHONY: migrate-local
migrate-local:
	DATABASE_URL="postgres://grid:grid@localhost:5432/gridlogger?sslmode=disable" sh scripts/migrate.sh

