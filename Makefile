dev: ## Run the app in dev mode
	@go run . --dev
.PHONY: dev

build: dist/app ## Build the app

dist/app: go.mod go.sum sekki.json $(shell find . -type f -name '*.go')
	@go build -tags netgo -ldflags '-s -w' -o $@

format: ## Format the code
	@go fmt ./...
.PHONY: format

lint: ## Lint the code
	@go vet ./...
.PHONY: lint

help: ## Show this help
	@echo "\nSpecify a command. The choices are:\n"
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[0;36m%-12s\033[m %s\n", $$1, $$2}'
	@echo ""
.PHONY: help
