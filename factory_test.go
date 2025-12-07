package vaultmux

import (
	"errors"
	"testing"
)

func TestRegisterBackend(t *testing.T) {
	// Save original state
	originalFactories := make(map[BackendType]BackendFactory)
	for k, v := range backendFactories {
		originalFactories[k] = v
	}
	defer func() {
		backendFactories = originalFactories
	}()

	testBackendType := BackendType("test-backend")
	testFactory := func(cfg Config) (Backend, error) {
		return nil, nil
	}

	RegisterBackend(testBackendType, testFactory)

	if _, ok := backendFactories[testBackendType]; !ok {
		t.Error("RegisterBackend() did not register the backend")
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "unknown backend",
			config: Config{
				Backend: BackendType("unknown"),
			},
			wantErr: true,
		},
		{
			name: "apply defaults",
			config: Config{
				Backend:    BackendPass,
				SessionTTL: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && backend == nil {
				t.Error("New() returned nil backend without error")
			}
		})
	}
}

func TestNew_SessionTTLDefault(t *testing.T) {
	// Register a test backend that captures the config
	var capturedConfig Config
	testType := BackendType("test-ttl")

	RegisterBackend(testType, func(cfg Config) (Backend, error) {
		capturedConfig = cfg
		return nil, errors.New("test backend")
	})

	cfg := Config{
		Backend:    testType,
		SessionTTL: 0, // Should default to 1800
	}

	_, _ = New(cfg)

	if capturedConfig.SessionTTL != 1800 {
		t.Errorf("SessionTTL = %d, want 1800", capturedConfig.SessionTTL)
	}
}

func TestMustNew_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNew() did not panic with invalid backend")
		}
	}()

	cfg := Config{
		Backend: BackendType("invalid"),
	}

	MustNew(cfg)
}

func TestBackendType_Constants(t *testing.T) {
	tests := []struct {
		backend BackendType
		want    string
	}{
		{BackendBitwarden, "bitwarden"},
		{BackendOnePassword, "1password"},
		{BackendPass, "pass"},
	}

	for _, tt := range tests {
		t.Run(string(tt.backend), func(t *testing.T) {
			if string(tt.backend) != tt.want {
				t.Errorf("BackendType = %q, want %q", tt.backend, tt.want)
			}
		})
	}
}
