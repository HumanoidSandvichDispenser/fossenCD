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

// AddKey binds an endpoint to the caller's account. Idempotent. Rejects a
// malformed endpoint id before it reaches the database.
func (s *KeyService) AddKey(ctx context.Context, userID uint, endpointID, label string) error {
	if !okayeg.ValidEndpointID(endpointID) {
		return ErrBadRequest
	}
	return s.users.AddKey(ctx, userID, endpointID, label)
}

// ListKeys returns the caller's registered keys.
func (s *KeyService) ListKeys(ctx context.Context, userID uint) ([]store.UserKey, error) {
	return s.users.ListKeys(ctx, userID)
}

// RemoveKey deletes one of the caller's keys, then closes its live sessions on
// every project the caller is a member of. The row (policy) drops before the
// live sessions, so a failure leaves the key gone rather than still authorized.
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
