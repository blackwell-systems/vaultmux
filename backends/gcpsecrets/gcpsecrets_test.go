package gcpsecrets

import (
	"context"
	"errors"
	"strings"
	"testing"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
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
			name: "missing project_id",
			options: map[string]string{
				"prefix": "test-",
			},
			wantErr:   true,
			errString: "project_id is required",
		},
		{
			name: "defaults",
			options: map[string]string{
				"project_id": "my-project",
			},
			want: &Backend{
				projectID: "my-project",
				prefix:    "vaultmux-",
			},
		},
		{
			name: "custom prefix",
			options: map[string]string{
				"project_id": "my-project",
				"prefix":     "myapp-",
			},
			want: &Backend{
				projectID: "my-project",
				prefix:    "myapp-",
			},
		},
		{
			name: "custom endpoint",
			options: map[string]string{
				"project_id": "my-project",
				"endpoint":   "localhost:8080",
			},
			want: &Backend{
				projectID: "my-project",
				prefix:    "vaultmux-",
				endpoint:  "localhost:8080",
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
				if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("New() error = %q, want error containing %q", err.Error(), tt.errString)
				}
				return
			}

			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			if got.projectID != tt.want.projectID {
				t.Errorf("projectID = %q, want %q", got.projectID, tt.want.projectID)
			}
			if got.prefix != tt.want.prefix {
				t.Errorf("prefix = %q, want %q", got.prefix, tt.want.prefix)
			}
			if got.endpoint != tt.want.endpoint {
				t.Errorf("endpoint = %q, want %q", got.endpoint, tt.want.endpoint)
			}
		})
	}
}

func TestBackend_Name(t *testing.T) {
	backend, _ := New(map[string]string{"project_id": "test"}, "")
	if got := backend.Name(); got != "gcpsecrets" {
		t.Errorf("Name() = %q, want %q", got, "gcpsecrets")
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
				projectID: "test-project",
				prefix:    tt.prefix,
			}
			got := backend.secretName(tt.item)
			if got != tt.want {
				t.Errorf("secretName(%q) = %q, want %q", tt.item, got, tt.want)
			}
		})
	}
}

func TestBackend_LocationManagement(t *testing.T) {
	backend, _ := New(map[string]string{"project_id": "test"}, "")
	ctx := context.Background()
	session := &gcpSession{}

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
	backend, _ := New(map[string]string{"project_id": "test"}, "")

	// Close without client initialized
	if err := backend.Close(); err != nil {
		t.Errorf("Close() with nil client error = %v, want nil", err)
	}
}

func TestBackend_Sync(t *testing.T) {
	backend, _ := New(map[string]string{"project_id": "test"}, "")
	session := &gcpSession{projectID: "test"}

	err := backend.Sync(context.Background(), session)
	if err != nil {
		t.Errorf("Sync() error = %v, want nil (no-op)", err)
	}
}

func TestSession_Token(t *testing.T) {
	session := &gcpSession{projectID: "my-project-123"}

	token := session.Token()
	if token != "my-project-123" {
		t.Errorf("Token() = %q, want %q", token, "my-project-123")
	}
}

func TestSession_ExpiresAt(t *testing.T) {
	session := &gcpSession{}

	expiresAt := session.ExpiresAt()
	if !expiresAt.IsZero() {
		t.Errorf("ExpiresAt() = %v, want zero time (GCP credentials don't expire)", expiresAt)
	}
}

func TestSession_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		session *gcpSession
		want    bool
	}{
		{
			name: "valid with project ID",
			session: &gcpSession{
				projectID: "test-project",
				backend:   &Backend{client: &secretmanager.Client{}},
			},
			want: true,
		},
		{
			name: "invalid no project ID",
			session: &gcpSession{
				projectID: "",
				backend:   &Backend{client: &secretmanager.Client{}},
			},
			want: false,
		},
		{
			name: "invalid no backend",
			session: &gcpSession{
				projectID: "test-project",
				backend:   nil,
			},
			want: false,
		},
		{
			name: "invalid no client",
			session: &gcpSession{
				projectID: "test-project",
				backend:   &Backend{},
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

// Integration test note:
// To run integration tests with a real GCP project:
// 1. Create a GCP project and enable Secret Manager API
// 2. Set GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
// 3. GCP_PROJECT_ID=your-project go test -v ./backends/gcpsecrets/
//
// Alternatively, use mocked SDK for offline testing.
// See gcpsecrets_integration_test.go for full CRUD integration tests.
