package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

// SSOProvider represents an SSO provider configuration
type SSOProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Config       *oauth2.Config
}

// SSOProviderType represents the type of SSO provider
type SSOProviderType string

const (
	SSOProviderGoogle  SSOProviderType = "google"
	SSOProviderGitHub  SSOProviderType = "github"
	SSOProviderOkta    SSOProviderType = "okta"
	SSOProviderAzureAD SSOProviderType = "azure-ad"
	SSOProviderGeneric SSOProviderType = "generic"
)

// SSOConfig represents SSO configuration
type SSOConfig struct {
	Provider     SSOProviderType `yaml:"provider" json:"provider"`
	ClientID     string          `yaml:"clientId" json:"clientId"`
	ClientSecret string          `yaml:"clientSecret" json:"clientSecret"`
	RedirectURL  string          `yaml:"redirectUrl" json:"redirectUrl"`
	Domain       string          `yaml:"domain,omitempty" json:"domain,omitempty"`     // For Okta, Azure AD
	TenantID     string          `yaml:"tenantId,omitempty" json:"tenantId,omitempty"` // For Azure AD
	Scopes       []string        `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

// NewSSOProvider creates a new SSO provider
func NewSSOProvider(config *SSOConfig) (*SSOProvider, error) {
	var oauthConfig *oauth2.Config

	switch config.Provider {
	case SSOProviderGoogle:
		oauthConfig = &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
		}

	case SSOProviderGitHub:
		oauthConfig = &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
			Scopes: []string{"user:email", "read:user"},
		}

	case SSOProviderAzureAD:
		if config.TenantID == "" {
			return nil, fmt.Errorf("tenant ID is required for Azure AD")
		}
		oauthConfig = &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Endpoint:     microsoft.AzureADEndpoint(config.TenantID),
			Scopes: []string{
				"openid",
				"profile",
				"email",
				"User.Read",
			},
		}

	case SSOProviderOkta:
		if config.Domain == "" {
			return nil, fmt.Errorf("domain is required for Okta")
		}
		oauthConfig = &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://%s/oauth2/v1/authorize", config.Domain),
				TokenURL: fmt.Sprintf("https://%s/oauth2/v1/token", config.Domain),
			},
			Scopes: []string{"openid", "profile", "email"},
		}

	default:
		return nil, fmt.Errorf("unsupported SSO provider: %s", config.Provider)
	}

	// Override scopes if provided
	if len(config.Scopes) > 0 {
		oauthConfig.Scopes = config.Scopes
	}

	return &SSOProvider{
		Name:         string(config.Provider),
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       oauthConfig.Scopes,
		Config:       oauthConfig,
	}, nil
}

// GetAuthURL returns the OAuth authorization URL
func (p *SSOProvider) GetAuthURL(state string) string {
	return p.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for a token
func (p *SSOProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.Config.Exchange(ctx, code)
}

// GetUserInfo retrieves user information from the provider
func (p *SSOProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (map[string]interface{}, error) {
	client := p.Config.Client(ctx, token)

	var userInfoURL string
	switch p.Name {
	case string(SSOProviderGoogle):
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	case string(SSOProviderGitHub):
		userInfoURL = "https://api.github.com/user"
	case string(SSOProviderAzureAD):
		userInfoURL = "https://graph.microsoft.com/v1.0/me"
	case string(SSOProviderOkta):
		// For Okta, extract from ID token claims
		return parseJWTClaims(token.Extra("id_token").(string))
	default:
		return nil, fmt.Errorf("unsupported provider for user info: %s", p.Name)
	}

	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

// NormalizeUserInfo converts provider-specific user info to standard format
func (p *SSOProvider) NormalizeUserInfo(userInfo map[string]interface{}) *SAMLResponse {
	// Convert attributes to string map
	attributes := make(map[string]string)
	for k, v := range userInfo {
		if str, ok := v.(string); ok {
			attributes[k] = str
		} else {
			attributes[k] = fmt.Sprintf("%v", v)
		}
	}

	response := &SAMLResponse{
		Attributes: attributes,
	}

	switch p.Name {
	case string(SSOProviderGoogle):
		response.Email = getString(userInfo, "email")
		response.Username = getString(userInfo, "email")
		response.DisplayName = getString(userInfo, "name")

	case string(SSOProviderGitHub):
		response.Email = getString(userInfo, "email")
		response.Username = getString(userInfo, "login")
		response.DisplayName = getString(userInfo, "name")
		if response.Email == "" {
			// Fetch email separately for GitHub
			response.Email = response.Username + "@github.com"
		}

	case string(SSOProviderAzureAD):
		response.Email = getString(userInfo, "mail")
		if response.Email == "" {
			response.Email = getString(userInfo, "userPrincipalName")
		}
		response.Username = getString(userInfo, "userPrincipalName")
		response.DisplayName = getString(userInfo, "displayName")

	case string(SSOProviderOkta):
		response.Email = getString(userInfo, "email")
		response.Username = getString(userInfo, "preferred_username")
		if response.Username == "" {
			response.Username = response.Email
		}
		response.DisplayName = getString(userInfo, "name")
	}

	// Use email as fallback for username
	if response.Username == "" {
		response.Username = response.Email
	}

	return response
}

// getString safely gets a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// parseJWTClaims parses JWT token claims (simplified)
func parseJWTClaims(token string) (map[string]interface{}, error) {
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode payload (second part)
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return claims, nil
}

// base64URLDecode decodes base64 URL-encoded string
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	return base64.StdEncoding.DecodeString(s)
}

// GetGitHubEmails fetches email addresses from GitHub API
func (p *SSOProvider) GetGitHubEmails(ctx context.Context, token *oauth2.Token) ([]string, error) {
	if p.Name != string(SSOProviderGitHub) {
		return nil, fmt.Errorf("not a GitHub provider")
	}

	client := p.Config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(emails))
	for _, e := range emails {
		if e.Verified {
			result = append(result, e.Email)
		}
	}

	return result, nil
}
