APP_IMAGE ?= ghcr.io/your-user/gridlogger-backend
FRONTEND_IMAGE ?= ghcr.io/your-user/gridlogger-frontend
FIRMWARE_IMAGE ?= ghcr.io/your-user/gridlogger-firmware
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

.PHONY: next-version
next-version:
	sh scripts/next-version.sh

.PHONY: rollout-version
rollout-version:
	test -n "$(VERSION)" || (echo "VERSION is required, e.g. make rollout-version VERSION=1.2.3"; exit 1)
	K8S_NAMESPACE="$(NAMESPACE)" BACKEND_IMAGE="$(APP_IMAGE)" FRONTEND_IMAGE="$(FRONTEND_IMAGE)" FIRMWARE_IMAGE="$(FIRMWARE_IMAGE)" \
		sh scripts/rollout-version.sh "$(VERSION)" $(if $(filter true,$(WITH_FIRMWARE)),--with-firmware,)
