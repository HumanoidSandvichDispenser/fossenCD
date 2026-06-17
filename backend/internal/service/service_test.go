package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/afero"
	"gorm.io/driver/sqlite"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

func newServices(t *testing.T) *Services {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on", t.Name())
	db, err := store.Open(sqlite.Open(dsn))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	return New(Options{
		DB:         db,
		FS:         afero.NewMemMapFs(),
		DataDir:    "data",
		SessionTTL: time.Hour,
		MintCtx:    context.Background(),
		Host:       noopHost{},
	})
}

type noopHost struct{}

func (noopHost) EnsureHost(string)          {}
func (noopHost) Stop(string)                {}
func (noopHost) Logs(string) (string, bool) { return "", false }
