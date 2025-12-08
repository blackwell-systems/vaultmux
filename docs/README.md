# Vaultmux

> **Unified interface for multiple secret management backends**

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev/)
[![Version](https://img.shields.io/github/v/release/blackwell-systems/vaultmux)](https://github.com/blackwell-systems/vaultmux/releases)
[![CI](https://github.com/blackwell-systems/vaultmux/workflows/CI/badge.svg)](https://github.com/blackwell-systems/vaultmux/actions)
[![codecov](https://codecov.io/gh/blackwell-systems/vaultmux/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/vaultmux)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Vaultmux is a Go library that provides a unified interface for interacting with multiple secret management backends. Write your code once and support Bitwarden, 1Password, and pass (the standard Unix password manager) with the same API.

## Features

- **Unified API** - Single interface works with any backend
- **Zero External Dependencies** - Only Go stdlib; backends use their own CLIs
- **Context Support** - All operations accept `context.Context` for cancellation/timeout
- **Session Caching** - Avoid repeated authentication prompts
- **Type-Safe** - Full static typing with Go interfaces
- **Testable** - Includes mock backend for unit testing (89%+ core coverage)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/blackwell-systems/vaultmux"
)

func main() {
    ctx := context.Background()

    // Create backend (auto-detects ~/.password-store)
    backend, err := vaultmux.New(vaultmux.Config{
        Backend: vaultmux.BackendPass,
        Prefix:  "myapp",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer backend.Close()

    // Initialize (checks CLI availability)
    if err := backend.Init(ctx); err != nil {
        log.Fatal(err)
    }

    // Authenticate (no-op for pass, interactive for Bitwarden/1Password)
    session, err := backend.Authenticate(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Store a secret
    err = backend.CreateItem(ctx, "API-Key", "sk-secret123", session)
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve it
    secret, err := backend.GetNotes(ctx, "API-Key", session)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Secret:", secret)
}
```

## Installation

```bash
go get github.com/blackwell-systems/vaultmux
```

## Supported Backends

| Backend | CLI Tool | Features |
|---------|----------|----------|
| **Bitwarden** | `bw` | Session tokens, folders, sync |
| **1Password** | `op` | Session tokens, vaults, auto-sync |
| **pass** | `pass` + `gpg` | Git-based, directories, offline |

## Documentation

- [Architecture](ARCHITECTURE.md) - Design principles and internals
- [Extending](EXTENDING.md) - Adding new backends

## Requirements

Each backend requires its CLI tool to be installed:

```bash
# Bitwarden
npm install -g @bitwarden/cli

# 1Password
# See: https://1password.com/downloads/command-line/

# pass
apt-get install pass  # Debian/Ubuntu
brew install pass      # macOS
```

## License

MIT License - see [LICENSE](https://github.com/blackwell-systems/vaultmux/blob/main/LICENSE) for details

## Related Projects

- [blackwell-systems/dotfiles](https://github.com/blackwell-systems/dotfiles) - The project vaultmux was extracted from
- [Bitwarden CLI](https://github.com/bitwarden/clients)
- [1Password CLI](https://developer.1password.com/docs/cli/)
- [pass](https://www.passwordstore.org/)
