// Package pai defines the Platform Auth Interface (PAI).
// This handles authentication method abstraction.
package pai

import (
	"context"
	"time"
)

// AuthMethod is the main interface for authentication.
// Implementations handle different authentication mechanisms.
type AuthMethod interface {
	// Type returns the type of this auth method (e.g., "jwt", "oauth", "saml")
	Type() string

	// Authenticate verifies credentials and returns an identity
	Authenticate(ctx context.Context, credentials interface{}) (*Identity, error)

	// Validate verifies a token and returns the associated identity
	Validate(ctx context.Context, token string) (*Identity, error)

	// Refresh exchanges a token for a new one
	Refresh(ctx context.Context, token string) (*TokenResponse, error)

	// Revoke invalidates a token
	Revoke(ctx context.Context, token string) error

	// Close releases auth method resources
	Close() error
}

// Identity represents an authenticated identity
type Identity struct {
	// ID is the unique identifier for this identity
	ID string

	// Type is the identity type (e.g., "user", "service", "machine")
	Type IdentityType

	// Name is the display name
	Name string

	// Email is the email address (if applicable)
	Email string

	// Groups contains group memberships
	Groups []string

	// Roles contains role assignments
	Roles []string

	// Metadata contains additional identity information
	Metadata map[string]interface{}

	// AuthenticatedAt is when the identity was authenticated
	AuthenticatedAt time.Time

	// ExpiresAt is when the identity authentication expires
	ExpiresAt time.Time

	// Issuer is who issued this identity
	Issuer string
}

// IdentityType represents the type of identity
type IdentityType string

const (
	IdentityTypeUser    IdentityType = "user"
	IdentityTypeService IdentityType = "service"
	IdentityTypeMachine IdentityType = "machine"
	IdentityTypeUnknown IdentityType = "unknown"
)

// TokenResponse contains the result of token operations
type TokenResponse struct {
	// AccessToken is the new access token
	AccessToken string

	// RefreshToken is a token that can be used to get new access tokens
	RefreshToken string

	// TokenType is the type of token (e.g., "Bearer")
	TokenType string

	// ExpiresIn is how long until the access token expires
	ExpiresIn time.Duration

	// Scope is the granted scope
	Scope []string

	// Identity is the associated identity
	Identity *Identity
}

// Credentials represents authentication credentials
type Credentials interface {
	// Type returns the credential type
	Type() string
}

// UsernamePasswordCredentials represents username/password credentials
type UsernamePasswordCredentials struct {
	Username string
	Password string
}

func (c UsernamePasswordCredentials) Type() string {
	return "username_password"
}

// APIKeyCredentials represents API key credentials
type APIKeyCredentials struct {
	Key string
}

func (c APIKeyCredentials) Type() string {
	return "api_key"
}

// TokenCredentials represents token-based credentials
type TokenCredentials struct {
	Token string
}

func (c TokenCredentials) Type() string {
	return "token"
}

// OAuthCredentials represents OAuth credentials
type OAuthCredentials struct {
	Code         string
	RedirectURI  string
	ClientID     string
	ClientSecret string
}

func (c OAuthCredentials) Type() string {
	return "oauth"
}

// SAMLCredentials represents SAML assertion credentials
type SAMLCredentials struct {
	Assertion string
}

func (c SAMLCredentials) Type() string {
	return "saml"
}

// Common errors for auth operations
var (
	// ErrInvalidCredentials indicates invalid credentials
	ErrInvalidCredentials = authError("invalid credentials")

	// ErrTokenExpired indicates the token has expired
	ErrTokenExpired = authError("token expired")

	// ErrTokenInvalid indicates the token is invalid
	ErrTokenInvalid = authError("invalid token")

	// ErrAuthMethodNotSupported indicates the auth method is not supported
	ErrAuthMethodNotSupported = authError("auth method not supported")

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = authError("permission denied")

	// ErrAccountLocked indicates the account is locked
	ErrAccountLocked = authError("account locked")

	// ErrAccountDisabled indicates the account is disabled
	ErrAccountDisabled = authError("account disabled")
)

type authError string

func (e authError) Error() string {
	return string(e)
}

// AuthConfig holds configuration for an auth method
type AuthConfig struct {
	// Type is the auth method type
	Type string

	// Config contains method-specific configuration
	Config map[string]interface{}
}

// Permission represents a permission check
type Permission struct {
	// Resource is the resource being accessed
	Resource string

	// Action is the action being performed
	Action string

	// Conditions are additional conditions
	Conditions map[string]interface{}
}

// Authorizer checks if an identity has permission
type Authorizer interface {
	// Authorize checks if the identity can perform the action
	Authorize(ctx context.Context, identity *Identity, permission Permission) (bool, error)
}
