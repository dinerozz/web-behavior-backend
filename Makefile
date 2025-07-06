.PHONY: run serve migrate migrate-down seed seed-down build test clean

BINARY_NAME=web-behavior
BINARY_PATH=bin/$(BINARY_NAME)

run: serve

serve:
	@echo "Starting server..."
	go run cmd/main.go serve

migrate:
	@echo "Running migrations..."
	go run cmd/main.go migrate

migrate-down:
	@echo "Rolling back migrations..."
	go run cmd/main.go migrate -d

build:
	@echo "Building the project..."
	@go build -o $(BINARY_PATH) cmd/main.go

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@go clean

test:
	@echo "Running tests..."
	@go test ./... -v

setup: build migrate seed

reset: clean build migrate-down