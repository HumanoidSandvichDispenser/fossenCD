// Package store holds the gorm models and database handle. Only user accounts,
// project metadata, and account peer public keys live here; okayeg private keys
// stay on disk in each project's .eg/ (see internal/okayeg), and user peers hold
// their own private keys locally. The package is database-agnostic.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open opens the database via the given dialector and migrates the schema.
func Open(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&User{}, &Project{}, &ProjectMember{}, &UserKey{}, &Session{}, &UserEmail{}); err != nil {
		return nil, err
	}
	if err := seedPrimaryEmails(db); err != nil {
		return nil, err
	}
	// endpoint_id used to be globally unique (one endpoint -> one account). It is
	// now unique only per user, so an endpoint may be registered by several
	// accounts. AutoMigrate adds the new composite index but leaves the old
	// single-column one in place, so drop it on databases that still carry it.
	if db.Migrator().HasIndex(&UserKey{}, "idx_user_keys_endpoint_id") {
		if err := db.Migrator().DropIndex(&UserKey{}, "idx_user_keys_endpoint_id"); err != nil {
			return nil, err
		}
	}
	return db, nil
}

// seedPrimaryEmails backfills a verified UserEmail row from each account's
// registration email. Accounts predating the email set kept their address only
// on users.email; new registrations write both.
func seedPrimaryEmails(db *gorm.DB) error {
	var users []User
	err := db.
		Where("NOT EXISTS (SELECT 1 FROM user_emails WHERE user_emails.user_id = users.id AND user_emails.email = users.email)").
		Find(&users).
		Error
	if err != nil {
		return err
	}

	now := time.Now()
	for _, u := range users {
		row := UserEmail{UserID: u.ID, Email: u.Email, VerifiedAt: &now}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// NewID returns a 16-byte random hex string, used for primary keys that
// shouldn't be enumerable.
func NewID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
