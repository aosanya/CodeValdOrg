// Package registrar provides the CodeValdOrg Cross heartbeat registrar.
// It wraps the SharedLib registrar and announces all admin routes to Cross.
// OAuth endpoints are NOT included — they are served directly by Org's HTTP
// listener and discoverable via /.well-known/oauth-authorization-server.
package registrar

import (
	"time"

	sharedregistrar "github.com/aosanya/CodeValdSharedLib/registrar"
	"github.com/aosanya/CodeValdSharedLib/types"
)

// Registrar is the SharedLib registrar type alias.
type Registrar = sharedregistrar.Registrar

// New constructs a Registrar that heartbeats to the CodeValdCross gRPC server
// at crossAddr. It registers all 27 admin routes from cross-registration.md.
func New(
	crossAddr, advertiseAddr, agencyID string,
	pingInterval, pingTimeout time.Duration,
) (sharedregistrar.Registrar, error) {
	produces := []string{"cross.org." + agencyID + ".token.revoked"}
	return sharedregistrar.New(
		crossAddr,
		advertiseAddr,
		agencyID,
		"codevaldorg",
		produces,
		nil,
		orgRoutes(agencyID),
		pingInterval,
		pingTimeout,
	)
}

// orgRoutes returns the 27 admin HTTP routes CodeValdOrg registers with Cross.
func orgRoutes(agencyID string) []types.RouteInfo {
	base := "/{agencyId}/org"
	ab := types.PathBinding{URLParam: "agencyId", Field: "agency_id"}
	userB := types.PathBinding{URLParam: "userId", Field: "user_id"}
	roleB := types.PathBinding{URLParam: "roleId", Field: "role_id"}
	membershipB := types.PathBinding{URLParam: "membershipId", Field: "membership_id"}
	clientB := types.PathBinding{URLParam: "clientId", Field: "client_id"}
	scopeB := types.PathBinding{URLParam: "scopeId", Field: "scope_id"}

	svc := "/codevaldorg.v1.OrgService/"

	return []types.RouteInfo{
		// Organization lifecycle
		{Method: "POST", Pattern: base, GrpcMethod: svc + "InitOrganization",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "GET", Pattern: base, GrpcMethod: svc + "GetOrganization",
			PathBindings: []types.PathBinding{ab}},
		{Method: "PATCH", Pattern: base, GrpcMethod: svc + "UpdateOrganization",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "POST", Pattern: base + "/disable", GrpcMethod: svc + "DisableOrganization",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "DELETE", Pattern: base, GrpcMethod: svc + "DeleteOrganization",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},

		// Users & invitations
		{Method: "POST", Pattern: base + "/users/invite", GrpcMethod: svc + "InviteUser",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "POST", Pattern: base + "/invitations/accept", GrpcMethod: svc + "AcceptInvitation",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "GET", Pattern: base + "/users", GrpcMethod: svc + "ListUsers",
			PathBindings: []types.PathBinding{ab}},
		{Method: "GET", Pattern: base + "/users/{userId}", GrpcMethod: svc + "GetUser",
			PathBindings: []types.PathBinding{ab, userB}},
		{Method: "POST", Pattern: base + "/users/{userId}/suspend", GrpcMethod: svc + "SuspendUser",
			PathBindings: []types.PathBinding{ab, userB}, IsWrite: true},
		{Method: "POST", Pattern: base + "/users/{userId}/reactivate", GrpcMethod: svc + "ReactivateUser",
			PathBindings: []types.PathBinding{ab, userB}, IsWrite: true},
		{Method: "DELETE", Pattern: base + "/users/{userId}", GrpcMethod: svc + "DeleteUser",
			PathBindings: []types.PathBinding{ab, userB}, IsWrite: true},

		// Roles
		{Method: "POST", Pattern: base + "/roles", GrpcMethod: svc + "CreateRole",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "GET", Pattern: base + "/roles", GrpcMethod: svc + "ListRoles",
			PathBindings: []types.PathBinding{ab}},
		{Method: "PATCH", Pattern: base + "/roles/{roleId}", GrpcMethod: svc + "UpdateRole",
			PathBindings: []types.PathBinding{ab, roleB}, IsWrite: true},
		{Method: "DELETE", Pattern: base + "/roles/{roleId}", GrpcMethod: svc + "DeleteRole",
			PathBindings: []types.PathBinding{ab, roleB}, IsWrite: true},

		// Scopes
		{Method: "POST", Pattern: base + "/scopes", GrpcMethod: svc + "RegisterScope",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "POST", Pattern: base + "/scopes/{scopeId}/deprecate", GrpcMethod: svc + "DeprecateScope",
			PathBindings: []types.PathBinding{ab, scopeB}, IsWrite: true},
		{Method: "GET", Pattern: base + "/scopes", GrpcMethod: svc + "ListScopes",
			PathBindings: []types.PathBinding{ab}},

		// Memberships
		{Method: "POST", Pattern: base + "/memberships", GrpcMethod: svc + "GrantMembership",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "DELETE", Pattern: base + "/memberships/{membershipId}", GrpcMethod: svc + "RevokeMembership",
			PathBindings: []types.PathBinding{ab, membershipB}, IsWrite: true},
		{Method: "GET", Pattern: base + "/memberships", GrpcMethod: svc + "ListMemberships",
			PathBindings: []types.PathBinding{ab}},

		// OAuth clients
		{Method: "POST", Pattern: base + "/oauth-clients", GrpcMethod: svc + "CreateOAuthClient",
			PathBindings: []types.PathBinding{ab}, IsWrite: true},
		{Method: "GET", Pattern: base + "/oauth-clients", GrpcMethod: svc + "ListOAuthClients",
			PathBindings: []types.PathBinding{ab}},
		{Method: "POST", Pattern: base + "/oauth-clients/{clientId}/rotate-secret",
			GrpcMethod:   svc + "RotateClientSecret",
			PathBindings: []types.PathBinding{ab, clientB}, IsWrite: true},
		{Method: "DELETE", Pattern: base + "/oauth-clients/{clientId}", GrpcMethod: svc + "DeleteOAuthClient",
			PathBindings: []types.PathBinding{ab, clientB}, IsWrite: true},

		// Audit
		{Method: "GET", Pattern: base + "/audit", GrpcMethod: svc + "ListAuditEvents",
			PathBindings: []types.PathBinding{ab}},
	}
}

// Ensure agencyID compile-time reference (avoids unused variable lint warning).
var _ = orgRoutes
