.PHONY: tidy fmt test vet openapi-check infra migrate seed bootstrap-admin api worker down

tidy:
	go mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

test:
	go test ./...

vet:
	go vet ./...

openapi-check:
	ruby -ryaml -e 'document = YAML.safe_load(File.read("openapi/openapi.yaml"), permitted_classes: [], permitted_symbols: [], aliases: true); abort("invalid OpenAPI version") unless document["openapi"] == "3.1.0"; puts "OpenAPI paths: #{document.fetch("paths").size}"'

infra:
	docker compose up -d postgres minio minio-init

migrate:
	docker compose --profile tools run --rm migrate

seed:
	docker compose --profile tools run --rm seed

bootstrap-admin:
	go run ./cmd/bootstrap-admin

api:
	go run ./cmd/api

worker:
	go run ./cmd/worker

down:
	docker compose down
