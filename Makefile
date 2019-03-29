build: ## Builds the service broker
	go build -i github.com/wso2/service-broker-apim/cmd/servicebroker

test: ## Runs the tests
	go test -v ./pkg/utils/*

linux: ## Builds a Linux executable
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -o servicebroker-linux github.com/wso2/service-broker-apim/cmd/servicebroker

darwin: ## Builds a Darwin executable
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 \
	go build -o servicebroker-darwin github.com/wso2/service-broker-apim/cmd/servicebroker

clean: ## Cleans up build artifacts
	rm -f servicebroker
	rm -f servicebroker-linux
	rm -f servicebroker-darwin

setup: ## Install golint
	go get -u golang.org/x/lint/golint

lint: ## Run golint on the code
	golint  pkg/* cmd/*

format:
	gofmt -w pkg/* cmd/*

help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

