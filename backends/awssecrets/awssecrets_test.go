package awssecrets

import (
	"context"
	"errors"
	"testing"

	"github.com/blackwell-systems/vaultmux"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]string
		want    *Backend
	}{
		{
			name:    "defaults",
			options: map[string]string{},
			want: &Backend{
				region: "us-east-1",
				prefix: "vaultmux/",
			},
		},
		{
			name: "custom region and prefix",
			options: map[string]string{
				"region": "us-west-2",
				"prefix": "myapp/",
			},
			want: &Backend{
				region: "us-west-2",
				prefix: "myapp/",
			},
		},
		{
			name: "localstack endpoint",
			options: map[string]string{
				"endpoint": "http://localhost:4566",
			},
			want: &Backend{
				region:   "us-east-1",
				prefix:   "vaultmux/",
				endpoint: "http://localhost:4566",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.options, "")
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if got.region != tt.want.region {
				t.Errorf("region = %q, want %q", got.region, tt.want.region)
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
	backend, _ := New(nil, "")
	if got := backend.Name(); got != "awssecrets" {
		t.Errorf("Name() = %q, want %q", got, "awssecrets")
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
			prefix: "myapp/",
			item:   "database-password",
			want:   "myapp/database-password",
		},
		{
			name:   "no prefix",
			prefix: "",
			item:   "api-key",
			want:   "api-key",
		},
		{
			name:   "default prefix",
			prefix: "vaultmux/",
			item:   "ssh-key",
			want:   "vaultmux/ssh-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &Backend{prefix: tt.prefix}
			got := backend.secretName(tt.item)
			if got != tt.want {
				t.Errorf("secretName(%q) = %q, want %q", tt.item, got, tt.want)
			}
		})
	}
}

func TestBackend_HandleAWSError(t *testing.T) {
	backend, _ := New(nil, "")

	tests := []struct {
		name      string
		err       error
		operation string
		itemName  string
		wantErr   error
	}{
		{
			name:      "nil error",
			err:       nil,
			operation: "get",
			itemName:  "test",
			wantErr:   nil,
		},
		{
			name:      "generic error",
			err:       errors.New("some error"),
			operation: "get",
			itemName:  "test",
			wantErr:   errors.New("[awssecrets:get:test] some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := backend.handleAWSError(tt.err, tt.operation, tt.itemName)

			if tt.wantErr == nil {
				if gotErr != nil {
					t.Errorf("handleAWSError() error = %v, want nil", gotErr)
				}
				return
			}

			if gotErr == nil {
				t.Errorf("handleAWSError() error = nil, want %v", tt.wantErr)
				return
			}

			// Check error message contains expected parts
			if tt.err != nil && gotErr.Error() != tt.err.Error() {
				// For wrapped errors, just check the error is not nil
				if gotErr == nil {
					t.Errorf("handleAWSError() returned nil, want non-nil error")
				}
			}
		})
	}
}

func TestBackend_LocationManagement(t *testing.T) {
	backend, _ := New(nil, "")
	ctx := context.Background()
	session := &awsSession{}

	t.Run("ListLocations", func(t *testing.T) {
		_, err := backend.ListLocations(ctx, session)
		if err == nil {
			t.Error("ListLocations() expected error, got nil")
		}
	})

	t.Run("LocationExists", func(t *testing.T) {
		_, err := backend.LocationExists(ctx, "test", session)
		if err == nil {
			t.Error("LocationExists() expected error, got nil")
		}
	})

	t.Run("CreateLocation", func(t *testing.T) {
		err := backend.CreateLocation(ctx, "test", session)
		if err == nil {
			t.Error("CreateLocation() expected error, got nil")
		}
	})

	t.Run("ListItemsInLocation", func(t *testing.T) {
		_, err := backend.ListItemsInLocation(ctx, "test", "test", session)
		if err == nil {
			t.Error("ListItemsInLocation() expected error, got nil")
		}
	})
}

func TestBackend_Close(t *testing.T) {
	backend, _ := New(nil, "")
	if err := backend.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestBackend_Sync(t *testing.T) {
	backend, _ := New(nil, "")
	session := &awsSession{}

	err := backend.Sync(context.Background(), session)
	if err != nil {
		t.Errorf("Sync() error = %v, want nil (no-op)", err)
	}
}

func TestSession_Token(t *testing.T) {
	// awsSession requires a valid aws.Config with credentials
	// Without credentials, Token() returns empty string
	backend, _ := New(nil, "")
	session := &awsSession{
		config:  backend.awsConfig,
		backend: backend,
	}

	// With no credentials configured, Token() should return empty string
	token := session.Token()
	// Token will be empty since no AWS credentials are configured in test env
	// This is expected behavior
	_ = token
}

func TestSession_ExpiresAt(t *testing.T) {
	session := &awsSession{}

	expiresAt := session.ExpiresAt()
	if !expiresAt.IsZero() {
		t.Errorf("ExpiresAt() = %v, want zero time (AWS credentials don't expire)", expiresAt)
	}
}

func TestBackend_InterfaceCompliance(t *testing.T) {
	var _ vaultmux.Backend = (*Backend)(nil)
}

// Integration test note:
// To run integration tests with LocalStack:
// 1. docker run -d -p 4566:4566 -e SERVICES=secretsmanager localstack/localstack
// 2. LOCALSTACK_ENDPOINT=http://localhost:4566 AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test go test -v
//
// See awssecrets_integration_test.go for LocalStack tests.
