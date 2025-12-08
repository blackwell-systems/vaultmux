package azurekeyvault

import (
	"context"
	"errors"
	"testing"

	"github.com/blackwell-systems/vaultmux"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		options   map[string]string
		want      *Backend
		wantErr   bool
		errString string
	}{
		{
			name: "missing vault_url",
			options: map[string]string{
				"prefix": "test-",
			},
			wantErr:   true,
			errString: "vault_url is required",
		},
		{
			name: "invalid vault_url format",
			options: map[string]string{
				"vault_url": "http://invalid.com",
			},
			wantErr:   true,
			errString: "vault_url must be in format",
		},
		{
			name: "defaults",
			options: map[string]string{
				"vault_url": "https://myvault.vault.azure.net/",
			},
			want: &Backend{
				vaultURL: "https://myvault.vault.azure.net/",
				prefix:   "vaultmux-",
			},
		},
		{
			name: "custom prefix",
			options: map[string]string{
				"vault_url": "https://testvault.vault.azure.net/",
				"prefix":    "myapp-",
			},
			want: &Backend{
				vaultURL: "https://testvault.vault.azure.net/",
				prefix:   "myapp-",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.options, "")

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error containing %q, got nil", tt.errString)
					return
				}
				if err.Error() == "" || !contains(err.Error(), tt.errString) {
					t.Errorf("New() error = %q, want error containing %q", err.Error(), tt.errString)
				}
				return
			}

			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			if got.vaultURL != tt.want.vaultURL {
				t.Errorf("vaultURL = %q, want %q", got.vaultURL, tt.want.vaultURL)
			}
			if got.prefix != tt.want.prefix {
				t.Errorf("prefix = %q, want %q", got.prefix, tt.want.prefix)
			}
		})
	}
}

func TestBackend_Name(t *testing.T) {
	backend, _ := New(map[string]string{"vault_url": "https://test.vault.azure.net/"}, "")
	if got := backend.Name(); got != "azurekeyvault" {
		t.Errorf("Name() = %q, want %q", got, "azurekeyvault")
	}
}

func TestBackend_SecretName(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		item   string
		want   string
	}{
		{
			name:   "with prefix",
			prefix: "myapp-",
			item:   "database-password",
			want:   "myapp-database-password",
		},
		{
			name:   "no prefix",
			prefix: "",
			item:   "api-key",
			want:   "api-key",
		},
		{
			name:   "default prefix",
			prefix: "vaultmux-",
			item:   "ssh-key",
			want:   "vaultmux-ssh-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &Backend{
				vaultURL: "https://test.vault.azure.net/",
				prefix:   tt.prefix,
			}
			got := backend.secretName(tt.item)
			if got != tt.want {
				t.Errorf("secretName(%q) = %q, want %q", tt.item, got, tt.want)
			}
		})
	}
}

func TestBackend_LocationManagement(t *testing.T) {
	backend, _ := New(map[string]string{"vault_url": "https://test.vault.azure.net/"}, "")
	ctx := context.Background()
	session := &azureSession{}

	t.Run("ListLocations", func(t *testing.T) {
		_, err := backend.ListLocations(ctx, session)
		if !errors.Is(err, vaultmux.ErrNotSupported) {
			t.Errorf("ListLocations() error = %v, want ErrNotSupported", err)
		}
	})

	t.Run("LocationExists", func(t *testing.T) {
		_, err := backend.LocationExists(ctx, "test", session)
		if !errors.Is(err, vaultmux.ErrNotSupported) {
			t.Errorf("LocationExists() error = %v, want ErrNotSupported", err)
		}
	})

	t.Run("CreateLocation", func(t *testing.T) {
		err := backend.CreateLocation(ctx, "test", session)
		if !errors.Is(err, vaultmux.ErrNotSupported) {
			t.Errorf("CreateLocation() error = %v, want ErrNotSupported", err)
		}
	})

	t.Run("ListItemsInLocation", func(t *testing.T) {
		_, err := backend.ListItemsInLocation(ctx, "test", "test", session)
		if !errors.Is(err, vaultmux.ErrNotSupported) {
			t.Errorf("ListItemsInLocation() error = %v, want ErrNotSupported", err)
		}
	})
}

func TestBackend_Close(t *testing.T) {
	backend, _ := New(map[string]string{"vault_url": "https://test.vault.azure.net/"}, "")
	if err := backend.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestBackend_Sync(t *testing.T) {
	backend, _ := New(map[string]string{"vault_url": "https://test.vault.azure.net/"}, "")
	session := &azureSession{}

	err := backend.Sync(context.Background(), session)
	if err != nil {
		t.Errorf("Sync() error = %v, want nil (no-op)", err)
	}
}

func TestSession_Token(t *testing.T) {
	session := &azureSession{vaultURL: "https://test.vault.azure.net/"}

	token := session.Token()
	if token != "https://test.vault.azure.net/" {
		t.Errorf("Token() = %q, want %q", token, "https://test.vault.azure.net/")
	}
}

func TestSession_ExpiresAt(t *testing.T) {
	session := &azureSession{}

	expiresAt := session.ExpiresAt()
	if !expiresAt.IsZero() {
		t.Errorf("ExpiresAt() = %v, want zero time (Azure credentials managed by SDK)", expiresAt)
	}
}

func TestSession_Refresh(t *testing.T) {
	session := &azureSession{}

	// Refresh is a no-op for Azure (SDK handles token refresh)
	err := session.Refresh(context.Background())
	if err != nil {
		t.Errorf("Refresh() error = %v, want nil (no-op)", err)
	}
}

func TestSession_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		session *azureSession
		want    bool
	}{
		{
			name: "invalid no vault URL",
			session: &azureSession{
				backend: &Backend{},
			},
			want: false,
		},
		{
			name: "invalid no credential",
			session: &azureSession{
				vaultURL: "https://test.vault.azure.net/",
				backend:  &Backend{},
			},
			want: false,
		},
		{
			name: "invalid no backend",
			session: &azureSession{
				vaultURL: "https://test.vault.azure.net/",
			},
			want: false,
		},
		{
			name: "invalid no client",
			session: &azureSession{
				vaultURL: "https://test.vault.azure.net/",
				backend:  &Backend{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.session.IsValid(context.Background())
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackend_InterfaceCompliance(t *testing.T) {
	var _ vaultmux.Backend = (*Backend)(nil)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Integration test note:
// To run integration tests with a real Azure Key Vault:
// 1. Create an Azure Key Vault and enable RBAC
// 2. Grant "Key Vault Secrets Officer" role to your identity
// 3. Set AZURE_VAULT_URL=https://your-vault.vault.azure.net/
// 4. Authenticate: az login OR set AZURE_TENANT_ID/AZURE_CLIENT_ID/AZURE_CLIENT_SECRET
// 5. go test -v ./backends/azurekeyvault/
//
// Alternatively, use mocked SDK for offline testing.
// See azurekeyvault_integration_test.go for full CRUD integration tests.
