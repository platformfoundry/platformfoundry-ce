package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey     []byte
	issuer        string
	tokenDuration time.Duration
}

// Claims represents JWT claims
type Claims struct {
	Username     string   `json:"username"`
	Email        string   `json:"email,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	Organization string   `json:"organization,omitempty"`
	jwt.RegisteredClaims
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, issuer string, duration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		issuer:        issuer,
		tokenDuration: duration,
	}
}

// GenerateToken generates a new JWT token for a user
func (m *JWTManager) GenerateToken(user *User) (string, error) {
	now := time.Now()
	claims := Claims{
		Username:     user.Username,
		Email:        user.Email,
		Roles:        user.Roles,
		Organization: user.Organization,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// RefreshToken generates a new token from an existing valid token
func (m *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Create new token with updated expiration
	now := time.Now()
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(m.tokenDuration))
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}
