.PHONY: \
	install \
	fmt \
	lint \
	test \
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

test: ## Run tests locally on the host machine
	go test ./...
	cd e2e && go test ./...

ci: fmt lint test ## Run all CI checks: format, lint, and test
	@echo "All CI checks passed."

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
