.PHONY: build test lint run-infra down
build:
	go build -o bin/ ./cmd/...
test:
	go test -race -count=1 ./...
lint:
	golangci-lint run
run-infra:
	docker compose up -d redis kafka dynamodb prometheus grafana
down:
	docker compose down -v
