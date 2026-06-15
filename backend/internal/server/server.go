package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/service"
)

const sessionCookie = "fossencd_session"

type userIDKey struct{}

type Server struct {
	svc    *service.Services
	secure bool
}

func New(svc *service.Services, secure bool) http.Handler {
	s := &Server{svc: svc, secure: secure}

	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("fossenCD", "0.1.0"))
	api.UseMiddleware(s.sessionMiddleware)

	s.registerAuth(api)
	s.registerProjects(api)
	return router
}

func (s *Server) sessionMiddleware(ctx huma.Context, next func(huma.Context)) {
	if c, err := huma.ReadCookie(ctx, sessionCookie); err == nil {
		if u, err := s.svc.Auth.ResolveSession(ctx.Context(), c.Value); err == nil {
			ctx = huma.WithValue(ctx, userIDKey{}, u.ID)
		}
	}
	next(ctx)
}

func requireUser(ctx context.Context) (uint, error) {
	if id, ok := ctx.Value(userIDKey{}).(uint); ok {
		return id, nil
	}
	return 0, huma.Error401Unauthorized("login required")
}

func (s *Server) sessionCookieFor(token string, expires time.Time) http.Cookie {
	return http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	}
}

func httpError(err error) error {
	switch {
	case errors.Is(err, service.ErrUnauthorized), errors.Is(err, service.ErrInvalidCredentials):
		return huma.Error401Unauthorized(err.Error())
	case errors.Is(err, service.ErrConflict):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, service.ErrNotFound):
		return huma.Error404NotFound(err.Error())
	default:
		return err
	}
}
