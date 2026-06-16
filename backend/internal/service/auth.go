package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/auth"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

type AuthService struct {
	db  *gorm.DB
	ttl time.Duration
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (store.User, store.Session, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	u := store.User{Username: username, Email: email, PasswordHash: hash}
	if err := s.db.WithContext(ctx).Create(&u).Error; err != nil {
		return store.User{}, store.Session{}, ErrConflict
	}
	return s.newSession(ctx, u)
}

func (s *AuthService) Login(ctx context.Context, username, password string) (store.User, store.Session, error) {
	var u store.User
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || !auth.CheckPassword(u.PasswordHash, password) {
		return store.User{}, store.Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	return s.newSession(ctx, u)
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.db.WithContext(ctx).Where("token = ?", token).Delete(&store.Session{}).Error
}

// SetPassword replaces the password of the user with the given username and
// revokes all of their existing sessions, forcing a re-login everywhere.
func (s *AuthService) SetPassword(ctx context.Context, username, password string) (store.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return store.User{}, err
	}
	var u store.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return store.User{}, ErrNotFound
		}
		return store.User{}, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&u).Update("password_hash", hash).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", u.ID).Delete(&store.Session{}).Error
	})
	if err != nil {
		return store.User{}, err
	}
	return u, nil
}

func (s *AuthService) ResolveSession(ctx context.Context, token string) (store.User, error) {
	var sess store.Session
	err := s.db.WithContext(ctx).
		Joins("User").
		Where("sessions.token = ? AND sessions.expires_at > ?", token, time.Now()).
		First(&sess).Error
	if err != nil {
		return store.User{}, ErrUnauthorized
	}
	return sess.User, nil
}

func (s *AuthService) User(ctx context.Context, id uint) (store.User, error) {
	var u store.User
	if err := s.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return store.User{}, ErrNotFound
	}
	return u, nil
}

func (s *AuthService) newSession(ctx context.Context, u store.User) (store.User, store.Session, error) {
	sess := store.Session{
		Token:     auth.NewSessionToken(),
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(s.ttl),
	}
	if err := s.db.WithContext(ctx).Create(&sess).Error; err != nil {
		return store.User{}, store.Session{}, err
	}
	return u, sess, nil
}
