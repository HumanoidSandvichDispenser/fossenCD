package service

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/spf13/afero"
	"gorm.io/gorm"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/teamtype"
)

type ProjectService struct {
	db      *gorm.DB
	fs      afero.Fs
	dataDir string
	relay   string
	mintCtx context.Context
}

func (s *ProjectService) Create(ctx context.Context, ownerID uint, name string) (store.Project, error) {
	p := store.Project{OwnerID: ownerID, Name: name}
	if err := s.db.WithContext(ctx).Create(&p).Error; err != nil {
		return store.Project{}, err
	}
	if _, err := teamtype.EnsureKey(s.fs, s.shareDir(p.ID)); err != nil {
		s.db.WithContext(ctx).Delete(&p)
		return store.Project{}, err
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context, ownerID uint) ([]store.Project, error) {
	var ps []store.Project
	err := s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("created_at desc").Find(&ps).Error
	return ps, err
}

func (s *ProjectService) Get(ctx context.Context, ownerID uint, id string) (store.Project, error) {
	var p store.Project
	err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return store.Project{}, ErrNotFound
	}
	return p, err
}

func (s *ProjectService) Delete(ctx context.Context, ownerID uint, id string) error {
	p, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&p).Error; err != nil {
		return err
	}
	return s.fs.RemoveAll(s.shareDir(p.ID))
}

func (s *ProjectService) Address(ctx context.Context, ownerID uint, id string) (string, error) {
	key, err := s.loadKey(ctx, ownerID, id)
	if err != nil {
		return "", err
	}
	return key.SecretAddress(), nil
}

// MintJoinCode returns the code and address immediately; delivery to a peer runs
// on the mint context for the server's lifetime.
func (s *ProjectService) MintJoinCode(ctx context.Context, ownerID uint, id string) (code, address string, err error) {
	key, err := s.loadKey(ctx, ownerID, id)
	if err != nil {
		return "", "", err
	}
	code, done, err := key.MintJoinCode(s.mintCtx, s.relay)
	if err != nil {
		return "", "", err
	}
	go func() { <-done }()
	return code, key.SecretAddress(), nil
}

func (s *ProjectService) loadKey(ctx context.Context, ownerID uint, id string) (teamtype.Key, error) {
	if _, err := s.Get(ctx, ownerID, id); err != nil {
		return teamtype.Key{}, err
	}
	return teamtype.LoadKey(s.fs, s.shareDir(id))
}

func (s *ProjectService) shareDir(id string) string {
	return filepath.Join(s.dataDir, id)
}
