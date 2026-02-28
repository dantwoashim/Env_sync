.PHONY: build test test-race lint vet cover release-snapshot clean

BINARY=envsync

## Build the envsync binary
build:
	go build -o $(BINARY) .

## Run all tests
test:
	go test ./... -count=1 -timeout=120s

## Run tests with race detector
test-race:
	go test ./... -race -count=1 -timeout=120s

## Run go vet
vet:
	go vet ./...

## Run golangci-lint (must be installed)
lint:
	golangci-lint run ./...

## Generate test coverage report
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## GoReleaser dry-run (snapshot)
release-snapshot:
	goreleaser --snapshot --clean

## Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY).exe coverage.out coverage.html
	rm -rf dist/

## Show help
help:
	@echo "Available targets:"
	@echo "  build            Build the envsync binary"
	@echo "  test             Run all tests"
	@echo "  test-race        Run tests with race detector"
	@echo "  vet              Run go vet"
	@echo "  lint             Run golangci-lint"
	@echo "  cover            Generate test coverage report"
	@echo "  release-snapshot GoReleaser dry-run"
	@echo "  clean            Clean build artifacts"
