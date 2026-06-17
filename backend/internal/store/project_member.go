package store

import "time"

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

type ProjectMember struct {
	ID        uint   `gorm:"primaryKey"`
	ProjectID string `gorm:"not null;index;uniqueIndex:idx_project_user"`
	UserID    uint   `gorm:"not null;index;uniqueIndex:idx_project_user"`
	Role      string `gorm:"not null"`
	CreatedAt time.Time

	Project Project `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	User    User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
