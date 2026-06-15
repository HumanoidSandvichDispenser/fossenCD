package store

import (
	"time"

	"gorm.io/gorm"
)

// Project maps an owner to an on-disk share dir. ID is a random hex string so
// share dirs aren't enumerable; the teamtype key lives at <dataDir>/<ID>/.teamtype/key.
type Project struct {
	ID        string `gorm:"primaryKey"`
	OwnerID   uint   `gorm:"index;not null"`
	Name      string `gorm:"not null"`
	CreatedAt time.Time
}

// BeforeCreate assigns a random ID if one wasn't set.
func (p *Project) BeforeCreate(*gorm.DB) error {
	if p.ID == "" {
		p.ID = NewID()
	}
	return nil
}
