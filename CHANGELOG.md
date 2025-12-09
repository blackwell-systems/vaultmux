# Changelog

All notable changes to vaultmux will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - TBD

### Performance

- **Status Caching for CLI Backends** - Significant performance improvement
  - IsAuthenticated() results cached for 5 seconds (configurable TTL)
  - Reduces subprocess overhead for Bitwarden, 1Password, and pass backends
  - Eliminates repeated `bw unlock --check`, `op whoami`, and `pass ls` calls
  - Thread-safe implementation with sync.RWMutex
  - Cache automatically updated after successful authentication
  - Typical performance improvement: 5-50ms saved per IsAuthenticated call
  - No impact on security (cache expires quickly, auth state verified periodically)

### Added

- **Status Cache Tests** - Comprehensive test suite for caching behavior
  - Cache hit/miss testing within and after TTL expiration
  - Concurrent access tests (100 goroutines reading/writing)
  - Alternating state tests (true/false transitions)
  - Verified thread-safe with go test -race
  - Added to all three CLI backends (Bitwarden, 1Password, pass)

### Testing

- **AWS Backend CI Integration** - LocalStack-based integration tests in GitHub Actions
  - Dedicated `integration-aws` job with LocalStack service container
  - Automatic health checks ensure LocalStack readiness before test execution
  - Full CRUD test coverage (Create, Get, Update, Delete, List with pagination)
  - Coverage increased from 23.7% to 79.1% for AWS backend
  - No AWS credentials or costs required for CI testing
  - Coverage reports uploaded to Codecov with `integration-aws` flag
  - Test execution time: ~3-5 seconds in CI environment
  - TESTING.md documentation updated with complete CI workflow details

## [0.3.3] - 2025-12-08

### Security

- **Command Injection Prevention** - Added input validation to all CLI backends
  - New ValidateItemName() function prevents shell metacharacter injection
  - Validates item names before passing to CLI commands (Bitwarden, 1Password, pass)
  - Blocks dangerous characters: ; | & $ ` < > ( ) { } [ ] ! * ? ~ # @ % ^ \ " '
  - Blocks control characters and null bytes
  - Maximum name length enforced (256 characters)
  - Applied to all GetItem, CreateItem, UpdateItem, DeleteItem, and CreateLocation methods
  - Protects against command chaining, variable expansion, and subshell execution
  - Comprehensive test suite with 40+ test cases including real-world secret names

- **Session File Permissions Hardening**
  - Session cache directories now created with 0700 permissions (owner access only)
  - Session cache files written with 0600 permissions (owner read/write only)
  - Prevents unauthorized access to cached session tokens
  - Directory creation errors now properly returned instead of silently ignored
  - Improved security for multi-user systems

- **Race Condition Fix** - AutoRefreshSession is now thread-safe
  - Added sync.Mutex to protect concurrent Token() and Refresh() calls
  - Prevents race conditions when multiple goroutines access the same session
  - All methods documented as safe for concurrent use
  - Verified with go test -race

### Fixed

- **Error Handling** - BackendError now implements errors.Is()
  - Allows errors.Is() to work through BackendError wrappers
  - Fixes error checking for sentinel errors (ErrNotFound, ErrAlreadyExists, etc.)
  - Example: `errors.Is(wrappedErr, vaultmux.ErrNotFound)` now works correctly
  - Improved error wrapping consistency across all backends

- **Session Cache** - Invalid JSON now returns error instead of silent nil
  - Corrupted session files are detected and reported
  - Invalid files are removed automatically
  - Test updated to expect error for better security posture

### Changed

- **Error Messages** - Improved error context in session operations
  - Parse errors now include "parse session cache" context
  - Directory creation errors include full context
  - Better debugging information for session-related failures

## [0.3.2] - 2025-12-08

### Changed

- **License** - Changed from MIT License to Apache License 2.0
  - LICENSE file updated to Apache License 2.0
  - All documentation and badges updated (README.md, BRAND.md, etc.)
  - Provides additional patent protection and contributor license terms
  - Code remains free and open source
  - Effective starting with v0.3.2 and all future releases
  - Version note added to README: v0.3.2+ is Apache 2.0, v0.3.0 and earlier is MIT

### Note

- This release formalizes the license change to Apache 2.0 for Go module immutability

## [0.3.1] - 2025-12-08

### Added

- **Testing Documentation** - Comprehensive guide for testing vaultmux backends
  - TESTING.md: Complete testing guide with LocalStack integration examples
  - LocalStack setup and configuration for AWS Secrets Manager testing
  - Backend testing strategies for CLI-based and SDK-based backends
  - Integration test patterns with environment variable conditional skipping
  - CI/CD testing examples and GitHub Actions workflow
  - Troubleshooting guide for common testing issues
  - Coverage goals and test organization best practices
  - Added to docsify documentation site navigation

### Changed

- **Documentation Site** - Enhanced sidebar navigation
  - Added Testing section between Extending and Roadmap
  - Testing Guide with detailed subsections (Philosophy, Quick Start, Backend Strategies, etc.)
  - Improved documentation flow and organization

### Technical Details

- **LocalStack Verification**: AWS backend integration tests verified against LocalStack
  - All CRUD operations tested (Create, Get, Update, Delete, List)
  - Pagination testing with multiple items
  - Error handling verification (ErrNotFound, ErrAlreadyExists)
  - Test execution time: ~3 seconds for full suite
  - 100% pass rate on integration tests

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
  - README.md: Added "Why Vaultmux?" section with unified API value proposition and use cases
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

### Fixed

- **Bitwarden Backend** - Replace `--session` CLI flag with `BW_SESSION` environment variable
  - The `--session` argument is unreliable with Bitwarden CLI
  - Using environment variable is the officially documented method
  - Applies to all Bitwarden CLI commands (unlock, sync, get, list, create, edit, delete)
  - Improves session management reliability and authentication consistency

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
  - Apache License 2.0
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

[unreleased]: https://github.com/blackwell-systems/vaultmux/compare/v0.3.3...HEAD
[0.3.3]: https://github.com/blackwell-systems/vaultmux/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/blackwell-systems/vaultmux/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/blackwell-systems/vaultmux/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/vaultmux/releases/tag/v0.1.0
