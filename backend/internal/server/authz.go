package server

import (
	"net/http"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/service"
)

// NewAuthzHandler serves the eg daemons' hook scripts, over a unix socket only
// the backend's container can reach. Two routes:
//
//	GET  /projects/{projectID}/peers/{endpointID}
//	POST /projects/{projectID}/peers/{endpointID}/session
//
// GET is the authz decision: 200 with body "pull push" grants the peer both
// directions; 403 refuses it, with a reason as the body for the hook to relay
// to the dialer. eg treats a non-2xx curl exit as a refusal too, so the gate
// fails closed.
//
// POST is the session event feed: form fields event ("hello"/"bye"), name, and
// email record the peer's claimed identity in the presence registry. It
// decides nothing.
//
// The unregistered reason carries an enroll link when publicURL is configured;
// without it the peer is told to ask the project owner instead.
func NewAuthzHandler(svc *service.Services, publicURL string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects/{projectID}/peers/{endpointID}", func(w http.ResponseWriter, r *http.Request) {
		endpointID := r.PathValue("endpointID")
		verdict, err := svc.Projects.AuthorizePeer(r.Context(), r.PathValue("projectID"), endpointID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if verdict == service.PeerGranted {
			w.Write([]byte("pull push\n"))
			return
		}
		http.Error(w, refusalReason(verdict, publicURL, endpointID), http.StatusForbidden)
	})
	mux.HandleFunc("POST /projects/{projectID}/peers/{endpointID}/session", func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("projectID")
		endpointID := r.PathValue("endpointID")

		switch r.FormValue("event") {
		case "hello":
			claim, err := svc.Emails.VerifyClaim(r.Context(), endpointID, r.FormValue("email"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			svc.Presence.Hello(projectID, service.Peer{
				EndpointID: endpointID,
				Name:       r.FormValue("name"),
				Email:      r.FormValue("email"),
				Verified:   claim.Verified,
				Username:   claim.Username,
			})
		case "bye":
			svc.Presence.Bye(projectID, endpointID)
		default:
			http.Error(w, "unknown event", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func refusalReason(verdict service.PeerAuthz, publicURL, endpointID string) string {
	switch verdict {
	case service.PeerMalformedID:
		return "malformed endpoint id"
	case service.PeerNotMember:
		return "You are not a member of this project. Ask the project owner to add you."
	case service.PeerUnregistered:
		msg := "This key is not registered with this fossenCD server."
		if publicURL != "" {
			const enrollPath = "/enroll?key="
			msg += "\nRegister it at " + publicURL + enrollPath + endpointID
		} else {
			msg += " Register it on this instance, or ask its administrator to."
		}
		return msg
	default:
		return "not authorized"
	}
}
