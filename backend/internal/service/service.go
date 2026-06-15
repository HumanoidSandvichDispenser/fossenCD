package service

import (
	"context"
	"time"

	"github.com/spf13/afero"
	"gorm.io/gorm"
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
}

func New(o Options) *Services {
	return &Services{
		Auth: &AuthService{db: o.DB, ttl: o.SessionTTL},
		Projects: &ProjectService{
			db:      o.DB,
			fs:      o.FS,
			dataDir: o.DataDir,
			relay:   o.Relay,
			mintCtx: o.MintCtx,
		},
	}
}
