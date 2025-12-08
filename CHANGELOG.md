# Changelog

All notable changes to vaultmux will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[unreleased]: https://github.com/blackwell-systems/vaultmux/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/blackwell-systems/vaultmux/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/vaultmux/releases/tag/v0.1.0
