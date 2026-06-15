package service

import (
	"context"
	"errors"
	"testing"
)

func TestRegisterThenLogin(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()

	u, sess, err := svc.Auth.Register(ctx, "ryan", "r@x.io", "hunter2hunter2")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.ID == 0 || sess.Token == "" {
		t.Fatalf("expected user id and session token, got %+v %+v", u, sess)
	}

	got, _, err := svc.Auth.Login(ctx, "ryan", "hunter2hunter2")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("login returned user %d, want %d", got.ID, u.ID)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	svc.Auth.Register(ctx, "ryan", "r@x.io", "hunter2hunter2")

	_, _, err := svc.Auth.Login(ctx, "ryan", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc := newServices(t)
	_, _, err := svc.Auth.Login(context.Background(), "nobody", "whatever12")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	svc.Auth.Register(ctx, "ryan", "r@x.io", "hunter2hunter2")

	_, _, err := svc.Auth.Register(ctx, "ryan", "other@x.io", "hunter2hunter2")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestResolveSession(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	u, sess, _ := svc.Auth.Register(ctx, "ryan", "r@x.io", "hunter2hunter2")

	got, err := svc.Auth.ResolveSession(ctx, sess.Token)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("resolved user %d, want %d", got.ID, u.ID)
	}
}

func TestResolveSessionAfterLogout(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	_, sess, _ := svc.Auth.Register(ctx, "ryan", "r@x.io", "hunter2hunter2")

	if err := svc.Auth.Logout(ctx, sess.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Auth.ResolveSession(ctx, sess.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestResolveSessionBadToken(t *testing.T) {
	svc := newServices(t)
	if _, err := svc.Auth.ResolveSession(context.Background(), "garbage"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}
