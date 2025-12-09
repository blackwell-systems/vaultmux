package vaultmux

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateItemName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errType error
	}{
		// Valid names
		{
			name:    "simple alphanumeric",
			input:   "myitem",
			wantErr: false,
		},
		{
			name:    "with dashes",
			input:   "my-item-name",
			wantErr: false,
		},
		{
			name:    "with underscores",
			input:   "my_item_name",
			wantErr: false,
		},
		{
			name:    "with dots",
			input:   "my.item.name",
			wantErr: false,
		},
		{
			name:    "with slashes (path-like)",
			input:   "folder/subfolder/item",
			wantErr: false,
		},
		{
			name:    "with colons",
			input:   "app:secret:key",
			wantErr: false,
		},
		{
			name:    "mixed valid characters",
			input:   "my-app_v1.0/secrets:api-key",
			wantErr: false,
		},
		{
			name:    "numbers only",
			input:   "12345",
			wantErr: false,
		},
		{
			name:    "long but valid name",
			input:   strings.Repeat("a", 256),
			wantErr: false,
		},

		// Invalid names - command injection attempts
		{
			name:    "semicolon (command chaining)",
			input:   "item; rm -rf /",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "pipe (command piping)",
			input:   "item | cat /etc/passwd",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "ampersand (background execution)",
			input:   "item & evil-command",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "dollar (variable expansion)",
			input:   "item$USER",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "backtick (command substitution)",
			input:   "item`whoami`",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "less than (input redirection)",
			input:   "item < /etc/passwd",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "greater than (output redirection)",
			input:   "item > /tmp/evil",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "parentheses (subshell)",
			input:   "item (echo hacked)",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "braces (command grouping)",
			input:   "item { echo hacked; }",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "brackets (pattern matching)",
			input:   "item[0-9]",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "exclamation (history expansion)",
			input:   "item!123",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "asterisk (glob)",
			input:   "item*",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "question mark (glob)",
			input:   "item?",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "tilde (home expansion)",
			input:   "~/item",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "hash (comment)",
			input:   "item # comment",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "at symbol",
			input:   "item@host",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "percent",
			input:   "item%value",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "caret",
			input:   "item^value",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "backslash (escape)",
			input:   "item\\escape",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "double quote",
			input:   "item\"quoted",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "single quote",
			input:   "item'quoted",
			wantErr: true,
			errType: ErrInvalidItemName,
		},

		// Invalid names - other issues
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "null byte",
			input:   "item\x00evil",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "newline",
			input:   "item\ncommand",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "tab character",
			input:   "item\tvalue",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "carriage return",
			input:   "item\rvalue",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "bell character",
			input:   "item\x07",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "delete character",
			input:   "item\x7F",
			wantErr: true,
			errType: ErrInvalidItemName,
		},
		{
			name:    "too long (257 chars)",
			input:   strings.Repeat("a", 257),
			wantErr: true,
			errType: ErrInvalidItemName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateItemName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateItemName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("ValidateItemName() error type = %T, want %T", err, tt.errType)
				}
			}
		})
	}
}

func TestValidateLocationName(t *testing.T) {
	// ValidateLocationName uses the same logic as ValidateItemName
	// Just test a few cases to ensure it's wired up correctly

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid folder name",
			input:   "my-folder",
			wantErr: false,
		},
		{
			name:    "invalid folder with semicolon",
			input:   "folder; rm -rf /",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocationName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLocationName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateItemName_RealWorldExamples tests realistic secret names
func TestValidateItemName_RealWorldExamples(t *testing.T) {
	validNames := []string{
		"api-key",
		"database_password",
		"jwt.secret",
		"app/production/db-password",
		"aws:access-key-id",
		"github-token-prod",
		"stripe_secret_key_live",
		"postgres.production.password",
		"redis://password",
		"myapp-v2.0-api-token",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			if err := ValidateItemName(name); err != nil {
				t.Errorf("ValidateItemName(%q) failed: %v (should be valid)", name, err)
			}
		})
	}
}
