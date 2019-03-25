build: ## Builds the service broker
	go build -i github.com/wso2/service-broker-apim/cmd/main

test: ## Runs the tests
	go test -v $(shell go list ./... | grep -v /vendor/ | grep -v /test/)

linux: ## Builds a Linux executable
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -o servicebroker-linux --ldflags="-s" github.com/wso2/service-broker-apim/cmd/main

darwin: ## Builds a Linux executable
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 \
	go build -o servicebroker-darwin --ldflags="-s" github.com/wso2/service-broker-apim/cmd/main

clean: ## Cleans up build artifacts
	rm -f servicebroker
	rm -f servicebroker-linux
	rm -f servicebroker-darwin

help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

