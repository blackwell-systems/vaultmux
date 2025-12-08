# AWS Secrets Manager Backend - Implementation Plan

**Version:** v0.3.0
**Status:** Planning Phase
**Date:** 2025-12-07

---

## Overview

This document outlines the complete implementation plan for the AWS Secrets Manager backend. This is vaultmux's first SDK-based backend (not CLI-wrapper), which validates that the `Backend` interface works with native API clients.

**Why AWS Secrets Manager First:**
- Validates SDK pattern (critical design validation)
- Industry-standard secret management for AWS workloads
- Well-documented Go SDK v2
- LocalStack enables local testing without AWS account
- Pattern applicable to Azure Key Vault and GCP Secret Manager

---

## Architecture Design

### Backend Struct

```go
package awssecrets

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
    "github.com/blackwell-systems/vaultmux"
)

// Backend implements vaultmux.Backend for AWS Secrets Manager.
type Backend struct {
    // AWS SDK client
    client *secretsmanager.Client

    // Configuration
    region      string
    prefix      string
    endpoint    string // For localstack testing

    // AWS config (credentials, region)
    awsConfig   aws.Config

    // Session cache (optional - IAM creds are long-lived)
    sessionFile string
}

// New creates a new AWS Secrets Manager backend.
func New(options map[string]string, sessionFile string) (*Backend, error) {
    region := options["region"]
    if region == "" {
        region = "us-east-1" // Default
    }

    prefix := options["prefix"]
    if prefix == "" {
        prefix = "vaultmux/"
    }

    endpoint := options["endpoint"] // For localstack: http://localhost:4566

    return &Backend{
        region:      region,
        prefix:      prefix,
        endpoint:    endpoint,
        sessionFile: sessionFile,
    }, nil
}
```

**Key Design Decisions:**

1. **Client Field**: Store `*secretsmanager.Client` to reuse connections
2. **Prefix**: Namespaces secrets (e.g., `vaultmux/my-app/item-name`)
3. **Endpoint**: Allows localstack override for testing
4. **AWS Config**: Loaded once during `Init()`, reused for all operations

---

### Session Implementation

AWS Secrets Manager uses IAM credentials, not session tokens. Our `Session` implementation wraps the AWS config:

```go
// awsSession represents IAM credentials for AWS.
type awsSession struct {
    config  aws.Config
    backend *Backend
}

func (s *awsSession) Token() string {
    // IAM credentials don't have a simple "token"
    // Return a representation for debugging
    creds, _ := s.config.Credentials.Retrieve(context.Background())
    return creds.AccessKeyID // Truncated for security
}

func (s *awsSession) IsValid(ctx context.Context) bool {
    // Check if credentials are still valid
    creds, err := s.config.Credentials.Retrieve(ctx)
    if err != nil {
        return false
    }

    // AWS credentials don't expire in the traditional sense
    // (except for temporary STS credentials)
    // For now, assume valid if retrieval succeeds
    return creds.AccessKeyID != ""
}

func (s *awsSession) Refresh(ctx context.Context) error {
    // Re-initialize AWS config to pick up new credentials
    return s.backend.initAWSConfig(ctx)
}

func (s *awsSession) ExpiresAt() time.Time {
    // Static credentials don't expire
    // STS temporary credentials have expiration, but SDK handles refresh
    return time.Time{} // Zero value = no expiration
}
```

**Session Pattern Insight:**

AWS IAM credentials are fundamentally different from Bitwarden/1Password tokens:
- **Long-lived**: Static credentials from `~/.aws/credentials` don't expire
- **SDK-managed**: Temporary credentials (STS, EC2 instance roles) are auto-refreshed by SDK
- **No explicit refresh**: AWS SDK handles credential renewal transparently

Our session is a thin wrapper that validates credentials are present.

---

## Method Implementations

### Init()

```go
func (b *Backend) Init(ctx context.Context) error {
    // Load AWS config (credentials, region)
    if err := b.initAWSConfig(ctx); err != nil {
        return vaultmux.WrapError("awssecrets", "init", "",
            fmt.Errorf("failed to load AWS config: %w", err))
    }

    // Create AWS Secrets Manager client
    b.client = secretsmanager.NewFromConfig(b.awsConfig, func(o *secretsmanager.Options) {
        if b.endpoint != "" {
            o.BaseEndpoint = aws.String(b.endpoint)
        }
    })

    // Verify connectivity (optional health check)
    _, err := b.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
        MaxResults: aws.Int32(1),
    })
    if err != nil {
        return vaultmux.WrapError("awssecrets", "init", "",
            fmt.Errorf("failed to connect to AWS Secrets Manager: %w", err))
    }

    return nil
}

func (b *Backend) initAWSConfig(ctx context.Context) error {
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(b.region),
    )
    if err != nil {
        return err
    }

    b.awsConfig = cfg
    return nil
}
```

**Init Responsibilities:**
1. Load AWS credentials (from env, shared config, or instance metadata)
2. Create Secrets Manager client with proper endpoint
3. Verify connectivity with lightweight API call
4. Return clear error if credentials missing or region unreachable

---

### IsAuthenticated() & Authenticate()

```go
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
    if b.awsConfig.Credentials == nil {
        return false
    }

    // Try to retrieve credentials
    _, err := b.awsConfig.Credentials.Retrieve(ctx)
    return err == nil
}

func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
    // AWS authentication happens via Init()
    // No interactive login needed (credentials from env/config)

    if !b.IsAuthenticated(ctx) {
        return nil, vaultmux.ErrNotAuthenticated
    }

    return &awsSession{
        config:  b.awsConfig,
        backend: b,
    }, nil
}
```

**Authentication Pattern:**

Unlike CLI-based backends (Bitwarden, 1Password), AWS doesn't have interactive authentication:
- Credentials come from: `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` env vars, `~/.aws/credentials`, or IAM instance role
- No password prompt
- `Authenticate()` validates credentials exist, doesn't fetch new ones

---

### GetItem()

```go
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
    if !session.IsValid(ctx) {
        return nil, vaultmux.ErrNotAuthenticated
    }

    secretName := b.secretName(name)

    result, err := b.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretName),
    })
    if err != nil {
        return nil, b.handleAWSError(err, "get", name)
    }

    return &vaultmux.Item{
        ID:    aws.ToString(result.ARN),
        Name:  name,
        Type:  vaultmux.ItemTypeSecureNote,
        Notes: aws.ToString(result.SecretString),
    }, nil
}

func (b *Backend) secretName(name string) string {
    if b.prefix != "" {
        return b.prefix + name
    }
    return name
}
```

**API Mapping:**
- `vaultmux.Item.ID` → AWS ARN (unique identifier)
- `vaultmux.Item.Name` → User-provided name (without prefix)
- `vaultmux.Item.Notes` → AWS `SecretString` field
- `vaultmux.Item.Type` → Always `ItemTypeSecureNote` (AWS doesn't have types)

---

### CreateItem()

```go
func (b *Backend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
    if !session.IsValid(ctx) {
        return vaultmux.ErrNotAuthenticated
    }

    secretName := b.secretName(name)

    // Check if already exists
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if exists {
        return vaultmux.ErrAlreadyExists
    }

    _, err = b.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
        Name:         aws.String(secretName),
        SecretString: aws.String(content),
        Tags: []types.Tag{
            {Key: aws.String("vaultmux"), Value: aws.String("true")},
            {Key: aws.String("prefix"), Value: aws.String(b.prefix)},
        },
    })
    if err != nil {
        return b.handleAWSError(err, "create", name)
    }

    return nil
}
```

**Tags for Filtering:**

We tag secrets with:
- `vaultmux: true` - Identifies vaultmux-managed secrets
- `prefix: <prefix>` - Enables prefix-based filtering in `ListItems()`

This allows `ListItems()` to efficiently filter secrets.

---

### UpdateItem()

```go
func (b *Backend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
    if !session.IsValid(ctx) {
        return vaultmux.ErrNotAuthenticated
    }

    secretName := b.secretName(name)

    // Check if exists
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if !exists {
        return vaultmux.ErrNotFound
    }

    _, err = b.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
        SecretId:     aws.String(secretName),
        SecretString: aws.String(content),
    })
    if err != nil {
        return b.handleAWSError(err, "update", name)
    }

    return nil
}
```

**Versioning Note:**

AWS Secrets Manager automatically versions secrets. Each `PutSecretValue` creates a new version. We use the latest version (default behavior).

---

### DeleteItem()

```go
func (b *Backend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
    if !session.IsValid(ctx) {
        return vaultmux.ErrNotAuthenticated
    }

    secretName := b.secretName(name)

    // Check if exists
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if !exists {
        return vaultmux.ErrNotFound
    }

    _, err = b.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
        SecretId:                   aws.String(secretName),
        ForceDeleteWithoutRecovery: aws.Bool(true), // Immediate deletion
    })
    if err != nil {
        return b.handleAWSError(err, "delete", name)
    }

    return nil
}
```

**Deletion Strategy:**

AWS Secrets Manager has two deletion modes:
1. **Scheduled deletion** (default, 7-30 day recovery window)
2. **Force delete** (`ForceDeleteWithoutRecovery: true`)

We use **force delete** for consistency with other backends (immediate effect).

---

### ListItems()

```go
func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
    if !session.IsValid(ctx) {
        return nil, vaultmux.ErrNotAuthenticated
    }

    var items []*vaultmux.Item
    var nextToken *string

    for {
        result, err := b.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
            Filters: []types.Filter{
                {
                    Key:    types.FilterNameStringTypeName,
                    Values: []string{b.prefix + "*"}, // Prefix filter
                },
                {
                    Key:    types.FilterNameStringTypeTagKey,
                    Values: []string{"vaultmux"},
                },
            },
            MaxResults: aws.Int32(100),
            NextToken:  nextToken,
        })
        if err != nil {
            return nil, b.handleAWSError(err, "list", "")
        }

        for _, secret := range result.SecretList {
            name := strings.TrimPrefix(aws.ToString(secret.Name), b.prefix)
            items = append(items, &vaultmux.Item{
                ID:   aws.ToString(secret.ARN),
                Name: name,
                Type: vaultmux.ItemTypeSecureNote,
                // Notes not included (requires separate GetSecretValue call)
            })
        }

        nextToken = result.NextToken
        if nextToken == nil {
            break
        }
    }

    return items, nil
}
```

**Pagination Handling:**

AWS returns up to 100 secrets per call. We loop until `NextToken` is nil.

---

### Error Handling

```go
func (b *Backend) handleAWSError(err error, operation, itemName string) error {
    if err == nil {
        return nil
    }

    var rnf *types.ResourceNotFoundException
    if errors.As(err, &rnf) {
        return vaultmux.ErrNotFound
    }

    var rae *types.ResourceExistsException
    if errors.As(err, &rae) {
        return vaultmux.ErrAlreadyExists
    }

    var ade *types.AccessDeniedException
    if errors.As(err, &ade) {
        return vaultmux.WrapError("awssecrets", operation, itemName,
            fmt.Errorf("access denied - check IAM permissions: %w", err))
    }

    // Generic error
    return vaultmux.WrapError("awssecrets", operation, itemName, err)
}
```

**AWS Error Mapping:**
- `ResourceNotFoundException` → `ErrNotFound`
- `ResourceExistsException` → `ErrAlreadyExists`
- `AccessDeniedException` → Wrapped error with IAM hint
- All others → Wrapped error with AWS message

---

## Configuration Options

The backend accepts these options in `Config.Options`:

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `region` | AWS region | `us-east-1` | `us-west-2` |
| `prefix` | Secret name prefix | `vaultmux/` | `myapp/` |
| `endpoint` | Custom endpoint (for localstack) | (none) | `http://localhost:4566` |

**Usage Example:**

```go
backend, err := vaultmux.New(vaultmux.Config{
    Backend: vaultmux.BackendAWSSecretsManager,
    Options: map[string]string{
        "region":   "us-west-2",
        "prefix":   "myapp/",
        "endpoint": "http://localhost:4566", // LocalStack
    },
})
```

---

## Testing Strategy

### 1. Unit Tests (Mocked SDK)

Test individual methods with mocked AWS SDK client:

```go
// awssecrets_test.go
type mockSecretsManagerClient struct {
    secrets map[string]string
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
    secretName := aws.ToString(params.SecretId)
    value, ok := m.secrets[secretName]
    if !ok {
        return nil, &types.ResourceNotFoundException{}
    }

    return &secretsmanager.GetSecretValueOutput{
        SecretString: aws.String(value),
        ARN:          aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:" + secretName),
    }, nil
}

func TestGetItem_Mock(t *testing.T) {
    backend := &Backend{
        client: &mockSecretsManagerClient{
            secrets: map[string]string{
                "vaultmux/test": "test-content",
            },
        },
        prefix: "vaultmux/",
    }

    item, err := backend.GetItem(context.Background(), "test", &awsSession{})
    if err != nil {
        t.Fatalf("GetItem() error = %v", err)
    }
    if item.Notes != "test-content" {
        t.Errorf("Notes = %q, want %q", item.Notes, "test-content")
    }
}
```

**Advantages:**
- Fast (no network calls)
- No external dependencies
- Tests business logic in isolation

---

### 2. Integration Tests (LocalStack)

Test against real AWS API (emulated):

```go
// awssecrets_integration_test.go
func TestWithLocalStack(t *testing.T) {
    if os.Getenv("LOCALSTACK_ENDPOINT") == "" {
        t.Skip("LOCALSTACK_ENDPOINT not set")
    }

    backend, err := New(map[string]string{
        "region":   "us-east-1",
        "endpoint": os.Getenv("LOCALSTACK_ENDPOINT"),
        "prefix":   "test/",
    }, "")
    if err != nil {
        t.Fatal(err)
    }

    ctx := context.Background()
    if err := backend.Init(ctx); err != nil {
        t.Fatal(err)
    }

    session, err := backend.Authenticate(ctx)
    if err != nil {
        t.Fatal(err)
    }

    // Full CRUD test
    t.Run("create", func(t *testing.T) {
        err := backend.CreateItem(ctx, "test-item", "test-content", session)
        if err != nil {
            t.Fatalf("CreateItem() error = %v", err)
        }
    })

    t.Run("get", func(t *testing.T) {
        item, err := backend.GetItem(ctx, "test-item", session)
        if err != nil {
            t.Fatalf("GetItem() error = %v", err)
        }
        if item.Notes != "test-content" {
            t.Errorf("Notes = %q, want %q", item.Notes, "test-content")
        }
    })

    t.Run("update", func(t *testing.T) {
        err := backend.UpdateItem(ctx, "test-item", "updated-content", session)
        if err != nil {
            t.Fatalf("UpdateItem() error = %v", err)
        }
    })

    t.Run("delete", func(t *testing.T) {
        err := backend.DeleteItem(ctx, "test-item", session)
        if err != nil {
            t.Fatalf("DeleteItem() error = %v", err)
        }
    })
}
```

**Setup:**
```bash
# Start localstack
docker run -d -p 4566:4566 -e SERVICES=secretsmanager localstack/localstack

# Run tests
LOCALSTACK_ENDPOINT=http://localhost:4566 \
AWS_ACCESS_KEY_ID=test \
AWS_SECRET_ACCESS_KEY=test \
go test -v ./backends/awssecrets/
```

---

### 3. GitHub Actions CI

```yaml
# .github/workflows/test-aws.yml
name: AWS Secrets Manager Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      localstack:
        image: localstack/localstack
        ports:
          - 4566:4566
        env:
          SERVICES: secretsmanager

    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Wait for LocalStack
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:4566/_localstack/health; do sleep 2; done'

      - name: Run AWS tests
        env:
          LOCALSTACK_ENDPOINT: http://localhost:4566
          AWS_ACCESS_KEY_ID: test
          AWS_SECRET_ACCESS_KEY: test
          AWS_REGION: us-east-1
        run: go test -v ./backends/awssecrets/
```

---

## Implementation Steps

### Phase 1: Foundation (Day 1-2)

1. **Create directory structure**
   ```bash
   mkdir -p backends/awssecrets
   touch backends/awssecrets/awssecrets.go
   touch backends/awssecrets/awssecrets_test.go
   touch backends/awssecrets/session.go
   ```

2. **Add AWS SDK dependency**
   ```bash
   go get github.com/aws/aws-sdk-go-v2/config
   go get github.com/aws/aws-sdk-go-v2/service/secretsmanager
   ```

3. **Implement Backend struct and New()**
   - Define struct fields
   - Parse configuration options
   - Validate inputs

4. **Implement Session type**
   - Wrap AWS config
   - Implement Session interface methods
   - Handle credential validation

---

### Phase 2: Core Methods (Day 3-4)

5. **Implement Init() and authentication**
   - Load AWS config
   - Create Secrets Manager client
   - Verify connectivity

6. **Implement GetItem() and GetNotes()**
   - Call `GetSecretValue` API
   - Map AWS response to `vaultmux.Item`
   - Handle errors

7. **Implement CreateItem()**
   - Check existence
   - Call `CreateSecret` API
   - Add tags

8. **Implement UpdateItem()**
   - Validate existence
   - Call `PutSecretValue` API

9. **Implement DeleteItem()**
   - Validate existence
   - Call `DeleteSecret` with force flag

---

### Phase 3: List and Errors (Day 5)

10. **Implement ListItems()**
    - Handle pagination
    - Filter by prefix and tags
    - Strip prefix from names

11. **Implement error handling**
    - Map AWS errors to vaultmux errors
    - Add IAM permission hints
    - Test error paths

12. **Implement ItemExists()**
    - Use GetItem() internally
    - Return bool without error on not-found

---

### Phase 4: Testing (Day 6-7)

13. **Write unit tests**
    - Mock AWS SDK client
    - Test all CRUD operations
    - Test error conditions

14. **Write integration tests**
    - Test with LocalStack
    - Full CRUD workflow
    - Pagination testing

15. **Add GitHub Actions workflow**
    - LocalStack service
    - Run integration tests
    - Code coverage reporting

---

### Phase 5: Registration and Docs (Day 8)

16. **Register backend**
    ```go
    func init() {
        vaultmux.RegisterBackend(vaultmux.BackendAWSSecretsManager,
            func(cfg vaultmux.Config) (vaultmux.Backend, error) {
                return New(cfg.Options, cfg.SessionFile)
            })
    }
    ```

17. **Add constant to factory.go**
    ```go
    BackendAWSSecretsManager BackendType = "awssecrets"
    ```

18. **Update documentation**
    - README.md: Add AWS to supported backends table
    - ARCHITECTURE.md: Add AWS column to feature matrix
    - EXTENDING.md: Reference AWS as SDK example

19. **Update CHANGELOG.md**
    - Document v0.3.0 release
    - Highlight first SDK-based backend

---

### Phase 6: Validation (Day 9-10)

20. **Manual testing**
    - Test with real AWS account (optional)
    - Verify IAM permissions documented correctly
    - Test in EC2/ECS environment (optional)

21. **Performance testing**
    - Measure API latency
    - Test with 100+ secrets (pagination)
    - Benchmark operations

22. **Security review**
    - Ensure no credentials logged
    - Validate session security
    - Check error messages don't leak sensitive info

---

## Location Management (Optional - v0.3.1)

AWS doesn't have native "folders" like 1Password vaults. Two approaches:

### Approach 1: Tag-Based Locations

```go
func (b *Backend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
    // Create a marker secret with location tag
    _, err := b.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
        Name:         aws.String(b.prefix + "_locations/" + name),
        SecretString: aws.String("{}"),
        Tags: []types.Tag{
            {Key: aws.String("vaultmux-location"), Value: aws.String(name)},
        },
    })
    return err
}

func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
    // Filter by location tag
    result, err := b.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
        Filters: []types.Filter{
            {
                Key:    types.FilterNameStringTypeTagValue,
                Values: []string{locValue},
            },
        },
    })
    // ... map results
}
```

### Approach 2: Path-Based Locations

```go
// Items in "work" location: "vaultmux/work/item-name"
// Items in "personal" location: "vaultmux/personal/item-name"

func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
    prefix := b.prefix + locValue + "/"
    // Filter by name prefix in ListSecrets
}
```

**Recommendation:** Start without location support (v0.3.0), add tag-based approach in v0.3.1 if needed.

---

## Success Criteria

**v0.3.0 is complete when:**

1. ✅ All `Backend` interface methods implemented
2. ✅ Session type working with IAM credentials
3. ✅ Unit tests with mocked SDK (>90% coverage)
4. ✅ Integration tests with LocalStack passing
5. ✅ GitHub Actions CI running LocalStack tests
6. ✅ Documentation updated (README, ARCHITECTURE, CHANGELOG)
7. ✅ Backend registered and usable via `vaultmux.New()`
8. ✅ Manual testing with LocalStack successful

**Stretch Goals:**
- Real AWS testing in CI (using secrets)
- Performance benchmarks
- Location management (tag-based)

---

## Risk Mitigation

**Risk: AWS SDK breaking changes**
- Mitigation: Pin SDK version, monitor deprecations

**Risk: LocalStack differences from real AWS**
- Mitigation: Document any known differences, add real AWS tests (optional)

**Risk: IAM permission confusion**
- Mitigation: Document exact permissions needed, provide example policy

**Risk: Cost concerns for testing**
- Mitigation: Emphasize LocalStack for dev, only use real AWS for final validation

---

## Next Steps

1. **Review this plan** - Get feedback on architecture decisions
2. **Set up LocalStack** - Verify testing environment works
3. **Start Phase 1** - Create directory structure and basic types
4. **Iterate daily** - Commit working code incrementally

---

**Estimated Timeline:** 10 days (part-time) or 5 days (full-time)

**Questions? Concerns?** Review this plan and identify any unclear sections before starting implementation.
