package service

import (
	"context"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/okayeg"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

// KeyService manages account-scoped okayeg peer keys: the public keys a user
// uploads so their browser and native peers may sync every project they belong
// to. The server never sees the matching private keys. Revocation fans out
// across the user's projects, closing live sessions so a device is killed
// everywhere at once.
type KeyService struct {
	users    *store.UserStore
	projects *store.ProjectStore
	host     okayeg.Host
}

// AddKey binds an endpoint to a user's account. Idempotent for a key they
// already own; the endpoint may belong to other accounts too. Returns
// ErrBadRequest for a malformed endpoint id.
func (s *KeyService) AddKey(ctx context.Context, userID uint, endpointID, label string) error {
	if !okayeg.ValidEndpointID(endpointID) {
		return ErrBadRequest
	}
	return s.users.AddKey(ctx, userID, endpointID, label)
}

// ListKeys returns a user's registered keys.
func (s *KeyService) ListKeys(ctx context.Context, userID uint) ([]store.UserKey, error) {
	return s.users.ListKeys(ctx, userID)
}

// RenameKey relabels one of a user's keys. The endpoint id is immutable.
func (s *KeyService) RenameKey(ctx context.Context, userID uint, endpointID, newLabel string) error {
	return s.users.RenameKey(ctx, userID, endpointID, newLabel)
}

// RemoveKey deletes a key and closes its live sessions on every project it
// authorized. The row (policy) drops before the sessions, so a failure leaves
// the key gone rather than still authorized.
func (s *KeyService) RemoveKey(ctx context.Context, userID uint, endpointID string) error {
	if err := s.users.RemoveKey(ctx, userID, endpointID); err != nil {
		return mapErr(err)
	}

	projects, err := s.projects.ListUserProjects(ctx, userID)
	if err != nil {
		return err
	}
	for _, p := range projects {
		if err := s.host.Revoke(p.ID, endpointID); err != nil {
			return err
		}
	}
	return nil
}
