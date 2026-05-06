package codevaldorg

import (
	"errors"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// Re-exported entitygraph errors so callers need not import entitygraph.
var (
	ErrEntityNotFound    = entitygraph.ErrEntityNotFound
	ErrImmutableType     = entitygraph.ErrImmutableType
)

// OAuth 2.0 endpoint errors (RFC 6749 §5.2).
var (
	ErrInvalidRequest         = errors.New("invalid_request")
	ErrInvalidClient          = errors.New("invalid_client")
	ErrInvalidGrant           = errors.New("invalid_grant")
	ErrUnauthorizedClient     = errors.New("unauthorized_client")
	ErrUnsupportedGrantType   = errors.New("unsupported_grant_type")
	ErrInvalidScope           = errors.New("invalid_scope")
	ErrAccessDenied           = errors.New("access_denied")
	ErrTemporarilyUnavailable = errors.New("temporarily_unavailable")
	ErrRateLimitExceeded      = errors.New("rate_limit_exceeded")
)

// Admin surface errors.
var (
	ErrOrgNotFound               = errors.New("organization not found")
	ErrOrgAlreadyExists          = errors.New("organization already exists")
	ErrOrgDisabled               = errors.New("organization disabled")
	ErrUserNotFound              = errors.New("user not found")
	ErrUserAlreadyExists         = errors.New("user already exists")
	ErrUserSuspended             = errors.New("user suspended")
	ErrRoleNotFound              = errors.New("role not found")
	ErrRoleAlreadyExists         = errors.New("role already exists")
	ErrRoleBuiltinImmutable      = errors.New("built-in role cannot be modified or deleted")
	ErrScopeNotFound             = errors.New("scope not found")
	ErrScopeNameCollision        = errors.New("scope name already registered by a different owner")
	ErrScopeReserved             = errors.New("scope prefix is reserved")
	ErrMembershipNotFound        = errors.New("membership not found")
	ErrInvitationNotFound        = errors.New("invitation not found")
	ErrInvitationExpired         = errors.New("invitation expired")
	ErrInvitationAlreadyAccepted = errors.New("invitation already accepted")
	ErrOAuthClientNotFound       = errors.New("oauth client not found")
	ErrRedirectURIMismatch       = errors.New("redirect_uri mismatch")
	ErrPKCERequired              = errors.New("pkce required for public clients")
	ErrPKCEMethodInvalid         = errors.New("code_challenge_method must be S256")
	ErrPKCEMismatch              = errors.New("code_verifier does not match code_challenge")
	ErrTokenRevoked              = errors.New("token revoked")
	ErrTokenExpired              = errors.New("token expired")
	ErrSuperAdminRequired        = errors.New("cannot remove the last super_admin")
)
