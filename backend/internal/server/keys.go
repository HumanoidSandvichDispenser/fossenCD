package server

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

// registerKeys mounts the account-scoped peer key routes. A key is an okayeg
// EndpointId (public key) the caller's browser or native peer uploads so it may
// sync every project the caller belongs to. Keys are listed and revoked here,
// not per project, because they are account-wide.
func (s *Server) registerKeys(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-keys", Method: http.MethodGet, Path: "/keys",
		Summary: "List the caller's registered okayeg peer keys",
	}, s.handleListKeys)

	huma.Register(api, huma.Operation{
		OperationID: "add-key", Method: http.MethodPost, Path: "/keys",
		Summary: "Register a peer public key (EndpointId) on the caller's account",
		DefaultStatus: http.StatusCreated,
	}, s.handleAddKey)

	huma.Register(api, huma.Operation{
		OperationID: "remove-key", Method: http.MethodDelete, Path: "/keys/{endpointId}",
		Summary: "Revoke a peer key and close its live sessions", DefaultStatus: http.StatusNoContent,
	}, s.handleRemoveKey)
}

type keyView struct {
	EndpointID string    `json:"endpoint_id"`
	Label      string    `json:"label"`
	CreatedAt  time.Time `json:"created_at"`
}

func viewKey(k store.UserKey) keyView {
	return keyView{EndpointID: k.EndpointID, Label: k.Label, CreatedAt: k.CreatedAt}
}

type keysOutput struct{ Body []keyView }

func (s *Server) handleListKeys(ctx context.Context, _ *struct{}) (*keysOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := s.svc.Keys.ListKeys(ctx, uid)
	if err != nil {
		return nil, httpError(err)
	}
	out := &keysOutput{Body: make([]keyView, 0, len(keys))}
	for _, k := range keys {
		out.Body = append(out.Body, viewKey(k))
	}
	return out, nil
}

type addKeyInput struct {
	Body struct {
		EndpointID string `json:"endpointId" minLength:"1"`
		Label      string `json:"label" maxLength:"128"`
	}
}

func (s *Server) handleAddKey(ctx context.Context, in *addKeyInput) (*struct{}, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Keys.AddKey(ctx, uid, in.Body.EndpointID, in.Body.Label); err != nil {
		return nil, httpError(err)
	}
	return nil, nil
}

type removeKeyInput struct {
	EndpointID string `path:"endpointId"`
}

func (s *Server) handleRemoveKey(ctx context.Context, in *removeKeyInput) (*struct{}, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Keys.RemoveKey(ctx, uid, in.EndpointID); err != nil {
		return nil, httpError(err)
	}
	return nil, nil
}
