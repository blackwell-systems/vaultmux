// Package gcpsecrets implements the vaultmux.Backend interface for Google Cloud Secret Manager.
//
// GCP Secret Manager is Google Cloud's managed secret storage service with features like
// automatic replication, IAM-based access control, and audit logging via Cloud Audit Logs.
//
// This backend uses the official Google Cloud Go SDK, making it the second SDK-based backend
// (after AWS Secrets Manager). The GCP SDK is notably simpler than AWS, with cleaner APIs.
package gcpsecrets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/blackwell-systems/vaultmux"
)

// Backend implements vaultmux.Backend for GCP Secret Manager.
type Backend struct {
	// GCP Secret Manager client
	client *secretmanager.Client

	// Configuration
	projectID string // GCP project ID (required, e.g., "my-project-123")
	prefix    string // Secret name prefix for namespacing (e.g., "myapp-")
	endpoint  string // Custom endpoint for testing (optional)

	// Session cache file (currently unused - GCP credentials are long-lived)
	sessionFile string
}

// New creates a new GCP Secret Manager backend.
//
// Supported options:
//   - project_id: GCP project ID (required)
//   - prefix: Secret name prefix for namespacing (default: "vaultmux-")
//   - endpoint: Custom endpoint URL (for fake-gcp-server testing, optional)
//
// Authentication uses Application Default Credentials (ADC):
//   - GOOGLE_APPLICATION_CREDENTIALS env var pointing to service account JSON
//   - gcloud CLI credentials (gcloud auth application-default login)
//   - GCE/GKE metadata server (automatic for Compute Engine/Kubernetes)
//
// Example:
//
//	backend, err := gcpsecrets.New(map[string]string{
//	    "project_id": "my-gcp-project",
//	    "prefix":     "myapp-",
//	}, "")
func New(options map[string]string, sessionFile string) (*Backend, error) {
	projectID := options["project_id"]
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required for GCP Secret Manager")
	}

	prefix := options["prefix"]
	if prefix == "" {
		prefix = "vaultmux-"
	}

	endpoint := options["endpoint"]

	return &Backend{
		projectID:   projectID,
		prefix:      prefix,
		endpoint:    endpoint,
		sessionFile: sessionFile,
	}, nil
}

// Name returns the backend identifier.
func (b *Backend) Name() string {
	return "gcpsecrets"
}

// Init initializes the GCP Secret Manager client and verifies connectivity.
func (b *Backend) Init(ctx context.Context) error {
	if err := b.initGCPClient(ctx); err != nil {
		return vaultmux.WrapError(b.Name(), "init", "",
			fmt.Errorf("failed to initialize GCP client: %w", err))
	}

	// Verify connectivity with lightweight API call (list with limit 1)
	parent := fmt.Sprintf("projects/%s", b.projectID)
	req := &secretmanagerpb.ListSecretsRequest{
		Parent:   parent,
		PageSize: 1,
	}

	iter := b.client.ListSecrets(ctx, req)
	_, err := iter.Next()

	// EOF is ok (no secrets exist yet)
	if err != nil && err != iterator.Done {
		return vaultmux.WrapError(b.Name(), "init", "",
			fmt.Errorf("failed to connect to GCP Secret Manager: %w", err))
	}

	return nil
}

// initGCPClient creates a new GCP Secret Manager client.
// Uses Application Default Credentials (ADC) for authentication.
func (b *Backend) initGCPClient(ctx context.Context) error {
	var opts []option.ClientOption

	// Custom endpoint for testing (e.g., gcp-secret-manager-mock)
	if b.endpoint != "" {
		opts = append(opts, option.WithEndpoint(b.endpoint))
		opts = append(opts, option.WithoutAuthentication()) // Skip auth for mock servers
		// Use insecure transport for local mock servers (no TLS)
		opts = append(opts, option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	client, err := secretmanager.NewClient(ctx, opts...)
	if err != nil {
		return err
	}

	b.client = client
	return nil
}

// Close releases GCP client resources.
func (b *Backend) Close() error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

// IsAuthenticated checks if GCP credentials are available.
// This is a lightweight check - actual credential validation happens on first API call.
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
	// If client is initialized, assume credentials are available
	// GCP SDK will fail gracefully on API calls if credentials are invalid
	return b.client != nil
}

// Authenticate returns a session wrapping GCP credentials.
// Unlike CLI-based backends, there's no interactive authentication -
// credentials come from GOOGLE_APPLICATION_CREDENTIALS, gcloud CLI, or GCE/GKE metadata.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	if !b.IsAuthenticated(ctx) {
		return nil, vaultmux.WrapError(b.Name(), "authenticate", "",
			fmt.Errorf("GCP credentials not found - set GOOGLE_APPLICATION_CREDENTIALS or run 'gcloud auth application-default login'"))
	}

	return &gcpSession{
		projectID: b.projectID,
		backend:   b,
	}, nil
}

// Sync is a no-op for GCP Secret Manager.
// GCP is always synchronized (cloud-native service).
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	return nil
}

// GetItem retrieves a secret from GCP Secret Manager.
// Returns the latest version of the secret.
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
	if !session.IsValid(ctx) {
		return nil, vaultmux.ErrNotAuthenticated
	}

	secretName := b.secretName(name)
	// GCP secret path format: projects/{project}/secrets/{secret}/versions/latest
	versionName := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", b.projectID, secretName)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: versionName,
	}

	result, err := b.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, b.handleGCPError(err, "get", name)
	}

	// Get secret metadata for full item info
	secretPath := fmt.Sprintf("projects/%s/secrets/%s", b.projectID, secretName)
	secret, err := b.client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: secretPath,
	})
	if err != nil {
		return nil, b.handleGCPError(err, "get-metadata", name)
	}

	return &vaultmux.Item{
		ID:    secret.Name, // Full resource name
		Name:  name,        // User-provided name (without prefix)
		Type:  vaultmux.ItemTypeSecureNote,
		Notes: string(result.Payload.Data),
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
// GCP API supports simple iteration (no complex pagination like AWS).
func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
	if !session.IsValid(ctx) {
		return nil, vaultmux.ErrNotAuthenticated
	}

	parent := fmt.Sprintf("projects/%s", b.projectID)
	req := &secretmanagerpb.ListSecretsRequest{
		Parent:   parent,
		PageSize: 100, // Max per page
	}

	var items []*vaultmux.Item
	iter := b.client.ListSecrets(ctx, req)

	for {
		secret, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, b.handleGCPError(err, "list", "")
		}

		// Extract secret name from full path: projects/{project}/secrets/{name}
		parts := strings.Split(secret.Name, "/")
		if len(parts) < 4 {
			continue
		}
		fullName := parts[3]

		// Filter by prefix
		if b.prefix != "" && !strings.HasPrefix(fullName, b.prefix) {
			continue
		}

		name := strings.TrimPrefix(fullName, b.prefix)
		items = append(items, &vaultmux.Item{
			ID:   secret.Name, // Full resource name
			Name: name,
			Type: vaultmux.ItemTypeSecureNote,
			// Notes not included (requires separate AccessSecretVersion call)
		})
	}

	return items, nil
}

// CreateItem creates a new secret in GCP Secret Manager.
// GCP requires two operations: CreateSecret (metadata) + AddSecretVersion (content).
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

	// Step 1: Create secret (metadata only)
	parent := fmt.Sprintf("projects/%s", b.projectID)
	createReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: secretName,
		Secret: &secretmanagerpb.Secret{
			Labels: map[string]string{
				"vaultmux": "true",
				"prefix":   b.prefix,
			},
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}

	secret, err := b.client.CreateSecret(ctx, createReq)
	if err != nil {
		return b.handleGCPError(err, "create", name)
	}

	// Step 2: Add secret version (actual content)
	addReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(content),
		},
	}

	_, err = b.client.AddSecretVersion(ctx, addReq)
	if err != nil {
		return b.handleGCPError(err, "add-version", name)
	}

	return nil
}

// UpdateItem updates an existing secret in GCP Secret Manager.
// GCP automatically creates a new version with each update (versioning is built-in).
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

	// Add new secret version (GCP's way of "updating")
	secretPath := fmt.Sprintf("projects/%s/secrets/%s", b.projectID, secretName)
	req := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretPath,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(content),
		},
	}

	_, err = b.client.AddSecretVersion(ctx, req)
	if err != nil {
		return b.handleGCPError(err, "update", name)
	}

	return nil
}

// DeleteItem deletes a secret from GCP Secret Manager.
// GCP deletion is immediate (unlike AWS which has recovery periods).
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

	secretPath := fmt.Sprintf("projects/%s/secrets/%s", b.projectID, secretName)
	req := &secretmanagerpb.DeleteSecretRequest{
		Name: secretPath,
	}

	err = b.client.DeleteSecret(ctx, req)
	if err != nil {
		return b.handleGCPError(err, "delete", name)
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

// handleGCPError maps GCP gRPC errors to vaultmux standard errors.
func (b *Backend) handleGCPError(err error, operation, itemName string) error {
	if err == nil {
		return nil
	}

	// Extract gRPC status code
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, wrap and return
		return vaultmux.WrapError(b.Name(), operation, itemName, err)
	}

	switch st.Code() {
	case codes.NotFound:
		return vaultmux.ErrNotFound

	case codes.AlreadyExists:
		return vaultmux.ErrAlreadyExists

	case codes.PermissionDenied:
		return vaultmux.WrapError(b.Name(), operation, itemName,
			fmt.Errorf("permission denied - check IAM permissions: %w", err))

	case codes.Unauthenticated:
		return vaultmux.WrapError(b.Name(), operation, itemName,
			fmt.Errorf("unauthenticated - check GCP credentials: %w", err))

	case codes.InvalidArgument:
		return vaultmux.WrapError(b.Name(), operation, itemName,
			fmt.Errorf("invalid argument: %w", err))

	default:
		// Generic error with gRPC code context
		return vaultmux.WrapError(b.Name(), operation, itemName,
			fmt.Errorf("GCP error [%s]: %w", st.Code(), err))
	}
}

// Location management stubs (GCP doesn't have native "folders" like 1Password vaults).
// These operations are not supported and return ErrNotSupported.
// Could be implemented using labels in the future, but not currently supported.

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

// init registers the GCP Secret Manager backend with vaultmux.
func init() {
	vaultmux.RegisterBackend(vaultmux.BackendGCPSecretManager,
		func(cfg vaultmux.Config) (vaultmux.Backend, error) {
			return New(cfg.Options, cfg.SessionFile)
		})
}
