// Package httphandler provides the OAuth 2.0 HTTP endpoints served directly
// by CodeValdOrg (not proxied through Cross). These include:
//   - GET/POST /{agencyId}/oauth/authorize
//   - POST /{agencyId}/oauth/token
//   - POST /{agencyId}/oauth/introspect
//   - POST /{agencyId}/oauth/revoke
//   - GET /{agencyId}/.well-known/oauth-authorization-server
//
// In v1 these are stubs that return 503. Full implementation is ORG-010.
package httphandler

import (
	"net/http"
)

// Handler is the HTTP mux for OAuth endpoints.
type Handler struct {
	mux      *http.ServeMux
	agencyID string
	issuerURL string
}

// New constructs the HTTP handler.
func New(agencyID, issuerURL string) *Handler {
	h := &Handler{
		mux:       http.NewServeMux(),
		agencyID:  agencyID,
		issuerURL: issuerURL,
	}
	h.mux.HandleFunc("/"+agencyID+"/oauth/authorize", h.handleAuthorize)
	h.mux.HandleFunc("/"+agencyID+"/oauth/token", h.handleToken)
	h.mux.HandleFunc("/"+agencyID+"/oauth/introspect", h.handleIntrospect)
	h.mux.HandleFunc("/"+agencyID+"/oauth/revoke", h.handleRevoke)
	h.mux.HandleFunc("/"+agencyID+"/.well-known/oauth-authorization-server", h.handleMeta)
	h.mux.HandleFunc("/healthz", h.handleHealth)
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"temporarily_unavailable"}`, http.StatusServiceUnavailable)
}

func (h *Handler) handleToken(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"temporarily_unavailable"}`, http.StatusServiceUnavailable)
}

func (h *Handler) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"temporarily_unavailable"}`, http.StatusServiceUnavailable)
}

func (h *Handler) handleRevoke(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"temporarily_unavailable"}`, http.StatusServiceUnavailable)
}

func (h *Handler) handleMeta(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	base := h.issuerURL + "/" + h.agencyID
	_, _ = w.Write([]byte(`{` +
		`"issuer":"` + base + `",` +
		`"authorization_endpoint":"` + base + `/oauth/authorize",` +
		`"token_endpoint":"` + base + `/oauth/token",` +
		`"introspection_endpoint":"` + base + `/oauth/introspect",` +
		`"revocation_endpoint":"` + base + `/oauth/revoke",` +
		`"response_types_supported":["code"],` +
		`"grant_types_supported":["authorization_code","client_credentials","refresh_token"],` +
		`"token_endpoint_auth_methods_supported":["client_secret_basic","client_secret_post","none"],` +
		`"code_challenge_methods_supported":["S256"]` +
		`}`))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
