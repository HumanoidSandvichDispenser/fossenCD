package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRegisterSeedsVerifiedPrimaryEmail(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	u, _, err := svc.Auth.Register(ctx, "alice", "alice@x.io", "hunter2hunter2")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	emails, err := svc.Emails.ListEmails(ctx, u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(emails) != 1 || emails[0].Email != "alice@x.io" || emails[0].VerifiedAt == nil {
		t.Fatalf("expected one verified primary email, got %+v", emails)
	}
}

func TestAddEmailVerifiesWithoutConfirmation(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	u, _, _ := svc.Auth.Register(ctx, "alice", "alice@x.io", "hunter2hunter2")
	if err := svc.Emails.AddEmail(ctx, u.ID, "work@x.io"); err != nil {
		t.Fatalf("add: %v", err)
	}

	emails, _ := svc.Emails.ListEmails(ctx, u.ID)
	if len(emails) != 2 || emails[1].Email != "work@x.io" || emails[1].VerifiedAt == nil {
		t.Fatalf("expected a second verified email, got %+v", emails)
	}
}

func TestAddEmailRejectsDuplicatesAcrossAccounts(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	alice, _, _ := svc.Auth.Register(ctx, "alice", "alice@x.io", "hunter2hunter2")
	bob, _, _ := svc.Auth.Register(ctx, "bob", "bob@x.io", "hunter2hunter2")

	if err := svc.Emails.AddEmail(ctx, bob.ID, "alice@x.io"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict claiming alice's email, got %v", err)
	}
	if err := svc.Emails.AddEmail(ctx, alice.ID, "alice@x.io"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict re-adding own email, got %v", err)
	}
	if err := svc.Emails.AddEmail(ctx, bob.ID, "not-an-email"); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest for malformed address, got %v", err)
	}
}

func TestRemoveEmailKeepsPrimary(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	u, _, _ := svc.Auth.Register(ctx, "alice", "alice@x.io", "hunter2hunter2")
	_ = svc.Emails.AddEmail(ctx, u.ID, "work@x.io")

	if err := svc.Emails.RemoveEmail(ctx, u.ID, "alice@x.io"); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest removing primary, got %v", err)
	}
	if err := svc.Emails.RemoveEmail(ctx, u.ID, "work@x.io"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := svc.Emails.RemoveEmail(ctx, u.ID, "work@x.io"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound removing twice, got %v", err)
	}
}

func TestVerifyClaimMatchesEndpointOwnersVerifiedSet(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	alice, _, _ := svc.Auth.Register(ctx, "alice", "alice@x.io", "hunter2hunter2")
	endpoint := strings.Repeat("ab", 32)
	if err := svc.Keys.AddKey(ctx, alice.ID, endpoint, "laptop"); err != nil {
		t.Fatalf("add key: %v", err)
	}

	claim, err := svc.Emails.VerifyClaim(ctx, endpoint, "alice@x.io")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !claim.Verified || claim.Username != "alice" {
		t.Fatalf("expected verified claim for alice, got %+v", claim)
	}

	// an email outside the owning account's verified set stays unverified,
	// even when it belongs to some other account
	_, _, _ = svc.Auth.Register(ctx, "bob", "bob@x.io", "hunter2hunter2")
	for _, email := range []string{"bob@x.io", "stranger@x.io", ""} {
		claim, err := svc.Emails.VerifyClaim(ctx, endpoint, email)
		if err != nil {
			t.Fatalf("verify %q: %v", email, err)
		}
		if claim.Verified {
			t.Fatalf("expected unverified claim for %q", email)
		}
	}

	// an unregistered endpoint verifies nothing
	claim, err = svc.Emails.VerifyClaim(ctx, strings.Repeat("cd", 32), "alice@x.io")
	if err != nil || claim.Verified {
		t.Fatalf("expected unverified claim for unknown endpoint, got %+v %v", claim, err)
	}
}
