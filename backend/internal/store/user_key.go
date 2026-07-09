package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserKey is an okayeg peer public key (an EndpointId) bound to a fossenCD
// account. Every peer, browser or native, generates its own ed keypair locally
// and uploads only the public half here, so the server never holds a private
// key. A key is account-scoped, not project-scoped: it authorizes its owner on
// every project they are a member of, and the authz command resolves an
// incoming endpoint to its owning account through this table. Multiple rows per
// user are distinct devices, each independently revocable in account settings.
type UserKey struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"not null;index"`
	EndpointID string `gorm:"not null;uniqueIndex"`
	Label      string
	CreatedAt  time.Time

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// AddKey binds an endpoint to a user. Idempotent on endpoint_id: a repeat is a
// no-op and does not reassign the owner or relabel the key. An endpoint belongs
// to exactly one account, so a second user cannot claim one already registered.
func (s *UserStore) AddKey(ctx context.Context, userID uint, endpointID, label string) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "endpoint_id"}},
			DoNothing: true,
		}).
		Create(&UserKey{UserID: userID, EndpointID: endpointID, Label: label}).
		Error
}

// RemoveKey drops the user's key for endpointID. Returns ErrNotFound if the
// user has no such key. Scoped to userID so a user can only delete their own.
func (s *UserStore) RemoveKey(ctx context.Context, userID uint, endpointID string) error {
	res := s.db.WithContext(ctx).
		Delete(&UserKey{}, "user_id = ? AND endpoint_id = ?", userID, endpointID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListKeys returns a user's keys in creation order.
func (s *UserStore) ListKeys(ctx context.Context, userID uint) ([]UserKey, error) {
	var keys []UserKey
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at asc").
		Find(&keys).
		Error
	return keys, err
}

// GetKeyByEndpoint resolves an endpoint to its owning key, or ErrNotFound. This
// is the authz lookup: endpoint -> account, after which the caller checks
// project membership.
func (s *UserStore) GetKeyByEndpoint(ctx context.Context, endpointID string) (*UserKey, error) {
	var k UserKey
	err := s.db.WithContext(ctx).First(&k, "endpoint_id = ?", endpointID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}
