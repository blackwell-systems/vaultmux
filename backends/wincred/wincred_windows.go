//go:build windows

// Package wincred implements the vaultmux.Backend interface for Windows Credential Manager.
package wincred

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/blackwell-systems/vaultmux"
)

func init() {
	vaultmux.RegisterBackend(vaultmux.BackendWindowsCredentialManager, func(cfg vaultmux.Config) (vaultmux.Backend, error) {
		return New(cfg.Prefix)
	})
}

// Backend implements vaultmux.Backend for Windows Credential Manager.
type Backend struct {
	prefix string
}

// New creates a new Windows Credential Manager backend.
func New(prefix string) (*Backend, error) {
	if prefix == "" {
		prefix = "vaultmux"
	}
	return &Backend{
		prefix: prefix,
	}, nil
}

// Name returns the backend name.
func (b *Backend) Name() string { return "wincred" }

// Init checks if PowerShell is available.
func (b *Backend) Init(ctx context.Context) error {
	// Check if powershell.exe is available
	cmd := exec.CommandContext(ctx, "powershell.exe", "-Command", "$PSVersionTable.PSVersion.Major")
	if err := cmd.Run(); err != nil {
		return vaultmux.ErrBackendNotInstalled
	}
	return nil
}

// Close is a no-op for Windows Credential Manager.
func (b *Backend) Close() error { return nil }

// IsAuthenticated always returns true as Windows Credential Manager uses OS-level auth.
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
	return true // OS handles authentication
}

// Authenticate returns a no-op session since Windows handles auth.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	return &winCredSession{}, nil
}

// Sync is a no-op for Windows Credential Manager (no remote sync).
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	return nil // No sync for local credential manager
}

// GetItem retrieves a vault item by name.
func (b *Backend) GetItem(ctx context.Context, name string, _ vaultmux.Session) (*vaultmux.Item, error) {
	notes, err := b.GetNotes(ctx, name, nil)
	if err != nil {
		return nil, err
	}
	if notes == "" {
		return nil, vaultmux.ErrNotFound
	}

	return &vaultmux.Item{
		Name:  name,
		Type:  vaultmux.ItemTypeSecureNote,
		Notes: notes,
	}, nil
}

// GetNotes retrieves the content of an item from Windows Credential Manager.
func (b *Backend) GetNotes(ctx context.Context, name string, _ vaultmux.Session) (string, error) {
	target := b.credentialTarget(name)

	// PowerShell script to get credential
	script := fmt.Sprintf(`
$cred = Get-StoredCredential -Target '%s' -ErrorAction SilentlyContinue
if ($cred) {
    $ptr = [System.Runtime.InteropServices.Marshal]::SecureStringToCoTaskMemUnicode($cred.Password)
    $password = [System.Runtime.InteropServices.Marshal]::PtrToStringUni($ptr)
    [System.Runtime.InteropServices.Marshal]::ZeroFreeCoTaskMemUnicode($ptr)
    Write-Output $password
} else {
    exit 1
}
`, target)

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", vaultmux.ErrNotFound
		}
		return "", vaultmux.WrapError("wincred", "get", name, err)
	}

	return strings.TrimSpace(string(out)), nil
}

// ItemExists checks if an item exists in Windows Credential Manager.
func (b *Backend) ItemExists(ctx context.Context, name string, _ vaultmux.Session) (bool, error) {
	target := b.credentialTarget(name)

	script := fmt.Sprintf(`
$cred = Get-StoredCredential -Target '%s' -ErrorAction SilentlyContinue
if ($cred) { exit 0 } else { exit 1 }
`, target)

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", script)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListItems lists all items in Windows Credential Manager under the prefix.
func (b *Backend) ListItems(ctx context.Context, _ vaultmux.Session) ([]*vaultmux.Item, error) {
	// PowerShell script to list credentials with our prefix
	script := fmt.Sprintf(`
$creds = Get-StoredCredential | Where-Object { $_.TargetName -like '%s:*' }
$creds | ForEach-Object {
    [PSCustomObject]@{
        Name = $_.TargetName.Substring(%d)
        Target = $_.TargetName
    }
} | ConvertTo-Json -Compress
`, b.prefix, len(b.prefix)+1) // +1 for the colon

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("wincred", "list", "", err)
	}

	if len(out) == 0 || strings.TrimSpace(string(out)) == "" {
		return []*vaultmux.Item{}, nil
	}

	// Parse JSON output
	var results []struct {
		Name   string `json:"Name"`
		Target string `json:"Target"`
	}

	// Handle single item (not an array)
	if !strings.HasPrefix(strings.TrimSpace(string(out)), "[") {
		var single struct {
			Name   string `json:"Name"`
			Target string `json:"Target"`
		}
		if err := json.Unmarshal(out, &single); err != nil {
			return nil, fmt.Errorf("parse credential list: %w", err)
		}
		results = []struct {
			Name   string `json:"Name"`
			Target string `json:"Target"`
		}{single}
	} else {
		if err := json.Unmarshal(out, &results); err != nil {
			return nil, fmt.Errorf("parse credential list: %w", err)
		}
	}

	items := make([]*vaultmux.Item, 0, len(results))
	for _, r := range results {
		items = append(items, &vaultmux.Item{
			Name: r.Name,
			Type: vaultmux.ItemTypeSecureNote,
		})
	}

	return items, nil
}

// CreateItem creates a new item in Windows Credential Manager.
func (b *Backend) CreateItem(ctx context.Context, name, content string, _ vaultmux.Session) error {
	exists, err := b.ItemExists(ctx, name, nil)
	if err != nil {
		return err
	}
	if exists {
		return vaultmux.ErrAlreadyExists
	}

	target := b.credentialTarget(name)

	// PowerShell script to create credential
	script := fmt.Sprintf(`
$password = ConvertTo-SecureString -String '%s' -AsPlainText -Force
$cred = New-Object System.Management.Automation.PSCredential('%s', $password)
New-StoredCredential -Target '%s' -Credential $cred -Type Generic -Persist LocalMachine
`, escapePowerShellString(content), "vaultmux", target)

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", script)
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("wincred", "create", name, err)
	}

	return nil
}

// UpdateItem updates an existing item in Windows Credential Manager.
func (b *Backend) UpdateItem(ctx context.Context, name, content string, _ vaultmux.Session) error {
	exists, err := b.ItemExists(ctx, name, nil)
	if err != nil {
		return err
	}
	if !exists {
		return vaultmux.ErrNotFound
	}

	target := b.credentialTarget(name)

	// PowerShell script to update credential (remove and recreate)
	script := fmt.Sprintf(`
Remove-StoredCredential -Target '%s' -ErrorAction SilentlyContinue
$password = ConvertTo-SecureString -String '%s' -AsPlainText -Force
$cred = New-Object System.Management.Automation.PSCredential('%s', $password)
New-StoredCredential -Target '%s' -Credential $cred -Type Generic -Persist LocalMachine
`, target, escapePowerShellString(content), "vaultmux", target)

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", script)
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("wincred", "update", name, err)
	}

	return nil
}

// DeleteItem removes an item from Windows Credential Manager.
func (b *Backend) DeleteItem(ctx context.Context, name string, _ vaultmux.Session) error {
	target := b.credentialTarget(name)

	script := fmt.Sprintf(`
Remove-StoredCredential -Target '%s' -ErrorAction SilentlyContinue
`, target)

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", script)
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("wincred", "delete", name, err)
	}

	return nil
}

// ListLocations returns empty list (Windows Credential Manager doesn't have folders).
func (b *Backend) ListLocations(ctx context.Context, _ vaultmux.Session) ([]string, error) {
	return []string{}, nil // No folder concept
}

// LocationExists always returns false (no folders).
func (b *Backend) LocationExists(ctx context.Context, name string, _ vaultmux.Session) (bool, error) {
	return false, nil // No folder concept
}

// CreateLocation is a no-op (no folders).
func (b *Backend) CreateLocation(ctx context.Context, name string, _ vaultmux.Session) error {
	return nil // No folder concept
}

// ListItemsInLocation returns empty list (no folders).
func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, _ vaultmux.Session) ([]*vaultmux.Item, error) {
	return []*vaultmux.Item{}, nil // No folder concept
}

// credentialTarget returns the Windows Credential Manager target name.
func (b *Backend) credentialTarget(name string) string {
	return fmt.Sprintf("%s:%s", b.prefix, name)
}

// escapePowerShellString escapes single quotes in PowerShell strings.
func escapePowerShellString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
