package store

import (
	"context"
	"time"

	"gorm.io/gorm/clause"
)

// UserKey is an okayeg peer public key (an EndpointId) bound to a fossenCD
// account. Every peer, browser or native, generates its own ed keypair locally
// and uploads only the public half here, so the server never holds a private
// key. A key is account-scoped, not project-scoped: it authorizes its owner on
// every project they are a member of, and the authz command resolves an
// incoming endpoint to its owning accounts through this table. Multiple rows
// per user are distinct devices, each independently revocable in account
// settings. The (user_id, endpoint_id) pair is unique, not endpoint_id alone:
// the same endpoint may be registered by several accounts, since holding its
// private key is the proof of control.
type UserKey struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"not null;uniqueIndex:idx_user_keys_user_endpoint,priority:1"`
	EndpointID string `gorm:"not null;uniqueIndex:idx_user_keys_user_endpoint,priority:2"`
	Label      string
	CreatedAt  time.Time

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// AddKey binds an endpoint to a user. Idempotent for a key they already own:
// re-adding is a no-op and keeps the existing label. The same endpoint may be
// registered by other accounts too, so this never conflicts across users.
func (s *UserStore) AddKey(ctx context.Context, userID uint, endpointID, label string) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "endpoint_id"}},
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

// RenameKey changes the label of a user's key. Returns ErrNotFound if the user
// has no such key. Scoped to userID so a user can only rename their own.
func (s *UserStore) RenameKey(ctx context.Context, userID uint, endpointID, newLabel string) error {
	res := s.db.WithContext(ctx).
		Model(&UserKey{}).
		Where("user_id = ? AND endpoint_id = ?", userID, endpointID).
		Update("label", newLabel)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// OwnersOfEndpoint returns the IDs of every account that has registered
// endpointID, empty if none. An endpoint can belong to more than one account,
// so authz checks membership across all of them.
func (s *UserStore) OwnersOfEndpoint(ctx context.Context, endpointID string) ([]uint, error) {
	var ids []uint
	err := s.db.WithContext(ctx).
		Model(&UserKey{}).
		Where("endpoint_id = ?", endpointID).
		Pluck("user_id", &ids).
		Error
	return ids, err
}
