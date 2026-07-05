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
	projects *store.ProjectStore
	users    *store.UserStore
	fs       afero.Fs
	dataDir  string
	host     okayeg.Host
}

func (s *ProjectService) Create(ctx context.Context, ownerID uint, name string) (store.Project, error) {
	p := store.Project{Name: name}
	if err := s.projects.Create(ctx, &p, ownerID); err != nil {
		return store.Project{}, err
	}

	// if the key can't be created, delete the project to avoid a broken state.
	// EnsureKey also makes the share dir a real eg repo (.eg/key), which is
	// what `eg serve` needs to self-initialize on first connect.
	if _, err := okayeg.EnsureKey(s.fs, s.shareDir(p.ID)); err != nil {
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

// Endpoint returns the host's EndpointId for the project: the public identity a
// browser peer dials with `connect(endpointId)`. It is safe to show any member.
// Ensures the eg daemon is up so the address is reachable.
func (s *ProjectService) Endpoint(ctx context.Context, userID uint, id string) (string, error) {
	key, err := s.loadKey(ctx, userID, id)
	if err != nil {
		return "", err
	}

	s.host.EnsureHost(id)
	return key.EndpointID(), nil
}

// RegisterPeer authorizes a browser's ed identity (its EndpointId) to sync the
// project, recording it in .eg/trust (the enforcement eg re-reads on every
// connect) and in the project_peer table (the queryable record). Requires
// membership; tags the registering user on the trust line. Idempotent.
func (s *ProjectService) RegisterPeer(ctx context.Context, userID uint, projectID, endpointID string) error {
	if err := s.access(ctx, userID, projectID); err != nil {
		return err
	}
	if !okayeg.ValidEndpointID(endpointID) {
		return ErrBadRequest
	}

	if err := okayeg.SetTrust(s.fs, s.shareDir(projectID), endpointID, userID); err != nil {
		return err
	}
	if err := s.projects.AddPeer(ctx, projectID, userID, endpointID); err != nil {
		return mapErr(err)
	}

	s.host.EnsureHost(projectID)
	return nil
}

// UnregisterPeer revokes a browser's access: removes the trust line (takes
// effect on the peer's next connect) and drops the project_peer row. The
// project owner may remove any peer; a member may only remove their own.
func (s *ProjectService) UnregisterPeer(ctx context.Context, userID uint, projectID, endpointID string) error {
	peer, err := s.projects.GetPeer(ctx, projectID, endpointID)
	if err != nil {
		return mapErr(err)
	}

	// membership is the floor; owner or the registering user may proceed
	isOwner, err := s.isOwner(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if !isOwner && peer.UserID != userID {
		return ErrUnauthorized
	}

	if err := okayeg.RemoveTrust(s.fs, s.shareDir(projectID), endpointID); err != nil {
		return err
	}
	if err := s.projects.RemovePeer(ctx, projectID, endpointID); err != nil {
		return mapErr(err)
	}

	return nil
}

// ListPeers returns the endpoints authorized on a project, with their
// registering user. Any member may view them.
func (s *ProjectService) ListPeers(ctx context.Context, userID uint, projectID string) ([]store.ProjectPeer, error) {
	if err := s.access(ctx, userID, projectID); err != nil {
		return nil, err
	}
	return s.projects.ListPeers(ctx, projectID)
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

// RemoveMember removes a user from the project and revokes every browser they
// registered, so a removed collaborator can't reconnect. Owner only.
func (s *ProjectService) RemoveMember(ctx context.Context, actorID uint, id string, userID uint) error {
	if err := s.requireOwner(ctx, actorID, id); err != nil {
		return err
	}

	peers, err := s.projects.PeersForUser(ctx, id, userID)
	if err != nil {
		return mapErr(err)
	}

	// revoke trust before the member row, so a failure leaves the user still a
	// member rather than a non-member who can still connect
	for _, p := range peers {
		if err := okayeg.RemoveTrust(s.fs, s.shareDir(id), p.EndpointID); err != nil {
			return err
		}
	}

	if err := s.projects.RemoveMember(ctx, id, userID); err != nil {
		return mapErr(err)
	}
	if len(peers) > 0 {
		// best-effort: the trust lines are already gone; the DB rows are clerical
		_ = s.projects.RemovePeersByUser(ctx, id, userID)
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

// isOwner reports membership+ownership; missing membership is an error.
func (s *ProjectService) isOwner(ctx context.Context, userID uint, id string) (bool, error) {
	m, err := s.projects.GetMember(ctx, id, userID)
	if err != nil {
		return false, mapErr(err)
	}
	return m.Role == store.RoleOwner, nil
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
