.PHONY: build run test clean

# Build the application
build:
	cd src && go build -o ../bin/nexus-mind

# Run the application
run: build
	./bin/nexus-mind

# Run all tests
test:
	cd src && go test -v ./...

# Run tests with race detection
test-race:
	cd src && go test -race -v ./...

# Run benchmarks
bench:
	cd src && go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f src/nexus-mind

# Create directories if they don't exist
dirs:
	mkdir -p bin
	mkdir -p data

# Run tests for a specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-pkg PKG=vector/index"; \
		exit 1; \
	fi
	cd src && go test -v ./$(PKG)

# Format code
fmt:
	cd src && go fmt ./...

# Check for lint issues
lint:
	cd src && go vet ./...

# Initialize the project
init: dirs

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the application"
	@echo "  run        - Run the application"
	@echo "  test       - Run all tests"
	@echo "  test-race  - Run tests with race detection"
	@echo "  bench      - Run benchmarks"
	@echo "  clean      - Clean build artifacts"
	@echo "  dirs       - Create necessary directories"
	@echo "  test-pkg   - Run tests for a specific package (e.g., make test-pkg PKG=vector/index)"
	@echo "  fmt        - Format code"
	@echo "  lint       - Check for lint issues"
	@echo "  init       - Initialize the project"
