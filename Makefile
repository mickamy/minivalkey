.PHONY: \
	install \
	fmt \
	lint \
	test \
	test-e2e \
	test-all \
	ci \
	help

install: ## Install dependencies
	go mod download
	cd e2e && go mod download

fmt: ## Format the code using gofmt
	go fmt ./...
	cd e2e && go fmt ./...

lint: ## Run linters (vet and staticcheck)
	go vet ./...
	go tool staticcheck ./...
	cd e2e && go vet ./...

test: ## Run unit tests
	go test ./...

test-e2e: ## Run end-to-end tests
	cd e2e && go test ./...

test-all: test test-e2e ## Run all tests: unit and end-to-end
	@echo "All tests passed."

ci: fmt lint test ## Run all CI checks: format, lint, and test
	@echo "All CI checks passed."

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
