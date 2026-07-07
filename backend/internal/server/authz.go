package server

import (
	"net/http"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/service"
)

// NewAuthzHandler serves the authz decision the eg daemons' hook script asks
// for, over a unix socket only the backend's container can reach. One route:
//
//	GET /projects/{projectID}/peers/{endpointID}
//
// 200 with body "pull push" grants the peer both directions; 403 refuses it.
// eg treats a non-2xx curl exit as a refusal, so the gate fails closed.
func NewAuthzHandler(svc *service.Services) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects/{projectID}/peers/{endpointID}", func(w http.ResponseWriter, r *http.Request) {
		ok, err := svc.Projects.AuthorizePeer(r.Context(), r.PathValue("projectID"), r.PathValue("endpointID"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "not authorized", http.StatusForbidden)
			return
		}
		w.Write([]byte("pull push\n"))
	})
	return mux
}
