package awssecrets

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// awsSession implements vaultmux.Session for AWS Secrets Manager.
// Unlike token-based backends (Bitwarden, 1Password), AWS uses IAM credentials
// which are either long-lived (static keys) or SDK-managed (STS temporary credentials).
type awsSession struct {
	config  aws.Config
	backend *Backend
}

// Token returns the AWS Access Key ID for debugging purposes.
// Note: This is not a traditional session token - AWS uses IAM credentials.
func (s *awsSession) Token() string {
	if s.config.Credentials == nil {
		return ""
	}

	creds, err := s.config.Credentials.Retrieve(context.Background())
	if err != nil {
		return ""
	}
	// Return AccessKeyID (not the full credentials for security)
	return creds.AccessKeyID
}

// IsValid checks if AWS credentials can be retrieved.
// For static IAM keys, this always returns true if credentials exist.
// For STS temporary credentials, the SDK handles automatic refresh.
func (s *awsSession) IsValid(ctx context.Context) bool {
	if s.config.Credentials == nil {
		return false
	}

	_, err := s.config.Credentials.Retrieve(ctx)
	return err == nil
}

// Refresh re-initializes the AWS config to pick up new credentials.
// This is primarily useful if credentials have changed in the environment.
func (s *awsSession) Refresh(ctx context.Context) error {
	return s.backend.initAWSConfig(ctx)
}

// ExpiresAt returns zero time because AWS credentials don't have simple expiration:
// - Static IAM keys never expire
// - STS temporary credentials are auto-refreshed by the SDK
// - EC2 instance role credentials are auto-refreshed by the SDK
func (s *awsSession) ExpiresAt() time.Time {
	return time.Time{} // Zero value indicates no expiration
}
