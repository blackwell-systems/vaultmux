// Package azurekeyvault implements the vaultmux.Backend interface for Azure Key Vault.
//
// Azure Key Vault is Microsoft Azure's managed secret storage service with features like
// HSM-backed storage, Azure AD integration, RBAC permissions, and soft-delete protection.
//
// This backend uses the official Azure SDK for Go, making it the third SDK-based backend
// (after AWS Secrets Manager and GCP Secret Manager). The Azure SDK uses interface-based
// design which makes testing straightforward with mocks.
package azurekeyvault

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	"github.com/blackwell-systems/vaultmux"
)

// Backend implements vaultmux.Backend for Azure Key Vault.
type Backend struct {
	// Azure Key Vault client
	client *azsecrets.Client

	// Configuration
	vaultURL string // Azure Key Vault URL (required, e.g., "https://myvault.vault.azure.net/")
	prefix   string // Secret name prefix for namespacing (e.g., "myapp-")

	// Azure AD credential (service principal, managed identity, CLI, etc.)
	credential azcore.TokenCredential

	// Session cache file (currently unused - Azure credentials are long-lived)
	sessionFile string
}

// New creates a new Azure Key Vault backend.
//
// Supported options:
//   - vault_url: Azure Key Vault URL (required, e.g., "https://myvault.vault.azure.net/")
//   - prefix: Secret name prefix for namespacing (default: "vaultmux-")
//   - tenant_id: Azure AD tenant ID (optional, for service principal auth)
//   - client_id: Azure AD client ID (optional, for service principal auth)
//   - client_secret: Azure AD client secret (optional, for service principal auth)
//
// Authentication uses DefaultAzureCredential by default, which tries in order:
//   - Environment variables (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)
//   - Managed Identity (for apps running on Azure)
//   - Azure CLI credentials (az login)
//
// Or explicitly via service principal (if tenant_id, client_id, client_secret provided).
//
// Example:
//
//	backend, err := azurekeyvault.New(map[string]string{
//	    "vault_url": "https://myvault.vault.azure.net/",
//	    "prefix":    "myapp-",
//	}, "")
func New(options map[string]string, sessionFile string) (*Backend, error) {
	vaultURL := options["vault_url"]
	if vaultURL == "" {
		return nil, fmt.Errorf("vault_url is required for Azure Key Vault")
	}

	// Validate vault URL format
	if !strings.HasPrefix(vaultURL, "https://") || !strings.HasSuffix(vaultURL, ".vault.azure.net/") {
		return nil, fmt.Errorf("vault_url must be in format: https://<vault-name>.vault.azure.net/")
	}

	prefix := options["prefix"]
	if prefix == "" {
		prefix = "vaultmux-"
	}

	return &Backend{
		vaultURL:    vaultURL,
		prefix:      prefix,
		sessionFile: sessionFile,
	}, nil
}

// Name returns the backend identifier.
func (b *Backend) Name() string {
	return "azurekeyvault"
}

// Init initializes the Azure Key Vault client and verifies connectivity.
func (b *Backend) Init(ctx context.Context) error {
	if err := b.initCredential(); err != nil {
		return vaultmux.WrapError(b.Name(), "init", "",
			fmt.Errorf("failed to initialize Azure credential: %w", err))
	}

	// Create Azure Key Vault client
	client, err := azsecrets.NewClient(b.vaultURL, b.credential, nil)
	if err != nil {
		return vaultmux.WrapError(b.Name(), "init", "",
			fmt.Errorf("failed to create Azure Key Vault client: %w", err))
	}
	b.client = client

	// Verify connectivity with lightweight API call (list with max 1)
	pager := b.client.NewListSecretPropertiesPager(nil)
	if pager.More() {
		_, err := pager.NextPage(ctx)
		// EOF is ok (no secrets exist yet), other errors indicate connectivity issues
		if err != nil {
			return vaultmux.WrapError(b.Name(), "init", "",
				fmt.Errorf("failed to connect to Azure Key Vault: %w", err))
		}
	}

	return nil
}

// initCredential initializes Azure AD credential.
// Uses DefaultAzureCredential which tries multiple auth methods automatically.
func (b *Backend) initCredential() error {
	// Use DefaultAzureCredential (tries env vars, managed identity, CLI, etc.)
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return err
	}

	b.credential = credential
	return nil
}

// Close releases Azure Key Vault client resources.
func (b *Backend) Close() error {
	// Azure SDK doesn't require explicit cleanup
	return nil
}

// IsAuthenticated checks if Azure credentials are available.
// This is a lightweight check - actual credential validation happens on first API call.
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
	// If credential is initialized, assume credentials are available
	// Azure SDK will fail gracefully on API calls if credentials are invalid
	return b.credential != nil
}

// Authenticate returns a session wrapping Azure AD credentials.
// Unlike CLI-based backends, there's no interactive authentication -
// credentials come from environment variables, managed identity, or Azure CLI.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	if !b.IsAuthenticated(ctx) {
		return nil, vaultmux.WrapError(b.Name(), "authenticate", "",
			fmt.Errorf("Azure credentials not found - set AZURE_TENANT_ID/AZURE_CLIENT_ID/AZURE_CLIENT_SECRET or run 'az login'"))
	}

	return &azureSession{
		credential: b.credential,
		vaultURL:   b.vaultURL,
		backend:    b,
	}, nil
}

// Sync is a no-op for Azure Key Vault.
// Azure is always synchronized (cloud-native service).
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	return nil
}

// GetItem retrieves a secret from Azure Key Vault.
// Returns the latest version of the secret.
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
	if !session.IsValid(ctx) {
		return nil, vaultmux.ErrNotAuthenticated
	}

	secretName := b.secretName(name)

	// Get secret (latest version)
	resp, err := b.client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return nil, b.handleAzureError(err, "get", name)
	}

	return &vaultmux.Item{
		ID:    string(*resp.Secret.ID),
		Name:  name,
		Type:  vaultmux.ItemTypeSecureNote,
		Notes: *resp.Secret.Value,
	}, nil
}

// GetNotes retrieves only the notes field of a secret (convenience method).
func (b *Backend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
	item, err := b.GetItem(ctx, name, session)
	if err != nil {
		return "", err
	}
	return item.Notes, nil
}

// ItemExists checks if a secret exists without retrieving its value.
func (b *Backend) ItemExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
	_, err := b.GetItem(ctx, name, session)
	if err != nil {
		if errors.Is(err, vaultmux.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListItems returns all secrets matching the configured prefix.
// Azure SDK uses pager pattern for pagination.
func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
	if !session.IsValid(ctx) {
		return nil, vaultmux.ErrNotAuthenticated
	}

	var items []*vaultmux.Item

	// Create pager for listing secret properties
	pager := b.client.NewListSecretPropertiesPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, b.handleAzureError(err, "list", "")
		}

		for _, secret := range page.Value {
			// Extract secret name from ID
			// ID format: https://<vault>.vault.azure.net/secrets/<name>/<version>
			parts := strings.Split(string(*secret.ID), "/")
			if len(parts) < 5 {
				continue
			}
			fullName := parts[4]

			// Filter by prefix
			if b.prefix != "" && !strings.HasPrefix(fullName, b.prefix) {
				continue
			}

			name := strings.TrimPrefix(fullName, b.prefix)
			items = append(items, &vaultmux.Item{
				ID:   string(*secret.ID),
				Name: name,
				Type: vaultmux.ItemTypeSecureNote,
				// Notes not included (requires separate GetSecret call)
			})
		}
	}

	return items, nil
}

// CreateItem creates a new secret in Azure Key Vault.
func (b *Backend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	if !session.IsValid(ctx) {
		return vaultmux.ErrNotAuthenticated
	}

	secretName := b.secretName(name)

	// Check if already exists
	exists, err := b.ItemExists(ctx, name, session)
	if err != nil {
		return err
	}
	if exists {
		return vaultmux.ErrAlreadyExists
	}

	// Create secret
	params := azsecrets.SetSecretParameters{
		Value: &content,
	}

	_, err = b.client.SetSecret(ctx, secretName, params, nil)
	if err != nil {
		return b.handleAzureError(err, "create", name)
	}

	return nil
}

// UpdateItem updates an existing secret in Azure Key Vault.
// Azure automatically creates a new version with each update (versioning is built-in).
func (b *Backend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	if !session.IsValid(ctx) {
		return vaultmux.ErrNotAuthenticated
	}

	secretName := b.secretName(name)

	// Check if exists
	exists, err := b.ItemExists(ctx, name, session)
	if err != nil {
		return err
	}
	if !exists {
		return vaultmux.ErrNotFound
	}

	// Update secret (creates new version automatically)
	params := azsecrets.SetSecretParameters{
		Value: &content,
	}

	_, err = b.client.SetSecret(ctx, secretName, params, nil)
	if err != nil {
		return b.handleAzureError(err, "update", name)
	}

	return nil
}

// DeleteItem deletes a secret from Azure Key Vault.
// Azure uses soft-delete by default (recoverable for configured retention period).
func (b *Backend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
	if !session.IsValid(ctx) {
		return vaultmux.ErrNotAuthenticated
	}

	secretName := b.secretName(name)

	// Check if exists
	exists, err := b.ItemExists(ctx, name, session)
	if err != nil {
		return err
	}
	if !exists {
		return vaultmux.ErrNotFound
	}

	// Delete secret (soft delete - recoverable for retention period)
	_, err = b.client.DeleteSecret(ctx, secretName, nil)
	if err != nil {
		return b.handleAzureError(err, "delete", name)
	}

	return nil
}

// secretName returns the full secret name with prefix applied.
func (b *Backend) secretName(name string) string {
	if b.prefix != "" {
		return b.prefix + name
	}
	return name
}

// handleAzureError maps Azure SDK errors to vaultmux standard errors.
func (b *Backend) handleAzureError(err error, operation, itemName string) error {
	if err == nil {
		return nil
	}

	// Check for Azure ResponseError
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		switch respErr.StatusCode {
		case 404:
			return vaultmux.ErrNotFound

		case 409:
			return vaultmux.ErrAlreadyExists

		case 403:
			return vaultmux.WrapError(b.Name(), operation, itemName,
				fmt.Errorf("permission denied - check Azure RBAC permissions: %w", err))

		case 401:
			return vaultmux.WrapError(b.Name(), operation, itemName,
				fmt.Errorf("unauthenticated - check Azure AD credentials: %w", err))

		case 400:
			return vaultmux.WrapError(b.Name(), operation, itemName,
				fmt.Errorf("invalid request: %w", err))

		default:
			// Generic error with status code context
			return vaultmux.WrapError(b.Name(), operation, itemName,
				fmt.Errorf("Azure error [%d]: %w", respErr.StatusCode, err))
		}
	}

	// Generic error
	return vaultmux.WrapError(b.Name(), operation, itemName, err)
}

// Location management stubs (Azure doesn't have native "folders" like 1Password vaults).
// These operations are not supported and return ErrNotSupported.
// Could be implemented using tags in the future, but not currently supported.

func (b *Backend) ListLocations(ctx context.Context, session vaultmux.Session) ([]string, error) {
	return nil, vaultmux.ErrNotSupported
}

func (b *Backend) LocationExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
	return false, vaultmux.ErrNotSupported
}

func (b *Backend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
	return vaultmux.ErrNotSupported
}

func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
	return nil, vaultmux.ErrNotSupported
}

// init registers the Azure Key Vault backend with vaultmux.
func init() {
	vaultmux.RegisterBackend(vaultmux.BackendAzureKeyVault,
		func(cfg vaultmux.Config) (vaultmux.Backend, error) {
			return New(cfg.Options, cfg.SessionFile)
		})
}
