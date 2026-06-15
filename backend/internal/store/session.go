package store

import "time"

// Session is a server-side login session.
type Session struct {
	Token     string `gorm:"primaryKey"`
	UserID    uint   `gorm:"index;not null"`
	User      User
	ExpiresAt time.Time
	CreatedAt time.Time
}
