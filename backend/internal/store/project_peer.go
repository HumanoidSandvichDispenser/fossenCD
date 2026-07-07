package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProjectPeer is a registered browser endpoint authorized to sync a project.
// Each browser holds its own ed identity (an okayeg EndpointId); the row
// records which user registered it. This table is what the authz command
// consults when the peer connects, so it is the single source of truth for
// who may sync.
type ProjectPeer struct {
	ID         uint   `gorm:"primaryKey"`
	ProjectID  string `gorm:"not null;index;uniqueIndex:idx_project_endpoint"`
	UserID     uint   `gorm:"not null;index"`
	EndpointID string `gorm:"not null;uniqueIndex:idx_project_endpoint"`
	CreatedAt  time.Time

	Project Project `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	User    User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// AddPeer registers an endpoint for a user on a project. Idempotent on
// (project, endpoint): a repeat does not reassign the registering user.
func (s *ProjectStore) AddPeer(ctx context.Context, projectID string, userID uint, endpointID string) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "project_id"},
				{Name: "endpoint_id"},
			},
			DoNothing: true,
		}).
		Create(&ProjectPeer{
			ProjectID:  projectID,
			UserID:     userID,
			EndpointID: endpointID,
		}).
		Error
}

// RemovePeer drops the (project, endpoint) row. Returns ErrNotFound if absent.
func (s *ProjectStore) RemovePeer(ctx context.Context, projectID, endpointID string) error {
	res := s.db.WithContext(ctx).
		Delete(&ProjectPeer{}, "project_id = ? AND endpoint_id = ?", projectID, endpointID)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// PeersForUser returns every endpoint a user has registered on a project.
// Used to revoke their live sessions when removing their membership.
func (s *ProjectStore) PeersForUser(ctx context.Context, projectID string, userID uint) ([]ProjectPeer, error) {
	var peers []ProjectPeer

	err := s.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Find(&peers).
		Error

	return peers, err
}

// RemovePeersByUser deletes every endpoint a user registered on a project.
// Returns ErrNotFound if they had none.
func (s *ProjectStore) RemovePeersByUser(ctx context.Context, projectID string, userID uint) error {
	res := s.db.WithContext(ctx).
		Delete(&ProjectPeer{}, "project_id = ? AND user_id = ?", projectID, userID)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// ListPeers returns the endpoints registered on a project, with their
// registering user preloaded, in creation order.
func (s *ProjectStore) ListPeers(ctx context.Context, projectID string) ([]ProjectPeer, error) {
	var peers []ProjectPeer

	err := s.db.WithContext(ctx).
		Preload("User").
		Where("project_id = ?", projectID).
		Order("created_at asc").
		Find(&peers).
		Error

	return peers, err
}

// GetPeer returns the row for (project, endpoint), or ErrNotFound.
func (s *ProjectStore) GetPeer(ctx context.Context, projectID, endpointID string) (*ProjectPeer, error) {
	var p ProjectPeer

	err := s.db.WithContext(ctx).
		First(&p, "project_id = ? AND endpoint_id = ?", projectID, endpointID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}
