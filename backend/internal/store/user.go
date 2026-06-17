package store

import "time"

// User is an account. PasswordHash is bcrypt. NOTE: GoogleSub is reserved for a later
// OIDC login path (nullable, unique).
type User struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;not null"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash []byte
	GoogleSub    *string `gorm:"uniqueIndex"`
	CreatedAt    time.Time
}
