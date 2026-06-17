package service

import (
	"context"
	"time"

	"github.com/spf13/afero"
	"gorm.io/gorm"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/teamtype"
)

type Services struct {
	Auth     *AuthService
	Projects *ProjectService
}

type Options struct {
	DB         *gorm.DB
	FS         afero.Fs
	DataDir    string
	Relay      string
	SessionTTL time.Duration
	MintCtx    context.Context
	Host       teamtype.Host
}

func New(o Options) *Services {
	projects := store.NewProjectStore(o.DB)
	users := store.NewUserStore(o.DB)

	return &Services{
		Auth: &AuthService{db: o.DB, ttl: o.SessionTTL},
		Projects: &ProjectService{
			projects: projects,
			users:    users,
			fs:       o.FS,
			dataDir:  o.DataDir,
			relay:    o.Relay,
			mintCtx:  o.MintCtx,
			host:     o.Host,
		},
	}
}
