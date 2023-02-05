dev: ## Run the app in dev mode
	@go run . --dev
.PHONY: dev

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
