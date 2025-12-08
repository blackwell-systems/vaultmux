# Changelog

All notable changes to vaultmux will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2025-12-08

### Added

- **Azure Key Vault Backend** - Third SDK-based backend (completes major cloud provider support)
  - Native Azure SDK for Go integration (`github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets`)
  - Azure AD authentication via DefaultAzureCredential (Managed Identity, Service Principal, Azure CLI)
  - HSM-backed secret storage (FIPS 140-2 Level 2 validated)
  - Azure RBAC integration (Key Vault Secrets Officer role)
  - Secret name prefixing for namespace isolation (`myapp-secret-name`)
  - Automatic versioning on every update (built-in to Azure)
  - Soft-delete and purge protection for compliance
  - Pager-based pagination for secret listing
  - Azure Monitor audit logging integration
  - Comprehensive error mapping (Azure ResponseError → vaultmux errors)
  - No session management complexity (credentials are long-lived or SDK-managed)
- **Azure Testing Strategy** - Third backend without local emulator
  - Unit tests with no Azure credentials required (configuration, interface compliance)
  - Integration tests skip gracefully when AZURE_VAULT_URL not set
  - Real Azure free tier testing (10k operations free per month)
  - Azure AD credential documentation (Environment variables, CLI, Managed Identity)
  - Alternative mocking via Azure SDK interfaces
- **Documentation Enhancements**
  - README.md: Added "Why Vaultmux?" section explaining dotfiles framework motivation
  - README.md: Updated to include Azure Key Vault in all backend tables and comparisons
  - docs/_coverpage.md: Updated from 6 to 7 backends
  - ARCHITECTURE.md: Added Azure to feature matrix, implementation diagram, and "When to Use" guidance
  - Azure Key Vault comparison highlighting HSM backing, RBAC, and soft-delete features

### Changed

- **go.mod** - Added Azure SDK dependencies
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore` v1.20.0
  - `github.com/Azure/azure-sdk-for-go/sdk/azidentity` v1.13.1
  - `github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets` v1.4.0
- **factory.go** - Added `BackendAzureKeyVault` constant
  - Updated Config comment to include "azurekeyvault"
  - Maintains backward compatibility
- **vaultmux.go** - Updated package documentation
  - SDK-based backends now includes Azure Key Vault
  - Updated Backend interface comment to list all 7 backends
  - Added Azure Key Vault configuration example

### Technical Details

- **Pattern Validation**: Third SDK-based backend (after AWS and GCP) confirms universal cloud provider support
- **Session Semantics**: Azure AD credentials (Managed Identity, Service Principal, CLI) validate flexible auth patterns
- **Error Handling**: Azure ResponseError with HTTP status codes map cleanly to vaultmux standard errors
- **Testing Strategy**: Real Azure testing (no emulator) with graceful skip when credentials unavailable
- **Integration Type**: Native SDK (not CLI subprocess) - same pattern as AWS and GCP
- **API Design**: Azure SDK uses interface-based design, excellent for mocking and testing

### Comparison to AWS/GCP Implementations

- **Authentication Complexity**: Multiple Azure AD credential types (similar complexity to AWS)
- **API Simplicity**: Synchronous operations (simpler than AWS, similar to GCP)
- **Unique Features**: HSM-backed storage, soft-delete protection, certificate management
- **SDK Quality**: Interface-based design (better for testing than AWS)
- **No Local Emulator**: Must use real Azure (like GCP, but free tier makes this viable)
- **Pager Pattern**: Iterator-based pagination (similar to GCP, different from AWS)

## [0.2.0] - 2025-12-07

### Added

- **Windows Credential Manager Backend** - Native Windows support without external dependencies
  - Uses PowerShell cmdlets (Get-StoredCredential, New-StoredCredential, Remove-StoredCredential)
  - OS-level authentication with Windows Hello and biometric support
  - Cross-platform build tags (`wincred_windows.go`, `wincred_unix.go`)
  - No session management needed (OS handles authentication)
  - Flat namespace with prefix-based credential targets (`prefix:itemname`)
  - Testable via WSL2 PowerShell interop
  - Graceful error on non-Windows platforms
- **Comprehensive Testing Strategy Documentation** - Multi-pronged approach for Windows backend
  - WSL2 testing via PowerShell interop (primary path)
  - Build tags for cross-platform compilation
  - Mock backend for TDD on any platform
  - GitHub Actions Windows runners for CI/CD
  - Manual testing workflow documented
- **Roadmap Documentation** (ROADMAP.md)
  - Future backend plans (AWS Secrets Manager, Azure Key Vault, HashiCorp Vault, KeePassXC)
  - Priority levels and implementation timeline
  - Feature comparison matrix for all backends (current and planned)
  - Windows Credential Manager marked as implemented
- **Docsify Documentation Site** - Professional documentation with Blackwell theme
  - docs/index.html with dark mode mermaid support
  - Coverpage with logo and feature highlights
  - Sidebar navigation for all documentation sections
  - Search, tabs, pagination, and zoom plugins
  - Consistent Blackwell Systems branding (#4a9eff theme)
- **Architecture Diagrams** - Comprehensive mermaid diagrams in dark mode
  - Component architecture (Consumer → Core → Backends → CLIs)
  - Authentication flow with decision paths
  - Session caching strategy
  - CRUD operation flows (GetNotes, CreateItem)
  - Backend registration pattern
  - Data flow sequences
  - Error handling pipeline
  - Test pyramid visualization
  - Backend comparison diagrams
- **Extending Guide** (EXTENDING.md) - Complete guide for adding new backends
  - Backend interface requirements
  - Session management patterns
  - Implementation steps with examples
  - Testing checklist
  - Best practices and common pitfalls

### Changed

- **Test Coverage Improvement** - Core package increased from 95.5% to 98.5%
  - Added MustNew success path test
  - Comprehensive AutoRefreshSession tests (success/failure paths)
  - Only uncovered code: defensive json.MarshalIndent error (unreachable)
- **Documentation Updates**
  - README.md: Added Windows Credential Manager to supported backends table
  - README.md: Added Platform column to backends table
  - README.md: Added Windows Credential Manager to requirements section
  - ARCHITECTURE.md: Added Windows Credential Manager to feature matrix
  - ARCHITECTURE.md: Added Windows Credential Manager to backend comparison diagram
  - ARCHITECTURE.md: Added "When to Use" guidance for Windows Credential Manager
  - Coverpage: Updated to mention 4 backends and cross-platform support
  - Sidebar: Added comprehensive navigation for all architecture sections
  - All docs/ directory files synced with root documentation

### Fixed

- **Mermaid Diagram Rendering** - Removed inline theme config that broke docsify-mermaid
  - Removed `%%{init: {'theme':'dark', ...}}%%` from all 11 diagrams
  - Diagrams now use global mermaid.initialize() configuration
  - All diagrams render correctly in docsify dark mode

## [0.1.0] - 2025-12-07

### Added

- **Initial Release** - Production-ready unified vault abstraction library
- **Three Backend Implementations**
  - Bitwarden backend (`backends/bitwarden/`)
    - Session token management with 30-minute TTL
    - Folder support via folderId
    - Sync via `bw sync` command
    - Email/password + 2FA authentication
  - 1Password backend (`backends/onepassword/`)
    - Session token with biometric support
    - Vault organizational units
    - Automatic synchronization
    - Service account support
  - pass backend (`backends/pass/`)
    - GPG-based encryption
    - Directory-based organization
    - Git integration for sync
    - No session management (GPG agent)
- **Core Library Features**
  - Unified `Backend` interface for all secret managers
  - `Session` interface with caching and auto-refresh
  - `LocationManager` interface for folders/vaults/directories
  - `Item` structure with metadata (ID, Name, Type, Notes, Fields)
  - Context-aware operations (cancellation and timeout support)
  - Typed errors (ErrNotFound, ErrAlreadyExists, ErrNotAuthenticated, etc.)
  - Factory pattern with backend registration
  - Session caching to disk (30-minute default TTL)
  - AutoRefreshSession wrapper for automatic session renewal
- **Mock Backend** - In-memory testing backend (`mock/`)
  - 100% test coverage
  - Pre-populate with test data
  - Simulate error conditions
  - No external dependencies
- **Comprehensive Test Suite**
  - 95.5% core package coverage
  - 100% mock backend coverage
  - Example tests in documentation
  - Table-driven tests throughout
- **Documentation**
  - README.md with quick start and usage examples
  - ARCHITECTURE.md with design principles and patterns
  - Go package documentation with examples
  - MIT License
- **CI/CD**
  - GitHub Actions workflow
  - golangci-lint integration
  - codecov integration
  - Automated releases
- **Zero External Dependencies** - Only Go stdlib; backends delegate to their CLIs

### Technical Details

- Go 1.21+ required
- Module: `github.com/blackwell-systems/vaultmux`
- 58 tests passing
- Cross-platform: Linux, macOS, Windows (WSL2)

[unreleased]: https://github.com/blackwell-systems/vaultmux/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/vaultmux/releases/tag/v0.1.0
