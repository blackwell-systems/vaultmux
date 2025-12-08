# Vaultmux Roadmap

> **Current Status:** v0.1.0 - Production-ready with 3 backends (Bitwarden, 1Password, pass)

This document outlines planned backend integrations and major features for vaultmux.

---

## Supported Backends (v0.1.0)

- **Bitwarden** - Open source, self-hostable (Vaultwarden)
- **1Password** - Enterprise-grade with biometric auth
- **pass** - Unix password store (GPG + git)

---

## Planned Backends

### High Priority: Windows Native Support

#### Windows Credential Manager
**Status:** Planned for v0.2.0
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

---

### Cloud Provider Secret Managers

#### AWS Secrets Manager
**Status:** Under Consideration
**Priority:** Medium-High

**Why:**
- De facto standard for AWS deployments
- Automatic rotation, audit logging
- Integration with IAM roles and permissions
- Pay-per-use pricing

**Implementation:**
- Use AWS SDK for Go v2
- Session: IAM credentials (from env, profile, or EC2 instance role)
- Folders: Use tags or secret name prefixes

**Challenges:**
- Requires AWS account and IAM setup
- Not free (though cheap for small usage)
- Latency for API calls

**Use Case:**
- Applications running on AWS (EC2, ECS, Lambda)
- Teams already using AWS infrastructure

---

#### Azure Key Vault
**Status:** Under Consideration
**Priority:** Medium-High

**Why:**
- Native Azure integration
- HSM-backed key storage
- Enterprise authentication (Azure AD)
- Popular in Microsoft/Azure shops

**Implementation:**
- Use Azure SDK for Go
- Session: Azure AD authentication
- Folders: Use Key Vault "vaults" or tags

**Challenges:**
- Requires Azure subscription
- Complex authentication setup
- API latency

**Use Case:**
- Azure-deployed applications
- Organizations using Azure AD
- Enterprise Windows + Azure environments

---

#### Google Cloud Secret Manager
**Status:** Under Consideration
**Priority:** Medium

**Why:**
- Native GCP integration
- Automatic encryption at rest
- IAM-based access control

**Implementation:**
- Use Google Cloud SDK for Go
- Session: Service account or ADC (Application Default Credentials)
- Folders: Use labels or naming conventions

**Use Case:**
- GCP-deployed applications
- Google Workspace organizations

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
| **Azure Key Vault** | No | Yes | No | No ($) | Azure AD | Azure deployments |
| **HashiCorp Vault** | No | Yes | Yes | Yes (OSS) | Token | Enterprise infra |
| **KeePassXC** | No | Yes | Yes | Yes | Master Password | Privacy-focused |

---

## Implementation Priority

### v0.2.0 (Next Release)
1. **Windows Credential Manager** - Address Windows support gap
2. Documentation improvements
3. Integration testing with real CLIs

### v0.3.0
1. **AWS Secrets Manager** - Cloud-native option
2. **Azure Key Vault** - Azure ecosystem support

### v0.4.0
1. **HashiCorp Vault** - Enterprise option
2. **KeePassXC** - Open source file-based option

### Future
- Additional cloud providers (GCP)
- Additional enterprise managers (Keeper, Doppler)
- Consumer managers (if demand exists)

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

**Last Updated:** 2025-12-07
**Current Version:** v0.1.0
