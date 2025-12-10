# Vaultmux Roadmap

> **Current Status:** v1.0.0-rc1 - Stable API, production-ready with 7 backends (Bitwarden, 1Password, pass, Windows Credential Manager, AWS Secrets Manager, Google Cloud Secret Manager, Azure Key Vault)

This document outlines planned backend integrations and major features for vaultmux.

---

## Supported Backends (v1.0.0-rc1)

- **Bitwarden** - Open source, self-hostable (Vaultwarden)
- **1Password** - Enterprise-grade with biometric auth
- **pass** - Unix password store (GPG + git)
- **Windows Credential Manager** - Native Windows integration (v0.2.0)
- **AWS Secrets Manager** - Cloud-native AWS secret storage (v0.2.0)
- **Google Cloud Secret Manager** - Cloud-native GCP secret storage (v0.2.0)
- **Azure Key Vault** - Microsoft Azure secret storage with HSM backing (v0.3.0)

---

## Planned Features

### v1.1.0 - Secret Rotation Automation (Q1 2026)

**Priority: HIGH**

- Rotation policies and schedules
- Rotation engine (background service)
- Provider plugins for databases, API keys, certificates
- Rotation workflow: generate → update → distribute → verify → revoke
- Configuration via `.vaultmux/rotation.yaml`

**Estimated Effort:** 6-8 weeks

### v1.2.0 - Web UI (Q2 2026)

**Priority: HIGH**

- Browser-based secret management
- Multi-backend aggregated view
- Search and filtering
- Secret history timeline
- Audit logging
- Built with Go backend + Svelte frontend

**Estimated Effort:** 8-10 weeks

### v1.3.0 - Secret Scanning (Q2 2026)

**Priority: MEDIUM**

- Scan git history for leaked secrets
- File system scanning
- Pattern detection (AWS keys, API keys, etc.)
- CI/CD integration
- Remediation suggestions

**Estimated Effort:** 4-6 weeks

### v1.4.0 - Secret Synchronization (Q3 2026)

**Priority: MEDIUM**

- Sync secrets between vaults
- One-way and bidirectional sync
- Selective sync by prefix/tags
- Continuous sync with change detection

**Estimated Effort:** 5-7 weeks

### v1.5.0 - Observability (Q3 2026)

**Priority: MEDIUM**

- Prometheus metrics endpoint
- OpenTelemetry tracing
- Structured logging
- Health checks
- Backend availability monitoring

**Estimated Effort:** 3-4 weeks

---

## Planned Backends

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

**We will NOT do:**
- Browser extension password managers (no CLI available)
- Deprecated/unmaintained tools
- Platforms without programmatic access
- Built-in vault replacement (we're an adapter, not a vault)

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

**Last Updated:** 2025-12-10
**Current Version:** v1.0.0-rc1 (release candidate)
