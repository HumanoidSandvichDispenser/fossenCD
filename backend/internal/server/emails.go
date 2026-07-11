package server

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

// registerEmails mounts a user's email set.
func (s *Server) registerEmails(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-emails", Method: http.MethodGet, Path: "/emails",
		Summary: "List the user's email addresses",
	}, s.handleListEmails)

	huma.Register(api, huma.Operation{
		OperationID: "add-email", Method: http.MethodPost, Path: "/emails",
		Summary:       "Add an email address to the user's account",
		DefaultStatus: http.StatusCreated,
	}, s.handleAddEmail)

	huma.Register(api, huma.Operation{
		OperationID: "remove-email", Method: http.MethodDelete, Path: "/emails/{email}",
		Summary: "Remove a non-primary email address", DefaultStatus: http.StatusNoContent,
	}, s.handleRemoveEmail)
}

type emailView struct {
	Email     string    `json:"email"`
	Verified  bool      `json:"verified"`
	Primary   bool      `json:"primary"`
	CreatedAt time.Time `json:"created_at"`
}

func viewEmail(e store.UserEmail, primary string) emailView {
	return emailView{
		Email:     e.Email,
		Verified:  e.VerifiedAt != nil,
		Primary:   e.Email == primary,
		CreatedAt: e.CreatedAt,
	}
}

type emailsOutput struct{ Body []emailView }

func (s *Server) handleListEmails(ctx context.Context, _ *struct{}) (*emailsOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	u, err := s.svc.Auth.User(ctx, uid)
	if err != nil {
		return nil, httpError(err)
	}
	emails, err := s.svc.Emails.ListEmails(ctx, uid)
	if err != nil {
		return nil, httpError(err)
	}

	out := &emailsOutput{Body: make([]emailView, 0, len(emails))}
	for _, e := range emails {
		out.Body = append(out.Body, viewEmail(e, u.Email))
	}
	return out, nil
}

type addEmailInput struct {
	Body struct {
		Email string `json:"email" format:"email" maxLength:"254"`
	}
}

func (s *Server) handleAddEmail(ctx context.Context, in *addEmailInput) (*struct{}, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Emails.AddEmail(ctx, uid, in.Body.Email); err != nil {
		return nil, httpError(err)
	}
	return nil, nil
}

type removeEmailInput struct {
	Email string `path:"email"`
}

func (s *Server) handleRemoveEmail(ctx context.Context, in *removeEmailInput) (*struct{}, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Emails.RemoveEmail(ctx, uid, in.Email); err != nil {
		return nil, httpError(err)
	}
	return nil, nil
}
