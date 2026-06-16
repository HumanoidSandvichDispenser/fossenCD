// Command passwd resets a user's password out-of-band (e.g. a locked-out user,
// before any email-based reset flow exists) and revokes their active sessions.
//
//	passwd <username>
//
// The new password is read twice from the terminal without echo. It uses the
// same FOSSENCD_DB path as the server.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
	"gorm.io/driver/sqlite"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/config"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/service"
	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

const minPasswordLen = 8

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: passwd <username>")
		os.Exit(2)
	}
	username := os.Args[1]

	password, err := readNewPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg := config.Load()
	db, err := store.Open(sqlite.Open(cfg.DBPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}

	svc := service.New(service.Options{DB: db})
	if _, err := svc.Auth.SetPassword(context.Background(), username, password); err != nil {
		fmt.Fprintln(os.Stderr, "set password:", err)
		os.Exit(1)
	}

	fmt.Printf("password reset for %q; active sessions revoked\n", username)
}

// readNewPassword prompts twice (no echo) and verifies the entries match.
func readNewPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("refusing to read password from a non-terminal stdin")
	}

	fmt.Fprint(os.Stderr, "new password: ")
	first, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if len(first) < minPasswordLen {
		return "", fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	fmt.Fprint(os.Stderr, "confirm password: ")
	second, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if string(first) != string(second) {
		return "", errors.New("passwords do not match")
	}
	return string(first), nil
}
