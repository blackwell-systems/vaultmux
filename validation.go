package vaultmux

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidItemName indicates the item name contains invalid characters.
var ErrInvalidItemName = errors.New("invalid item name")

// ValidateItemName checks if an item name is safe for use in CLI commands.
// It prevents command injection by rejecting names with shell metacharacters.
//
// Valid characters: alphanumeric, dash, underscore, dot, slash, colon
// Invalid: shell metacharacters like ; | & $ ` < > ( ) { } [ ] ! * ? ~ # @ % ^ \ " '
//
// This validation is critical for CLI-based backends (Bitwarden, 1Password, pass)
// to prevent command injection attacks.
func ValidateItemName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidItemName)
	}

	// Check for shell metacharacters that could be used for injection
	dangerousChars := `;|&$` + "`<>(){}[]!*?~#@%^\\\"'"
	for _, char := range dangerousChars {
		if strings.ContainsRune(name, char) {
			return fmt.Errorf("%w: contains forbidden character %q", ErrInvalidItemName, char)
		}
	}

	// Check for null bytes (can cause issues in C-based CLIs)
	if strings.ContainsRune(name, '\x00') {
		return fmt.Errorf("%w: contains null byte", ErrInvalidItemName)
	}

	// Check for newlines and control characters
	for _, char := range name {
		if char < 32 || char == 127 { // ASCII control characters
			return fmt.Errorf("%w: contains control character", ErrInvalidItemName)
		}
	}

	// Check length (reasonable limit to prevent abuse)
	if len(name) > 256 {
		return fmt.Errorf("%w: name too long (max 256 characters)", ErrInvalidItemName)
	}

	return nil
}

// ValidateLocationName validates location/folder names using the same rules as item names.
func ValidateLocationName(name string) error {
	return ValidateItemName(name)
}
