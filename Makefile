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
	kubectl apply -k k8s/infisical
	kubectl apply -k k8s/overlays/prod

.PHONY: infisical-install
infisical-install:
	helm repo add infisical-helm-charts 'https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/'
	helm repo update
	helm upgrade --install infisical-secrets-operator infisical-helm-charts/secrets-operator \
	  --namespace infisical-operator-system \
	  --create-namespace

.PHONY: infisical-apply
infisical-apply:
	kubectl apply -k k8s/infisical

.PHONY: infisical-status
infisical-status:
	kubectl get infisicalsecrets.secrets.infisical.com -A
	kubectl get secret -n $(NAMESPACE) gridlogger-secrets

.PHONY: migrate-local
migrate-local:
	DATABASE_URL="postgres://grid:grid@localhost:5432/gridlogger?sslmode=disable" sh scripts/migrate.sh

.PHONY: secrets-setup
secrets-setup:
	kubectl create secret generic app-secrets --from-env-file=.env