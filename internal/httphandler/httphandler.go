// Package httphandler provides the OAuth 2.0 HTTP endpoints served directly
// by CodeValdOrg (not proxied through Cross). These include:
//   - GET/POST /{agencyId}/oauth/authorize
//   - POST /{agencyId}/oauth/token
//   - POST /{agencyId}/oauth/introspect
//   - POST /{agencyId}/oauth/revoke
//   - GET /{agencyId}/.well-known/oauth-authorization-server
package httphandler

import (
	"encoding/json"
	"net/http"
	"strings"

	codevaldorg "github.com/aosanya/CodeValdOrg"
)

// Handler is the HTTP mux for OAuth endpoints.
type Handler struct {
	mux       *http.ServeMux
	agencyID  string
	issuerURL string
	mgr       codevaldorg.OrgManager
}

// New constructs the HTTP handler. mgr may be nil (endpoints return 503).
func New(agencyID, issuerURL string, mgr codevaldorg.OrgManager) *Handler {
	h := &Handler{
		mux:       http.NewServeMux(),
		agencyID:  agencyID,
		issuerURL: issuerURL,
		mgr:       mgr,
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

// handleAuthorize handles GET/POST /{agencyId}/oauth/authorize.
// The caller must supply user_id (already authenticated by the frontend session layer).
// For redirect-based flows the frontend is responsible for the browser redirect;
// this endpoint returns a JSON body with the code for backend-to-backend use.
func (h *Handler) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if h.mgr == nil {
		writeOAuthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form")
		return
	}
	req := codevaldorg.AuthorizeRequest{
		ClientID:            r.FormValue("client_id"),
		RedirectURI:         r.FormValue("redirect_uri"),
		ResponseType:        r.FormValue("response_type"),
		State:               r.FormValue("state"),
		CodeChallenge:       r.FormValue("code_challenge"),
		CodeChallengeMethod: r.FormValue("code_challenge_method"),
		UserID:              r.FormValue("user_id"),
	}
	if sc := r.FormValue("scope"); sc != "" {
		req.Scopes = strings.Fields(sc)
	}
	if req.ResponseType != "code" {
		writeOAuthError(w, http.StatusBadRequest, "unsupported_response_type", "only 'code' is supported")
		return
	}
	code, state, err := h.mgr.Authorize(r.Context(), req)
	if err != nil {
		status, oauthErr := mapToHTTPError(err)
		writeOAuthError(w, status, oauthErr, "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": code, "state": state})
}

// handleToken handles POST /{agencyId}/oauth/token (RFC 6749).
func (h *Handler) handleToken(w http.ResponseWriter, r *http.Request) {
	if h.mgr == nil {
		writeOAuthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "")
		return
	}
	if r.Method != http.MethodPost {
		writeOAuthError(w, http.StatusMethodNotAllowed, "invalid_request", "method must be POST")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form")
		return
	}

	clientID, clientSecret := extractClientCredentials(r)

	req := codevaldorg.TokenRequest{
		GrantType:    r.FormValue("grant_type"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		CodeVerifier: r.FormValue("code_verifier"),
		RefreshToken: r.FormValue("refresh_token"),
	}
	if sc := r.FormValue("scope"); sc != "" {
		req.Scopes = strings.Fields(sc)
	}

	result, err := h.mgr.Token(r.Context(), req)
	if err != nil {
		status, oauthErr := mapToHTTPError(err)
		writeOAuthError(w, status, oauthErr, "")
		return
	}

	resp := map[string]any{
		"access_token": result.AccessToken,
		"token_type":   result.TokenType,
		"expires_in":   result.ExpiresIn,
	}
	if len(result.Scopes) > 0 {
		resp["scope"] = strings.Join(result.Scopes, " ")
	}
	if result.RefreshToken != "" {
		resp["refresh_token"] = result.RefreshToken
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, resp)
}

// handleIntrospect handles POST /{agencyId}/oauth/introspect (RFC 7662).
func (h *Handler) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if h.mgr == nil {
		writeOAuthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "")
		return
	}
	if r.Method != http.MethodPost {
		writeOAuthError(w, http.StatusMethodNotAllowed, "invalid_request", "method must be POST")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form")
		return
	}
	token := r.FormValue("token")
	if token == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}
	result, err := h.mgr.Introspect(r.Context(), token)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	resp := map[string]any{"active": result.Active}
	if result.Active {
		resp["token_type"] = result.TokenType
		resp["exp"] = result.Exp.Unix()
		resp["iat"] = result.Iat.Unix()
		if result.Sub != "" {
			resp["sub"] = result.Sub
		}
		if result.ClientID != "" {
			resp["client_id"] = result.ClientID
		}
		if len(result.Scopes) > 0 {
			resp["scope"] = strings.Join(result.Scopes, " ")
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleRevoke handles POST /{agencyId}/oauth/revoke (RFC 7009).
func (h *Handler) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if h.mgr == nil {
		writeOAuthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "")
		return
	}
	if r.Method != http.MethodPost {
		writeOAuthError(w, http.StatusMethodNotAllowed, "invalid_request", "method must be POST")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form")
		return
	}
	token := r.FormValue("token")
	if token == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}
	reason := r.FormValue("reason")
	if err := h.mgr.Revoke(r.Context(), token, reason); err != nil {
		status, oauthErr := mapToHTTPError(err)
		writeOAuthError(w, status, oauthErr, "")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleMeta(w http.ResponseWriter, r *http.Request) {
	base := h.issuerURL + "/" + h.agencyID
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"introspection_endpoint":                base + "/oauth/introspect",
		"revocation_endpoint":                   base + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"code_challenge_methods_supported":      []string{"S256"},
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// ── helpers ───────────────────────────────────────────────────────────────────

// extractClientCredentials pulls client_id/client_secret from Basic auth or form params.
func extractClientCredentials(r *http.Request) (clientID, clientSecret string) {
	if id, secret, ok := r.BasicAuth(); ok {
		return id, secret
	}
	return r.FormValue("client_id"), r.FormValue("client_secret")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func writeOAuthError(w http.ResponseWriter, status int, errCode, description string) {
	resp := map[string]string{"error": errCode}
	if description != "" {
		resp["error_description"] = description
	}
	writeJSON(w, status, resp)
}

func mapToHTTPError(err error) (int, string) {
	switch err {
	case codevaldorg.ErrInvalidClient:
		return http.StatusUnauthorized, "invalid_client"
	case codevaldorg.ErrInvalidGrant:
		return http.StatusBadRequest, "invalid_grant"
	case codevaldorg.ErrInvalidScope:
		return http.StatusBadRequest, "invalid_scope"
	case codevaldorg.ErrInvalidRequest:
		return http.StatusBadRequest, "invalid_request"
	case codevaldorg.ErrUnauthorizedClient:
		return http.StatusBadRequest, "unauthorized_client"
	case codevaldorg.ErrUnsupportedGrantType:
		return http.StatusBadRequest, "unsupported_grant_type"
	case codevaldorg.ErrAccessDenied:
		return http.StatusForbidden, "access_denied"
	case codevaldorg.ErrPKCERequired, codevaldorg.ErrPKCEMethodInvalid, codevaldorg.ErrPKCEMismatch:
		return http.StatusBadRequest, "invalid_grant"
	case codevaldorg.ErrRedirectURIMismatch:
		return http.StatusBadRequest, "invalid_request"
	case codevaldorg.ErrTemporarilyUnavailable, codevaldorg.ErrRateLimitExceeded:
		return http.StatusServiceUnavailable, "temporarily_unavailable"
	default:
		return http.StatusInternalServerError, "server_error"
	}
}
