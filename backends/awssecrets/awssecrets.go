// Package awssecrets implements the vaultmux.Backend interface for AWS Secrets Manager.
//
// AWS Secrets Manager is AWS's native secret management service with features like
// automatic rotation, audit logging, and IAM-based access control.
//
// This backend uses the AWS SDK for Go v2, making it the first SDK-based backend
// (as opposed to CLI wrappers like Bitwarden, 1Password, and pass).
package awssecrets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/blackwell-systems/vaultmux"
)

// Backend implements vaultmux.Backend for AWS Secrets Manager.
type Backend struct {
	// AWS Secrets Manager client
	client *secretsmanager.Client

	// Configuration
	region   string // AWS region (e.g., us-east-1, us-west-2)
	prefix   string // Secret name prefix for namespacing (e.g., "myapp/")
	endpoint string // Custom endpoint URL for LocalStack testing

	// AWS config (credentials, region)
	awsConfig aws.Config

	// Session cache file (currently unused - AWS credentials are long-lived)
	sessionFile string
}

// New creates a new AWS Secrets Manager backend.
//
// Supported options:
//   - region: AWS region (default: us-east-1)
//   - prefix: Secret name prefix for namespacing (default: "vaultmux/")
//   - endpoint: Custom endpoint URL (for LocalStack testing)
//
// Example:
//
//	backend, err := awssecrets.New(map[string]string{
//	    "region": "us-west-2",
//	    "prefix": "myapp/",
//	}, "")
func New(options map[string]string, sessionFile string) (*Backend, error) {
	region := options["region"]
	if region == "" {
		region = "us-east-1"
	}

	prefix := options["prefix"]
	if prefix == "" {
		prefix = "vaultmux/"
	}

	endpoint := options["endpoint"]

	return &Backend{
		region:      region,
		prefix:      prefix,
		endpoint:    endpoint,
		sessionFile: sessionFile,
	}, nil
}

// Name returns the backend identifier.
func (b *Backend) Name() string {
	return "awssecrets"
}

// Init initializes the AWS Secrets Manager client and verifies connectivity.
func (b *Backend) Init(ctx context.Context) error {
	// Load AWS configuration (credentials, region)
	if err := b.initAWSConfig(ctx); err != nil {
		return vaultmux.WrapError(b.Name(), "init", "",
			fmt.Errorf("failed to load AWS config: %w", err))
	}

	// Create Secrets Manager client
	b.client = secretsmanager.NewFromConfig(b.awsConfig, func(o *secretsmanager.Options) {
		if b.endpoint != "" {
			o.BaseEndpoint = aws.String(b.endpoint)
		}
	})

	// Verify connectivity with lightweight API call
	_, err := b.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		return vaultmux.WrapError(b.Name(), "init", "",
			fmt.Errorf("failed to connect to AWS Secrets Manager: %w", err))
	}

	return nil
}

// initAWSConfig loads AWS configuration from environment, shared config, or instance metadata.
func (b *Backend) initAWSConfig(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(b.region),
	)
	if err != nil {
		return err
	}

	b.awsConfig = cfg
	return nil
}

// Close releases resources. AWS SDK clients don't require explicit cleanup.
func (b *Backend) Close() error {
	return nil
}

// IsAuthenticated checks if AWS credentials are available.
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
	if b.awsConfig.Credentials == nil {
		return false
	}

	_, err := b.awsConfig.Credentials.Retrieve(ctx)
	return err == nil
}

// Authenticate returns a session wrapping AWS credentials.
// Unlike CLI-based backends, there's no interactive authentication -
// credentials come from environment variables, shared config, or instance roles.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	if !b.IsAuthenticated(ctx) {
		return nil, vaultmux.WrapError(b.Name(), "authenticate", "",
			fmt.Errorf("AWS credentials not found - set AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY or configure ~/.aws/credentials"))
	}

	return &awsSession{
		config:  b.awsConfig,
		backend: b,
	}, nil
}

// Sync is a no-op for AWS Secrets Manager.
// AWS is always synchronized (cloud-native service).
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	return nil
}

// GetItem retrieves a secret from AWS Secrets Manager.
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
	if !session.IsValid(ctx) {
		return nil, vaultmux.ErrNotAuthenticated
	}

	secretName := b.secretName(name)

	result, err := b.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, b.handleAWSError(err, "get", name)
	}

	return &vaultmux.Item{
		ID:    aws.ToString(result.ARN),
		Name:  name,
		Type:  vaultmux.ItemTypeSecureNote,
		Notes: aws.ToString(result.SecretString),
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
// Handles pagination automatically for large secret collections.
func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
	if !session.IsValid(ctx) {
		return nil, vaultmux.ErrNotAuthenticated
	}

	var items []*vaultmux.Item
	var nextToken *string

	for {
		// Note: LocalStack doesn't support wildcard filtering (e.g., "prefix/*")
		// so we list all secrets and filter in Go code
		result, err := b.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
			MaxResults: aws.Int32(100),
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, b.handleAWSError(err, "list", "")
		}

		for _, secret := range result.SecretList {
			secretName := aws.ToString(secret.Name)

			// Filter by prefix
			if b.prefix != "" && !strings.HasPrefix(secretName, b.prefix) {
				continue
			}

			name := strings.TrimPrefix(secretName, b.prefix)
			items = append(items, &vaultmux.Item{
				ID:   aws.ToString(secret.ARN),
				Name: name,
				Type: vaultmux.ItemTypeSecureNote,
				// Notes field not populated - requires separate GetSecretValue call
			})
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return items, nil
}

// CreateItem creates a new secret in AWS Secrets Manager.
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

	_, err = b.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(content),
		Tags: []types.Tag{
			{Key: aws.String("vaultmux"), Value: aws.String("true")},
			{Key: aws.String("prefix"), Value: aws.String(b.prefix)},
		},
	})
	if err != nil {
		return b.handleAWSError(err, "create", name)
	}

	return nil
}

// UpdateItem updates an existing secret in AWS Secrets Manager.
// AWS automatically creates a new version with each update.
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

	_, err = b.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretName),
		SecretString: aws.String(content),
	})
	if err != nil {
		return b.handleAWSError(err, "update", name)
	}

	return nil
}

// DeleteItem deletes a secret from AWS Secrets Manager.
// Uses ForceDeleteWithoutRecovery for immediate deletion (consistent with other backends).
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

	_, err = b.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(secretName),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	if err != nil {
		return b.handleAWSError(err, "delete", name)
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

// handleAWSError maps AWS SDK errors to vaultmux standard errors.
func (b *Backend) handleAWSError(err error, operation, itemName string) error {
	if err == nil {
		return nil
	}

	// Resource not found
	var rnf *types.ResourceNotFoundException
	if errors.As(err, &rnf) {
		return vaultmux.ErrNotFound
	}

	// Resource already exists
	var rae *types.ResourceExistsException
	if errors.As(err, &rae) {
		return vaultmux.ErrAlreadyExists
	}

	// Invalid request
	var ire *types.InvalidRequestException
	if errors.As(err, &ire) {
		return vaultmux.WrapError(b.Name(), operation, itemName,
			fmt.Errorf("invalid request: %w", err))
	}

	// Invalid parameter (could indicate IAM permissions issue)
	var ipe *types.InvalidParameterException
	if errors.As(err, &ipe) {
		return vaultmux.WrapError(b.Name(), operation, itemName,
			fmt.Errorf("invalid parameter: %w", err))
	}

	// Generic error
	return vaultmux.WrapError(b.Name(), operation, itemName, err)
}

// Location management stubs (AWS doesn't have native "folders" like 1Password vaults)
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

// init registers the AWS Secrets Manager backend with vaultmux.
func init() {
	vaultmux.RegisterBackend(vaultmux.BackendAWSSecretsManager,
		func(cfg vaultmux.Config) (vaultmux.Backend, error) {
			return New(cfg.Options, cfg.SessionFile)
		})
}
