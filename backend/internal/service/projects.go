package service

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/okayeg"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

type ProjectService struct {
	projects    *store.ProjectStore
	users       *store.UserStore
	fs          afero.Fs
	dataDir     string
	host        okayeg.Host
	authzScript string
	authzSocket string
}

func (s *ProjectService) Create(ctx context.Context, ownerID uint, name string) (store.Project, error) {
	p := store.Project{Name: name}
	if err := s.projects.Create(ctx, &p, ownerID); err != nil {
		return store.Project{}, err
	}

	// if the repo can't be seeded, delete the project to avoid a broken state.
	// EnsureKey makes the share dir a real eg repo (.eg/key), which is what
	// `eg serve` needs to self-initialize on first connect; the config points
	// eg's connection gate at our authz command, so the DB decides who syncs.
	if _, err := okayeg.EnsureKey(s.fs, s.shareDir(p.ID)); err != nil {
		_ = s.projects.Delete(ctx, p.ID)
		return store.Project{}, err
	}
	if err := s.writeAuthzConfig(p.ID); err != nil {
		_ = s.projects.Delete(ctx, p.ID)
		return store.Project{}, err
	}
	return p, nil
}

// writeAuthzConfig writes the project's .eg/config.toml naming the authz
// command eg runs per connection: the script, the backend's authz socket, and
// the project id. No-op when no script is configured (eg then uses .eg/trust).
func (s *ProjectService) writeAuthzConfig(projectID string) error {
	if s.authzScript == "" {
		return nil
	}
	argv := []string{s.authzScript, s.authzSocket, projectID}
	return okayeg.WriteAuthzConfig(s.fs, s.shareDir(projectID), argv)
}

// AuthorizePeer is the decision behind the authz socket: whether endpointID may
// sync projectID. It holds when the endpoint is a key registered to some account
// and that account is a member of the project. eg asks once per incoming
// connection. A key authorizes its owner on every project they belong to, so the
// decision is endpoint -> owning account -> membership.
func (s *ProjectService) AuthorizePeer(ctx context.Context, projectID, endpointID string) (bool, error) {
	if !okayeg.ValidEndpointID(endpointID) {
		return false, nil
	}
	key, err := s.users.GetKeyByEndpoint(ctx, endpointID)
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := s.projects.GetMember(ctx, projectID, key.UserID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
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

// Endpoint returns the host's EndpointId for the project: the public identity a
// browser peer dials with `connect(endpointId)`. It is safe to show any member.
// Ensures the eg daemon is up so the address is reachable.
func (s *ProjectService) Endpoint(ctx context.Context, userID uint, id string) (string, error) {
	key, err := s.loadKey(ctx, userID, id)
	if err != nil {
		return "", err
	}

	// re-written on every start path so projects created before the authz
	// hook existed (or with a stale script/socket path) heal on first use
	if err := s.writeAuthzConfig(id); err != nil {
		return "", err
	}
	s.host.EnsureHost(id)
	return key.EndpointID(), nil
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

// RemoveMember removes a user from the project and closes any live sessions
// their keys hold on it, so a removed collaborator can't keep syncing. The keys
// are account-owned and stay registered (they remain valid on the user's other
// projects); dropping the membership is what makes the authz gate refuse their
// next connection here. Owner only.
func (s *ProjectService) RemoveMember(ctx context.Context, actorID uint, id string, userID uint) error {
	if err := s.requireOwner(ctx, actorID, id); err != nil {
		return err
	}

	keys, err := s.users.ListKeys(ctx, userID)
	if err != nil {
		return err
	}

	// drop the membership before closing sessions, so the gate denies reconnects
	// by the time we revoke; a failure here leaves them a member rather than a
	// non-member whose peers can still connect
	if err := s.projects.RemoveMember(ctx, id, userID); err != nil {
		return mapErr(err)
	}

	for _, k := range keys {
		if err := s.host.Revoke(id, k.EndpointID); err != nil {
			return err
		}
	}

	return nil
}

func (s *ProjectService) ListMembers(ctx context.Context, userID uint, id string) ([]store.ProjectMember, error) {
	if err := s.access(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.projects.GetMembers(ctx, id)
}

// Logs returns recent daemon output for the project and whether a host is
// running. Any member may view it.
func (s *ProjectService) Logs(ctx context.Context, userID uint, id string) (output string, running bool, err error) {
	if err := s.access(ctx, userID, id); err != nil {
		return "", false, err
	}
	output, running = s.host.Logs(id)
	return output, running, nil
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

func (s *ProjectService) loadKey(ctx context.Context, userID uint, id string) (okayeg.Key, error) {
	if err := s.access(ctx, userID, id); err != nil {
		return okayeg.Key{}, err
	}

	return okayeg.LoadKey(s.fs, s.shareDir(id))
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
