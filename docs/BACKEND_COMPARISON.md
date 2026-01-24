# Backend Comparison

Detailed comparison of all backends supported by vaultmux.

---

## Feature Comparison Matrix

| Feature | Bitwarden | 1Password | pass | Windows Cred Mgr | AWS Secrets Manager | GCP Secret Manager | Azure Key Vault |
|---------|-----------|-----------|------|------------------|---------------------|--------------------|--------------------|
| **Integration** | CLI (`bw`) | CLI (`op`) | CLI (`pass`) | PowerShell | SDK (aws-sdk-go-v2) | SDK (cloud.google.com/go) | SDK (azure-sdk-for-go) |
| **Auth Method** | Email/password + 2FA | Account + biometrics | GPG key | Windows Hello/PIN | IAM credentials | ADC (Application Default) | Azure AD / Managed Identity |
| **Session Duration** | Until lock | 30 minutes | GPG agent TTL | OS-managed | Long-lived (IAM keys) | Token-based (ADC refresh) | Token-based (AD refresh) |
| **Sync** | `bw sync` | Automatic | `pass git pull/push` | N/A | Always synchronized | Always synchronized | Always synchronized |
| **Offline Mode** | Yes (cached) | Limited | Yes (local files) | Yes (local only) | No (requires AWS API) | No (requires GCP API) | No (requires Azure API) |
| **Folders/Organization** | Folders (folderId) | Vaults | Directories | Prefix-based | Prefix-based + tags | Labels + versions | Tags + versions |
| **Sharing** | Organizations | Vaults | Git repos | Machine-local | IAM policies | IAM bindings | RBAC (role assignments) |
| **Free Tier** | Yes | No | Yes (FOSS) | Yes (built-in) | No (~$0.40/secret/month) | No (pricing per access) | No (pricing per operation) |
| **Self-Host** | Yes (Vaultwarden) | No | Yes (any git host) | N/A (OS feature) | No (AWS only) | No (GCP only) | No (Azure only) |
| **Platform** | All | All | Unix | Windows | All | All | All |
| **Encryption** | AES-256 (client-side) | AES-256 (client-side) | GPG (user's key) | Windows DPAPI | AES-256 (AWS KMS) | AES-256 (Google KMS) | AES-256 (Azure KMS) |
| **Audit Trail** | Premium feature | Business plans | Git history | Windows Event Log | CloudTrail | Cloud Audit Logs | Azure Monitor |
| **Secret Rotation** | Manual | Manual | Manual | Manual | Automatic (Lambda) | Manual (custom automation) | Automatic (Key Vault rotation) |
| **Versioning** | Limited | Yes | Git-based | No | Yes (automatic) | Yes (automatic) | Yes (automatic) |
| **Cost (est.)** | Free (personal) | $7.99/user/month | Free | Free | $0.40/secret/month + $0.05/10k requests | ~$0.06/10k accesses | ~$0.03/10k operations |

---

## Integration Details

### CLI-Based Backends

These backends use command-line tools installed on the system:

#### Bitwarden (`bw`)
- **Installation:** `npm install -g @bitwarden/cli`
- **Authentication:** Interactive (email/password + 2FA)
- **Session Management:** Session tokens cached in file
- **Performance:** Moderate (spawns CLI process per operation)
- **Best For:** Personal use, small teams, self-hosted deployments

#### 1Password (`op`)
- **Installation:** Download from https://1password.com/downloads/command-line/
- **Authentication:** Interactive (Touch ID, Face ID, or password)
- **Session Management:** 30-minute tokens, auto-refresh
- **Performance:** Fast (optimized CLI)
- **Best For:** macOS/iOS users, teams already using 1Password

#### pass (`pass` + `gpg`)
- **Installation:** `apt-get install pass` (Debian/Ubuntu), `brew install pass` (macOS)
- **Authentication:** GPG key passphrase (cached by gpg-agent)
- **Session Management:** GPG agent handles caching
- **Performance:** Very fast (local files, no network)
- **Best For:** Unix developers, git-based workflows, offline environments

#### Windows Credential Manager (PowerShell)
- **Installation:** Built into Windows (no installation)
- **Authentication:** Windows Hello (fingerprint, face, PIN)
- **Session Management:** OS-managed (persistent)
- **Performance:** Fast (native OS API)
- **Best For:** Windows applications, desktop apps

---

### SDK-Based Backends

These backends use native Go SDKs (no external CLI required):

#### AWS Secrets Manager
- **Installation:** Go SDK via `go get` (automatic)
- **Authentication:** IAM credentials (environment, ~/.aws/credentials, instance role)
- **Session Management:** Long-lived credentials (hours/days)
- **Performance:** Fast (native SDK, connection pooling)
- **Best For:** AWS deployments, cloud-native applications, multi-region setups

#### GCP Secret Manager
- **Installation:** Go SDK via `go get` (automatic)
- **Authentication:** Application Default Credentials (gcloud auth, service accounts)
- **Session Management:** ADC token refresh (automatic)
- **Performance:** Fast (native SDK)
- **Best For:** GCP deployments, Google Cloud Platform integration

#### Azure Key Vault
- **Installation:** Go SDK via `go get` (automatic)
- **Authentication:** Azure AD (DefaultAzureCredential, managed identities)
- **Session Management:** AD token refresh (automatic)
- **Performance:** Fast (native SDK)
- **Best For:** Azure deployments, Microsoft 365 integration

---

## Authentication Comparison

### Interactive Authentication (Requires User Input)

| Backend | Method | User Experience |
|---------|--------|-----------------|
| **Bitwarden** | Email/password + 2FA code | User types credentials, 2FA code |
| **1Password** | Touch ID / Face ID / Password | Biometric prompt or password entry |
| **pass** | GPG passphrase | User types GPG key passphrase (once per session) |
| **Windows Credential Manager** | Windows Hello | Fingerprint, face scan, or PIN |

**Use Case:** Local development, desktop applications, CLI tools

---

### Non-Interactive Authentication (No User Input)

| Backend | Method | Configuration |
|---------|--------|--------------|
| **AWS Secrets Manager** | IAM credentials | Environment vars, ~/.aws/credentials, EC2 instance role |
| **GCP Secret Manager** | Application Default Credentials | gcloud auth, GOOGLE_APPLICATION_CREDENTIALS |
| **Azure Key Vault** | Managed Identity / Service Principal | AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET |

**Use Case:** Production deployments, CI/CD pipelines, serverless functions

---

## Session Management

### Short-Lived Sessions (Re-authentication Required)

| Backend | Session Duration | Refresh Mechanism |
|---------|-----------------|-------------------|
| **1Password** | 30 minutes | Auto-prompt for credentials |
| **pass** | GPG agent TTL (default: 10 min) | GPG agent re-prompts |

---

### Long-Lived Sessions (Persistent)

| Backend | Session Duration | Refresh Mechanism |
|---------|-----------------|-------------------|
| **Bitwarden** | Until explicit lock/logout | Manual lock required |
| **Windows Credential Manager** | Until Windows logoff | OS-managed |
| **AWS Secrets Manager** | Hours/days (IAM key) | IAM key rotation (manual/automatic) |
| **GCP Secret Manager** | Token-based (hours) | ADC auto-refresh |
| **Azure Key Vault** | Token-based (hours) | Azure AD auto-refresh |

---

## Offline Support

### Full Offline (No Network Required)

- **pass:** Fully offline (local files encrypted with GPG)
- **Windows Credential Manager:** Fully offline (OS-level storage)

### Partial Offline (Cached)

- **Bitwarden:** Can read cached vault after `bw sync` (requires network for initial sync)
- **1Password:** Limited offline access (requires periodic authentication)

### No Offline (Network Required)

- **AWS Secrets Manager:** Always requires AWS API access
- **GCP Secret Manager:** Always requires GCP API access
- **Azure Key Vault:** Always requires Azure API access

---

## Cost Analysis

### Free Options

| Backend | Cost | Limitations |
|---------|------|-------------|
| **Bitwarden** | Free (personal) | 1 user, basic features |
| **pass** | Free (FOSS) | Self-hosted, requires GPG setup |
| **Windows Credential Manager** | Free (built-in) | Windows only, machine-local |

---

### Paid Options

| Backend | Pricing Model | Estimated Monthly Cost (1000 secrets, 100k accesses) |
|---------|--------------|------------------------------------------------------|
| **1Password** | Per user | $7.99/user/month |
| **Bitwarden Premium** | Per user | $10/user/year |
| **AWS Secrets Manager** | Per secret + per access | $400 (storage) + $0.50 (API calls) = ~$400.50 |
| **GCP Secret Manager** | Per access | ~$6 |
| **Azure Key Vault** | Per operation | ~$3 |

**Note:** Cloud vault costs scale with usage. For high-volume applications, consider caching strategies.

---

## Security Features

### Encryption at Rest

| Backend | Encryption Method | Key Management |
|---------|-------------------|----------------|
| **Bitwarden** | AES-256 (client-side) | User's master password |
| **1Password** | AES-256 (client-side) | Secret Key + Master Password |
| **pass** | GPG (user's key) | User's GPG key |
| **Windows Credential Manager** | Windows DPAPI | OS-managed keys |
| **AWS Secrets Manager** | AES-256 (AWS KMS) | AWS-managed keys (or customer-managed) |
| **GCP Secret Manager** | AES-256 (Google KMS) | Google-managed keys |
| **Azure Key Vault** | AES-256 (Azure KMS) | Azure-managed keys (or HSM-backed) |

---

### Audit Trail

| Backend | Audit Capability | Details |
|---------|------------------|---------|
| **Bitwarden** | Premium feature | Event logs (who accessed what, when) |
| **1Password** | Business plans | Activity logs, sign-in attempts |
| **pass** | Git history | Git commits show changes (who, when, what) |
| **Windows Credential Manager** | Windows Event Log | System events (credential access) |
| **AWS Secrets Manager** | CloudTrail | Full audit trail (access, modifications, deletions) |
| **GCP Secret Manager** | Cloud Audit Logs | Access logs, admin activity logs |
| **Azure Key Vault** | Azure Monitor | Access logs, diagnostic logs |

---

## Use Case Recommendations

### Development (Local Machine)

**Best Choices:**
1. **pass** - Fast, offline, no dependencies
2. **Bitwarden** - If already using for personal passwords
3. **1Password** - If team already uses 1Password

**Avoid:** Cloud vaults (AWS, GCP, Azure) - unnecessary for local dev, costs money

---

### CI/CD Pipelines

**Best Choices:**
1. **Mock backend** - For unit tests (no vault needed)
2. **AWS/GCP/Azure** - For integration tests (use CI secrets for credentials)

**Avoid:** Interactive backends (Bitwarden, 1Password, pass) - require user input

---

### Production (Cloud)

**Best Choices:**
1. **AWS Secrets Manager** - If deployed on AWS
2. **GCP Secret Manager** - If deployed on GCP
3. **Azure Key Vault** - If deployed on Azure

**Why:** Native integration, IAM-based access, automatic rotation, audit trails

---

### Production (On-Premises)

**Best Choices:**
1. **Bitwarden (self-hosted Vaultwarden)** - Full-featured, no cloud dependency
2. **pass** - Lightweight, git-based, no server required

**Avoid:** Cloud vaults (AWS, GCP, Azure) - require internet, monthly costs

---

### Desktop Applications

**Best Choices:**
1. **Windows Credential Manager** - On Windows
2. **1Password** - On macOS (Touch ID integration)
3. **pass** - On Linux (standard Unix tool)

**Why:** Native OS integration, biometric authentication, user-friendly

---

### Open Source Projects

**Best Choices:**
1. **pass** - Most contributors already have it
2. **Bitwarden** - Free, cross-platform

**Avoid:** Paid options (1Password) - excludes contributors without subscription

---

## Migration Complexity

### Easy Migrations (Low Effort)

- **pass → Bitwarden:** Export pass entries, import to Bitwarden
- **Bitwarden → 1Password:** Built-in import tools
- **Local vaults → Cloud vaults:** Copy secrets programmatically with vaultmux

---

### Moderate Migrations (Medium Effort)

- **Any vault → AWS/GCP/Azure:** Write migration script, replicate secrets
- **Cloud vault → Cloud vault:** Cross-cloud replication script

---

### Difficult Migrations (High Effort)

- **Custom vault → vaultmux:** Implement custom backend (see [EXTENDING.md](EXTENDING.md))

---

## Performance Benchmarks

**Disclaimer:** Benchmarks are approximate and vary by hardware, network, and configuration.

| Backend | Init (ms) | Authenticate (ms) | Get Secret (ms) | Create Secret (ms) |
|---------|-----------|-------------------|-----------------|--------------------|
| **Mock** | < 1 | < 1 | < 1 | < 1 |
| **pass** | 5-10 | 10-50 (GPG prompt) | 10-20 | 20-30 |
| **Bitwarden** | 50-100 | 500-1000 (network) | 100-200 | 200-300 |
| **1Password** | 50-100 | 200-500 | 50-100 | 100-200 |
| **Windows Credential Manager** | 10-20 | 50-100 | 20-30 | 30-50 |
| **AWS Secrets Manager** | 100-200 | 200-500 | 100-200 | 200-400 |
| **GCP Secret Manager** | 100-200 | 200-500 | 100-200 | 200-400 |
| **Azure Key Vault** | 100-200 | 200-500 | 100-200 | 200-400 |

**Key Takeaway:** Local backends (pass, Windows Credential Manager) are fastest. Cloud backends have network latency.

---

## Summary Table

| Backend | Best For | Pros | Cons |
|---------|----------|------|------|
| **Bitwarden** | Personal use, small teams | Free, cross-platform, self-hostable | Slower than native tools |
| **1Password** | macOS users, enterprises | Fast, polished UX, biometrics | Paid, not self-hostable |
| **pass** | Unix developers | Fast, offline, git-based | Unix only, requires GPG setup |
| **Windows Credential Manager** | Windows apps | Native, fast, free | Windows only |
| **AWS Secrets Manager** | AWS deployments | IAM integration, rotation, audit | Costs scale with secrets |
| **GCP Secret Manager** | GCP deployments | ADC integration, versioning | Costs scale with accesses |
| **Azure Key Vault** | Azure deployments | HSM-backed, RBAC, rotation | Costs scale with operations |

---

## Next Steps

- **[Use Cases](USE_CASES.md)** - See real-world scenarios
- **[Decision Guide](DECISION_GUIDE.md)** - Choose the right backend for your project
- **[Main README](../README.md)** - Installation and quick start
