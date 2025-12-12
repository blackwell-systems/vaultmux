# Vaultmux Documentation

> **The definitive Go library for multi-vault secret management**

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/vaultmux.svg)](https://pkg.go.dev/github.com/blackwell-systems/vaultmux)
[![CI](https://github.com/blackwell-systems/vaultmux/workflows/CI/badge.svg)](https://github.com/blackwell-systems/vaultmux/actions)
[![codecov](https://codecov.io/gh/blackwell-systems/vaultmux/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/vaultmux)

Vaultmux provides a unified interface for interacting with multiple secret management systems. Write your code once and support Bitwarden, 1Password, pass, Windows Credential Manager, AWS Secrets Manager, Google Cloud Secret Manager, and Azure Key Vault with the same API.

## Getting Started

**New to vaultmux?** Start with the [main README](https://github.com/blackwell-systems/vaultmux#readme) for:
- Installation instructions
- Quick start guide
- API examples
- Supported backends
- Configuration options

## Documentation

### Core Guides

- **[Architecture](ARCHITECTURE.md)** - Design decisions, backend integration patterns, and internal structure
- **[Testing](TESTING.md)** - Running tests, writing backend tests, integration testing strategies
- **[Extending](EXTENDING.md)** - Adding new backends, implementing custom integrations
- **[Roadmap](ROADMAP.md)** - Planned features, backend priorities, contribution ideas

### Reference

- **[Changelog](CHANGELOG.md)** - Version history and release notes
- **[Brand Guidelines](BRAND.md)** - Logo usage, trademark policy
- **[API Documentation](https://pkg.go.dev/github.com/blackwell-systems/vaultmux)** - Go package documentation

## Quick Links

- [GitHub Repository](https://github.com/blackwell-systems/vaultmux)
- [Report an Issue](https://github.com/blackwell-systems/vaultmux/issues)
- [Discussions](https://github.com/blackwell-systems/vaultmux/discussions)
- [Releases](https://github.com/blackwell-systems/vaultmux/releases)

## Supported Backends

| Backend | Integration | Platform |
|---------|-------------|----------|
| **Bitwarden** | CLI (`bw`) | All |
| **1Password** | CLI (`op`) | All |
| **pass** | CLI (`pass` + `gpg`) | Unix |
| **Windows Credential Manager** | PowerShell | Windows |
| **AWS Secrets Manager** | SDK (aws-sdk-go-v2) | All |
| **Google Cloud Secret Manager** | SDK (cloud.google.com/go) | All |
| **Azure Key Vault** | SDK (azure-sdk-for-go) | All |

## Example

```go
// Create backend
backend, _ := vaultmux.New(vaultmux.Config{
    Backend: vaultmux.BackendPass,
    Prefix:  "myapp",
})
defer backend.Close()

// Get secret
secret, _ := backend.GetNotes(ctx, "api-key", session)
```

See the [main README](https://github.com/blackwell-systems/vaultmux#readme) for complete examples.

## Contributing

Contributions are welcome! Check the [Roadmap](ROADMAP.md) for planned features, or see [Extending](EXTENDING.md) to add a new backend.

## License

Apache License 2.0. See [LICENSE](https://github.com/blackwell-systems/vaultmux/blob/main/LICENSE).
