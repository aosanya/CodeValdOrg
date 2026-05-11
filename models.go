// Package codevaldorg provides identity, OAuth 2.0, and access-management
// services for CodeVald agencies. One process per agency; AGENCY_ID is baked
// in at startup.
package codevaldorg

import (
	"time"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
	"github.com/aosanya/CodeValdSharedLib/registrar"
)

// OrgDataManager is the entitygraph DataManager scoped to the org domain.
type OrgDataManager = entitygraph.DataManager

// OrgSchemaManager is the entitygraph SchemaManager scoped to the org domain.
type OrgSchemaManager = entitygraph.SchemaManager

// CrossPublisher is the registrar used by the org domain to publish events.
type CrossPublisher = registrar.Registrar

// Clock allows tests to inject a deterministic time source.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// NewClock returns a Clock backed by time.Now.
func NewClock() Clock { return realClock{} }

// ── Domain structs ────────────────────────────────────────────────────────────

// Organization is the root identity entity for an agency.
type Organization struct {
	ID           string
	AgencyID     string
	Name         string
	Enabled      bool
	Description  string
	ContactEmail string
	LogoURL      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DisabledAt   *time.Time
	DeletedAt    *time.Time
}

// User is a person or service identity within an organization.
type User struct {
	ID          string
	AgencyID    string
	Email       string
	Status      string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// PasswordCredential holds an Argon2id PHC hash for a User.
type PasswordCredential struct {
	ID           string
	AgencyID     string
	PasswordHash string
	CreatedAt    time.Time
	LastUsedAt   *time.Time
	RevokedAt    *time.Time
	ExpiresAt    *time.Time
}

// Role is a named permission set; built-in roles cannot be modified or deleted.
type Role struct {
	ID          string
	AgencyID    string
	Name        string
	Builtin     bool
	DisplayName string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Scope is a registered permission string in the form <service>:<action>.
type Scope struct {
	ID           string
	AgencyID     string
	Name         string
	RegisteredBy string
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeprecatedAt *time.Time
}

// Membership binds a User to a Role.
type Membership struct {
	ID        string
	AgencyID  string
	UserID    string
	RoleID    string
	GrantedAt time.Time
	GrantedBy string
	RevokedAt *time.Time
	RevokedBy string
}

// Invitation is an outstanding user invitation with a one-time token.
type Invitation struct {
	ID         string
	AgencyID   string
	TokenHash  string
	Status     string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AcceptedAt *time.Time
	RevokedAt  *time.Time
}

// OAuthClient is a registered OAuth 2.0 client.
type OAuthClient struct {
	ID                string
	AgencyID          string
	ClientID          string
	ClientType        string
	Name              string
	AllowedGrantTypes []string
	Description       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DisabledAt        *time.Time
	DeletedAt         *time.Time
}

// AuditEvent is an immutable forensic log entry.
type AuditEvent struct {
	ID        string
	AgencyID  string
	EventType string
	ActorID   string
	SubjectID string
	Outcome   string
	EventAt   time.Time
	Payload   string
}

// AuthorizationCode is an immutable PKCE authorization code.
type AuthorizationCode struct {
	ID            string
	AgencyID      string
	CodeHash      string
	CodeChallenge string
	RedirectURI   string
	ExpiresAt     time.Time
	State         string
	CreatedAt     time.Time
	ConsumedAt    *time.Time
}

// TokenResult is returned by Token flows.
type TokenResult struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	ExpiresIn    int32
	Scopes       []string
}

// IntrospectResult is returned by the Introspect flow.
type IntrospectResult struct {
	Active    bool
	Scopes    []string
	Sub       string
	ClientID  string
	Exp       time.Time
	Iat       time.Time
	TokenType string
}

// ── Request types ─────────────────────────────────────────────────────────────

// InitOrganizationRequest is the input for creating an Organization.
type InitOrganizationRequest struct {
	Name         string
	Description  string
	ContactEmail string
	LogoURL      string
}

// UpdateOrganizationRequest patches Organization fields.
type UpdateOrganizationRequest struct {
	Name         string
	Description  string
	ContactEmail string
	LogoURL      string
}

// InviteUserRequest creates a User in "invited" status plus an Invitation.
type InviteUserRequest struct {
	Email         string
	DisplayName   string
	InvitedByUserID string
	RoleIDs       []string
	ExpiresIn     time.Duration
}

// ListRequest is a generic pagination request.
type ListRequest struct {
	PageToken string
	PageSize  int32
}

// ListUsersRequest filters and paginates users.
type ListUsersRequest struct {
	StatusFilter string
	PageToken    string
	PageSize     int32
}

// ListMembershipsRequest filters memberships by user.
type ListMembershipsRequest struct {
	UserID    string
	PageToken string
	PageSize  int32
}

// CreateRoleRequest creates a custom (non-builtin) Role.
type CreateRoleRequest struct {
	Name        string
	DisplayName string
	Description string
}

// UpdateRoleRequest patches mutable Role fields.
type UpdateRoleRequest struct {
	DisplayName string
	Description string
}

// RegisterScopeRequest registers or idempotently updates a Scope.
type RegisterScopeRequest struct {
	Name         string
	RegisteredBy string
	Description  string
}

// GrantMembershipRequest binds a User to a Role.
type GrantMembershipRequest struct {
	UserID    string
	RoleID    string
	GrantedBy string
}

// CreateOAuthClientRequest registers a new OAuth client.
type CreateOAuthClientRequest struct {
	Name              string
	ClientType        string
	Description       string
	AllowedGrantTypes []string
	RedirectURIs      []string
	AllowedScopeIDs   []string
}

// AuthorizeRequest is the input for the Authorization Code + PKCE flow.
type AuthorizeRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Scopes              []string
	UserID              string
}

// TokenRequest is the input for all token grant flows.
type TokenRequest struct {
	GrantType    string
	ClientID     string
	ClientSecret string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	Scopes       []string
}

// AuditEventFilter scopes a ListAuditEvents query.
type AuditEventFilter struct {
	EventTypeFilter string
	ActorIDFilter   string
	SubjectIDFilter string
	PageToken       string
	PageSize        int32
}

// ManagerConfig carries constructor-time configuration for OrgManager.
type ManagerConfig struct {
	AgencyID          string
	IssuerURL         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	AuthCodeTTL       time.Duration
	ClientSecretGrace time.Duration
	Argon2Time        uint32
	Argon2MemoryKiB   uint32
	Argon2Threads     uint8
}
