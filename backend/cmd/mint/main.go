// Command mint loads a teamtype share's key, mints a wormhole join code, and
// keeps the rendezvous connection alive so one peer can claim it. The peer
// receives the share's secret address and connects by iroh.
//
//	mint <share-dir> [relay-url]
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/afero"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/teamtype"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mint <share-dir> [relay-url]")
		os.Exit(2)
	}
	dir := os.Args[1]
	relay := ""
	if len(os.Args) > 2 {
		relay = os.Args[2]
	}

	key, err := teamtype.LoadKey(afero.NewOsFs(), dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load key:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	code, done, err := key.MintJoinCode(ctx, relay)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mint:", err)
		os.Exit(1)
	}

	fmt.Println("address:", key.SecretAddress())
	fmt.Println("join code:", code)
	fmt.Fprintln(os.Stderr, "serving claim; Ctrl-C to stop")

	select {
	case err := <-done:
		if err != nil {
			fmt.Fprintln(os.Stderr, "delivery failed:", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "delivered secret address to peer")
	case <-ctx.Done():
	}
}
