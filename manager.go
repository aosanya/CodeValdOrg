package codevaldorg

import "context"

// OrgManager is the flat domain interface for all org operations.
// The agencyID is baked into the implementation via ManagerConfig.AgencyID;
// no method takes agencyID as a parameter.
type OrgManager interface {
	// Organization lifecycle
	InitOrganization(ctx context.Context, req InitOrganizationRequest) (Organization, error)
	GetOrganization(ctx context.Context) (Organization, error)
	UpdateOrganization(ctx context.Context, req UpdateOrganizationRequest) (Organization, error)
	DisableOrganization(ctx context.Context) (Organization, error)
	DeleteOrganization(ctx context.Context) error

	// User & invitation
	InviteUser(ctx context.Context, req InviteUserRequest) (Invitation, error)
	AcceptInvitation(ctx context.Context, token string) (User, error)
	GetUser(ctx context.Context, userID string) (User, error)
	ListUsers(ctx context.Context, req ListUsersRequest) ([]User, error)
	SuspendUser(ctx context.Context, userID string) (User, error)
	ReactivateUser(ctx context.Context, userID string) (User, error)
	DeleteUser(ctx context.Context, userID string) error

	// Roles
	CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error)
	UpdateRole(ctx context.Context, roleID string, req UpdateRoleRequest) (Role, error)
	DeleteRole(ctx context.Context, roleID string) error
	ListRoles(ctx context.Context, req ListRequest) ([]Role, error)

	// Scopes
	RegisterScope(ctx context.Context, req RegisterScopeRequest) (Scope, error)
	DeprecateScope(ctx context.Context, scopeID string) (Scope, error)
	ListScopes(ctx context.Context, req ListRequest) ([]Scope, error)

	// Memberships
	GrantMembership(ctx context.Context, req GrantMembershipRequest) (Membership, error)
	RevokeMembership(ctx context.Context, membershipID string) (Membership, error)
	ListMemberships(ctx context.Context, req ListMembershipsRequest) ([]Membership, error)

	// OAuth clients
	CreateOAuthClient(ctx context.Context, req CreateOAuthClientRequest) (OAuthClient, string, error)
	RotateClientSecret(ctx context.Context, clientID string) (string, error)
	ListOAuthClients(ctx context.Context, req ListRequest) ([]OAuthClient, error)
	DeleteOAuthClient(ctx context.Context, clientID string) error

	// OAuth 2.0 protocol
	Authorize(ctx context.Context, req AuthorizeRequest) (string, string, error)
	Token(ctx context.Context, req TokenRequest) (TokenResult, error)
	Introspect(ctx context.Context, token string) (IntrospectResult, error)
	Revoke(ctx context.Context, token, reason string) error

	// Audit
	ListAuditEvents(ctx context.Context, filter AuditEventFilter) ([]AuditEvent, error)
}

// orgManager is the concrete OrgManager implementation.
type orgManager struct {
	dm    OrgDataManager
	sm    OrgSchemaManager
	pub   CrossPublisher
	clock Clock
	cfg   ManagerConfig
}

// NewOrgManager constructs a production OrgManager.
func NewOrgManager(dm OrgDataManager, sm OrgSchemaManager, pub CrossPublisher, clock Clock, cfg ManagerConfig) OrgManager {
	return &orgManager{
		dm:    dm,
		sm:    sm,
		pub:   pub,
		clock: clock,
		cfg:   cfg,
	}
}
