.DEFAULT_GOAL := help

deploy: ## Deploy the function to GCP production environment
	./deploy.sh send-email

deploy_staging: ## Deploy the function to GCP staging environment
	./deploy.sh staging-send-email

binaries: ## Compile server binaries for local environment
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./_example/server-linux-amd64 ./_example
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./_example/server-linux-arm64 ./_example
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./_example/server-darwin-amd64 ./_example
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./_example/server-darwin-arm64 ./_example
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./_example/server-windows-amd64.exe ./_example

tools: ## Install tools required for development
	go install github.com/vektra/mockery/v2

mocks: ## Generate mocks for testing
	go generate -x -run mockery ./...

help: ## Show help
	@grep -h '\s##\s' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: deploy deploy_staging binaries tools mocks help
