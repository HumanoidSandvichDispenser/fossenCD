package server

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

type userView struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func viewUser(u store.User) userView {
	return userView{ID: u.ID, Username: u.Username, Email: u.Email}
}

type authOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      userView
}

func (s *Server) registerAuth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "register", Method: http.MethodPost, Path: "/auth/register",
		Summary: "Create an account and start a session",
	}, s.handleRegister)

	huma.Register(api, huma.Operation{
		OperationID: "login", Method: http.MethodPost, Path: "/auth/login",
		Summary: "Log in and start a session",
	}, s.handleLogin)

	huma.Register(api, huma.Operation{
		OperationID: "logout", Method: http.MethodPost, Path: "/auth/logout",
		Summary: "End the current session",
	}, s.handleLogout)

	huma.Register(api, huma.Operation{
		OperationID: "me", Method: http.MethodGet, Path: "/auth/me",
		Summary: "Return the current user",
	}, s.handleMe)
}

type registerInput struct {
	Body struct {
		Username string `json:"username" minLength:"3" maxLength:"32"`
		Email    string `json:"email" format:"email"`
		Password string `json:"password" minLength:"8" maxLength:"128"`
	}
}

func (s *Server) handleRegister(ctx context.Context, in *registerInput) (*authOutput, error) {
	u, sess, err := s.svc.Auth.Register(ctx, in.Body.Username, in.Body.Email, in.Body.Password)
	if err != nil {
		return nil, httpError(err)
	}
	return &authOutput{SetCookie: s.sessionCookieFor(sess.Token, sess.ExpiresAt), Body: viewUser(u)}, nil
}

type loginInput struct {
	Body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
}

func (s *Server) handleLogin(ctx context.Context, in *loginInput) (*authOutput, error) {
	u, sess, err := s.svc.Auth.Login(ctx, in.Body.Username, in.Body.Password)
	if err != nil {
		return nil, httpError(err)
	}
	return &authOutput{SetCookie: s.sessionCookieFor(sess.Token, sess.ExpiresAt), Body: viewUser(u)}, nil
}

type logoutInput struct {
	Cookie http.Cookie `cookie:"fossencd_session"`
}

type logoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

func (s *Server) handleLogout(ctx context.Context, in *logoutInput) (*logoutOutput, error) {
	if in.Cookie.Value != "" {
		if err := s.svc.Auth.Logout(ctx, in.Cookie.Value); err != nil {
			return nil, httpError(err)
		}
	}
	return &logoutOutput{SetCookie: s.sessionCookieFor("", time.Unix(0, 0))}, nil
}

type meOutput struct{ Body userView }

func (s *Server) handleMe(ctx context.Context, _ *struct{}) (*meOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.svc.Auth.User(ctx, uid)
	if err != nil {
		return nil, httpError(err)
	}
	return &meOutput{Body: viewUser(u)}, nil
}
