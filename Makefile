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

.PHONY: deploy
deploy:
	kubectl apply -f k8s/traefik/traefik-helmchartconfig.yaml
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl -n $(NAMESPACE) create secret generic gridlogger-secrets \
	  --from-literal=POSTGRES_USER="USERNAME" \
	  --from-literal=POSTGRES_PASSWORD="CHANGE_ME" \
	  --from-literal=POSTGRES_DB="gridlogger" \
	  --from-literal=DATABASE_URL="postgres://grid:CHANGE_ME@timescaledb.$(NAMESPACE).svc.cluster.local:5432/gridlogger?sslmode=disable" \
	  --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -k k8s/overlays/prod

.PHONY: migrate-local
migrate-local:
	DATABASE_URL="postgres://grid:grid@localhost:5432/gridlogger?sslmode=disable" sh scripts/migrate.sh

.PHONY: secrets-setup
secrets-setup:
	kubectl create secret generic app-secrets --from-env-file=.env