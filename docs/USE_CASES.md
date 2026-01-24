# Vaultmux Use Cases

Real-world scenarios where vaultmux solves concrete problems. Each use case includes the problem context, solution with code examples, and explanation of benefits.

---

## Table of Contents

- [Multi-Cloud Deployments](#use-case-1-multi-cloud-deployments)
- [Cross-Platform Applications](#use-case-2-cross-platform-applications)
- [Team Flexibility / Open Source Projects](#use-case-3-team-flexibility--open-source-projects)
- [Development → Staging → Production Workflow](#use-case-4-development--staging--production-workflow)
- [Migration Between Secret Managers](#use-case-5-migration-between-secret-managers)
- [Testing Without Real Credentials](#use-case-6-testing-without-real-credentials)
- [Vendor Neutrality for SaaS Products](#use-case-7-vendor-neutrality-for-saas-products)
- [Hybrid Cloud / On-Premises](#use-case-8-hybrid-cloud--on-premises)
- [CLI Tools with User Choice](#use-case-9-cli-tools-with-user-choice)
- [Disaster Recovery / Business Continuity](#use-case-10-disaster-recovery--business-continuity)

---

## Use Case 1: Multi-Cloud Deployments

### Problem

You're building a SaaS product. Customer A runs on AWS, Customer B runs on GCP, Customer C runs on Azure. Each customer wants secrets stored in their cloud provider's native secret management service.

**Without vaultmux, you need separate code paths:**

```go
func getSecret(customer string, name string) (string, error) {
    switch customer {
    case "customer-a":
        // AWS-specific code
        cfg, _ := awsconfig.LoadDefaultConfig(context.Background())
        client := awssecretsmanager.NewFromConfig(cfg)
        input := &awssecretsmanager.GetSecretValueInput{
            SecretId: aws.String(name),
        }
        result, err := client.GetSecretValue(context.Background(), input)
        if err != nil {
            return "", err
        }
        return *result.SecretString, nil
        
    case "customer-b":
        // GCP-specific code
        ctx := context.Background()
        client, _ := secretmanager.NewClient(ctx)
        defer client.Close()
        req := &secretmanagerpb.AccessSecretVersionRequest{
            Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name),
        }
        result, err := client.AccessSecretVersion(ctx, req)
        if err != nil {
            return "", err
        }
        return string(result.Payload.Data), nil
        
    case "customer-c":
        // Azure-specific code
        cred, _ := azidentity.NewDefaultAzureCredential(nil)
        client, _ := azsecrets.NewClient(vaultURL, cred, nil)
        resp, err := client.GetSecret(context.Background(), name, "", nil)
        if err != nil {
            return "", err
        }
        return *resp.Value, nil
    }
    
    return "", fmt.Errorf("unknown customer: %s", customer)
}
```

**Problems:**
- Three different APIs to learn, maintain, and debug
- Adding a new cloud provider requires significant new code
- Different error handling patterns for each backend
- Testing requires credentials for all three clouds
- Code duplication (retry logic, error handling, logging)

### Solution with Vaultmux

```go
import "github.com/blackwell-systems/vaultmux"

// Customer configuration stored in database
type Customer struct {
    ID          string
    VaultConfig vaultmux.Config
}

func getSecret(customer Customer, name string) (string, error) {
    // Create backend from customer's configuration
    backend, err := vaultmux.New(customer.VaultConfig)
    if err != nil {
        return "", fmt.Errorf("failed to create backend: %w", err)
    }
    defer backend.Close()
    
    // Initialize and authenticate
    if err := backend.Init(context.Background()); err != nil {
        return "", fmt.Errorf("failed to initialize backend: %w", err)
    }
    
    session, err := backend.Authenticate(context.Background())
    if err != nil {
        return "", fmt.Errorf("failed to authenticate: %w", err)
    }
    
    // Get secret - same code for all backends
    secret, err := backend.GetNotes(context.Background(), name, session)
    if err != nil {
        return "", fmt.Errorf("failed to get secret: %w", err)
    }
    
    return secret, nil
}

// Customer configurations
customerA := Customer{
    ID: "customer-a",
    VaultConfig: vaultmux.Config{
        Backend: vaultmux.BackendAWS,
        Options: map[string]string{
            "region": "us-west-2",
            "prefix": "customer-a/",
        },
    },
}

customerB := Customer{
    ID: "customer-b",
    VaultConfig: vaultmux.Config{
        Backend: vaultmux.BackendGCP,
        Options: map[string]string{
            "project_id": "customer-b-project",
            "prefix":     "customer-b-",
        },
    },
}

customerC := Customer{
    ID: "customer-c",
    VaultConfig: vaultmux.Config{
        Backend: vaultmux.BackendAzure,
        Options: map[string]string{
            "vault_url": "https://customer-c-vault.vault.azure.net/",
        },
    },
}
```

### Benefits

- **Single code path** - One implementation works for all customers
- **Easy to add new backends** - Support new cloud provider by adding config
- **Consistent error handling** - Same error types across backends
- **Testable** - Use mock backend for unit tests
- **Customer flexibility** - Customers choose their cloud without code changes
- **No vendor lock-in** - Switch cloud providers with configuration change

### When This Matters

- Multi-tenant SaaS products
- Enterprises with hybrid cloud strategies
- Consulting firms deploying to customer infrastructure
- Products sold to enterprises with strict cloud policies
- Avoiding vendor lock-in in competitive bidding

---

## Use Case 2: Cross-Platform Applications

### Problem

You're building a desktop application that runs on Windows, macOS, and Linux. Each platform has its own native credential storage, and users expect your app to integrate with their OS.

**Without vaultmux:**

```go
func getSecret(name string) (string, error) {
    switch runtime.GOOS {
    case "windows":
        // PowerShell credential manager code
        cmd := exec.Command("powershell", "-Command", 
            fmt.Sprintf(`(Get-StoredCredential -Target "%s").GetNetworkCredential().Password`, name))
        output, err := cmd.Output()
        if err != nil {
            return "", err
        }
        return strings.TrimSpace(string(output)), nil
        
    case "darwin":
        // macOS Keychain code
        cmd := exec.Command("security", "find-generic-password", 
            "-a", os.Getenv("USER"), "-s", name, "-w")
        output, err := cmd.Output()
        if err != nil {
            return "", err
        }
        return strings.TrimSpace(string(output)), nil
        
    case "linux":
        // Secret Service API or gnome-keyring code
        conn, _ := dbus.SessionBus()
        obj := conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")
        // ... complex DBus code ...
        
    default:
        return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
}
```

**Problems:**
- Platform-specific code everywhere
- Must test on all three platforms
- Different security models (Windows Hello vs Touch ID vs GPG)
- Inconsistent user experience
- Hard to maintain (three different APIs)

### Solution with Vaultmux

```go
import (
    "runtime"
    "github.com/blackwell-systems/vaultmux"
)

func detectPlatformBackend() vaultmux.BackendType {
    switch runtime.GOOS {
    case "windows":
        return vaultmux.BackendWinCred  // Windows Credential Manager
    case "darwin":
        // On macOS, check if user has 1Password installed
        if _, err := exec.LookPath("op"); err == nil {
            return vaultmux.BackendOnePassword
        }
        // Fall back to pass if installed
        if _, err := exec.LookPath("pass"); err == nil {
            return vaultmux.BackendPass
        }
        return vaultmux.BackendWinCred  // Use Keychain via generic credential
    case "linux":
        // Prefer pass (most common on Linux)
        if _, err := exec.LookPath("pass"); err == nil {
            return vaultmux.BackendPass
        }
        // Fall back to Bitwarden if installed
        if _, err := exec.LookPath("bw"); err == nil {
            return vaultmux.BackendBitwarden
        }
    }
    return ""
}

func getSecret(name string) (string, error) {
    // Create platform-appropriate backend
    config := vaultmux.Config{
        Backend: detectPlatformBackend(),
        Prefix:  "myapp",
    }
    
    backend, err := vaultmux.New(config)
    if err != nil {
        return "", fmt.Errorf("failed to create backend: %w", err)
    }
    defer backend.Close()
    
    if err := backend.Init(context.Background()); err != nil {
        return "", fmt.Errorf("failed to initialize: %w", err)
    }
    
    session, err := backend.Authenticate(context.Background())
    if err != nil {
        return "", fmt.Errorf("failed to authenticate: %w", err)
    }
    
    // Same code on all platforms
    return backend.GetNotes(context.Background(), name, session)
}
```

### User Experience

**Windows users:**
```
Your app uses Windows Credential Manager
Authenticate with Windows Hello (fingerprint, face, PIN)
Credentials stored in Windows protected storage
```

**macOS users:**
```
Your app uses 1Password (if installed)
Authenticate with Touch ID or password
Credentials stored in 1Password vault
```

**Linux users:**
```
Your app uses pass (if installed)
Authenticate with GPG key
Credentials stored in ~/.password-store (encrypted with GPG)
```

### Benefits

- **Native integration** - Users get OS-native credential storage
- **Single codebase** - No platform-specific code paths
- **Consistent API** - Same code on Windows/macOS/Linux
- **User choice** - Respects existing credential workflow
- **Security** - Leverages OS security features (Windows Hello, Touch ID)

### When This Matters

- Desktop applications (Electron, Qt, native apps)
- CLI tools used across platforms
- Developer tools (git credential helpers, cloud CLI tools)
- Password managers or credential sync tools

---

## Use Case 3: Team Flexibility / Open Source Projects

### Problem

You maintain an open source CLI tool. Contributors use different secret managers:
- Alice uses Bitwarden (free, open source)
- Bob uses pass (Unix standard)
- Carol uses 1Password (paid, enterprise)
- David uses AWS Secrets Manager (cloud-native)

**Without vaultmux:**

You pick one backend (e.g., Bitwarden) and force everyone to use it. This creates friction:
- Contributors must install and set up Bitwarden
- Some contributors already have pass/1Password set up
- Cloud-first users prefer AWS/GCP integration
- Result: Lower adoption, fewer contributors

### Solution with Vaultmux

```go
// Let users choose their backend via config file
type AppConfig struct {
    Backend vaultmux.BackendType `yaml:"backend"`
    Options map[string]string    `yaml:"options"`
}

func loadConfig() (AppConfig, error) {
    data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".mytool", "config.yaml"))
    if err != nil {
        // Default config
        return AppConfig{
            Backend: vaultmux.BackendPass, // Sensible default
        }, nil
    }
    
    var config AppConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return AppConfig{}, err
    }
    
    return config, nil
}

// CLI commands to manage backend
func cmdSetBackend(backendType string) error {
    config := AppConfig{
        Backend: vaultmux.BackendType(backendType),
    }
    
    data, _ := yaml.Marshal(config)
    configPath := filepath.Join(os.Getenv("HOME"), ".mytool", "config.yaml")
    
    os.MkdirAll(filepath.Dir(configPath), 0755)
    return os.WriteFile(configPath, data, 0600)
}

// All secret operations use configured backend
func getSecret(name string) (string, error) {
    config, _ := loadConfig()
    
    backend, err := vaultmux.New(vaultmux.Config{
        Backend: config.Backend,
        Options: config.Options,
    })
    if err != nil {
        return "", err
    }
    defer backend.Close()
    
    backend.Init(context.Background())
    session, _ := backend.Authenticate(context.Background())
    
    return backend.GetNotes(context.Background(), name, session)
}
```

### User Experience

**Alice (Bitwarden user):**
```bash
$ mytool config set-backend bitwarden
Backend set to: bitwarden

$ mytool secret set github-token "ghp_abc123"
Authenticating with Bitwarden...
Stored: github-token

$ mytool secret get github-token
ghp_abc123
```

**Bob (pass user):**
```bash
$ mytool config set-backend pass
Backend set to: pass

$ mytool secret set github-token "ghp_xyz789"
Stored: github-token

$ mytool secret get github-token
ghp_xyz789
```

**Carol (1Password user):**
```bash
$ mytool config set-backend 1password
Backend set to: 1password

$ mytool secret set github-token "ghp_def456"
Authenticating with 1Password...
Stored: github-token

$ mytool secret get github-token
ghp_def456
```

### Real-World Examples

**Dotfiles managers:**
- Store SSH keys, GPG keys, API tokens
- Users have existing vaults - don't force them to migrate
- Example: [blackwell-systems/dotfiles](https://github.com/blackwell-systems/dotfiles)

**Cloud CLI tools:**
- Terraform wrappers, kubectl plugins, cloud management tools
- Some users have cloud credentials in AWS Secrets Manager
- Others prefer local pass for security

**Password generators:**
- Generate passwords and store them
- Users choose where to store (1Password, Bitwarden, pass)

### Benefits

- **Lower adoption barrier** - Don't force users to switch tools
- **More contributors** - Works with existing workflows
- **User respect** - Let users choose their security model
- **Dogfooding** - Maintainers can use different backends too

### When This Matters

- Open source projects (CLIs, tools, libraries)
- Internal tools at companies with mixed tooling
- Developer tools used by diverse teams
- Education/tutorial projects (students use different tools)

---

## Use Case 4: Development → Staging → Production Workflow

### Problem

Different environments have different secret management requirements:

- **Development:** Local laptop, no cloud credentials, fast iteration
- **CI/CD:** Automated tests, no interactive prompts, ephemeral
- **Staging:** Cloud-based, mirrors production, IAM-based auth
- **Production:** Highly secure, audited, encrypted at rest, rotation

**Without vaultmux:**

```go
func getSecret(name string) (string, error) {
    env := os.Getenv("ENVIRONMENT")
    
    if env == "production" || env == "staging" {
        // AWS Secrets Manager code
        cfg, _ := awsconfig.LoadDefaultConfig(context.Background())
        client := awssecretsmanager.NewFromConfig(cfg)
        // ... AWS-specific code ...
    } else if env == "ci" {
        // Read from environment variables (insecure!)
        return os.Getenv("SECRET_" + name), nil
    } else {
        // Development - read from .env file (even less secure!)
        return readEnvFile(name)
    }
}
```

**Problems:**
- Development uses insecure local files
- CI/CD exposes secrets in environment variables
- Different code paths for each environment
- Can't test production secret flow locally

### Solution with Vaultmux

```go
func getBackendConfig() vaultmux.Config {
    env := os.Getenv("ENVIRONMENT")
    
    switch env {
    case "development":
        return vaultmux.Config{
            Backend: vaultmux.BackendPass,
            StorePath: filepath.Join(os.Getenv("HOME"), ".password-store"),
            Prefix: "myapp-dev",
        }
        
    case "ci":
        return vaultmux.Config{
            Backend: "mock", // Mock backend for tests
        }
        
    case "staging":
        return vaultmux.Config{
            Backend: vaultmux.BackendAWS,
            Options: map[string]string{
                "region": "us-east-1",
                "prefix": "myapp-staging/",
            },
        }
        
    case "production":
        return vaultmux.Config{
            Backend: vaultmux.BackendAWS,
            Options: map[string]string{
                "region": "us-west-2",
                "prefix": "myapp-production/",
            },
        }
        
    default:
        // Default to pass for unknown environments
        return vaultmux.Config{Backend: vaultmux.BackendPass}
    }
}

func getSecret(name string) (string, error) {
    config := getBackendConfig()
    
    backend, err := vaultmux.New(config)
    if err != nil {
        return "", fmt.Errorf("failed to create backend: %w", err)
    }
    defer backend.Close()
    
    if err := backend.Init(context.Background()); err != nil {
        return "", fmt.Errorf("failed to initialize: %w", err)
    }
    
    session, err := backend.Authenticate(context.Background())
    if err != nil {
        return "", fmt.Errorf("failed to authenticate: %w", err)
    }
    
    return backend.GetNotes(context.Background(), name, session)
}
```

### Workflow

**Developer experience:**

```bash
# Development: Use local pass
$ export ENVIRONMENT=development
$ go run main.go
Using backend: pass
Secret retrieved from ~/.password-store/myapp-dev/api-key

# No AWS credentials needed
# Fast iteration (no network calls)
# Offline development works
```

**CI/CD pipeline:**

```yaml
# .github/workflows/test.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run tests
        env:
          ENVIRONMENT: ci
        run: go test ./...
        # Uses mock backend
        # No real vault credentials needed
        # Fast tests (no network calls)
```

**Staging deployment:**

```bash
# Staging: AWS Secrets Manager (us-east-1)
$ export ENVIRONMENT=staging
$ ./deploy.sh
Using backend: aws (region: us-east-1)
Secrets retrieved from AWS Secrets Manager
Application deployed to staging

# IAM role provides credentials
# Audited via CloudTrail
# Encrypted at rest via KMS
```

**Production deployment:**

```bash
# Production: AWS Secrets Manager (us-west-2)
$ export ENVIRONMENT=production
$ ./deploy.sh
Using backend: aws (region: us-west-2)
Secrets retrieved from AWS Secrets Manager
Application deployed to production

# Separate AWS region for isolation
# Different IAM role (tighter permissions)
# Same code as staging
```

### Benefits

- **Secure local development** - No secrets in `.env` files or environment variables
- **Fast CI/CD** - Mock backend (no real vault, instant tests)
- **Production-like staging** - Same AWS backend, different region/prefix
- **No code changes** - Environment variable changes configuration
- **Offline development** - pass works without internet
- **Cost optimization** - No cloud vault charges for development

### When This Matters

- Web applications (APIs, microservices)
- CI/CD pipelines with secret requirements
- Developer tools that need secrets locally
- Applications with staging environments

---

## Use Case 5: Migration Between Secret Managers

### Problem

You're migrating from Bitwarden to AWS Secrets Manager. Reasons might include:
- Enterprise requirement (centralized secret management)
- Compliance (audit trail, encryption at rest, rotation)
- Scale (hundreds of microservices, thousands of secrets)

**Challenge:** Zero-downtime migration without rewriting application code.

### Solution with Vaultmux: Phased Migration

#### Phase 1: Dual-Read (Fallback Strategy)

```go
type SecretService struct {
    newBackend vaultmux.Backend  // AWS Secrets Manager
    oldBackend vaultmux.Backend  // Bitwarden
    newSession vaultmux.Session
    oldSession vaultmux.Session
}

func (s *SecretService) GetSecret(ctx context.Context, name string) (string, error) {
    // Try new backend first
    secret, err := s.newBackend.GetNotes(ctx, name, s.newSession)
    if err == nil {
        log.Printf("Retrieved %s from new backend (AWS)", name)
        return secret, nil
    }
    
    // Log but don't fail on new backend errors
    if !errors.Is(err, vaultmux.ErrNotFound) {
        log.Printf("New backend error for %s: %v, falling back to old backend", name, err)
    }
    
    // Fall back to old backend
    secret, err = s.oldBackend.GetNotes(ctx, name, s.oldSession)
    if err != nil {
        return "", fmt.Errorf("both backends failed for %s: %w", name, err)
    }
    
    log.Printf("Retrieved %s from old backend (Bitwarden), consider migrating", name)
    return secret, nil
}
```

**Deploy this code.** Application continues reading from Bitwarden (old backend). No downtime.

#### Phase 2: Migration Script

```go
func migrateSecrets(ctx context.Context) error {
    bwConfig := vaultmux.Config{Backend: vaultmux.BackendBitwarden}
    awsConfig := vaultmux.Config{
        Backend: vaultmux.BackendAWS,
        Options: map[string]string{
            "region": "us-west-2",
            "prefix": "myapp/",
        },
    }
    
    bw, _ := vaultmux.New(bwConfig)
    defer bw.Close()
    aws, _ := vaultmux.New(awsConfig)
    defer aws.Close()
    
    bw.Init(ctx)
    aws.Init(ctx)
    
    bwSession, _ := bw.Authenticate(ctx)
    awsSession, _ := aws.Authenticate(ctx)
    
    // List all items in Bitwarden
    items, err := bw.ListItems(ctx, bwSession)
    if err != nil {
        return fmt.Errorf("failed to list items: %w", err)
    }
    
    log.Printf("Found %d items to migrate", len(items))
    
    // Migrate each item
    for i, item := range items {
        // Get secret from Bitwarden
        content, err := bw.GetNotes(ctx, item.Name, bwSession)
        if err != nil {
            log.Printf("Failed to get %s: %v", item.Name, err)
            continue
        }
        
        // Check if already exists in AWS
        exists, _ := aws.ItemExists(ctx, item.Name, awsSession)
        if exists {
            log.Printf("Skipping %s (already exists in AWS)", item.Name)
            continue
        }
        
        // Store in AWS Secrets Manager
        err = aws.CreateItem(ctx, item.Name, content, awsSession)
        if err != nil {
            log.Printf("Failed to create %s in AWS: %v", item.Name, err)
            continue
        }
        
        log.Printf("[%d/%d] Migrated: %s", i+1, len(items), item.Name)
    }
    
    log.Printf("Migration complete")
    return nil
}
```

**Run once:**
```bash
$ go run migrate.go
Found 150 items to migrate
[1/150] Migrated: api-key
[2/150] Migrated: database-password
[3/150] Migrated: jwt-secret
...
[150/150] Migrated: stripe-api-key
Migration complete
```

All secrets now exist in both Bitwarden and AWS.

#### Phase 3: Verification (1-2 Weeks)

Keep dual-read code deployed. Monitor logs:

```bash
$ grep "Retrieved.*from.*backend" application.log
2024-01-15 10:23:45 Retrieved api-key from new backend (AWS)
2024-01-15 10:24:12 Retrieved database-password from new backend (AWS)
2024-01-15 10:25:33 Retrieved jwt-secret from new backend (AWS)
```

If you see "old backend (Bitwarden)" in logs → that secret wasn't migrated properly.

#### Phase 4: Switch to New Backend Only

```go
type SecretService struct {
    backend vaultmux.Backend  // AWS Secrets Manager only
    session vaultmux.Session
}

func (s *SecretService) GetSecret(ctx context.Context, name string) (string, error) {
    return s.backend.GetNotes(ctx, name, s.session)
}
```

**Deploy this code.** Application now reads exclusively from AWS. No more Bitwarden dependency.

#### Phase 5: Cleanup

- Decommission Bitwarden account
- Remove Bitwarden CLI from servers
- Remove Bitwarden credentials
- Update documentation

### Timeline

```
Week 1:  Deploy dual-read code (Phase 1)
Week 2:  Run migration script (Phase 2)
Week 3-4: Monitor logs, verify new backend works (Phase 3)
Week 5:  Deploy new-backend-only code (Phase 4)
Week 6:  Cleanup Bitwarden (Phase 5)
```

### Benefits

- **Zero downtime** - Application never stops working
- **Safe rollback** - Can revert to old backend if issues arise
- **Gradual migration** - Migrate one service at a time
- **Verification period** - Catch issues before full cutover
- **Same API** - vaultmux abstracts both backends

### When This Matters

- Migrating from Bitwarden to AWS/GCP/Azure
- Moving from pass to cloud vaults
- Consolidating multiple vaults into one
- Changing vendors (e.g., 1Password to AWS)

---

## Use Case 6: Testing Without Real Credentials

### Problem

Your application uses AWS Secrets Manager. You want to write unit tests, but:
- Tests shouldn't require AWS credentials
- CI/CD shouldn't have production vault access
- Tests should be fast (no network calls)
- Need to test error conditions (vault unavailable, expired session)

**Without vaultmux:**

```go
func TestApplication(t *testing.T) {
    // Option 1: Use real AWS Secrets Manager (bad)
    // - Requires AWS credentials in CI/CD
    // - Slow (network calls)
    // - Can't test error conditions easily
    
    // Option 2: Mock AWS SDK (tedious)
    // - Must mock secretsmanager.Client interface
    // - Complex mock setup
    // - Tightly coupled to AWS SDK
}
```

### Solution with Vaultmux

```go
import (
    "testing"
    "github.com/blackwell-systems/vaultmux/mock"
)

func TestApplicationLogic(t *testing.T) {
    // Create mock backend
    backend := mock.New()
    
    // Pre-populate with test data
    backend.SetItem("api-key", "test-api-key-12345")
    backend.SetItem("database-password", "test-db-pass")
    backend.SetItem("jwt-secret", "test-jwt-secret")
    
    // Initialize application with mock backend
    app := NewApp(backend)
    
    // Test application logic
    err := app.Initialize(context.Background())
    if err != nil {
        t.Fatalf("Failed to initialize: %v", err)
    }
    
    // Verify secret was retrieved correctly
    if app.APIKey != "test-api-key-12345" {
        t.Errorf("Expected api-key to be test-api-key-12345, got %s", app.APIKey)
    }
}

func TestErrorHandling(t *testing.T) {
    backend := mock.New()
    
    // Simulate vault outage
    backend.GetError = errors.New("connection refused: vault unavailable")
    
    app := NewApp(backend)
    err := app.Initialize(context.Background())
    
    // Application should handle vault errors gracefully
    if err == nil {
        t.Error("Expected error when vault is unavailable")
    }
    
    if !strings.Contains(err.Error(), "vault unavailable") {
        t.Errorf("Expected error message about vault, got: %v", err)
    }
}

func TestSecretNotFound(t *testing.T) {
    backend := mock.New()
    // Don't set any secrets
    
    app := NewApp(backend)
    err := app.Initialize(context.Background())
    
    // Application should handle missing secrets
    if !errors.Is(err, vaultmux.ErrNotFound) {
        t.Errorf("Expected ErrNotFound, got: %v", err)
    }
}

func TestSessionExpiration(t *testing.T) {
    backend := mock.New()
    backend.SetItem("api-key", "test-key")
    
    // First call succeeds
    backend.GetError = nil
    app := NewApp(backend)
    app.Initialize(context.Background())
    
    // Simulate session expiration
    backend.GetError = vaultmux.ErrSessionExpired
    
    // Application should re-authenticate
    secret, err := app.GetSecret("api-key")
    if err != nil {
        t.Errorf("Application failed to handle session expiration: %v", err)
    }
    if secret != "test-key" {
        t.Errorf("Expected test-key after re-auth, got: %s", secret)
    }
}
```

### Application Code (Testable)

```go
type App struct {
    backend vaultmux.Backend
    APIKey  string
}

// NewApp accepts any backend (mock or real)
func NewApp(backend vaultmux.Backend) *App {
    return &App{backend: backend}
}

func (a *App) Initialize(ctx context.Context) error {
    if err := a.backend.Init(ctx); err != nil {
        return fmt.Errorf("failed to initialize backend: %w", err)
    }
    
    session, err := a.backend.Authenticate(ctx)
    if err != nil {
        return fmt.Errorf("failed to authenticate: %w", err)
    }
    
    apiKey, err := a.backend.GetNotes(ctx, "api-key", session)
    if err != nil {
        return fmt.Errorf("failed to get api-key: %w", err)
    }
    
    a.APIKey = apiKey
    return nil
}
```

### CI/CD Pipeline

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.23
      
      - name: Run tests
        run: go test -v ./...
        # No AWS credentials needed!
        # Uses mock backend automatically
        # Fast tests (no network calls)
```

### Benefits

- **No credentials in CI/CD** - Mock backend doesn't need real vault
- **Fast tests** - No network calls (instant feedback)
- **Test error conditions** - Simulate vault outages, expired sessions, network errors
- **Deterministic** - Same results every run (no flaky tests)
- **No external dependencies** - Tests run offline

### When This Matters

- Unit testing applications with secret dependencies
- CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins)
- Local development (no cloud credentials needed)
- Integration testing (test secret logic without vault)

---

## Use Case 7: Vendor Neutrality for SaaS Products

### Problem

You're building a B2B SaaS product. Enterprise customers have strict requirements:
- Customer A: Must use AWS (already has AWS infrastructure)
- Customer B: Must use GCP (Google Workspace integration)
- Customer C: Must use Azure (Microsoft 365 enterprise agreement)
- Customer D: Must stay on-premises (compliance/regulatory)

**Without vaultmux:**

You pick one cloud provider (e.g., AWS) and force all customers onto AWS. This limits your market:
- GCP-first customers won't sign (vendor conflict)
- Azure-committed enterprises excluded (Microsoft agreements)
- Regulated industries can't use cloud (on-prem required)

### Solution with Vaultmux

```go
// Customer configuration stored in database
type Customer struct {
    ID              string
    Name            string
    VaultBackend    string
    VaultOptions    map[string]string
}

// Retrieve customer configuration
func getCustomerVaultConfig(customerID string) (vaultmux.Config, error) {
    customer, err := database.GetCustomer(customerID)
    if err != nil {
        return vaultmux.Config{}, err
    }
    
    return vaultmux.Config{
        Backend: vaultmux.BackendType(customer.VaultBackend),
        Options: customer.VaultOptions,
    }, nil
}

// Application code (works for all customers)
func getCustomerSecret(customerID, secretName string) (string, error) {
    config, err := getCustomerVaultConfig(customerID)
    if err != nil {
        return "", fmt.Errorf("failed to get vault config: %w", err)
    }
    
    backend, err := vaultmux.New(config)
    if err != nil {
        return "", fmt.Errorf("failed to create backend: %w", err)
    }
    defer backend.Close()
    
    backend.Init(context.Background())
    session, _ := backend.Authenticate(context.Background())
    
    return backend.GetNotes(context.Background(), secretName, session)
}
```

### Customer Configurations

**Customer A (AWS):**

```go
customerA := Customer{
    ID:           "customer-a",
    Name:         "Acme Corp",
    VaultBackend: "awssecrets",
    VaultOptions: map[string]string{
        "region":    "us-west-2",
        "prefix":    "acme-corp/",
        "role_arn":  "arn:aws:iam::123456789012:role/AcmeVaultAccess",
    },
}
```

**Customer B (GCP):**

```go
customerB := Customer{
    ID:           "customer-b",
    Name:         "Globex Corporation",
    VaultBackend: "gcpsecrets",
    VaultOptions: map[string]string{
        "project_id": "globex-production",
        "prefix":     "globex-",
    },
}
```

**Customer C (Azure):**

```go
customerC := Customer{
    ID:           "customer-c",
    Name:         "Initech LLC",
    VaultBackend: "azure",
    VaultOptions: map[string]string{
        "vault_url": "https://initech-vault.vault.azure.net/",
        "tenant_id": "12345678-1234-1234-1234-123456789012",
    },
}
```

**Customer D (On-Premises):**

```go
customerD := Customer{
    ID:           "customer-d",
    Name:         "SecureBank",
    VaultBackend: "bitwarden",
    VaultOptions: map[string]string{
        "server": "https://vault.securebank.internal",  // Self-hosted Vaultwarden
    },
}
```

### Onboarding Flow

**Sales call:**
```
Sales: "Where do you want to store secrets?"
Customer: "We're an AWS shop, must use AWS Secrets Manager."
Sales: "No problem! We support AWS natively."

[Sales enters customer vault config into admin dashboard]
[Application automatically uses customer's AWS Secrets Manager]
[Customer secrets never leave their AWS account]
```

**Customer onboarding:**

```bash
# Step 1: Customer creates IAM role in their AWS account
$ aws iam create-role --role-name MyCompanyVaultAccess \
    --assume-role-policy-document file://trust-policy.json

# Step 2: Customer provides role ARN to your SaaS
Role ARN: arn:aws:iam::123456789012:role/MyCompanyVaultAccess

# Step 3: Your application assumes this role to access their secrets
[Application uses vaultmux with customer's IAM role]
[Secrets retrieved from customer's AWS account]
[Customer maintains full control]
```

### Benefits

- **Vendor neutrality** - Support AWS, GCP, Azure, on-prem
- **Customer data sovereignty** - Secrets stay in customer's account/region
- **Compliance-friendly** - Customer controls encryption, access, auditing
- **Competitive advantage** - Win deals competitors can't (vendor flexibility)
- **Enterprise sales** - Meet strict procurement requirements

### Real-World Adoption Pattern

**Startup phase:**
```
Year 1-2: Use AWS Secrets Manager for all customers (simple)
Year 3:   Enterprise customer requires GCP → Add GCP support
Year 4:   Healthcare customer requires on-prem → Add Bitwarden support
Year 5:   Azure-committed Fortune 500 → Add Azure support
```

**With vaultmux:** Adding new backend = config change, not code rewrite.

### When This Matters

- B2B SaaS products (enterprise customers)
- Highly regulated industries (finance, healthcare, government)
- Multi-cloud strategy (avoid vendor lock-in)
- Competitive differentiation (vendor flexibility)
- Customer data residency requirements (EU, China, local laws)

---

## Use Case 8: Hybrid Cloud / On-Premises

### Problem

Your company has:
- **Cloud workloads** (AWS, GCP) for public-facing services
- **On-premises workloads** (data center) for regulated data
- **Compliance requirement:** Certain secrets cannot leave on-prem

**Challenge:** Same application code runs in both environments.

### Solution with Vaultmux

```go
func getBackendForEnvironment() vaultmux.Config {
    deploymentType := os.Getenv("DEPLOYMENT_TYPE")
    
    switch deploymentType {
    case "cloud-aws":
        return vaultmux.Config{
            Backend: vaultmux.BackendAWS,
            Options: map[string]string{
                "region": "us-west-2",
                "prefix": "myapp-cloud/",
            },
        }
        
    case "cloud-gcp":
        return vaultmux.Config{
            Backend: vaultmux.BackendGCP,
            Options: map[string]string{
                "project_id": "mycompany-production",
                "prefix":     "myapp-cloud-",
            },
        }
        
    case "on-premises":
        return vaultmux.Config{
            Backend: vaultmux.BackendBitwarden,
            Options: map[string]string{
                "server": "https://vault.company.internal",  // Self-hosted Vaultwarden
            },
        }
        
    case "on-premises-pass":
        return vaultmux.Config{
            Backend: vaultmux.BackendPass,
            StorePath: "/opt/company/secrets/.password-store",
        }
    }
    
    return vaultmux.Config{Backend: vaultmux.BackendPass}
}
```

### Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Code                         │
│                 (Same binary everywhere)                    │
└─────────────────────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            │               │               │
            ▼               ▼               ▼
    ┌───────────────┐ ┌───────────────┐ ┌───────────────┐
    │   Cloud AWS   │ │   Cloud GCP   │ │ On-Premises   │
    ├───────────────┤ ├───────────────┤ ├───────────────┤
    │ AWS Secrets   │ │ GCP Secret    │ │ Vaultwarden   │
    │   Manager     │ │   Manager     │ │ (self-hosted) │
    └───────────────┘ └───────────────┘ └───────────────┘
         (Public)         (Public)        (Private network)
```

### Use Case: Healthcare Application

**Requirements:**
- Patient data must stay on-premises (HIPAA)
- Analytics data can go to cloud (de-identified)
- Same application handles both

**Implementation:**

```go
type SecretScope string

const (
    ScopePublic  SecretScope = "public"   // Can use cloud vaults
    ScopePrivate SecretScope = "private"  // Must use on-prem vault
)

func getSecret(name string, scope SecretScope) (string, error) {
    var config vaultmux.Config
    
    if scope == ScopePrivate {
        // Private secrets always use on-premises vault
        config = vaultmux.Config{
            Backend: vaultmux.BackendBitwarden,
            Options: map[string]string{
                "server": "https://vault.hospital.internal",
            },
        }
    } else {
        // Public secrets can use cloud vault
        deploymentType := os.Getenv("DEPLOYMENT_TYPE")
        if strings.HasPrefix(deploymentType, "cloud") {
            config = getCloudBackend()
        } else {
            config = getOnPremBackend()
        }
    }
    
    backend, _ := vaultmux.New(config)
    defer backend.Close()
    
    backend.Init(context.Background())
    session, _ := backend.Authenticate(context.Background())
    
    return backend.GetNotes(context.Background(), name, session)
}

// Usage
func main() {
    // Patient database credentials - must stay on-prem
    dbPassword, _ := getSecret("patient-db-password", ScopePrivate)
    
    // Analytics API key - can use cloud
    analyticsKey, _ := getSecret("analytics-api-key", ScopePublic)
}
```

### Benefits

- **Compliance** - Keep regulated secrets on-premises
- **Flexibility** - Cloud workloads use cloud vaults
- **Cost optimization** - Don't pay for cloud vaults when on-prem works
- **Migration path** - Gradual cloud adoption over time

### When This Matters

- Regulated industries (healthcare, finance, government)
- Companies with existing data center investments
- Data residency requirements (GDPR, data sovereignty laws)
- Hybrid cloud strategies

---

## Use Case 9: CLI Tools with User Choice

### Problem

You're building a CLI tool that needs to store credentials (API tokens, SSH keys, etc.). Users already have password managers set up - don't make them switch.

**Without vaultmux:**

You hardcode one backend (e.g., "use Bitwarden") and alienate users who prefer pass or 1Password.

### Solution with Vaultmux

```go
// CLI command: mytool config set-backend <backend>
func cmdConfigSetBackend(backendType string) error {
    valid := []string{"bitwarden", "1password", "pass", "awssecrets", "gcpsecrets"}
    
    if !contains(valid, backendType) {
        return fmt.Errorf("invalid backend: %s (valid: %s)", backendType, strings.Join(valid, ", "))
    }
    
    config := AppConfig{Backend: backendType}
    return saveConfig(config)
}

// CLI command: mytool secret set <name> <value>
func cmdSecretSet(name, value string) error {
    config, _ := loadConfig()
    
    backend, err := vaultmux.New(vaultmux.Config{
        Backend: vaultmux.BackendType(config.Backend),
        Prefix:  "mytool",
    })
    if err != nil {
        return fmt.Errorf("failed to create backend: %w", err)
    }
    defer backend.Close()
    
    backend.Init(context.Background())
    session, _ := backend.Authenticate(context.Background())
    
    return backend.CreateItem(context.Background(), name, value, session)
}

// CLI command: mytool secret get <name>
func cmdSecretGet(name string) error {
    config, _ := loadConfig()
    
    backend, err := vaultmux.New(vaultmux.Config{
        Backend: vaultmux.BackendType(config.Backend),
        Prefix:  "mytool",
    })
    if err != nil {
        return fmt.Errorf("failed to create backend: %w", err)
    }
    defer backend.Close()
    
    backend.Init(context.Background())
    session, _ := backend.Authenticate(context.Background())
    
    secret, err := backend.GetNotes(context.Background(), name, session)
    if err != nil {
        return err
    }
    
    fmt.Println(secret)
    return nil
}
```

### User Experience

**User with Bitwarden:**

```bash
$ mytool config set-backend bitwarden
✓ Backend set to: bitwarden

$ mytool secret set github-token "ghp_abc123"
Authenticating with Bitwarden...
✓ Stored: github-token

$ mytool secret get github-token
ghp_abc123
```

**User with pass:**

```bash
$ mytool config set-backend pass
✓ Backend set to: pass

$ mytool secret set github-token "ghp_xyz789"
✓ Stored: github-token

$ mytool secret get github-token
ghp_xyz789
```

**User with AWS (cloud-first developer):**

```bash
$ mytool config set-backend awssecrets
✓ Backend set to: awssecrets

$ mytool secret set github-token "ghp_def456"
✓ Stored: github-token (region: us-west-2)

$ mytool secret get github-token
ghp_def456
```

### Real-World Examples

**1. Dotfiles Manager**

Store SSH keys, GPG keys, API tokens in dotfiles repo. Users choose backend:

```bash
# Clone dotfiles
$ git clone https://github.com/me/dotfiles
$ cd dotfiles

# Set up secrets backend
$ ./install.sh
Which secret backend? (bitwarden/1password/pass): pass
✓ Using pass for secrets

# Secrets stored in ~/.password-store/dotfiles/
$ pass dotfiles/ssh-key
[SSH private key content]
```

**2. Cloud CLI Wrapper**

Wrapper around AWS/GCP/Azure CLIs that stores credentials:

```bash
# First-time setup
$ cloudctl config set-backend 1password
✓ Using 1Password for credentials

# Store AWS credentials
$ cloudctl aws configure
AWS Access Key ID: AKIAIOSFODNN7EXAMPLE
AWS Secret Access Key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
✓ Stored in 1Password (vault: cloudctl)

# Use credentials
$ cloudctl aws s3 ls
[Lists S3 buckets using credentials from 1Password]
```

**3. Git Credential Helper**

Git credential helper supporting multiple backends:

```bash
$ git config --global credential.helper cloudvault
$ cloudvault config set-backend pass
✓ Using pass for Git credentials

# Git automatically stores/retrieves credentials via pass
$ git push
Username: user@example.com
Password: [Retrieved from pass]
```

### Benefits

- **User respect** - Don't force tool changes
- **Lower friction** - Works with existing setup
- **Better security** - Users trust their existing vault
- **Wider adoption** - Appeal to more users

### When This Matters

- CLI tools (developer tools, infrastructure tools)
- Dotfiles managers
- Git credential helpers
- Cloud CLI wrappers
- Any tool storing user credentials

---

## Use Case 10: Disaster Recovery / Business Continuity

### Problem

Your application relies on AWS Secrets Manager (us-west-2). What happens if:
- AWS region outage (us-west-2 unavailable)
- API rate limiting (too many requests)
- Network partition (can't reach AWS)

**Without failover:** Application stops working entirely.

### Solution with Vaultmux: Multi-Region Failover

```go
type VaultConfig struct {
    Primary vaultmux.Config
    Backup  vaultmux.Config
}

func getSecretWithFailover(name string, config VaultConfig) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Try primary vault (us-west-2)
    primary, err := vaultmux.New(config.Primary)
    if err == nil {
        defer primary.Close()
        primary.Init(ctx)
        session, _ := primary.Authenticate(ctx)
        
        secret, err := primary.GetNotes(ctx, name, session)
        if err == nil {
            log.Printf("Retrieved %s from primary vault (us-west-2)", name)
            return secret, nil
        }
        
        // Log error but continue to backup
        log.Printf("Primary vault failed for %s: %v, trying backup", name, err)
    }
    
    // Failover to backup vault (us-east-1)
    backup, err := vaultmux.New(config.Backup)
    if err != nil {
        return "", fmt.Errorf("both vaults failed: primary error and backup create failed: %w", err)
    }
    defer backup.Close()
    
    backup.Init(ctx)
    session, err := backup.Authenticate(ctx)
    if err != nil {
        return "", fmt.Errorf("backup vault authentication failed: %w", err)
    }
    
    secret, err := backup.GetNotes(ctx, name, session)
    if err != nil {
        return "", fmt.Errorf("backup vault failed: %w", err)
    }
    
    log.Printf("Retrieved %s from backup vault (us-east-1) after primary failure", name)
    return secret, nil
}

// Configuration
config := VaultConfig{
    Primary: vaultmux.Config{
        Backend: vaultmux.BackendAWS,
        Options: map[string]string{
            "region": "us-west-2",  // Primary region
            "prefix": "myapp/",
        },
    },
    Backup: vaultmux.Config{
        Backend: vaultmux.BackendAWS,
        Options: map[string]string{
            "region": "us-east-1",  // Backup region
            "prefix": "myapp/",
        },
    },
}
```

### Scenario: Regional Outage

**Normal operation:**
```
2024-01-15 10:00:00 Retrieved api-key from primary vault (us-west-2)
2024-01-15 10:05:00 Retrieved database-password from primary vault (us-west-2)
2024-01-15 10:10:00 Retrieved jwt-secret from primary vault (us-west-2)
```

**us-west-2 outage occurs at 10:15:**
```
2024-01-15 10:15:00 Primary vault failed for api-key: timeout, trying backup
2024-01-15 10:15:01 Retrieved api-key from backup vault (us-east-1)
2024-01-15 10:20:00 Primary vault failed for database-password: timeout, trying backup
2024-01-15 10:20:01 Retrieved database-password from backup vault (us-east-1)
```

**Application continues running - zero downtime.**

### Secret Replication

Keep primary and backup in sync:

```go
func replicateSecrets() error {
    primaryBackend, _ := vaultmux.New(primaryConfig)
    defer primaryBackend.Close()
    backupBackend, _ := vaultmux.New(backupConfig)
    defer backupBackend.Close()
    
    primaryBackend.Init(context.Background())
    backupBackend.Init(context.Background())
    
    primarySession, _ := primaryBackend.Authenticate(context.Background())
    backupSession, _ := backupBackend.Authenticate(context.Background())
    
    // List all items in primary
    items, _ := primaryBackend.ListItems(context.Background(), primarySession)
    
    for _, item := range items {
        // Get secret from primary
        content, err := primaryBackend.GetNotes(context.Background(), item.Name, primarySession)
        if err != nil {
            log.Printf("Failed to get %s from primary: %v", item.Name, err)
            continue
        }
        
        // Check if exists in backup
        exists, _ := backupBackend.ItemExists(context.Background(), item.Name, backupSession)
        
        if exists {
            // Update in backup
            backupBackend.UpdateItem(context.Background(), item.Name, content, backupSession)
            log.Printf("Updated %s in backup vault", item.Name)
        } else {
            // Create in backup
            backupBackend.CreateItem(context.Background(), item.Name, content, backupSession)
            log.Printf("Created %s in backup vault", item.Name)
        }
    }
    
    return nil
}

// Run replication every hour
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        if err := replicateSecrets(); err != nil {
            log.Printf("Replication failed: %v", err)
        }
    }
}()
```

### Benefits

- **High availability** - Regional outages don't stop application
- **Automatic failover** - No manual intervention required
- **Fast recovery** - Backup vault responds in seconds
- **Cost-effective** - Only pay for secrets in two regions

### Advanced: Multi-Cloud Failover

Failover across cloud providers:

```go
config := VaultConfig{
    Primary: vaultmux.Config{
        Backend: vaultmux.BackendAWS,
        Options: map[string]string{"region": "us-west-2"},
    },
    Backup: vaultmux.Config{
        Backend: vaultmux.BackendGCP,
        Options: map[string]string{"project_id": "backup-project"},
    },
}

// If AWS is down, failover to GCP
// vaultmux abstracts the difference
```

### When This Matters

- Mission-critical applications (can't tolerate downtime)
- Multi-region deployments
- Disaster recovery planning
- SLA requirements (99.99%+ uptime)
- Business continuity requirements

---

## Summary Table

| Use Case | Problem Solved | When To Use |
|----------|----------------|-------------|
| **Multi-Cloud Deployments** | Support AWS/GCP/Azure without code duplication | Multi-tenant SaaS, vendor neutrality |
| **Cross-Platform Applications** | Native credential storage on Windows/macOS/Linux | Desktop apps, CLI tools |
| **Team Flexibility** | Contributors use different secret managers | Open source projects, diverse teams |
| **Dev/Staging/Prod Workflow** | Different vaults per environment | Web apps, CI/CD pipelines |
| **Migration** | Zero-downtime vault migration | Changing secret managers |
| **Testing** | Fast tests without real credentials | Unit tests, CI/CD |
| **Vendor Neutrality** | Customer chooses their cloud provider | B2B SaaS, enterprise sales |
| **Hybrid Cloud** | On-premises + cloud workloads | Compliance requirements, regulated industries |
| **CLI Tools** | Respect user's existing vault | Developer tools, dotfiles managers |
| **Disaster Recovery** | Failover when primary vault unavailable | Mission-critical apps, HA requirements |

---

## Next Steps

**Evaluate vaultmux for your use case:**

1. **Identify your scenario** - Which use case matches your needs?
2. **Try locally** - Install vaultmux and test with pass or Bitwarden
3. **Read decision guide** - [Should I Use Vaultmux?](DECISION_GUIDE.md)
4. **Check patterns** - [Common Patterns](PATTERNS.md) for implementation recipes
5. **Plan migration** - [Migration Guides](MIGRATIONS.md) if switching backends

**Still have questions?** Open an issue on GitHub or see the [API Reference](../README.md#api-reference).
