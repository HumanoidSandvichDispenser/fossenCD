package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/afero"
	"gorm.io/driver/sqlite"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/config"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/okayeg"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/server"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/service"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(sqlite.Open(cfg.DBPath + "?_foreign_keys=on"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	host := okayeg.NewHostManager(ctx, okayeg.HostOptions{
		Bin:     cfg.EgBin,
		DataDir: cfg.DataDir,
	})

	// The eg daemons run the authz script with the paths baked into each
	// project's .eg/config.toml, so both must be absolute: eg inherits our cwd
	// today, but nothing should depend on that.
	authzScript, err := filepath.Abs(cfg.AuthzScript)
	if err != nil {
		log.Fatalf("authz script path: %v", err)
	}
	authzSocket := cfg.AuthzSocket
	if authzSocket == "" {
		authzSocket = filepath.Join(cfg.DataDir, "authz.sock")
	}
	if authzSocket, err = filepath.Abs(authzSocket); err != nil {
		log.Fatalf("authz socket path: %v", err)
	}

	svc := service.New(service.Options{
		DB:          db,
		FS:          afero.NewOsFs(),
		DataDir:     cfg.DataDir,
		SessionTTL:  7 * 24 * time.Hour,
		Host:        host,
		AuthzScript: authzScript,
		AuthzSocket: authzSocket,
	})

	// the authz socket: where each eg daemon's hook script asks whether a
	// connecting peer may sync (see internal/server/authz.go). A unix socket,
	// so the decision is never reachable from outside the container.
	_ = os.Remove(authzSocket) // a stale socket from a previous run blocks Listen
	authzLn, err := net.Listen("unix", authzSocket)
	if err != nil {
		log.Fatalf("authz socket: %v", err)
	}
	authzSrv := &http.Server{Handler: server.NewAuthzHandler(svc)}
	go func() {
		if err := authzSrv.Serve(authzLn); err != nil && err != http.ErrServerClosed {
			log.Fatalf("authz server: %v", err)
		}
	}()

	handler := server.New(svc, false)

	srv := &http.Server{Addr: cfg.Addr, Handler: handler}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutCtx)
		authzSrv.Shutdown(shutCtx)
	}()

	log.Printf("listening on %s", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
