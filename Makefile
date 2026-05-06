.PHONY: fmt test run infra-up infra-down migrate

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

run:
	go run ./cmd/eventlog-service -config ./config.example.yaml

infra-up:
	cd deployments/docker-compose && docker compose up -d

infra-down:
	cd deployments/docker-compose && docker compose down

migrate:
	docker exec -i onec-eventlog-clickhouse clickhouse-client --multiquery < migrations/clickhouse/001_eventlog_events.sql
