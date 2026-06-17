package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

// GetByLogin resolves a user by username or email, for inviting members.
func (s *UserStore) GetByLogin(ctx context.Context, login string) (*User, error) {
	var u User
	err := s.db.WithContext(ctx).
		First(&u, "username = ? OR email = ?", login, login).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
