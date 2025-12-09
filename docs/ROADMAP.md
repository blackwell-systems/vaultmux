# Vaultmux Roadmap

> **Current Status:** v0.3.0 - Production-ready with 7 backends (Bitwarden, 1Password, pass, Windows Credential Manager, AWS Secrets Manager, Google Cloud Secret Manager, Azure Key Vault)

This document outlines planned backend integrations and major features for vaultmux.

---

## Supported Backends (v0.3.0)

- **Bitwarden** - Open source, self-hostable (Vaultwarden)
- **1Password** - Enterprise-grade with biometric auth
- **pass** - Unix password store (GPG + git)
- **Windows Credential Manager** - Native Windows integration (v0.2.0)
- **AWS Secrets Manager** - Cloud-native AWS secret storage (v0.2.0)
- **Google Cloud Secret Manager** - Cloud-native GCP secret storage (v0.2.0)
- **Azure Key Vault** - Microsoft Azure secret storage with HSM backing (v0.3.0)

---

## Planned Backends

### High Priority: Windows Native Support

#### Windows Credential Manager
**Status:** ✅ **IMPLEMENTED** (v0.2.0)
**Priority:** High - Critical for Windows users

**Why:**
- Built into every Windows installation (no CLI to install)
- Native OS integration with best security practices
- Zero-cost solution for Windows-based development
- Integrates with Windows Hello and TPM

**Implementation:**
- Use PowerShell cmdlets: `Get-StoredCredential`, `New-StoredCredential`
- Or native Win32 API via `golang.org/x/sys/windows`
- Session: No explicit session (OS-level auth)
- Folders: Credential Manager "targets" (similar to pass directories)

**Challenges:**
- Windows-only (won't work on Linux/macOS)
- Limited to Windows Credential Manager scope
- May need UAC elevation for some operations

**Use Case:**
```go
backend, _ := vaultmux.New(vaultmux.Config{
    Backend: vaultmux.BackendWindowsCredentialManager,
    Prefix:  "MyApp",
})
// Automatically uses Windows Hello / fingerprint for auth
```

**Testing Strategy:**

Since Windows Credential Manager is Windows-specific, we'll use a multi-pronged testing approach:

1. **WSL2 Testing (Primary Path):**
   - WSL2 can call Windows Credential Manager via PowerShell interop
   - Develop on Linux environment, test against real Windows APIs
   - Example: `powershell.exe -Command "Get-StoredCredential -Target 'test'"`
   - This is our primary testing approach since WSL2 is available

2. **Build Tags for Cross-Platform Code:**
   ```go
   // backends/wincred/wincred_windows.go
   //go:build windows

   // backends/wincred/wincred_unix.go
   //go:build !windows
   // Returns clear error: "Windows Credential Manager only available on Windows"
   ```

3. **Mock Backend for Development:**
   - Create mock implementation for unit testing on any platform
   - Fast TDD iteration without Windows dependency
   - Mock simulates Windows Credential Manager behavior

4. **GitHub Actions Windows Runners:**
   - Automated testing on real Windows for CI/CD
   - Free for open source repositories
   - Tests run on `windows-latest` runner

5. **Implementation Approach:**
   - PowerShell cmdlets via `exec.Command("powershell.exe", "-Command", script)`
   - Or native Win32 API via `golang.org/x/sys/windows` package
   - Graceful degradation on non-Windows platforms

**Development Workflow:**
- Local TDD: Use mock backend on Mac/Linux
- Integration testing: WSL2 with real Windows Credential Manager
- CI/CD: GitHub Actions Windows runner
- Final validation: Manual testing on native Windows

This approach allows cross-platform development while ensuring production code is tested against real Windows Credential Manager.

---

### Cloud Provider Secret Managers

#### AWS Secrets Manager
**Status:** ✅ **IMPLEMENTED** (v0.3.0)
**Priority:** High - First SDK-based backend

**Why:**
- De facto standard for AWS deployments
- Automatic rotation, audit logging, versioning
- Integration with IAM roles and permissions
- Pay-per-use pricing (~$0.40/secret/month + $0.05/10k API calls)
- Validates vaultmux interface works with SDKs (not just CLI wrappers)

**Implementation:**
- Use AWS SDK for Go v2: `github.com/aws/aws-sdk-go-v2/service/secretsmanager`
- Session: IAM credentials (from env vars, shared config, or EC2/ECS instance role)
- Item naming: ARN-based or secret name with prefix
- Folders: Use tags (`vaultmux:location=folder-name`) or path prefixes (`prefix/item-name`)
- Region: Configurable, defaults to `AWS_REGION` env var or `us-east-1`

**API Mapping:**
```go
GetItem()    → secretsmanager.GetSecretValue()
CreateItem() → secretsmanager.CreateSecret()
UpdateItem() → secretsmanager.PutSecretValue()
DeleteItem() → secretsmanager.DeleteSecret()
ListItems()  → secretsmanager.ListSecrets() with filters
```

**Challenges:**
- Requires AWS account and IAM setup (solved via localstack for testing)
- Not free for production (but cheap: ~$0.40/secret/month)
- API latency (~100-300ms per call)
- Secret names must be unique within region
- Deleted secrets remain in "pending deletion" state for 7-30 days

**Use Case:**
```go
backend, _ := vaultmux.New(vaultmux.Config{
    Backend: vaultmux.BackendAWSSecretsManager,
    Options: map[string]string{
        "region": "us-west-2",
        "prefix": "myapp/",
    },
})
// Uses IAM credentials from environment
```

**Testing Strategy:**

AWS Secrets Manager requires AWS account and incurs costs, so we'll use local emulation for development:

1. **LocalStack (Primary Testing - Docker-based):**
   - Full AWS service emulation including Secrets Manager
   - Free tier supports all Secrets Manager operations
   - Runs locally via Docker container

   **Setup:**
   ```bash
   # Install localstack
   pip install localstack

   # Start localstack with Secrets Manager
   localstack start -d

   # Or use Docker directly
   docker run --rm -d \
     -p 4566:4566 \
     -e SERVICES=secretsmanager \
     localstack/localstack

   # Configure AWS CLI to use localstack
   export AWS_ENDPOINT_URL=http://localhost:4566
   export AWS_ACCESS_KEY_ID=test
   export AWS_SECRET_ACCESS_KEY=test
   export AWS_REGION=us-east-1

   # Test connection
   aws secretsmanager list-secrets --endpoint-url http://localhost:4566
   ```

   **In Tests:**
   ```go
   // backends/awssecrets/awssecrets_test.go
   func TestWithLocalStack(t *testing.T) {
       if os.Getenv("LOCALSTACK_ENDPOINT") == "" {
           t.Skip("LOCALSTACK_ENDPOINT not set")
       }

       backend, _ := New(map[string]string{
           "endpoint": os.Getenv("LOCALSTACK_ENDPOINT"), // http://localhost:4566
           "region":   "us-east-1",
       }, "")

       // Full CRUD testing against local AWS
   }
   ```

2. **Moto (Python Mock - Alternative):**
   - Python library that mocks AWS services
   - Lighter weight than localstack
   - Can be used as pytest fixture or standalone server

   **Setup:**
   ```bash
   # Install moto
   pip install 'moto[server,secretsmanager]'

   # Start moto server
   moto_server secretsmanager -p 5000

   # Configure endpoint
   export AWS_ENDPOINT_URL=http://localhost:5000
   ```

   **Use Case:** Python-heavy teams or CI environments where localstack is too heavy

3. **AWS SDK Mocking (Unit Tests):**
   - Mock AWS SDK calls directly in Go tests
   - Fast, no external dependencies
   - Limited to SDK behavior testing

   ```go
   // Use aws-sdk-go-v2 middleware for mocking
   type mockSecretsManager struct {
       secrets map[string]string
   }

   func (m *mockSecretsManager) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
       // Return mocked response
   }
   ```

4. **GitHub Actions with LocalStack:**
   ```yaml
   - name: Start LocalStack
     run: |
       docker run -d -p 4566:4566 -e SERVICES=secretsmanager localstack/localstack
       sleep 10  # Wait for LocalStack to be ready

   - name: Run AWS Secrets Manager tests
     env:
       LOCALSTACK_ENDPOINT: http://localhost:4566
     run: go test -v ./backends/awssecrets/...
   ```

5. **Optional: Real AWS Testing (CI only):**
   - Use AWS free tier for integration tests in CI
   - Requires AWS credentials as GitHub secrets
   - Clean up secrets after tests
   - Only run on main branch to minimize costs

**Development Workflow:**
- Local TDD: Use mocked AWS SDK (fast, no setup)
- Integration testing: LocalStack via Docker (comprehensive)
- CI/CD: LocalStack in GitHub Actions (automated)
- Optional: Real AWS testing for final validation (pre-release only)

**IAM Permissions Needed (Production):**
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "secretsmanager:GetSecretValue",
      "secretsmanager:CreateSecret",
      "secretsmanager:PutSecretValue",
      "secretsmanager:DeleteSecret",
      "secretsmanager:ListSecrets",
      "secretsmanager:TagResource"
    ],
    "Resource": "*"
  }]
}
```

**Advantages of This Testing Approach:**
- Zero AWS costs for development
- Full API coverage testing
- Fast iteration (localstack starts in seconds)
- Identical API to real AWS
- No manual AWS account setup required for contributors

---

#### Azure Key Vault
**Status:** ✅ **IMPLEMENTED** (v0.3.0)
**Priority:** High - Third SDK-based backend, completes major cloud provider support

**Why:**
- Native Azure integration for Microsoft cloud deployments
- HSM-backed secret storage (FIPS 140-2 Level 2 validated)
- Enterprise authentication via Azure AD
- Popular in Microsoft/Azure shops
- Soft-delete and purge protection for compliance
- Azure Monitor integration for audit logging

**Implementation:**
- Use Azure SDK for Go: `github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets`
- Session: Azure AD via DefaultAzureCredential (Managed Identity, Service Principal, Azure CLI)
- Item naming: Secret name with prefix (e.g., `myapp-secret-name`)
- Folders: Not supported natively (flat namespace per vault, could use tags in future)
- Vault URL: Required configuration (e.g., `https://myvault.vault.azure.net/`)

**API Mapping:**
```go
GetItem()    → client.GetSecret(name, "", nil)
CreateItem() → client.SetSecret(name, params, nil)
UpdateItem() → client.SetSecret(name, params, nil) // Creates new version
DeleteItem() → client.DeleteSecret(name, nil)
ListItems()  → client.NewListSecretPropertiesPager(nil)
```

**Challenges:**
- Requires Azure subscription and Key Vault setup
- Multiple Azure AD authentication patterns (similar complexity to AWS)
- No local emulator (like GCP, must use real Azure for integration tests)
- RBAC permissions setup (Key Vault Secrets Officer role required)

**Use Case:**
```go
backend, _ := vaultmux.New(vaultmux.Config{
    Backend: vaultmux.BackendAzureKeyVault,
    Options: map[string]string{
        "vault_url": "https://myvault.vault.azure.net/",
        "prefix":    "myapp-",
    },
})
// Uses Azure AD credentials from environment, CLI, or Managed Identity
```

**Testing Strategy:**
- Unit tests with no Azure credentials required (configuration, interface compliance)
- Integration tests skip gracefully when AZURE_VAULT_URL not set
- Real Azure free tier testing (10k operations free per month)
- Azure SDK is interface-based, excellent for mocking

---

#### Google Cloud Secret Manager
**Status:** ✅ **IMPLEMENTED** (v0.2.0)
**Priority:** High - Second SDK-based backend, validates cloud provider universality

**Why:**
- Native GCP integration for Google Cloud deployments
- Automatic encryption at rest with Google-managed keys
- IAM-based access control with fine-grained permissions
- Simpler API than AWS (cleaner SDK design)
- Automatic secret versioning on every update
- First 6 secret versions free per month (~$0.06/version thereafter)
- Validates vaultmux interface works across multiple cloud SDKs

**Implementation:**
- Use Google Cloud Go SDK: `cloud.google.com/go/secretmanager/apiv1`
- Session: Application Default Credentials (ADC) - service account JSON, gcloud CLI, or GCE/GKE metadata
- Item naming: Secret name with prefix (e.g., `vaultmux-item-name`)
- Folders: Not supported natively (could use labels in future)
- Project ID: Required for all operations (specified in config)

**API Mapping:**
```go
GetItem()    → AccessSecretVersion(latest) + GetSecret(metadata)
CreateItem() → CreateSecret() + AddSecretVersion()
UpdateItem() → AddSecretVersion() (creates new version automatically)
DeleteItem() → DeleteSecret()
ListItems()  → ListSecrets() with prefix filter
```

**Challenges:**
- Requires GCP project and IAM setup (no local emulator like AWS LocalStack)
- Not free for production (but cheap: first 6 versions free, $0.06/version after)
- Two-step secret creation: CreateSecret (metadata) then AddSecretVersion (content)
- No native folder/vault concept (flat namespace per project)
- Deleted secrets are immediate (no recovery period like AWS)

**Use Case:**
```go
backend, _ := vaultmux.New(vaultmux.Config{
    Backend: vaultmux.BackendGCPSecretManager,
    Options: map[string]string{
        "project_id": "my-gcp-project",
        "prefix":     "myapp-",
    },
})
// Uses ADC from GOOGLE_APPLICATION_CREDENTIALS or gcloud CLI
```

**Testing Strategy:**

Unlike AWS (LocalStack) or Windows (WSL2), GCP Secret Manager has no good local emulator. Testing approach:

1. **Unit Tests (Primary - No GCP Required):**
   - Test configuration, error handling, interface compliance
   - Mock GCP SDK behavior for fast TDD
   - No GCP project or credentials needed

   ```go
   func TestNew(t *testing.T) {
       // Test project_id validation, prefix defaults, etc.
   }

   func TestBackend_InterfaceCompliance(t *testing.T) {
       var _ vaultmux.Backend = (*Backend)(nil)
   }
   ```

2. **Integration Tests (Skip Without Credentials):**
   - Full CRUD tests against real GCP project
   - Skipped gracefully when GCP_PROJECT_ID not set
   - Can use GCP free tier (first 6 secrets free)

   **Setup:**
   ```bash
   # Create GCP project and enable Secret Manager API
   gcloud services enable secretmanager.googleapis.com

   # Create service account with permissions
   gcloud iam service-accounts create vaultmux-test

   # Grant secretmanager.admin role
   gcloud projects add-iam-policy-binding PROJECT_ID \
     --member="serviceAccount:vaultmux-test@PROJECT_ID.iam.gserviceaccount.com" \
     --role="roles/secretmanager.admin"

   # Download service account key
   gcloud iam service-accounts keys create sa-key.json \
     --iam-account=vaultmux-test@PROJECT_ID.iam.gserviceaccount.com

   # Run integration tests
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/sa-key.json
   export GCP_PROJECT_ID=your-project-id
   go test -v ./backends/gcpsecrets/
   ```

   **In Tests:**
   ```go
   func TestIntegration(t *testing.T) {
       projectID := os.Getenv("GCP_PROJECT_ID")
       if projectID == "" {
           t.Skip("GCP_PROJECT_ID not set - skipping integration tests")
       }

       // Full CRUD testing against real GCP
   }
   ```

3. **GitHub Actions (Optional - Real GCP):**
   - Use GCP service account key as GitHub secret
   - Run integration tests on CI
   - Clean up secrets after tests
   - Only run on main branch to minimize API costs

**IAM Permissions Needed (Production):**
```json
{
  "roles": [
    "roles/secretmanager.admin"
  ],
  "OR_minimum_permissions": [
    "secretmanager.secrets.create",
    "secretmanager.secrets.get",
    "secretmanager.secrets.list",
    "secretmanager.secrets.delete",
    "secretmanager.versions.add",
    "secretmanager.versions.access"
  ]
}
```

**Advantages:**
- Simpler API than AWS (fewer methods, cleaner design)
- Excellent Go SDK quality (Google's official SDKs are top-tier)
- Built-in versioning (no separate version management)
- Fast iteration with real GCP free tier

**Comparison to AWS Secrets Manager:**
- **Simpler**: 5 core methods vs 7 for AWS
- **Faster to implement**: ~3-4 days vs ~5 days for AWS
- **Better SDK**: Google's Go SDKs are cleaner than AWS v2
- **No local emulator**: Must use real GCP (but free tier available)
- **Two-step creation**: CreateSecret + AddSecretVersion vs single AWS CreateSecret

---

### Enterprise Password Managers

#### HashiCorp Vault
**Status:** Under Consideration
**Priority:** Medium-High

**Why:**
- Industry-standard enterprise secret management
- Dynamic secrets, encryption as a service
- Multi-cloud support
- Open source + enterprise editions

**Implementation:**
- Use Vault Go client
- Session: Vault token (from various auth methods)
- Folders: Use Vault secret engines and paths

**Challenges:**
- Requires Vault server installation/subscription
- Complex setup for teams unfamiliar with Vault
- Overkill for simple use cases

**Use Case:**
- Large organizations with existing Vault infrastructure
- Teams needing dynamic secrets or advanced features

---

#### Keeper Security
**Status:** Under Consideration
**Priority:** Medium

**Why:**
- Strong enterprise password manager
- Commander CLI available
- Good Windows/Active Directory integration

**Implementation:**
- Use Keeper Commander CLI
- Similar pattern to Bitwarden backend

**Use Case:**
- Organizations standardized on Keeper

---

### Open Source / Self-Hosted

#### KeePass / KeePassXC
**Status:** Under Consideration
**Priority:** Medium

**Why:**
- Popular open-source password manager
- File-based (local or network drive)
- No cloud dependencies
- Cross-platform

**Implementation:**
- Use `keepassxc-cli` command-line tool
- Or parse `.kdbx` files directly with Go library
- Session: Master password unlock
- Folders: KeePass groups

**Challenges:**
- Database must be accessible on filesystem
- No built-in sync (users handle via Dropbox/git/etc.)
- Multiple database formats (KeePass vs KeePassXC)

**Use Case:**
- Privacy-conscious users
- Air-gapped or offline environments
- Teams using shared network drives

---

#### Doppler
**Status:** Under Consideration
**Priority:** Low-Medium

**Why:**
- Built for developers
- Environment variable management
- Good CI/CD integration

**Implementation:**
- Use Doppler CLI or REST API
- Session: Service token
- Folders: Projects and configs

**Use Case:**
- Development teams managing env vars
- CI/CD secret injection

---

### Consumer Password Managers

#### Dashlane
**Status:** Low Priority
**Priority:** Low

**Why:**
- Popular consumer password manager
- Has CLI tool

**Challenges:**
- CLI tool is business-tier only
- Less developer-focused

---

#### LastPass
**Status:** Low Priority
**Priority:** Low

**Why:**
- Still widely used despite recent issues
- CLI available

**Challenges:**
- Recent security concerns
- Declining popularity in developer community

---

## Backend Feature Matrix

| Backend | Windows Native | Cross-Platform | Self-Host | Free Tier | Session Type | Best For |
|---------|---------------|----------------|-----------|-----------|--------------|----------|
| **Bitwarden** | No | Yes | Yes | Yes | Token | Open source teams |
| **1Password** | No | Yes | No | No | Token + Biometric | Enterprise |
| **pass** | No | Yes (Unix) | Yes | Yes | GPG Agent | Unix power users |
| **Win Credential Mgr** | Yes | No | N/A | Yes | OS Auth | Windows devs |
| **AWS Secrets Mgr** | No | Yes | No | No ($) | IAM | AWS deployments |
| **GCP Secret Mgr** | No | Yes | No | Partial ($) | Service Account | GCP deployments |
| **Azure Key Vault** | No | Yes | No | No ($) | Azure AD | Azure deployments |
| **HashiCorp Vault** | No | Yes | Yes | Yes (OSS) | Token | Enterprise infra |
| **KeePassXC** | No | Yes | Yes | Yes | Master Password | Privacy-focused |

---

## Implementation Priority

### v0.2.0 ✅ RELEASED (2025-12-08)
1. **Windows Credential Manager** - Native Windows support ✅
2. **AWS Secrets Manager** - First SDK-based backend ✅
3. **Google Cloud Secret Manager** - Second SDK-based backend ✅
4. **Error Handling Standardization** - ErrNotSupported sentinel error, consistent WrapError usage ✅
5. Documentation improvements (Docsify site, architecture diagrams) ✅
6. Test coverage improvements (98%+ core coverage) ✅
7. LocalStack testing infrastructure for AWS ✅
8. Application Default Credentials (ADC) patterns for GCP ✅

### v0.3.0 ✅ RELEASED (2025-12-08)
1. **Azure Key Vault** - Completes big three cloud providers ✅
2. Azure AD authentication patterns (Managed Identity, Service Principal, CLI) ✅
3. Vault URL configuration support ✅
4. HSM-backed secret storage with FIPS 140-2 Level 2 validation ✅
5. Soft-delete and purge protection ✅
6. Brand guidelines documentation (BRAND.md) ✅
7. "Why Vaultmux?" motivation section in README ✅

### v0.3.3 ✅ RELEASED (2025-12-08)
1. **Security Fixes** - Critical security improvements ✅
   - Add input validation to prevent command injection in CLI backends ✅
   - Fix session file permissions (0700 for directories, 0600 for files) ✅
   - Fix AutoRefreshSession race condition (add mutex) ✅
   - Implement secure error handling for session operations ✅
2. **Error Handling Improvements** ✅
   - Implement BackendError.Is() for proper sentinel error checking ✅
   - Standardize error wrapping across all backends ✅
   - Add context to error messages ✅

### v0.4.0 🚀 PERFORMANCE & OPTIMIZATION (Planned)
1. **CLI Backend Optimizations**
   - Add batch operations (GetItems, CreateItems) to reduce process spawning
   - Implement status caching (5s TTL) to avoid repeated subprocess calls
   - Explore bw/op server mode for persistent connections
   - Reduce N+1 query patterns in ListItems operations
2. **SDK Backend Improvements**
   - Document concurrency safety guarantees
   - Add connection pooling for high-concurrency scenarios
   - Implement request batching where supported
3. **Pagination Support**
   - Add ListOptions with Limit/Offset/Filter to ListItems()
   - Prevent memory issues with large item collections
   - Support streaming/cursor-based pagination for large datasets

### v0.4.1 ✅ TESTING & QUALITY
1. **Test Coverage Improvements** (Target: 80%+ all backends)
   - Add CLI backend integration tests (conditional on CLI availability)
   - Add concurrent access tests with race detector
   - Add context cancellation tests for all operations
   - Add error path testing for all backends
2. **Benchmarking**
   - Add performance benchmarks for all backends
   - Document performance characteristics
   - Establish performance regression testing
3. **Documentation**
   - Document concurrency safety per backend
   - Add performance comparison guide
   - Add security best practices guide

### v0.5.0 📦 NEW BACKENDS
1. **HashiCorp Vault** - Enterprise option
2. **KeePassXC** - Open source file-based option
3. Dynamic secret support exploration
4. Additional enterprise managers (Keeper, Doppler)

### v0.6.0 ✨ API ENHANCEMENTS
1. **Enhanced Session Management**
   - Fix Session.Token() to accept context parameter
   - Implement session refresh notifications
   - Add session expiration callbacks
2. **Advanced Features**
   - Secret versioning API (where supported)
   - Secret rotation helpers
   - Audit logging hooks
   - Metadata/tags support
3. **Developer Experience**
   - Add OpenTelemetry tracing support
   - Improve error messages with remediation hints
   - Add CLI tool for interactive vault operations

### Future Considerations
- Additional enterprise managers (Doppler, Keeper Security, Dashlane)
- Consumer managers (if demand exists)
- Multi-vault orchestration (read from multiple backends simultaneously)
- Secret synchronization between backends
- Encryption wrapper backends (encrypt-then-store patterns)

---

## Contributing

Want to implement a backend? See [EXTENDING.md](EXTENDING.md) for implementation guide.

**Good first backends:**
- Windows Credential Manager (high impact, clear scope)
- KeePassXC (pure Go implementation possible)

**Complex backends:**
- Cloud provider integration (requires SDK knowledge)
- Enterprise systems (may need enterprise licenses for testing)

---

## Non-Goals

**We will NOT support:**
- Browser extension password managers (no CLI available)
- Deprecated/unmaintained tools
- Platforms without programmatic access

**We WILL support:**
- Any backend with a stable CLI or Go SDK
- Backends with clear authentication patterns
- Tools commonly used in development workflows

---

## Feedback

Have a backend you'd like to see supported? [Open an issue](https://github.com/blackwell-systems/vaultmux/issues) with:
- Backend name and URL
- CLI tool or API availability
- Your use case
- Existing integrations/SDKs

---

**Last Updated:** 2025-12-08
**Current Version:** v0.3.3 (released)
