package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserContextKey is the context key for storing user information
	UserContextKey contextKey = "user"
	// ClaimsContextKey is the context key for storing JWT claims
	ClaimsContextKey contextKey = "claims"
)

// AuthMiddleware provides HTTP authentication middleware
type AuthMiddleware struct {
	jwtManager  *JWTManager
	apiKeyStore *APIKeyStore
	userStore   *UserStore
	enabled     bool
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtManager *JWTManager, apiKeyStore *APIKeyStore, userStore *UserStore, enabled bool) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager:  jwtManager,
		apiKeyStore: apiKeyStore,
		userStore:   userStore,
		enabled:     enabled,
	}
}

// Authenticate is the main authentication middleware
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if authentication is disabled
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from request
		token, tokenType := m.extractToken(r)
		if token == "" {
			m.unauthorized(w, "missing authentication token")
			return
		}

		var user *User
		var claims *Claims

		// Validate based on token type
		switch tokenType {
		case "Bearer": // JWT token
			var err error
			claims, err = m.jwtManager.ValidateToken(token)
			if err != nil {
				m.unauthorized(w, "invalid token: "+err.Error())
				return
			}

			// Get user from store
			user, err = m.userStore.GetUser(claims.Username)
			if err != nil {
				m.unauthorized(w, "user not found")
				return
			}

			if !user.Enabled {
				m.unauthorized(w, "user disabled")
				return
			}

		case "ApiKey": // API key
			apiKey, err := m.apiKeyStore.ValidateAPIKey(token)
			if err != nil {
				m.unauthorized(w, "invalid API key")
				return
			}

			// Get user from API key
			user, err = m.userStore.GetUser(apiKey.Username)
			if err != nil {
				m.unauthorized(w, "user not found")
				return
			}

			// Convert API key to claims format
			claims = &Claims{
				Username:     apiKey.Username,
				Roles:        apiKey.Roles,
				Organization: apiKey.Organization,
			}

		default:
			m.unauthorized(w, "unsupported authentication type")
			return
		}

		// Add user and claims to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		ctx = context.WithValue(ctx, ClaimsContextKey, claims)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware checks if user has required role
func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsContextKey).(*Claims)
			if !ok {
				m.forbidden(w, "no authentication context")
				return
			}

			// Check if user has required role
			hasRole := false
			for _, r := range claims.Roles {
				if r == role || r == "admin" { // admin has all roles
					hasRole = true
					break
				}
			}

			if !hasRole {
				m.forbidden(w, "insufficient permissions: requires role "+role)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrganization middleware checks if user belongs to organization
func (m *AuthMiddleware) RequireOrganization(org string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsContextKey).(*Claims)
			if !ok {
				m.forbidden(w, "no authentication context")
				return
			}

			if claims.Organization != org {
				m.forbidden(w, "access denied: wrong organization")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken extracts authentication token from request
// Returns: (token, tokenType)
// Supports:
// - Authorization: Bearer <jwt-token>
// - Authorization: ApiKey <api-key>
// - X-API-Key: <api-key>
func (m *AuthMiddleware) extractToken(r *http.Request) (string, string) {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 {
			authType := parts[0]
			token := parts[1]

			switch authType {
			case "Bearer":
				return token, "Bearer"
			case "ApiKey":
				return token, "ApiKey"
			}
		}
	}

	// Try X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey, "ApiKey"
	}

	return "", ""
}

// unauthorized sends a 401 Unauthorized response
func (m *AuthMiddleware) unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// forbidden sends a 403 Forbidden response
func (m *AuthMiddleware) forbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// GetUserFromContext retrieves the authenticated user from request context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(UserContextKey).(*User)
	return user, ok
}

// GetClaimsFromContext retrieves the JWT claims from request context
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}
