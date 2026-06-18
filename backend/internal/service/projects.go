package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/afero"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/teamtype"
)

// joinCodeTTL is how long an unclaimed code lives; a repeat mint resets it.
const joinCodeTTL = 2 * time.Minute

type ProjectService struct {
	projects *store.ProjectStore
	users    *store.UserStore
	fs       afero.Fs
	dataDir  string
	relay    string
	mintCtx  context.Context
	host     teamtype.Host

	// live join codes keyed by user+project, guarded by the mutex
	codes struct {
		sync.Mutex
		pending map[string]*pendingJoin
	}
}

// pendingJoin is a live, unclaimed code. cancel closes the wormhole mailbox
// (releasing the nameplate); timer fires cleanup on expiry.
type pendingJoin struct {
	code    string
	address string
	cancel  context.CancelFunc
	timer   *time.Timer
	expires time.Time
}

// extend resets the code's TTL and returns its new expiry. Call under the lock.
func (pj *pendingJoin) extend() time.Time {
	pj.timer.Reset(joinCodeTTL)
	pj.expires = time.Now().Add(joinCodeTTL)
	return pj.expires
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

// MintJoinCode returns a join code, the secret address it delivers, and when the
// code expires. The code lives for joinCodeTTL; a repeat call from the same user
// reuses and extends it. Delivery to a peer runs on the mint context.
func (s *ProjectService) MintJoinCode(
	ctx context.Context,
	userID uint,
	id string,
) (code, address string, expiresAt time.Time, err error) {
	key, err := s.loadKey(ctx, userID, id)
	if err != nil {
		return "", "", time.Time{}, err
	}
	rkey := fmt.Sprintf("%d:%s", userID, id)

	// reuse an outstanding code, extending its TTL
	s.codes.Lock()
	if pj, ok := s.codes.pending[rkey]; ok {
		exp := pj.extend()
		c, a := pj.code, pj.address
		s.codes.Unlock()
		return c, a, exp, nil
	}
	s.codes.Unlock()

	mintCtx, cancel := context.WithCancel(s.mintCtx)
	code, done, err := key.MintJoinCode(mintCtx, s.relay)
	if err != nil {
		cancel()
		return "", "", time.Time{}, err
	}
	address = key.SecretAddress()

	pj := &pendingJoin{
		code:    code,
		address: address,
		cancel:  cancel,
		expires: time.Now().Add(joinCodeTTL),
	}
	pj.timer = time.AfterFunc(joinCodeTTL, func() {
		s.dropCode(rkey, pj)
	})

	// a concurrent mint may have won; if so, discard ours and return theirs
	s.codes.Lock()
	if existing, ok := s.codes.pending[rkey]; ok {
		exp := existing.extend()
		s.codes.Unlock()
		pj.timer.Stop()
		cancel()
		return existing.code, existing.address, exp, nil
	}
	if s.codes.pending == nil {
		s.codes.pending = map[string]*pendingJoin{}
	}
	s.codes.pending[rkey] = pj
	s.codes.Unlock()

	// drop the code once claimed (or cancelled by expiry)
	go func() {
		<-done
		s.dropCode(rkey, pj)
	}()

	return code, address, pj.expires, nil
}

// dropCode removes pj if it's still the active code for rkey and tears it down.
func (s *ProjectService) dropCode(rkey string, pj *pendingJoin) {
	s.codes.Lock()
	if s.codes.pending[rkey] == pj {
		delete(s.codes.pending, rkey)
	}
	s.codes.Unlock()
	pj.timer.Stop()
	pj.cancel()
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
