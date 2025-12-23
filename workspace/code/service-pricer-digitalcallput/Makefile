.PHONY: test

MODULE=github.com/regentmarkets/service-pricer-digitalcallput

help: ## Display this help message
	@echo "Usage: make [target]"
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

lint-proto: ## Lints protobuf files to ensure they meet the project's standards.
	buf lint proto

grpc-generate: ## Generates Go code from protobuf files using buf.
	buf generate

grpc-dep-update: ## Updates the dependencies in the buf.lock file.
	buf dep update

test: ## Runs all Go tests in the project with verbose output.
	go test -v -race -coverpkg ./... -coverprofile cover.out ./...
	# exclude api and cmd from the coverage
	grep -Ev '${MODULE}/(api|cmd)' <cover.out >cover-filtered.out
	go tool cover -func cover-filtered.out | awk 'END { print "Test coverage "$$3; if($$3+0<80) { print "Coverage is not sufficient"; exit 1 } }'

lint: ## Lints all Go files in the project to ensure they meet the project's standards.
	golangci-lint run ./...
