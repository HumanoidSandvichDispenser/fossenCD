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
		Summary: "List the caller's projects",
	}, s.handleListProjects)

	huma.Register(api, huma.Operation{
		OperationID: "create-project", Method: http.MethodPost, Path: "/projects",
		Summary: "Create a project and its teamtype key",
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
		OperationID: "project-address", Method: http.MethodGet, Path: "/projects/{id}/address",
		Summary: "Get the secret address for web/wasm peers",
	}, s.handleProjectAddress)

	huma.Register(api, huma.Operation{
		OperationID: "project-join-code", Method: http.MethodPost, Path: "/projects/{id}/join-code",
		Summary: "Mint a wormhole join code for terminal peers",
	}, s.handleJoinCode)

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
		Summary: "Remove a collaborator (owner only)", DefaultStatus: http.StatusNoContent,
	}, s.handleRemoveMember)

	huma.Register(api, huma.Operation{
		OperationID: "project-logs", Method: http.MethodGet, Path: "/projects/{id}/logs",
		Summary: "View recent stdout/stderr of the project's teamtype daemon",
	}, s.handleProjectLogs)
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

type addressBody struct {
	Address string `json:"address"`
}

type addressOutput struct{ Body addressBody }

func (s *Server) handleProjectAddress(ctx context.Context, in *idInput) (*addressOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	addr, err := s.svc.Projects.Address(ctx, uid, in.ID)
	if err != nil {
		return nil, httpError(err)
	}
	return &addressOutput{Body: addressBody{Address: addr}}, nil
}

type joinCodeBody struct {
	Code    string `json:"code"`
	Address string `json:"address"`
}

type joinCodeOutput struct{ Body joinCodeBody }

func (s *Server) handleJoinCode(ctx context.Context, in *idInput) (*joinCodeOutput, error) {
	uid, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	code, addr, err := s.svc.Projects.MintJoinCode(ctx, uid, in.ID)
	if err != nil {
		return nil, httpError(err)
	}
	return &joinCodeOutput{Body: joinCodeBody{Code: code, Address: addr}}, nil
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
