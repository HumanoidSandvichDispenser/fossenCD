package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectStore struct{
	db *gorm.DB
}

func NewProjectStore(db *gorm.DB) *ProjectStore {
	return &ProjectStore{db: db}
}

// Create creates a new project and adds the owner as a member.
func (s *ProjectStore) Create(ctx context.Context, p *Project, ownerID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return err
		}

		return tx.Create(&ProjectMember{
			ProjectID: p.ID,
			UserID:    ownerID,
			Role:      RoleOwner,
		}).Error
	})
}

// Get returns the project with the given ID, or an error if it doesn't exist.
func (s *ProjectStore) Get(ctx context.Context, id string) (*Project, error) {
	var p Project

	err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// ListUserProjects returns all projects that the given user is a member of,
// ordered by creation time.
func (s *ProjectStore) ListUserProjects(ctx context.Context, userID uint) ([]Project, error) {
	var projects []Project

	err := s.db.WithContext(ctx).
		Joins("JOIN project_members ON project_members.project_id = projects.id").
		Where("project_members.user_id = ?", userID).
		Order("projects.created_at desc").
		Find(&projects).
		Error

	return projects, err
}

// Delete deletes the project with the given ID. The FK cascade removes its
// members, but the caller is responsible for stopping any active host and deleting
// the share dir.
func (s *ProjectStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&Project{}, "id = ?", id).Error
}

func (s *ProjectStore) GetMembers(ctx context.Context, projectID string) ([]ProjectMember, error) {
	var members []ProjectMember
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("project_id = ?", projectID).
		Find(&members).
		Error
	return members, err
}

// GetMember returns the user's membership row, or ErrNotFound if they aren't one.
// Used for access checks (exists?) and authorization (role).
func (s *ProjectStore) GetMember(ctx context.Context, projectID string, userID uint) (*ProjectMember, error) {
	var m ProjectMember
	err := s.db.WithContext(ctx).
		First(&m, "project_id = ? AND user_id = ?", projectID, userID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// AddMember adds the given user to the project with the specified role. Idempotent.
func (s *ProjectStore) AddMember(ctx context.Context, projectID string, userID uint, role string) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "project_id"},
				{Name: "user_id"},
			},
			// for adding members, idempotency should NOT change their roles, so do nothing
			DoNothing: true,
		}).
		Create(&ProjectMember{
			ProjectID: projectID,
			UserID:    userID,
			Role:      role,
		}).
		Error
}

// RemoveMember removes the given user from the project. Returns ErrNotFound if
// they aren't a member, or ErrLastOwner if removing them would leave the project
// without an owner (transfer ownership first).
func (s *ProjectStore) RemoveMember(ctx context.Context, projectID string, userID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m ProjectMember
		err := tx.First(&m, "project_id = ? AND user_id = ?", projectID, userID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		if m.Role == RoleOwner {
			var owners int64
			if err := tx.Model(&ProjectMember{}).
				Where("project_id = ? AND role = ?", projectID, RoleOwner).
				Count(&owners).Error; err != nil {
				return err
			}
			if owners <= 1 {
				return ErrLastOwner
			}
		}

		return tx.Delete(&m).Error
	})
}
