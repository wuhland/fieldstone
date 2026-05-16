.PHONY: dev test test-integration test-race generate lint build migrate-up fmt vet swagger install-tools

dev:
	docker compose --env-file .env \
		-f infra/docker-compose.yml -f infra/docker-compose.dev.yml \
		--profile core --profile permits up --build --force-recreate

# Rebuild and restart a single service without touching the rest.
# Usage: make rebuild SERVICE=gateway
rebuild:
	@if [ -z "$(SERVICE)" ]; then echo "Usage: SERVICE=gateway make rebuild"; exit 1; fi
	docker compose --env-file .env \
		-f infra/docker-compose.yml -f infra/docker-compose.dev.yml \
		--profile core --profile permits \
		up --build --force-recreate --no-deps $(SERVICE)

test:
	go test ./...

test-integration:
	go test ./... -tags integration

test-race:
	go test -race ./...

generate:
	go generate ./...

# Regenerate the OpenAPI spec from handler annotations.
# Requires swag: run `make install-tools` first.
swagger:
	swag init \
		--generalInfo swag.go \
		--dir "services/gateway,services/permits/handlers,services/requests/handlers,services/records/handlers,services/identity/handlers,services/webhooks/handlers,services/audit/handlers" \
		--output services/gateway/docs \
		--parseDependency
	@rm -f services/gateway/docs/docs.go  # not needed; we embed swagger.json directly
	@echo "Spec written to services/gateway/docs/swagger.json"
	@echo "View at http://localhost:8080/docs after make dev"

install-tools:
	go install github.com/swaggo/swag/cmd/swag@v1.16.3

lint:
	golangci-lint run

build:
	go build ./services/...

migrate-up:
	@echo "Running migrations for all services..."
	@for service in identity permits requests records audit webhooks; do \
		echo "→ $$service"; \
		migrate -path services/$$service/db/migrations \
		        -database "$$(grep $${service^^}_DATABASE_DSN .env | cut -d= -f2-)" up; \
	done

migrate-down:
	@if [ -z "$(SERVICE)" ]; then echo "Usage: SERVICE=permits make migrate-down"; exit 1; fi
	migrate -path services/$(SERVICE)/db/migrations \
	        -database "$$(grep $$(echo $(SERVICE) | tr '[:lower:]' '[:upper:]')_DATABASE_DSN .env | cut -d= -f2-)" down 1

fmt:
	gofmt -w .

vet:
	go vet ./...
