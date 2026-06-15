// Package store holds the gorm models and database handle. Only user accounts
// and project metadata live here; teamtype keys stay on disk (see internal/teamtype).
// The package is database-agnostic: callers pass the gorm.Dialector.
package store

import (
	"crypto/rand"
	"encoding/hex"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open opens the database via the given dialector and migrates the schema.
func Open(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&User{}, &Project{}, &Session{}); err != nil {
		return nil, err
	}
	return db, nil
}

// NewID returns a 16-byte random hex string, used for primary keys that
// shouldn't be enumerable.
func NewID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
