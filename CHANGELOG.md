# Changelog

All notable changes to vaultmux will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2025-12-08

### Added

- **Google Cloud Secret Manager Backend** - Second SDK-based backend (validates cloud provider universality)
  - Native Google Cloud Go SDK integration (`cloud.google.com/go/secretmanager/apiv1`)
  - Application Default Credentials (ADC) support (service account JSON, gcloud CLI, GCE/GKE metadata)
  - GCP project ID-based session management
  - Secret name prefixing for namespace isolation (`myapp-secret-name`)
  - Automatic secret versioning on every update (built-in to GCP)
  - Two-step secret creation (CreateSecret for metadata + AddSecretVersion for content)
  - gRPC error code mapping (codes.NotFound, codes.AlreadyExists, etc.)
  - Iterator-based pagination for secret listing
  - Label-based secret organization (`vaultmux:true`, `prefix:<value>`)
  - Custom endpoint support for testing (fake-gcp-server compatible)
  - Immediate deletion (no recovery period like AWS)
  - Comprehensive error mapping (GCP gRPC errors → vaultmux errors)
  - No session management complexity (credentials are long-lived or SDK-managed)
- **Real GCP Testing Strategy** - First backend without local emulator
  - Unit tests with no GCP credentials required (configuration, interface compliance)
  - Integration tests skip gracefully when GCP_PROJECT_ID not set
  - Real GCP free tier testing (first 6 secret versions free)
  - Service account IAM permission documentation
  - GitHub Actions CI workflow examples with real GCP
- **GCP Implementation Documentation**
  - ROADMAP.md: Comprehensive GCP section with testing strategy
  - ROADMAP.md: API mapping (GetItem → AccessSecretVersion + GetSecret)
  - ROADMAP.md: IAM permissions templates (secretmanager.admin role)
  - ROADMAP.md: Cost analysis (~$0.06/secret version after free tier)
  - ROADMAP.md: Comparison to AWS (simpler API, cleaner SDK, no local emulator)

### Changed

- **go.mod** - Added Google Cloud SDK dependencies
  - `cloud.google.com/go/secretmanager` v1.16.0
  - `google.golang.org/api` v0.257.0
  - `google.golang.org/grpc` v1.77.0
  - Go version upgraded from 1.23 to 1.24.0 (GCP SDK requirement)
- **README.md** - Updated for 6 backends
  - Updated tagline from "Unified interface for multiple secret management backends" to "The definitive Go library for multi-vault secret management"
  - Updated intro paragraph to mention all 5 backends explicitly
  - Related Projects section restructured (Supported Backends + Similar Projects)
  - Added detailed backend descriptions in Related Projects
- **docs/_coverpage.md** - Updated for 5 backends
  - New tagline: "The definitive Go library for multi-vault secret management"
  - Updated feature list to show 5 backends
  - Emphasized multiple integration patterns (CLI, SDK, OS APIs)
- **ROADMAP.md** - GCP Secret Manager marked "IN PROGRESS (v0.4.0)"
  - Current status updated from v0.2.0 to v0.4.0
  - Added GCP to supported backends list
  - AWS marked "IMPLEMENTED (v0.3.0)"
  - Comprehensive GCP testing strategy (no emulator, real GCP approach)
  - API mapping documentation
  - IAM permissions JSON policy template
  - Comparison to AWS implementation
  - Updated backend feature matrix with GCP row
  - Updated implementation priority (v0.3.0 released, v0.4.0 in progress)
  - Last updated date: 2025-12-08
- **factory.go** - Added `BackendGCPSecretManager` constant
  - Updated Config comment to include "gcpsecrets"
  - Maintains backward compatibility

### Technical Details

- **Pattern Validation**: Second SDK-based backend (after AWS) confirms interface universality across clouds
- **Session Semantics**: GCP project ID + service account credentials validate flexible session patterns
- **Error Handling**: GCP gRPC status codes map cleanly to vaultmux standard errors
- **Testing Strategy**: Real GCP testing (no emulator) with graceful skip when credentials unavailable
- **Integration Type**: Native SDK (not CLI subprocess) - same pattern as AWS
- **API Simplicity**: GCP SDK is notably cleaner than AWS v2 (fewer methods, better design)

### Developer Experience

- **Local Testing**: Unit tests run with no GCP setup required
- **Integration Testing**: Real GCP free tier enables zero-cost testing (first 6 versions free)
- **Fast Tests**: GCP API is fast (~50-100ms latency)
- **Better SDK**: Google's Go SDKs are excellent quality (cleaner than AWS)
- **No Emulator Needed**: GCP free tier eliminates need for local emulation

### Comparison to AWS Implementation

- **Simpler API**: 5 core operations vs AWS's more complex API surface
- **Cleaner SDK**: Google's SDK design is more intuitive than AWS v2
- **Built-in Versioning**: GCP versions automatically on update (no separate API)
- **Two-Step Creation**: CreateSecret + AddSecretVersion (different from AWS single CreateSecret)
- **No Local Emulator**: Must use real GCP (but free tier makes this viable)
- **Faster Implementation**: ~3-4 days vs ~5 days for AWS (simpler patterns)

## [0.3.0] - 2025-12-07

### Added

- **AWS Secrets Manager Backend** - First SDK-based backend (validates interface universality)
  - Native AWS SDK for Go v2 integration (`github.com/aws/aws-sdk-go-v2/service/secretsmanager`)
  - IAM credential support (environment variables, shared config, instance roles)
  - Automatic pagination for large secret collections (100+ secrets)
  - Secret name prefixing for namespace isolation (`myapp/secret-name`)
  - Tag-based secret organization (`vaultmux:true`, `prefix:<value>`)
  - Configurable AWS region support
  - LocalStack endpoint override for local testing
  - Force deletion (immediate, no recovery period)
  - Comprehensive error mapping (AWS exceptions → vaultmux errors)
  - No session management needed (IAM credentials are long-lived or SDK-managed)
- **LocalStack Testing Infrastructure** - Zero-cost AWS testing
  - Docker-based AWS service emulation
  - Complete Secrets Manager API coverage
  - Integration tests with LocalStack endpoint override
  - Alternative moto (Python mock) support documented
  - CI/CD workflow examples for GitHub Actions
- **AWS Implementation Documentation** (docs/AWS_IMPLEMENTATION_PLAN.md)
  - 10-day phased implementation schedule
  - Complete method-by-method code examples
  - Session pattern explanation for IAM credentials
  - Testing strategy (LocalStack, moto, mocked SDK)
  - IAM permissions policy templates
  - Error handling patterns
  - Production deployment guidance

### Changed

- **go.mod** - Added AWS SDK v2 dependencies
  - `github.com/aws/aws-sdk-go-v2` v1.40.1
  - `github.com/aws/aws-sdk-go-v2/config` v1.32.3
  - `github.com/aws/aws-sdk-go-v2/service/secretsmanager` v1.40.3
  - Go version upgraded from 1.21 to 1.23 (AWS SDK requirement)
- **README.md** - Updated for 5 backends
  - Added AWS Secrets Manager to supported backends table
  - Added "Integration" column (CLI vs SDK vs PowerShell vs OS)
  - Updated backend comparison table with AWS column
  - Added "When to Use AWS Secrets Manager" guidance
  - Updated security considerations for IAM credentials
  - Changed "Zero Dependencies" to "Minimal Dependencies" (accurate now)
- **ROADMAP.md** - AWS Secrets Manager marked "IN PROGRESS (v0.3.0)"
  - Comprehensive LocalStack/moto testing strategy
  - API mapping documentation (GetItem → GetSecretValue, etc.)
  - IAM permissions JSON policy template
  - Cost analysis (~$0.40/secret/month + $0.05/10k API calls)
  - Implementation priority updated (v0.3.0 in progress)
- **factory.go** - Added `BackendAWSSecretsManager` constant
  - Updated Config comment to include "awssecrets"
  - Maintains backward compatibility

### Technical Details

- **Pattern Validation**: First SDK-based backend proves `Backend` interface works beyond CLI wrappers
- **Session Semantics**: IAM credentials (long-lived, SDK-managed) validate session flexibility
- **Error Handling**: AWS typed errors (`ResourceNotFoundException`, `ResourceExistsException`) map cleanly to vaultmux standard errors
- **Testing Strategy**: LocalStack enables full integration testing without AWS account
- **Integration Type**: Native SDK (not CLI subprocess) - different pattern from Bitwarden/1Password/pass

### Developer Experience

- **Local Testing**: `docker run localstack/localstack` + endpoint override enables offline development
- **No AWS Account Needed**: Contributors can develop and test without AWS credentials
- **Fast Tests**: LocalStack starts in seconds, integration tests run quickly
- **Identical API**: LocalStack provides same API as production AWS

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

[unreleased]: https://github.com/blackwell-systems/vaultmux/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/vaultmux/releases/tag/v0.1.0
