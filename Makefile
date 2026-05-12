.PHONY: dev test test-integration test-race generate lint build migrate-up fmt vet

dev:
	docker compose -f infra/docker-compose.yml -f infra/docker-compose.dev.yml \
		--profile core --profile permits up

test:
	go test ./...

test-integration:
	go test ./... -tags integration

test-race:
	go test -race ./...

generate:
	go generate ./...

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
