package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/afero"
	"gorm.io/driver/sqlite"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/config"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/server"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/service"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/teamtype"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(sqlite.Open(cfg.DBPath + "?_foreign_keys=on"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	host := teamtype.NewHostManager(ctx, teamtype.HostOptions{
		Bin:           cfg.TeamtypeBin,
		DataDir:       cfg.DataDir,
		WormholeRelay: cfg.Relay,
		IrohRelay:     cfg.IrohRelay,
		IrohPkarr:     cfg.IrohPkarr,
		IrohDNSDomain: cfg.IrohDNSDomain,
	})

	svc := service.New(service.Options{
		DB:         db,
		FS:         afero.NewOsFs(),
		DataDir:    cfg.DataDir,
		Relay:      cfg.Relay,
		SessionTTL: 7 * 24 * time.Hour,
		MintCtx:    ctx,
		Host:       host,
	})

	handler := server.New(svc, false)

	srv := &http.Server{Addr: cfg.Addr, Handler: handler}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutCtx)
	}()

	log.Printf("listening on %s", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
