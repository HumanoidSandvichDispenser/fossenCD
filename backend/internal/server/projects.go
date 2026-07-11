package server

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

type projectView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func viewProject(p store.Project) projectView {
	return projectView{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt}
}

func (s *Server) registerProjects(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-projects", Method: http.MethodGet, Path: "/projects",
		Summary: "List the user's projects",
	}, s.handleListProjects)

	huma.Register(api, huma.Operation{
		OperationID: "create-project", Method: http.MethodPost, Path: "/projects",
		Summary: "Create a project and its okayeg key",
	}, s.handleCreateProject)

	huma.Register(api, huma.Operation{
		OperationID: "get-project", Method: http.MethodGet, Path: "/projects/{id}",
		Summary: "Get a project",
	}, s.handleGetProject)

	huma.Register(api, huma.Operation{
		OperationID: "delete-project", Method: http.MethodDelete, Path: "/projects/{id}",
		Summary: "Delete a project and its share dir", DefaultStatus: http.StatusNoContent,
	}, s.handleDeleteProject)

	huma.Register(api, huma.Operation{
		OperationID: "project-endpoint", Method: http.MethodGet, Path: "/projects/{id}/endpoint",
		Summary: "Get the host endpoint id for browser/wasm peers to dial",
	}, s.handleProjectEndpoint)

	huma.Register(api, huma.Operation{
		OperationID: "list-members", Method: http.MethodGet, Path: "/projects/{id}/members",
		Summary: "List a project's collaborators",
	}, s.handleListMembers)

	huma.Register(api, huma.Operation{
		OperationID: "add-member", Method: http.MethodPost, Path: "/projects/{id}/members",
		Summary: "Add a collaborator by username or email (owner only)",
	}, s.handleAddMember)

	huma.Register(api, huma.Operation{
		OperationID: "remove-member", Method: http.MethodDelete, Path: "/projects/{id}/members/{userId}",
		Summary: "Remove a collaborator and revoke their peers (owner only)", DefaultStatus: http.StatusNoContent,
	}, s.handleRemoveMember)

	huma.Register(api, huma.Operation{
		OperationID: "project-logs", Method: http.MethodGet, Path: "/projects/{id}/logs",
		Summary: "View recent stdout/stderr of the project's eg daemon",
	}, s.handleProjectLogs)

	huma.Register(api, huma.Operation{
		OperationID: "verify-peers", Method: http.MethodPost, Path: "/projects/{id}/peers/verify",
		Summary: "Check peers' claimed emails against their keys' owning accounts",
	}, s.handleVerifyPeers)
}

// peerClaim is one peer as its presence entry shows it: the endpoint id the
// peer connected with and the email it claims, blank when it claims none.
type peerClaim struct {
	EndpointID string `json:"endpoint_id" minLength:"1"`
	Email      string `json:"email"`
}

type peerVerdict struct {
	EndpointID string `json:"endpoint_id"`
	Verified   bool   `json:"verified"`
	// Username of the account the claimed email verified against.
	Username string `json:"username"`
}

type verifyPeersInput struct {
	ID   string `path:"id"`
	Body struct {
		Claims []peerClaim `json:"claims" maxItems:"128"`
	}
}

type verifyPeersOutput struct{ Body []peerVerdict }

func (s *Server) handleVerifyPeers(ctx context.Context, in *verifyPeersInput) (*verifyPeersOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	// membership gate; the claims themselves came to the caller over presence
	if _, err := s.svc.Projects.Get(ctx, uid, in.ID); err != nil {
		return nil, httpError(err)
	}

	out := &verifyPeersOutput{Body: make([]peerVerdict, 0, len(in.Body.Claims))}
	for _, c := range in.Body.Claims {
		claim, err := s.svc.Emails.VerifyClaim(ctx, c.EndpointID, c.Email)
		if err != nil {
			return nil, httpError(err)
		}
		out.Body = append(out.Body, peerVerdict{
			EndpointID: c.EndpointID,
			Verified:   claim.Verified,
			Username:   claim.Username,
		})
	}
	return out, nil
}

type idInput struct {
	ID string `path:"id"`
}

type projectsOutput struct{ Body []projectView }

func (s *Server) handleListProjects(ctx context.Context, _ *struct{}) (*projectsOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	ps, err := s.svc.Projects.List(ctx, uid)
	if err != nil {
		return nil, httpError(err)
	}
	out := &projectsOutput{Body: make([]projectView, 0, len(ps))}
	for _, p := range ps {
		out.Body = append(out.Body, viewProject(p))
	}
	return out, nil
}

type createProjectInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"128"`
	}
}

type projectOutput struct{ Body projectView }

func (s *Server) handleCreateProject(ctx context.Context, in *createProjectInput) (*projectOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.svc.Projects.Create(ctx, uid, in.Body.Name)
	if err != nil {
		return nil, httpError(err)
	}
	return &projectOutput{Body: viewProject(p)}, nil
}

func (s *Server) handleGetProject(ctx context.Context, in *idInput) (*projectOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.svc.Projects.Get(ctx, uid, in.ID)
	if err != nil {
		return nil, httpError(err)
	}
	return &projectOutput{Body: viewProject(p)}, nil
}

func (s *Server) handleDeleteProject(ctx context.Context, in *idInput) (*struct{}, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Projects.Delete(ctx, uid, in.ID); err != nil {
		return nil, httpError(err)
	}
	return nil, nil
}

type endpointBody struct {
	EndpointID string `json:"endpointId"`
}

type endpointOutput struct{ Body endpointBody }

func (s *Server) handleProjectEndpoint(ctx context.Context, in *idInput) (*endpointOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	endpoint, err := s.svc.Projects.Endpoint(ctx, uid, in.ID)
	if err != nil {
		return nil, httpError(err)
	}
	return &endpointOutput{Body: endpointBody{EndpointID: endpoint}}, nil
}

type memberView struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func viewMember(m store.ProjectMember) memberView {
	return memberView{UserID: m.UserID, Username: m.User.Username, Email: m.User.Email, Role: m.Role}
}

type membersOutput struct{ Body []memberView }

// membersOf loads the project's members and shapes them for the response. Used
// by both the list and add handlers so they return the same view.
func (s *Server) membersOf(ctx context.Context, uid uint, id string) (*membersOutput, error) {
	members, err := s.svc.Projects.ListMembers(ctx, uid, id)
	if err != nil {
		return nil, httpError(err)
	}
	out := &membersOutput{Body: make([]memberView, 0, len(members))}
	for _, m := range members {
		out.Body = append(out.Body, viewMember(m))
	}
	return out, nil
}

func (s *Server) handleListMembers(ctx context.Context, in *idInput) (*membersOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.membersOf(ctx, uid, in.ID)
}

type addMemberInput struct {
	ID   string `path:"id"`
	Body struct {
		Login string `json:"login" minLength:"1"`
	}
}

func (s *Server) handleAddMember(ctx context.Context, in *addMemberInput) (*membersOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Projects.AddMember(ctx, uid, in.ID, in.Body.Login); err != nil {
		return nil, httpError(err)
	}
	return s.membersOf(ctx, uid, in.ID)
}

type removeMemberInput struct {
	ID     string `path:"id"`
	UserID uint   `path:"userId"`
}

func (s *Server) handleRemoveMember(ctx context.Context, in *removeMemberInput) (*struct{}, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Projects.RemoveMember(ctx, uid, in.ID, in.UserID); err != nil {
		return nil, httpError(err)
	}
	return nil, nil
}

type logsBody struct {
	Running bool   `json:"running"`
	Output  string `json:"output"`
}

type logsOutput struct{ Body logsBody }

func (s *Server) handleProjectLogs(ctx context.Context, in *idInput) (*logsOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	output, running, err := s.svc.Projects.Logs(ctx, uid, in.ID)
	if err != nil {
		return nil, httpError(err)
	}
	return &logsOutput{Body: logsBody{Running: running, Output: output}}, nil
}