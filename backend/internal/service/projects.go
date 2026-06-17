package service

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/teamtype"
)

type ProjectService struct {
	projects *store.ProjectStore
	users    *store.UserStore
	fs       afero.Fs
	dataDir  string
	relay    string
	mintCtx  context.Context
	host     teamtype.Host
}

func (s *ProjectService) Create(ctx context.Context, ownerID uint, name string) (store.Project, error) {
	p := store.Project{Name: name}
	if err := s.projects.Create(ctx, &p, ownerID); err != nil {
		return store.Project{}, err
	}

	// if the key can't be created, delete the project to avoid a broken state
	if _, err := teamtype.EnsureKey(s.fs, s.shareDir(p.ID)); err != nil {
		_ = s.projects.Delete(ctx, p.ID)
		return store.Project{}, err
	}
	return p, nil
}

func (s *ProjectService) List(ctx context.Context, userID uint) ([]store.Project, error) {
	return s.projects.ListUserProjects(ctx, userID)
}

func (s *ProjectService) Get(ctx context.Context, userID uint, id string) (store.Project, error) {
	if err := s.access(ctx, userID, id); err != nil {
		return store.Project{}, err
	}
	p, err := s.projects.Get(ctx, id)
	if err != nil {
		return store.Project{}, mapErr(err)
	}
	return *p, nil
}

func (s *ProjectService) Delete(ctx context.Context, userID uint, id string) error {
	if err := s.requireOwner(ctx, userID, id); err != nil {
		return err
	}
	if err := s.projects.Delete(ctx, id); err != nil {
		return err
	}
	s.host.Stop(id)
	return s.fs.RemoveAll(s.shareDir(id))
}

func (s *ProjectService) Address(ctx context.Context, userID uint, id string) (string, error) {
	key, err := s.loadKey(ctx, userID, id)
	if err != nil {
		return "", err
	}
	s.host.EnsureHost(id)
	return key.SecretAddress(), nil
}

// MintJoinCode returns the code and address immediately; delivery to a peer runs
// on the mint context for the server's lifetime.
func (s *ProjectService) MintJoinCode(ctx context.Context, userID uint, id string) (code, address string, err error) {
	key, err := s.loadKey(ctx, userID, id)
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

// AddMember adds login (username or email) to the project as a member. Owner only.
func (s *ProjectService) AddMember(ctx context.Context, actorID uint, id, login string) error {
	if err := s.requireOwner(ctx, actorID, id); err != nil {
		return err
	}
	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		return mapErr(err)
	}
	return s.projects.AddMember(ctx, id, u.ID, store.RoleMember)
}

// RemoveMember removes a user from the project. Owner only.
func (s *ProjectService) RemoveMember(ctx context.Context, actorID uint, id string, userID uint) error {
	if err := s.requireOwner(ctx, actorID, id); err != nil {
		return err
	}
	return mapErr(s.projects.RemoveMember(ctx, id, userID))
}

func (s *ProjectService) ListMembers(ctx context.Context, userID uint, id string) ([]store.ProjectMember, error) {
	if err := s.access(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.projects.GetMembers(ctx, id)
}

// access returns nil if the user is a member of the project.
func (s *ProjectService) access(ctx context.Context, userID uint, id string) error {
	if _, err := s.projects.GetMember(ctx, id, userID); err != nil {
		return mapErr(err)
	}
	return nil
}

// requireOwner returns ErrNotFound if the user isn't a member, or ErrUnauthorized
// if they're a member without the owner role.
func (s *ProjectService) requireOwner(ctx context.Context, userID uint, id string) error {
	m, err := s.projects.GetMember(ctx, id, userID)
	if err != nil {
		return mapErr(err)
	}
	if m.Role != store.RoleOwner {
		return ErrUnauthorized
	}
	return nil
}

func (s *ProjectService) loadKey(ctx context.Context, userID uint, id string) (teamtype.Key, error) {
	if err := s.access(ctx, userID, id); err != nil {
		return teamtype.Key{}, err
	}
	return teamtype.LoadKey(s.fs, s.shareDir(id))
}

func (s *ProjectService) shareDir(id string) string {
	return filepath.Join(s.dataDir, id)
}

// mapErr translates store sentinels to service sentinels.
func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, store.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, store.ErrLastOwner):
		return ErrConflict
	default:
		return err
	}
}
