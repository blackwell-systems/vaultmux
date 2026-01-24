# Should You Use Vaultmux?

A decision framework to help you determine if vaultmux is the right choice for your project.

---

## Quick Decision Tree

```
Do you need to support multiple secret management backends?
├─ YES → Continue reading
└─ NO → Use native SDK directly (skip vaultmux)

Will you deploy to different environments with different vaults?
├─ YES (dev: pass, prod: AWS) → vaultmux is ideal
└─ NO (AWS everywhere) → Consider native SDK

Do you need to test secret logic without real vault credentials?
├─ YES (unit tests, CI/CD) → vaultmux mock backend helps
└─ NO → vaultmux still useful, but less critical

Are you building for open source or diverse teams?
├─ YES (contributors use different tools) → vaultmux enables flexibility
└─ NO → Assess other factors

Might you migrate between secret managers in the future?
├─ YES (avoiding lock-in) → vaultmux makes migration easier
└─ NO → Less benefit, but still portable

Do you want abstraction over vault-specific APIs?
├─ YES (unified interface) → vaultmux provides this
└─ NO (need vault-specific features) → Use native SDK
```

---

## When to Use Vaultmux

### ✓ Use vaultmux when:

#### 1. You Have Multiple Deployment Targets

**Scenario:**
- Development uses `pass` (local, fast, no cloud credentials)
- CI/CD uses mock backend (fast tests, no vault)
- Staging uses AWS Secrets Manager (mirrors production)
- Production uses AWS Secrets Manager (different region/account)

**Why vaultmux helps:**
- Same code works everywhere
- No `if environment == "dev"` branches
- Configuration selects backend

**Example:**
```go
// One function, works in all environments
config := getBackendFromEnv()  // Returns different config per env
backend, _ := vaultmux.New(config)
secret, _ := backend.GetNotes(ctx, "api-key", session)
```

---

#### 2. You Support Multiple Clouds or Platforms

**Scenario:**
- Customer A runs on AWS
- Customer B runs on GCP
- Customer C runs on Azure
- Customer D runs on-premises

**Why vaultmux helps:**
- Support all customers with one codebase
- No per-cloud code paths
- Easy to add new cloud providers

**Without vaultmux:**
```go
// Separate code for each cloud
if customer.Cloud == "AWS" {
    awsSecret := getAWSSecret()
} else if customer.Cloud == "GCP" {
    gcpSecret := getGCPSecret()
} else if customer.Cloud == "Azure" {
    azureSecret := getAzureSecret()
}
```

**With vaultmux:**
```go
// One code path
backend := vaultmux.New(customer.VaultConfig)
secret := backend.GetNotes(ctx, "api-key", session)
```

---

#### 3. You Need Testable Secret Management

**Scenario:**
- Unit tests shouldn't require AWS credentials
- CI/CD should run fast (no network calls)
- Want to test error conditions (vault outage, expired session)

**Why vaultmux helps:**
- Mock backend for tests
- No external dependencies in tests
- Simulate errors easily

**Example:**
```go
func TestApp(t *testing.T) {
    // Create mock backend with test data
    backend := mock.New()
    backend.SetItem("api-key", "test-key-123")
    
    // Test application logic
    app := NewApp(backend)
    app.Initialize(context.Background())
    
    // No AWS credentials needed!
    // Fast tests (no network)
    // Deterministic results
}
```

---

#### 4. You Want Vendor Neutrality

**Scenario:**
- Avoiding lock-in to specific cloud provider
- Want flexibility to switch backends later
- Support customer-provided infrastructure

**Why vaultmux helps:**
- Switch backends with config change
- No code rewrite when migrating
- Abstraction protects from vendor specifics

---

#### 5. You're Building Open Source or Team Tools

**Scenario:**
- Contributors use different secret managers
- Don't want to force specific tooling
- Respect existing workflows

**Why vaultmux helps:**
- Users choose their preferred backend
- Lower adoption barrier
- Works with existing setups

---

#### 6. You Have Cross-Platform Requirements

**Scenario:**
- Desktop app on Windows, macOS, Linux
- Each platform has native credential storage
- Want to integrate with OS features (Windows Hello, Touch ID)

**Why vaultmux helps:**
- Auto-detect platform backend
- Native OS integration
- Consistent API across platforms

---

### ✗ Don't use vaultmux when:

#### 1. You're Committed to a Single Backend

**Scenario:**
- Only use AWS Secrets Manager
- Never plan to switch
- Need AWS-specific features (automatic rotation with Lambda)

**Why not vaultmux:**
- Native SDK (`aws-sdk-go-v2`) has more features
- Direct SDK is slightly faster (no abstraction layer)
- More control over AWS-specific options

**Use instead:** `github.com/aws/aws-sdk-go-v2/service/secretsmanager`

---

#### 2. You Need Vault-Specific Features

**Scenario:**
- HashiCorp Vault dynamic secrets
- AWS Secrets Manager automatic rotation
- Azure Key Vault HSM-backed keys
- GCP Secret Manager automatic replication

**Why not vaultmux:**
- Vaultmux provides lowest-common-denominator API
- Advanced features not exposed
- Direct SDK gives full access

**Use instead:** Native SDK for your chosen vault

---

#### 3. You Want a Secret Management Server

**Scenario:**
- Need centralized secret server
- Want dynamic credentials (database credentials that expire)
- Need secret leasing and renewal
- Want audit trail and access control policies

**Why not vaultmux:**
- Vaultmux is a client library, not a server
- Doesn't provide centralized management
- No dynamic secret generation

**Use instead:** HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager

---

#### 4. Performance is Critical in Tight Loops

**Scenario:**
- Retrieving secrets thousands of times per second
- Ultra-low latency requirements (< 1ms)
- High-frequency trading, real-time systems

**Why not vaultmux:**
- Abstraction adds minimal overhead (but not zero)
- Interface calls slightly slower than direct SDK calls

**Mitigation:** Cache secrets at application level (fetch once, reuse)

**Note:** For most applications, this overhead is negligible (microseconds).

---

#### 5. You Only Use Environment Variables or Config Files

**Scenario:**
- Secrets passed via environment variables
- No vault integration needed
- Simple deployment (single server, no orchestration)

**Why not vaultmux:**
- Overhead of vault abstraction unnecessary
- Environment variables simpler for basic cases

**Use instead:** `os.Getenv()` or config file parsers

**However:** Consider vaultmux if you want:
- Encryption at rest
- Secret rotation
- Audit trail
- Better security than plaintext config

---

## Comparison to Alternatives

### vs HashiCorp Vault

| Aspect | Vaultmux | HashiCorp Vault |
|--------|----------|-----------------|
| **Type** | Client library | Secret management server |
| **Purpose** | Abstract multiple backends | Centralized secret management |
| **Dynamic secrets** | No | Yes (database creds, cloud creds) |
| **Secret leasing** | No | Yes (automatic renewal) |
| **Audit trail** | Backend-dependent | Built-in |
| **Deployment** | Library (no server) | Requires Vault server |
| **Backend support** | 8 backends | One backend (Vault itself) |
| **Use together?** | Yes (Vaultmux can use Vault as a backend) | Yes |

**Use vaultmux when:** You want to support user's existing vaults  
**Use HashiCorp Vault when:** You need centralized server with dynamic secrets

---

### vs Native SDKs (aws-sdk-go, cloud.google.com/go/secretmanager)

| Aspect | Vaultmux | Native SDK |
|--------|----------|------------|
| **Portability** | High (works with 8 backends) | Low (locked to one vendor) |
| **Features** | Lowest common denominator | Full vendor feature set |
| **Performance** | Slight overhead (interface calls) | Direct (no abstraction) |
| **Testing** | Mock backend included | Must mock SDK yourself |
| **Learning curve** | One API for all | Learn each SDK separately |
| **Vendor lock-in** | None | High |

**Use vaultmux when:** You need portability, flexibility, or testing  
**Use native SDK when:** You're committed to one backend and need all features

---

### vs Environment Variables

| Aspect | Vaultmux | Environment Variables |
|--------|----------|----------------------|
| **Security** | Encrypted at rest | Plaintext |
| **Rotation** | Supported (backend-dependent) | Manual |
| **Audit trail** | Backend-dependent | None |
| **Visibility** | Not in process list | Visible in `ps aux` |
| **Access control** | IAM/RBAC (backend-dependent) | File permissions only |
| **Simplicity** | Requires vault setup | Very simple |

**Use vaultmux when:** You need encryption, rotation, or audit trail  
**Use env vars when:** Simple deployment, no security requirements

---

### vs Config Files (.env, .yaml, .json)

| Aspect | Vaultmux | Config Files |
|--------|----------|--------------|
| **Security** | Encrypted at rest | Plaintext (easily committed to git) |
| **Rotation** | Supported | Manual file edits |
| **Accidental exposure** | Low (not in repo) | High (easy to commit) |
| **Distribution** | Vault handles it | Must distribute files securely |
| **Versioning** | Backend-dependent | Git (but exposes secrets) |

**Use vaultmux when:** You need secure secret storage  
**Use config files when:** Local development only (never production!)

---

## Decision Matrix

### Your Project Characteristics

Rate each factor (High/Medium/Low/None):

| Factor | Rating | Vaultmux Benefit |
|--------|--------|------------------|
| **Multiple deployment environments** (dev/staging/prod) | ? | High → Strong benefit |
| **Multiple cloud providers** (AWS, GCP, Azure) | ? | High → Strong benefit |
| **Team diversity** (different secret managers) | ? | High → Strong benefit |
| **Portability needs** (might switch backends) | ? | High → Strong benefit |
| **Testing requirements** (mock backend useful) | ? | High → Strong benefit |
| **Cross-platform** (Windows/macOS/Linux) | ? | High → Strong benefit |
| **Committed to one backend** (only AWS forever) | ? | High → Weak benefit |
| **Need vendor-specific features** (dynamic secrets) | ? | High → Not suitable |

**Scoring:**
- 4+ "Strong benefit" → Vaultmux is excellent fit
- 2-3 "Strong benefit" → Vaultmux is good fit
- 1 "Strong benefit" → Vaultmux may help, assess trade-offs
- 0 "Strong benefit" → Consider native SDK instead

---

## Common Scenarios

### Scenario 1: Startup Building MVP

**Context:**
- Small team (2-5 developers)
- Using AWS for everything
- Move fast, break things
- No immediate multi-cloud plans

**Recommendation:** Start with native AWS SDK

**Why:**
- Simpler (one less abstraction)
- Full AWS feature access
- Faster iteration (less to learn)

**However:** Consider vaultmux if:
- Developers want to use `pass` locally (avoid AWS creds on laptop)
- Planning open source release (users might not use AWS)
- Want easy testing (mock backend helpful)

---

### Scenario 2: Enterprise B2B SaaS

**Context:**
- Selling to Fortune 500 customers
- Customers demand vendor flexibility
- Some require on-premises deployment
- Compliance requirements vary by customer

**Recommendation:** Use vaultmux from day one

**Why:**
- Customer A requires AWS → vaultmux supports
- Customer B requires GCP → vaultmux supports
- Customer C requires on-prem → vaultmux supports
- Sales advantage (vendor neutral positioning)

---

### Scenario 3: Open Source CLI Tool

**Context:**
- Public GitHub repository
- Contributors worldwide
- Diverse tooling preferences
- Want high adoption

**Recommendation:** Use vaultmux

**Why:**
- Don't force contributors to install specific vault
- Let users choose their preferred backend
- Lower barrier to contribution
- Respects existing workflows

---

### Scenario 4: Internal Corporate Tool

**Context:**
- Company standardized on HashiCorp Vault
- All teams use Vault
- No plans to change
- Need Vault-specific features (dynamic DB creds)

**Recommendation:** Use HashiCorp Vault SDK directly

**Why:**
- Standardized on one backend (no need for abstraction)
- Need Vault-specific features (vaultmux doesn't expose)
- Internal tool (no external flexibility requirements)

---

### Scenario 5: Migration Project

**Context:**
- Currently using Bitwarden
- Migrating to AWS Secrets Manager over 6 months
- Can't afford downtime
- Need gradual rollout

**Recommendation:** Use vaultmux for migration

**Why:**
- Dual-read pattern (try AWS, fallback to Bitwarden)
- Zero-downtime migration
- Easy rollback if issues
- Same code works with both backends

---

## Cost Considerations

### Vaultmux Costs

**Development time:**
- Initial: +2-4 hours (learning vaultmux API)
- Ongoing: Minimal (one API for all backends)

**Runtime overhead:**
- Performance: < 1% slower than native SDK (interface calls)
- Memory: Negligible (thin abstraction layer)

**Dependencies:**
- Core: Zero dependencies (standard library only)
- Backends: Optional (AWS SDK, GCP SDK via go modules)

### Alternative Costs

**Native SDK approach:**
- Initial: 1 hour per backend (AWS, GCP, Azure = 3 hours)
- Ongoing: High (maintain separate code paths, test all)
- Migration: High (rewrite code to switch backends)

**No abstraction approach:**
- Initial: Low (direct implementation)
- Ongoing: Very high (vendor lock-in, hard to test, hard to migrate)
- Migration: Very high (complete rewrite)

---

## Risk Assessment

### Risks of Using Vaultmux

**Minimal abstraction overhead:**
- Risk: Slightly slower than native SDK
- Mitigation: Cache secrets at application level
- Impact: Negligible for most applications (microseconds)

**Lowest common denominator API:**
- Risk: Can't use vendor-specific features
- Mitigation: Use native SDK for vendor-specific needs
- Impact: Only matters if you need advanced features

**Dependency on vaultmux project:**
- Risk: Project might be abandoned
- Mitigation: Apache 2.0 license (can fork), small codebase (easy to maintain)
- Impact: Low (stable API, infrequent breaking changes)

### Risks of Not Using Vaultmux

**Vendor lock-in:**
- Risk: Tied to specific vault (AWS, GCP, etc.)
- Impact: High switching costs if you need to migrate
- Mitigation: None (rewrite code to switch)

**Testing difficulty:**
- Risk: Unit tests require real vault credentials
- Impact: Slow tests, flaky CI/CD, security issues (creds in CI)
- Mitigation: Manual mocking (tedious)

**Code duplication:**
- Risk: Separate code for each backend
- Impact: High maintenance burden, more bugs
- Mitigation: Abstraction layer (reinventing vaultmux)

**Limited flexibility:**
- Risk: Can't support users with different vaults
- Impact: Missed opportunities (customers, contributors)
- Mitigation: Major refactor to add flexibility later

---

## Migration Path

### From Native SDK to Vaultmux

**Step 1:** Add vaultmux dependency
```bash
go get github.com/blackwell-systems/vaultmux
```

**Step 2:** Create wrapper (compatibility layer)
```go
// Existing code:
secret := awsGetSecret("api-key")

// New wrapper:
func getSecret(name string) string {
    backend, _ := vaultmux.New(vaultmux.Config{Backend: vaultmux.BackendAWS})
    defer backend.Close()
    backend.Init(context.Background())
    session, _ := backend.Authenticate(context.Background())
    secret, _ := backend.GetNotes(context.Background(), name, session)
    return secret
}

// Update callsites gradually:
secret := getSecret("api-key")
```

**Step 3:** Migrate callsites incrementally

**Step 4:** Remove native SDK dependency

**Effort:** Low (drop-in replacement in most cases)

### From Vaultmux to Native SDK

**Why migrate back:**
- Need vendor-specific features vaultmux doesn't expose
- Committed to single backend long-term
- Want direct SDK control

**Step 1:** Add native SDK dependency

**Step 2:** Replace vaultmux calls with native SDK calls

**Step 3:** Remove vaultmux dependency

**Effort:** Medium (must learn native SDK API)

---

## Final Checklist

Before deciding, ask yourself:

**✓ Consider vaultmux if:**
- [ ] I deploy to multiple environments (dev/staging/prod)
- [ ] I might support multiple clouds (AWS, GCP, Azure)
- [ ] I want testable secret management (mock backend)
- [ ] I'm building open source or team tools (user choice)
- [ ] I might migrate between backends someday
- [ ] I want vendor neutrality
- [ ] I need cross-platform support (Windows/macOS/Linux)

**✗ Skip vaultmux if:**
- [ ] I'm committed to one backend forever (AWS only)
- [ ] I need vendor-specific advanced features (dynamic secrets)
- [ ] I want a secret management server (use HashiCorp Vault)
- [ ] Performance in tight loops is critical (< 1ms latency)
- [ ] I only use environment variables (no vault integration)

**Still unsure?**
- Try vaultmux locally with `pass` or `bitwarden`
- Read [Use Cases](USE_CASES.md) to see if your scenario matches
- Check the [Quick Start](../README.md#quick-start) for implementation examples
- Open a GitHub issue with your specific scenario

---

## Summary

**Vaultmux is ideal for:**
- Projects with multiple deployment targets
- Multi-cloud or multi-platform applications
- Open source projects with diverse users
- Teams wanting vault flexibility
- Applications needing testable secret management
- Scenarios where vendor lock-in is a concern

**Vaultmux is not ideal for:**
- Projects committed to single backend with no flexibility needs
- Applications requiring vendor-specific advanced features
- Scenarios where you need a secret management server (not just a client library)

**The key question:** Do you need flexibility and portability in secret management?
- **Yes** → Vaultmux is a great fit
- **No** → Consider native SDK for your chosen backend

**Next steps:**
1. [Browse use cases](USE_CASES.md) to find your scenario
2. [Try the quick start](../README.md#quick-start) to see if vaultmux fits your workflow
3. [Read the backend comparison](BACKEND_COMPARISON.md) to understand trade-offs
4. [Check the architecture guide](ARCHITECTURE.md) for design patterns
