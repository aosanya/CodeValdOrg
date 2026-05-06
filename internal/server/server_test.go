package server_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	codevaldorg "github.com/aosanya/CodeValdOrg"
	pb "github.com/aosanya/CodeValdOrg/gen/go/codevaldorg/v1"
	"github.com/aosanya/CodeValdOrg/internal/server"
)

// fakeManager is a minimal OrgManager stub for server-layer tests.
type fakeManager struct {
	getOrgErr error
	getOrgVal codevaldorg.Organization
}

func (f *fakeManager) InitOrganization(ctx context.Context, req codevaldorg.InitOrganizationRequest) (codevaldorg.Organization, error) {
	return codevaldorg.Organization{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) GetOrganization(ctx context.Context) (codevaldorg.Organization, error) {
	return f.getOrgVal, f.getOrgErr
}
func (f *fakeManager) UpdateOrganization(ctx context.Context, req codevaldorg.UpdateOrganizationRequest) (codevaldorg.Organization, error) {
	return codevaldorg.Organization{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) DisableOrganization(ctx context.Context) (codevaldorg.Organization, error) {
	return codevaldorg.Organization{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) DeleteOrganization(ctx context.Context) error {
	return codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) InviteUser(ctx context.Context, req codevaldorg.InviteUserRequest) (codevaldorg.Invitation, error) {
	return codevaldorg.Invitation{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) AcceptInvitation(ctx context.Context, token string) (codevaldorg.User, error) {
	return codevaldorg.User{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) GetUser(ctx context.Context, userID string) (codevaldorg.User, error) {
	return codevaldorg.User{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) ListUsers(ctx context.Context, req codevaldorg.ListUsersRequest) ([]codevaldorg.User, error) {
	return nil, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) SuspendUser(ctx context.Context, userID string) (codevaldorg.User, error) {
	return codevaldorg.User{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) ReactivateUser(ctx context.Context, userID string) (codevaldorg.User, error) {
	return codevaldorg.User{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) DeleteUser(ctx context.Context, userID string) error {
	return codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) CreateRole(ctx context.Context, req codevaldorg.CreateRoleRequest) (codevaldorg.Role, error) {
	return codevaldorg.Role{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) UpdateRole(ctx context.Context, roleID string, req codevaldorg.UpdateRoleRequest) (codevaldorg.Role, error) {
	return codevaldorg.Role{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) DeleteRole(ctx context.Context, roleID string) error {
	return codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) ListRoles(ctx context.Context, req codevaldorg.ListRequest) ([]codevaldorg.Role, error) {
	return nil, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) RegisterScope(ctx context.Context, req codevaldorg.RegisterScopeRequest) (codevaldorg.Scope, error) {
	return codevaldorg.Scope{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) DeprecateScope(ctx context.Context, scopeID string) (codevaldorg.Scope, error) {
	return codevaldorg.Scope{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) ListScopes(ctx context.Context, req codevaldorg.ListRequest) ([]codevaldorg.Scope, error) {
	return nil, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) GrantMembership(ctx context.Context, req codevaldorg.GrantMembershipRequest) (codevaldorg.Membership, error) {
	return codevaldorg.Membership{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) RevokeMembership(ctx context.Context, membershipID string) (codevaldorg.Membership, error) {
	return codevaldorg.Membership{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) ListMemberships(ctx context.Context, req codevaldorg.ListMembershipsRequest) ([]codevaldorg.Membership, error) {
	return nil, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) CreateOAuthClient(ctx context.Context, req codevaldorg.CreateOAuthClientRequest) (codevaldorg.OAuthClient, string, error) {
	return codevaldorg.OAuthClient{}, "", codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) RotateClientSecret(ctx context.Context, clientID string) (string, error) {
	return "", codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) ListOAuthClients(ctx context.Context, req codevaldorg.ListRequest) ([]codevaldorg.OAuthClient, error) {
	return nil, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) DeleteOAuthClient(ctx context.Context, clientID string) error {
	return codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) Authorize(ctx context.Context, req codevaldorg.AuthorizeRequest) (string, string, error) {
	return "", "", codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) Token(ctx context.Context, req codevaldorg.TokenRequest) (codevaldorg.TokenResult, error) {
	return codevaldorg.TokenResult{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) Introspect(ctx context.Context, token string) (codevaldorg.IntrospectResult, error) {
	return codevaldorg.IntrospectResult{}, codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) Revoke(ctx context.Context, token, reason string) error {
	return codevaldorg.ErrTemporarilyUnavailable
}
func (f *fakeManager) ListAuditEvents(ctx context.Context, filter codevaldorg.AuditEventFilter) ([]codevaldorg.AuditEvent, error) {
	return nil, codevaldorg.ErrTemporarilyUnavailable
}

func TestOrgServer_RejectsMismatchedAgencyID(t *testing.T) {
	mgr := &fakeManager{}
	srv := server.New(mgr, "agency-abc")

	_, err := srv.GetOrganization(context.Background(), &pb.GetOrganizationRequest{
		AgencyId: "agency-different",
	})
	if err == nil {
		t.Fatal("expected error for mismatched agency_id")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", st.Code())
	}
}

func TestOrgServer_GetOrganization_NotFound(t *testing.T) {
	mgr := &fakeManager{getOrgErr: codevaldorg.ErrOrgNotFound}
	srv := server.New(mgr, "agency-abc")

	_, err := srv.GetOrganization(context.Background(), &pb.GetOrganizationRequest{
		AgencyId: "agency-abc",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestOrgServer_GetOrganization_OK(t *testing.T) {
	mgr := &fakeManager{
		getOrgVal: codevaldorg.Organization{
			ID:       "org-1",
			AgencyID: "agency-abc",
			Name:     "Test Org",
			Enabled:  true,
		},
	}
	srv := server.New(mgr, "agency-abc")

	resp, err := srv.GetOrganization(context.Background(), &pb.GetOrganizationRequest{
		AgencyId: "agency-abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetName() != "Test Org" {
		t.Errorf("expected name 'Test Org', got %q", resp.GetName())
	}
}
