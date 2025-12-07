.PHONY: test lint fmt vet clean coverage

# Run all tests
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean ./...

# Run all checks
check: fmt vet lint test

# Integration tests (requires backends to be installed)
test-integration:
	@echo "Running pass integration tests..."
	VAULTMUX_TEST_PASS=1 go test -v -tags=integration ./...

# Run quick tests (no race detector)
test-quick:
	go test ./...

# Install tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help
help:
	@echo "Available targets:"
	@echo "  test            - Run all tests with race detector"
	@echo "  test-quick      - Run tests without race detector"
	@echo "  test-integration- Run integration tests (requires backend CLIs)"
	@echo "  coverage        - Generate coverage report"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  vet             - Run go vet"
	@echo "  check           - Run all checks (fmt, vet, lint, test)"
	@echo "  clean           - Clean build artifacts"
	@echo "  install-tools   - Install required development tools"
