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
		DB:          db,
		FS:          afero.NewMemMapFs(),
		DataDir:     "data",
		SessionTTL:  time.Hour,
		Host:        &noopHost{},
		AuthzScript: "/app/scripts/eg-authz",
		AuthzSocket: "/data/authz.sock",
	})
}

// noopHost is an okayeg.Host that records EnsureHost and Revoke calls without
// spawning anything, so service tests stay hermetic.
type noopHost struct {
	ensured map[string]int
	revoked []string // "projectID endpointID" per Revoke call
}

func (h *noopHost) EnsureHost(id string) {
	if h.ensured == nil {
		h.ensured = map[string]int{}
	}
	h.ensured[id]++
}

func (h *noopHost) Stop(string) {}

func (h *noopHost) Revoke(id, endpointID string) error {
	h.revoked = append(h.revoked, id+" "+endpointID)
	return nil
}

func (h *noopHost) Logs(string) (string, bool) { return "", false }

var _ okayeg.Host = (*noopHost)(nil)
