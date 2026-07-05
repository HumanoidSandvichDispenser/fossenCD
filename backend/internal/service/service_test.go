package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/afero"
	"gorm.io/driver/sqlite"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/okayeg"
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
		Host:       &noopHost{},
	})
}

// noopHost is an okayeg.Host that records EnsureHost calls without spawning
// anything, so service tests stay hermetic.
type noopHost struct {
	ensured map[string]int
}

func (h *noopHost) EnsureHost(id string) {
	if h.ensured == nil {
		h.ensured = map[string]int{}
	}
	h.ensured[id]++
}

func (h *noopHost) Stop(string)                {}
func (h *noopHost) Logs(string) (string, bool) { return "", false }

var _ okayeg.Host = (*noopHost)(nil)
