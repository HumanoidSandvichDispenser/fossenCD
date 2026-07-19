package service

import (
	"time"

	"github.com/spf13/afero"
	"gorm.io/gorm"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/okayeg"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

type Services struct {
	Auth     *AuthService
	Projects *ProjectService
	Keys     *KeyService
	Emails   *EmailService
}

type Options struct {
	DB         *gorm.DB
	FS         afero.Fs
	DataDir    string
	SessionTTL time.Duration
	Host       okayeg.Host

	// AuthzScript and AuthzSocket build each project's .eg/config.toml authz
	// command: the script is run by eg per incoming connection and forwards
	// the decision to the backend over the unix socket. With AuthzScript empty
	// no config is written and eg falls back to its trust file.
	AuthzScript string
	AuthzSocket string

	// RequireEmailConfirmation withholds the verified state on added emails
	// until a confirmation flow resolves it. Off means adding an address
	// verifies it immediately.
	RequireEmailConfirmation bool
}

func New(o Options) *Services {
	projects := store.NewProjectStore(o.DB)
	users := store.NewUserStore(o.DB)
	auth := &AuthService{db: o.DB, ttl: o.SessionTTL}

	return &Services{
		Auth: auth,
		Projects: &ProjectService{
			projects:    projects,
			users:       users,
			fs:          o.FS,
			dataDir:     o.DataDir,
			host:        o.Host,
			authzScript: o.AuthzScript,
			authzSocket: o.AuthzSocket,
		},
		Keys: &KeyService{
			users:    users,
			projects: projects,
			host:     o.Host,
		},
		Emails: &EmailService{
			db:                  users,
			users:               auth,
			requireConfirmation: o.RequireEmailConfirmation,
		},
	}
}
