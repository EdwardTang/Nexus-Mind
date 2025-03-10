# Nexus-Mind Vector Store Makefile

.PHONY: build test clean

build:
	@echo "Building nexus-mind..."
	@cd src && go build -o ../bin/vectorstore ./vectorstore

test:
	@echo "Running tests..."
	@cd src && go test -v ./vectorstore

test-with-race:
	@echo "Running tests with race detector..."
	@cd src && go test -race -v ./vectorstore

bench:
	@echo "Running benchmarks..."
	@cd src && go test -bench=. ./vectorstore

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@go clean
