// Package auth handles password hashing and session-token generation. Sessions
// themselves are opaque random tokens persisted by the store and carried in an
// HttpOnly cookie.
package auth

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of pw.
func HashPassword(pw string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}

// CheckPassword reports whether pw matches hash.
func CheckPassword(hash []byte, pw string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(pw)) == nil
}

// NewSessionToken returns a 32-byte cryptographically random URL-safe token.
func NewSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
