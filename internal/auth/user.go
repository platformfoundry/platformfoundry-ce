package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a platform user
type User struct {
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"passwordHash"`
	Roles        []string  `json:"roles"`
	Organization string    `json:"organization,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Enabled      bool      `json:"enabled"`
}

// UserStore manages user persistence
type UserStore struct {
	filePath string
	users    map[string]*User
}

// NewUserStore creates a new user store
func NewUserStore(configDir string) (*UserStore, error) {
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, ".platformfoundry")
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(configDir, "users.json")
	store := &UserStore{
		filePath: filePath,
		users:    make(map[string]*User),
	}

	// Load existing users
	if err := store.load(); err != nil {
		// If file doesn't exist, create empty store
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

// CreateUser creates a new user with hashed password
func (s *UserStore) CreateUser(username, email, password string, roles []string, organization string) error {
	if _, exists := s.users[username]; exists {
		return fmt.Errorf("user %s already exists", username)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Roles:        roles,
		Organization: organization,
		CreatedAt:    now,
		UpdatedAt:    now,
		Enabled:      true,
	}

	s.users[username] = user
	return s.save()
}

// GetUser retrieves a user by username
func (s *UserStore) GetUser(username string) (*User, error) {
	user, exists := s.users[username]
	if !exists {
		return nil, fmt.Errorf("user %s not found", username)
	}
	return user, nil
}

// ListUsers returns all users
func (s *UserStore) ListUsers() []*User {
	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

// UpdateUser updates an existing user
func (s *UserStore) UpdateUser(user *User) error {
	if _, exists := s.users[user.Username]; !exists {
		return fmt.Errorf("user %s not found", user.Username)
	}

	user.UpdatedAt = time.Now()
	s.users[user.Username] = user
	return s.save()
}

// DeleteUser removes a user
func (s *UserStore) DeleteUser(username string) error {
	if _, exists := s.users[username]; !exists {
		return fmt.Errorf("user %s not found", username)
	}

	delete(s.users, username)
	return s.save()
}

// Authenticate verifies username and password
func (s *UserStore) Authenticate(username, password string) (*User, error) {
	user, err := s.GetUser(username)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: invalid credentials")
	}

	if !user.Enabled {
		return nil, fmt.Errorf("authentication failed: user disabled")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("authentication failed: invalid credentials")
	}

	return user, nil
}

// UpdatePassword changes a user's password
func (s *UserStore) UpdatePassword(username, newPassword string) error {
	user, err := s.GetUser(username)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	s.users[username] = user
	return s.save()
}

// load reads users from disk
func (s *UserStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var users map[string]*User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	s.users = users
	return nil
}

// save writes users to disk
func (s *UserStore) save() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}
