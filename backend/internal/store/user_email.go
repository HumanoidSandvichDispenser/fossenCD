package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// UserEmail is one address in an account's email set. Globally unique: an
// address belongs to exactly one account.
type UserEmail struct {
	ID uint `gorm:"primaryKey"`

	UserID uint `gorm:"not null;index"`

	Email string `gorm:"not null;uniqueIndex"`

	// VerifiedAt is when ownership of the address was confirmed, nil while
	// unconfirmed.
	VerifiedAt *time.Time

	CreatedAt time.Time

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// ListEmails returns a user's emails in creation order.
func (s *UserStore) ListEmails(ctx context.Context, userID uint) ([]UserEmail, error) {
	var emails []UserEmail
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at asc").
		Find(&emails).
		Error
	return emails, err
}

// AddEmail adds an address to a user's email set, verified when verifiedAt is
// non-nil. Returns ErrConflict when the address is already registered, to any
// account.
func (s *UserStore) AddEmail(ctx context.Context, userID uint, email string, verifiedAt *time.Time) error {
	err := s.db.WithContext(ctx).
		Create(&UserEmail{UserID: userID, Email: email, VerifiedAt: verifiedAt}).
		Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrConflict
	}
	return err
}

// RemoveEmail drops an address from a user's email set. Returns ErrNotFound if
// the user has no such address. Scoped to userID so a user can only remove
// their own.
func (s *UserStore) RemoveEmail(ctx context.Context, userID uint, email string) error {
	res := s.db.WithContext(ctx).
		Delete(&UserEmail{}, "user_id = ? AND email = ?", userID, email)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// VerifiedOwnerOfClaim resolves a peer's claimed email against the accounts
// owning its endpoint: the user that both registered endpointID and has email
// verified, or ErrNotFound.
func (s *UserStore) VerifiedOwnerOfClaim(ctx context.Context, endpointID, email string) (*User, error) {
	var u User
	err := s.db.WithContext(ctx).
		Joins("JOIN user_emails ON user_emails.user_id = users.id").
		Joins("JOIN user_keys ON user_keys.user_id = users.id").
		Where("user_keys.endpoint_id = ?", endpointID).
		Where("user_emails.email = ? AND user_emails.verified_at IS NOT NULL", email).
		First(&u).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
