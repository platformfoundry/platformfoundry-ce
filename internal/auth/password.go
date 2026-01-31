package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// bcryptHash hashes a password using bcrypt with the specified cost
func bcryptHash(password []byte, cost int) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, cost)
}

// bcryptCompare compares a hashed password with a plaintext password
func bcryptCompare(hashedPassword, password []byte) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}
