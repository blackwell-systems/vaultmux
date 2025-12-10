# Testing Guide

This document describes the testing strategy for vaultmux, including how to test backends locally without requiring cloud credentials.

## Testing Philosophy

Vaultmux uses a multi-tiered testing approach:

1. **Unit Tests** - Test individual functions and methods in isolation
2. **Integration Tests** - Test complete backend workflows against real or mock services
3. **Interface Compliance** - Verify all backends implement the `Backend` interface correctly

## Quick Start

```bash
# Run all tests (unit tests only, integration tests skip)
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific backend tests
go test -v ./backends/bitwarden/
go test -v ./backends/awssecrets/
```

## Backend Testing Strategies

### CLI-Based Backends (Bitwarden, 1Password, pass)

These backends delegate to external CLI tools and are tested in two ways:

**Unit Tests** (always run):
- Test configuration parsing
- Test error handling
- Test command construction
- Mock session behavior

**Integration Tests** (require CLI installed):
- Skip automatically if CLI not found
- Test against real CLI tools
- Require valid credentials/setup

Example for Bitwarden:
```bash
# Unit tests (no bw CLI required)
go test ./backends/bitwarden/

# Integration tests (requires bw CLI + login)
bw login
BW_TEST_INTEGRATION=1 go test -v ./backends/bitwarden/
```

### SDK-Based Backends (AWS, GCP, Azure)

SDK-based backends use official cloud SDKs and support both local mock testing and real cloud testing.

#### AWS Secrets Manager + LocalStack

LocalStack provides a local AWS environment for testing without AWS credentials or costs.

**Setup LocalStack:**

```bash
# Start LocalStack with Secrets Manager
docker run -d --rm \
  -p 4566:4566 \
  -e SERVICES=secretsmanager \
  --name vaultmux-localstack \
  localstack/localstack

# Wait for ready
sleep 10
curl http://localhost:4566/_localstack/health
```

**Run Integration Tests:**

```bash
# Point AWS SDK to LocalStack
LOCALSTACK_ENDPOINT=http://localhost:4566 \
AWS_ACCESS_KEY_ID=test \
AWS_SECRET_ACCESS_KEY=test \
AWS_REGION=us-east-1 \
go test -v ./backends/awssecrets/

# Expected output:
# PASS: TestIntegration (3.03s)
# PASS: TestIntegration_Pagination (0.08s)
# ok    github.com/blackwell-systems/vaultmux/backends/awssecrets    3.133s
```

**Cleanup:**

```bash
docker stop vaultmux-localstack
```

**What Gets Tested:**

- ✅ Backend initialization and connection
- ✅ Authentication and session management
- ✅ CRUD operations (Create, Get, Update, Delete)
- ✅ Secret listing with prefix filtering
- ✅ Pagination handling
- ✅ Error handling (ErrNotFound, ErrAlreadyExists)
- ✅ Secret name prefixing
- ✅ Metadata handling

**Benefits of LocalStack:**

- No AWS credentials required
- No AWS costs
- Fast test execution (3-5 seconds)
- Deterministic results
- Works in CI/CD pipelines
- Tests real AWS SDK code paths

#### GCP Secret Manager + Emulator

GCP Secret Manager testing uses the [gcp-secret-manager-emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator), a standalone package that implements the gRPC Secret Manager API.

**Setup GCP Secret Manager Emulator:**

```bash
# Option 1: Install and run binary
go install github.com/blackwell-systems/gcp-secret-manager-emulator/cmd/server@latest
server --port 9090

# Option 2: Use Docker (multi-arch: amd64, arm64)
docker run -d -p 9090:9090 --name gcp-emulator \
  ghcr.io/blackwell-systems/gcp-secret-manager-emulator:latest

# Option 3: Docker from Docker Hub (when available)
docker run -d -p 9090:9090 --name gcp-emulator \
  blackwellsystems/gcp-secret-manager-emulator:latest
```

**Run Integration Tests:**

```bash
# Point GCP SDK to mock server
GCP_MOCK_ENDPOINT=localhost:9090 \
go test -v ./backends/gcpsecrets/

# Expected output:
# PASS: TestIntegration (0.01s)
# PASS: TestIntegration_Pagination (0.01s)
# ok    github.com/blackwell-systems/vaultmux/backends/gcpsecrets    0.034s
```

**Cleanup:**

```bash
# If using Docker
docker stop gcp-emulator

# If using binary, Ctrl+C or:
pkill server
```

**What Gets Tested:**

- ✅ Backend initialization and connection
- ✅ Authentication (mock mode, no real credentials)
- ✅ CRUD operations (Create, Get, Update, Delete)
- ✅ Secret listing
- ✅ Pagination handling
- ✅ Error handling (ErrNotFound, ErrAlreadyExists)
- ✅ Version management (latest alias resolution)
- ✅ Metadata handling

**Benefits of Emulator:**

- No GCP credentials required
- No GCP costs
- Fast test execution (<1 second)
- Deterministic results
- Works in CI/CD pipelines
- Tests real gRPC SDK code paths
- Standalone package - reusable across projects
- Multi-architecture Docker images (amd64, arm64)
- Actively maintained with comprehensive test coverage

**Integration Tests with Real GCP** (optional):

If you want to test against real GCP Secret Manager:

```bash
# Set GCP credentials
export GCP_PROJECT_ID=your-project-id
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Run integration tests (will use real GCP, not mock)
go test -v ./backends/gcpsecrets/
```

#### Azure Key Vault

Azure tests use the Azure SDK without a local emulator:

**Unit Tests** (always run):
```bash
go test ./backends/azurekeyvault/
```

**Integration Tests** (requires Azure subscription):
```bash
# Set Azure credentials
export AZURE_VAULT_URL=https://your-vault.vault.azure.net/

# Azure CLI authentication (or other methods)
az login

# Run integration tests
go test -v ./backends/azurekeyvault/
```

### Windows Credential Manager

Windows-specific backend tested via build tags:

**On Windows:**
```bash
go test ./backends/wincred/
```

**On Unix (skips with graceful error):**
```bash
go test ./backends/wincred/
# Output: wincred_unix.go implementation returns ErrBackendNotInstalled
```

## Test Organization

### File Naming Convention

- `*_test.go` - Unit tests (always run)
- `*_integration_test.go` - Integration tests (conditional skip)

### Test Skipping Pattern

Integration tests check for required environment variables and skip gracefully:

```go
func TestIntegration(t *testing.T) {
    endpoint := os.Getenv("LOCALSTACK_ENDPOINT")
    if endpoint == "" {
        t.Skip("LOCALSTACK_ENDPOINT not set - skipping integration tests")
    }
    // ... test implementation
}
```

## CI/CD Testing

### GitHub Actions

The CI pipeline runs:

1. **Unit Tests** - All backends, all platforms (Ubuntu, macOS, Windows)
2. **Integration Tests (AWS)** - Dedicated job with LocalStack service container
3. **Integration Tests (GCP)** - Dedicated job with custom mock server
4. **Linting** - golangci-lint with 5-minute timeout
5. **Build Verification** - Ensures all packages compile
6. **Format Check** - Validates gofmt compliance
7. **Go Vet** - Static analysis for common errors
8. **Coverage** - Reports to Codecov (unit + AWS + GCP integration)

**AWS Integration Test Job**:

The CI workflow includes a dedicated `integration-aws` job that runs AWS backend tests against LocalStack:

```yaml
integration-aws:
  name: AWS Integration Tests (LocalStack)
  runs-on: ubuntu-latest
  services:
    localstack:
      image: localstack/localstack:latest
      ports:
        - 4566:4566
      env:
        SERVICES: secretsmanager
      options: >-
        --health-cmd "curl -f http://localhost:4566/_localstack/health || exit 1"
        --health-interval 10s
        --health-timeout 5s
        --health-retries 5
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.23'

    - name: Wait for LocalStack
      run: |
        timeout 60 bash -c 'until curl -f http://localhost:4566/_localstack/health 2>/dev/null | grep -q "secretsmanager.*available"; do sleep 2; done'

    - name: Run AWS integration tests
      env:
        LOCALSTACK_ENDPOINT: http://localhost:4566
        AWS_ACCESS_KEY_ID: test
        AWS_SECRET_ACCESS_KEY: test
        AWS_REGION: us-east-1
      run: |
        go test -v -race -coverprofile=coverage-aws.out ./backends/awssecrets/
        go tool cover -func=coverage-aws.out | grep total

    - name: Upload AWS coverage
      uses: codecov/codecov-action@v4
      with:
        file: ./coverage-aws.out
        flags: integration-aws
```

**Coverage Impact**: AWS backend coverage increased from 23.7% to 79.1% with LocalStack integration tests running in CI.

**GCP Integration Test Job**:

The CI workflow includes a dedicated `integration-gcp` job that runs GCP backend tests against the standalone GCP Secret Manager emulator:

```yaml
integration-gcp:
  name: GCP Integration Tests (Emulator)
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'

    - name: Install GCP Secret Manager emulator
      run: |
        echo "Installing GCP Secret Manager emulator..."
        go install github.com/blackwell-systems/gcp-secret-manager-emulator/cmd/server@v0.1.0
        mv ~/go/bin/server ./gcp-mock
        echo "Installed successfully"

    - name: Start GCP emulator
      run: |
        ./gcp-mock --port 9090 > gcp-mock.log 2>&1 &
        echo $! > gcp-mock.pid
        sleep 2

    - name: Run GCP integration tests
      env:
        GCP_MOCK_ENDPOINT: localhost:9090
      run: |
        go test -v -race -coverprofile=coverage-gcp.out ./backends/gcpsecrets/
        go tool cover -func=coverage-gcp.out | grep total

    - name: Stop GCP emulator
      if: always()
      run: kill $(cat gcp-mock.pid) 2>/dev/null || true

    - name: Upload GCP coverage
      uses: codecov/codecov-action@v4
      with:
        file: ./coverage-gcp.out
        flags: integration-gcp
```

**Coverage Impact**: GCP backend coverage increased from ~15% to 80.0% with emulator integration tests running in CI.

**About the GCP Secret Manager Emulator**: Unlike AWS (which has LocalStack), Google doesn't provide an official Secret Manager emulator. The [gcp-secret-manager-emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator) is a standalone gRPC implementation that:
- Implements the official Secret Manager v1 API (core operations)
- Has zero dependencies on vaultmux - completely standalone
- Is published as a reusable Go package
- Provides Docker images for both amd64 and arm64
- Is tested with the real GCP SDK client
- Executes tests in <1 second
- Can be used by any project that needs GCP Secret Manager testing
- Is actively maintained with 87% test coverage

The emulator was extracted from vaultmux in v1.0.0-rc2 to make it available to the broader Go community.

## Coverage Goals

| Package | Target | Current |
|---------|--------|---------|
| Core (`vaultmux`) | 95%+ | 98.5% |
| Mock Backend | 100% | 100% |
| CLI Backends | 80%+ | Varies |
| SDK Backends | 90%+ | 95%+ |

## Writing New Tests

### Adding Integration Tests for New Backends

1. **Create `*_integration_test.go`**
2. **Add environment variable check**
3. **Test full CRUD cycle**
4. **Test error conditions**
5. **Clean up resources**

Example template:

```go
func TestIntegration(t *testing.T) {
    // 1. Check for required env var
    endpoint := os.Getenv("BACKEND_ENDPOINT")
    if endpoint == "" {
        t.Skip("BACKEND_ENDPOINT not set - skipping")
    }

    // 2. Initialize backend
    backend, err := New(map[string]string{
        "endpoint": endpoint,
    }, "")
    if err != nil {
        t.Fatal(err)
    }

    ctx := context.Background()

    // 3. Test authentication
    session, err := backend.Authenticate(ctx)
    if err != nil {
        t.Fatal(err)
    }

    // 4. Test CRUD operations
    t.Run("CreateItem", func(t *testing.T) {
        err := backend.CreateItem(ctx, "test", "content", session)
        if err != nil {
            t.Fatal(err)
        }
    })

    // 5. Cleanup
    defer backend.DeleteItem(ctx, "test", session)
}
```

## Troubleshooting

### LocalStack Connection Issues

**Problem**: Tests fail with "connection refused"

**Solution**:
```bash
# Check LocalStack is running
docker ps | grep localstack

# Check health endpoint
curl http://localhost:4566/_localstack/health

# Restart LocalStack
docker stop vaultmux-localstack
docker run -d --rm -p 4566:4566 -e SERVICES=secretsmanager --name vaultmux-localstack localstack/localstack
```

### Test Timeouts

**Problem**: Tests timeout waiting for cloud services

**Solution**:
- Use context timeouts in tests
- Check network connectivity
- Verify credentials are valid
- For LocalStack, ensure container is fully started

### Flaky Tests

**Problem**: Tests pass sometimes, fail others

**Common Causes**:
- Race conditions (use `-race` flag)
- Resource cleanup issues
- Network instability
- Clock/time dependencies

**Solution**:
```bash
# Run with race detector
go test -race ./...

# Run multiple times
go test -count=10 ./backends/awssecrets/
```

## Performance Testing

Run benchmarks to ensure backends perform well:

```bash
# Run benchmarks
go test -bench=. ./...

# With memory profiling
go test -bench=. -benchmem ./...
```

## Test Data Management

### Prefixes for Test Isolation

Backends use prefixes to isolate test data:

- AWS: `test-vaultmux/` prefix
- GCP: `test-vaultmux-` prefix
- Azure: `test-vaultmux-` prefix

This allows running tests without polluting production namespaces.

### Cleanup Strategy

Integration tests should always clean up:

```go
defer func() {
    _ = backend.DeleteItem(ctx, itemName, session)
}()
```

For bulk operations, use test-specific prefixes that can be mass-deleted.

## Further Reading

- [GitHub Actions Workflow](.github/workflows/test.yml)
- [LocalStack Documentation](https://docs.localstack.cloud/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [Architecture Guide](ARCHITECTURE.md)
- [Contributing Guide](CONTRIBUTING.md)
