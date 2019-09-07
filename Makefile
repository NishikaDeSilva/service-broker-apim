
all: deps tests build

build: ## Builds the service broker
	go build -i github.com/wso2/service-broker-apim/cmd/servicebroker

tests: ## Runs the tests
	go test -v ./pkg/...

integration-test-start:
	./test/run-tests.sh

integration-test-setup-down:
	docker-compose -f ./test/integration-test-setup.yaml down

debug-setup-up:
	docker-compose -f ./test/debug-setup.yaml up -d

debug-setup-down:
	docker-compose -f ./test/debug-setup.yaml down

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

deps:
	dep ensure

setup-lint: ## Install golint
	go get -u golang.org/x/lint/golint

lint: ## Run golint on the code
	golint  pkg/* cmd/*

format: ## Run gofmt on the code
	gofmt -w pkg/* cmd/*

help: ## Shows the help
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
        awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

