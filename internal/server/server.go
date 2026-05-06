// Package server implements the OrgService gRPC handler.
// All business logic is delegated to OrgManager; this layer only
// translates between proto messages and domain types, validates that
// the request agency_id matches the baked-in agencyID, and maps
// domain errors to gRPC status codes via mappers.go.
package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	codevaldorg "github.com/aosanya/CodeValdOrg"
	pb "github.com/aosanya/CodeValdOrg/gen/go/codevaldorg/v1"
)

// OrgServer implements pb.OrgServiceServer by delegating to an OrgManager.
type OrgServer struct {
	pb.UnimplementedOrgServiceServer
	mgr      codevaldorg.OrgManager
	agencyID string
}

// New constructs an OrgServer.
func New(mgr codevaldorg.OrgManager, agencyID string) *OrgServer {
	return &OrgServer{mgr: mgr, agencyID: agencyID}
}

// checkAgencyID returns PermissionDenied if the request agencyID doesn't match.
func (s *OrgServer) checkAgencyID(requestAgencyID string) error {
	if requestAgencyID != s.agencyID {
		return status.Errorf(codes.PermissionDenied, "agency_id mismatch")
	}
	return nil
}

// ts converts a time.Time to a proto Timestamp (nil-safe).
func ts(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// tsPtr converts a *time.Time to a proto Timestamp.
func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// ── Organization Lifecycle ────────────────────────────────────────────────────

func (s *OrgServer) InitOrganization(ctx context.Context, req *pb.InitOrganizationRequest) (*pb.Organization, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	org, err := s.mgr.InitOrganization(ctx, codevaldorg.InitOrganizationRequest{
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		ContactEmail: req.GetContactEmail(),
		LogoURL:      req.GetLogoUrl(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoOrg(org), nil
}

func (s *OrgServer) GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.Organization, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	org, err := s.mgr.GetOrganization(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoOrg(org), nil
}

func (s *OrgServer) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.Organization, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	org, err := s.mgr.UpdateOrganization(ctx, codevaldorg.UpdateOrganizationRequest{
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		ContactEmail: req.GetContactEmail(),
		LogoURL:      req.GetLogoUrl(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoOrg(org), nil
}

func (s *OrgServer) DisableOrganization(ctx context.Context, req *pb.DisableOrganizationRequest) (*pb.Organization, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	org, err := s.mgr.DisableOrganization(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoOrg(org), nil
}

func (s *OrgServer) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	if err := s.mgr.DeleteOrganization(ctx); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteOrganizationResponse{AgencyId: s.agencyID}, nil
}

// ── User & Invitation ─────────────────────────────────────────────────────────

func (s *OrgServer) InviteUser(ctx context.Context, req *pb.InviteUserRequest) (*pb.Invitation, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	inv, err := s.mgr.InviteUser(ctx, codevaldorg.InviteUserRequest{
		Email:           req.GetEmail(),
		DisplayName:     req.GetDisplayName(),
		InvitedByUserID: req.GetInvitedByUserId(),
		RoleIDs:         req.GetRoleIds(),
		ExpiresIn:       time.Duration(req.GetExpiresInSeconds()) * time.Second,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoInvitation(inv), nil
}

func (s *OrgServer) AcceptInvitation(ctx context.Context, req *pb.AcceptInvitationRequest) (*pb.User, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	user, err := s.mgr.AcceptInvitation(ctx, req.GetToken())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoUser(user), nil
}

func (s *OrgServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	user, err := s.mgr.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoUser(user), nil
}

func (s *OrgServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	users, err := s.mgr.ListUsers(ctx, codevaldorg.ListUsersRequest{
		StatusFilter: req.GetStatusFilter(),
		PageToken:    req.GetPageToken(),
		PageSize:     req.GetPageSize(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		pbUsers[i] = toProtoUser(u)
	}
	return &pb.ListUsersResponse{Users: pbUsers}, nil
}

func (s *OrgServer) SuspendUser(ctx context.Context, req *pb.SuspendUserRequest) (*pb.User, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	user, err := s.mgr.SuspendUser(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoUser(user), nil
}

func (s *OrgServer) ReactivateUser(ctx context.Context, req *pb.ReactivateUserRequest) (*pb.User, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	user, err := s.mgr.ReactivateUser(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoUser(user), nil
}

func (s *OrgServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	if err := s.mgr.DeleteUser(ctx, req.GetUserId()); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteUserResponse{AgencyId: s.agencyID, UserId: req.GetUserId()}, nil
}

// ── Roles & Scopes ────────────────────────────────────────────────────────────

func (s *OrgServer) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.Role, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	role, err := s.mgr.CreateRole(ctx, codevaldorg.CreateRoleRequest{
		Name:        req.GetName(),
		DisplayName: req.GetDisplayName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoRole(role), nil
}

func (s *OrgServer) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.Role, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	role, err := s.mgr.UpdateRole(ctx, req.GetRoleId(), codevaldorg.UpdateRoleRequest{
		DisplayName: req.GetDisplayName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoRole(role), nil
}

func (s *OrgServer) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	if err := s.mgr.DeleteRole(ctx, req.GetRoleId()); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteRoleResponse{AgencyId: s.agencyID, RoleId: req.GetRoleId()}, nil
}

func (s *OrgServer) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	roles, err := s.mgr.ListRoles(ctx, codevaldorg.ListRequest{
		PageToken: req.GetPageToken(),
		PageSize:  req.GetPageSize(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbRoles := make([]*pb.Role, len(roles))
	for i, r := range roles {
		pbRoles[i] = toProtoRole(r)
	}
	return &pb.ListRolesResponse{Roles: pbRoles}, nil
}

func (s *OrgServer) RegisterScope(ctx context.Context, req *pb.RegisterScopeRequest) (*pb.Scope, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	scope, err := s.mgr.RegisterScope(ctx, codevaldorg.RegisterScopeRequest{
		Name:         req.GetName(),
		RegisteredBy: req.GetRegisteredBy(),
		Description:  req.GetDescription(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoScope(scope), nil
}

func (s *OrgServer) DeprecateScope(ctx context.Context, req *pb.DeprecateScopeRequest) (*pb.Scope, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	scope, err := s.mgr.DeprecateScope(ctx, req.GetScopeId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoScope(scope), nil
}

func (s *OrgServer) ListScopes(ctx context.Context, req *pb.ListScopesRequest) (*pb.ListScopesResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	scopes, err := s.mgr.ListScopes(ctx, codevaldorg.ListRequest{
		PageToken: req.GetPageToken(),
		PageSize:  req.GetPageSize(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbScopes := make([]*pb.Scope, len(scopes))
	for i, sc := range scopes {
		pbScopes[i] = toProtoScope(sc)
	}
	return &pb.ListScopesResponse{Scopes: pbScopes}, nil
}

// ── Membership ────────────────────────────────────────────────────────────────

func (s *OrgServer) GrantMembership(ctx context.Context, req *pb.GrantMembershipRequest) (*pb.Membership, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	mem, err := s.mgr.GrantMembership(ctx, codevaldorg.GrantMembershipRequest{
		UserID:    req.GetUserId(),
		RoleID:    req.GetRoleId(),
		GrantedBy: req.GetGrantedBy(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoMembership(mem), nil
}

func (s *OrgServer) RevokeMembership(ctx context.Context, req *pb.RevokeMembershipRequest) (*pb.Membership, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	mem, err := s.mgr.RevokeMembership(ctx, req.GetMembershipId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoMembership(mem), nil
}

func (s *OrgServer) ListMemberships(ctx context.Context, req *pb.ListMembershipsRequest) (*pb.ListMembershipsResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	mems, err := s.mgr.ListMemberships(ctx, codevaldorg.ListMembershipsRequest{
		UserID:    req.GetUserId(),
		PageToken: req.GetPageToken(),
		PageSize:  req.GetPageSize(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbMems := make([]*pb.Membership, len(mems))
	for i, m := range mems {
		pbMems[i] = toProtoMembership(m)
	}
	return &pb.ListMembershipsResponse{Memberships: pbMems}, nil
}

// ── OAuth Clients ─────────────────────────────────────────────────────────────

func (s *OrgServer) CreateOAuthClient(ctx context.Context, req *pb.CreateOAuthClientRequest) (*pb.CreateOAuthClientResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	client, secret, err := s.mgr.CreateOAuthClient(ctx, codevaldorg.CreateOAuthClientRequest{
		Name:              req.GetName(),
		ClientType:        req.GetClientType(),
		Description:       req.GetDescription(),
		AllowedGrantTypes: req.GetAllowedGrantTypes(),
		RedirectURIs:      req.GetRedirectUris(),
		AllowedScopeIDs:   req.GetAllowedScopeIds(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.CreateOAuthClientResponse{
		Client:          toProtoOAuthClient(client),
		PlaintextSecret: secret,
	}, nil
}

func (s *OrgServer) RotateClientSecret(ctx context.Context, req *pb.RotateClientSecretRequest) (*pb.RotateClientSecretResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	secret, err := s.mgr.RotateClientSecret(ctx, req.GetClientId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.RotateClientSecretResponse{PlaintextSecret: secret}, nil
}

func (s *OrgServer) ListOAuthClients(ctx context.Context, req *pb.ListOAuthClientsRequest) (*pb.ListOAuthClientsResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	clients, err := s.mgr.ListOAuthClients(ctx, codevaldorg.ListRequest{
		PageToken: req.GetPageToken(),
		PageSize:  req.GetPageSize(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbClients := make([]*pb.OAuthClient, len(clients))
	for i, c := range clients {
		pbClients[i] = toProtoOAuthClient(c)
	}
	return &pb.ListOAuthClientsResponse{Clients: pbClients}, nil
}

func (s *OrgServer) DeleteOAuthClient(ctx context.Context, req *pb.DeleteOAuthClientRequest) (*pb.DeleteOAuthClientResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	if err := s.mgr.DeleteOAuthClient(ctx, req.GetClientId()); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.DeleteOAuthClientResponse{AgencyId: s.agencyID, ClientId: req.GetClientId()}, nil
}

// ── OAuth 2.0 Protocol ────────────────────────────────────────────────────────

func (s *OrgServer) Authorize(ctx context.Context, req *pb.AuthorizeRequest) (*pb.AuthorizeResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	code, state, err := s.mgr.Authorize(ctx, codevaldorg.AuthorizeRequest{
		ClientID:            req.GetClientId(),
		RedirectURI:         req.GetRedirectUri(),
		ResponseType:        req.GetResponseType(),
		State:               req.GetState(),
		CodeChallenge:       req.GetCodeChallenge(),
		CodeChallengeMethod: req.GetCodeChallengeMethod(),
		Scopes:              req.GetScopes(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.AuthorizeResponse{Code: code, State: state}, nil
}

func (s *OrgServer) Token(ctx context.Context, req *pb.TokenRequest) (*pb.TokenResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	result, err := s.mgr.Token(ctx, codevaldorg.TokenRequest{
		GrantType:    req.GetGrantType().String(),
		ClientID:     req.GetClientId(),
		ClientSecret: req.GetClientSecret(),
		Code:         req.GetCode(),
		RedirectURI:  req.GetRedirectUri(),
		CodeVerifier: req.GetCodeVerifier(),
		RefreshToken: req.GetRefreshToken(),
		Scopes:       req.GetScopes(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.TokenResponse{
		AccessToken:  result.AccessToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		RefreshToken: result.RefreshToken,
		Scopes:       result.Scopes,
	}, nil
}

func (s *OrgServer) Introspect(ctx context.Context, req *pb.IntrospectRequest) (*pb.IntrospectResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	result, err := s.mgr.Introspect(ctx, req.GetToken())
	if err != nil {
		return nil, toGRPCError(err)
	}
	resp := &pb.IntrospectResponse{
		Active:    result.Active,
		Scopes:    result.Scopes,
		Sub:       result.Sub,
		ClientId:  result.ClientID,
		TokenType: result.TokenType,
	}
	if result.Active {
		resp.Exp = ts(result.Exp)
		resp.Iat = ts(result.Iat)
	}
	return resp, nil
}

func (s *OrgServer) Revoke(ctx context.Context, req *pb.RevokeRequest) (*pb.RevokeResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	if err := s.mgr.Revoke(ctx, req.GetToken(), req.GetReason()); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.RevokeResponse{AgencyId: s.agencyID}, nil
}

// ── Audit ─────────────────────────────────────────────────────────────────────

func (s *OrgServer) ListAuditEvents(ctx context.Context, req *pb.ListAuditEventsRequest) (*pb.ListAuditEventsResponse, error) {
	if err := s.checkAgencyID(req.GetAgencyId()); err != nil {
		return nil, err
	}
	events, err := s.mgr.ListAuditEvents(ctx, codevaldorg.AuditEventFilter{
		EventTypeFilter: req.GetEventTypeFilter(),
		ActorIDFilter:   req.GetActorIdFilter(),
		SubjectIDFilter: req.GetSubjectIdFilter(),
		PageToken:       req.GetPageToken(),
		PageSize:        req.GetPageSize(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	pbEvents := make([]*pb.AuditEvent, len(events))
	for i, e := range events {
		pbEvents[i] = toProtoAuditEvent(e)
	}
	return &pb.ListAuditEventsResponse{Events: pbEvents}, nil
}

// ── Proto conversion helpers ──────────────────────────────────────────────────

func toProtoOrg(o codevaldorg.Organization) *pb.Organization {
	return &pb.Organization{
		AgencyId:     o.AgencyID,
		Name:         o.Name,
		Enabled:      o.Enabled,
		Description:  o.Description,
		ContactEmail: o.ContactEmail,
		LogoUrl:      o.LogoURL,
		CreatedAt:    ts(o.CreatedAt),
		UpdatedAt:    ts(o.UpdatedAt),
		DisabledAt:   tsPtr(o.DisabledAt),
		DeletedAt:    tsPtr(o.DeletedAt),
	}
}

func toProtoUser(u codevaldorg.User) *pb.User {
	return &pb.User{
		AgencyId:    u.AgencyID,
		UserId:      u.ID,
		Email:       u.Email,
		Status:      toProtoUserStatus(u.Status),
		DisplayName: u.DisplayName,
		CreatedAt:   ts(u.CreatedAt),
		UpdatedAt:   ts(u.UpdatedAt),
		DeletedAt:   tsPtr(u.DeletedAt),
	}
}

func toProtoUserStatus(s string) pb.UserStatus {
	switch s {
	case "invited":
		return pb.UserStatus_USER_STATUS_INVITED
	case "active":
		return pb.UserStatus_USER_STATUS_ACTIVE
	case "suspended":
		return pb.UserStatus_USER_STATUS_SUSPENDED
	case "deleted":
		return pb.UserStatus_USER_STATUS_DELETED
	default:
		return pb.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

func toProtoRole(r codevaldorg.Role) *pb.Role {
	return &pb.Role{
		AgencyId:    r.AgencyID,
		RoleId:      r.ID,
		Name:        r.Name,
		Builtin:     r.Builtin,
		DisplayName: r.DisplayName,
		Description: r.Description,
		CreatedAt:   ts(r.CreatedAt),
		UpdatedAt:   ts(r.UpdatedAt),
		DeletedAt:   tsPtr(r.DeletedAt),
	}
}

func toProtoScope(sc codevaldorg.Scope) *pb.Scope {
	return &pb.Scope{
		AgencyId:     sc.AgencyID,
		ScopeId:      sc.ID,
		Name:         sc.Name,
		RegisteredBy: sc.RegisteredBy,
		Description:  sc.Description,
		CreatedAt:    ts(sc.CreatedAt),
		UpdatedAt:    ts(sc.UpdatedAt),
		DeprecatedAt: tsPtr(sc.DeprecatedAt),
	}
}

func toProtoMembership(m codevaldorg.Membership) *pb.Membership {
	return &pb.Membership{
		AgencyId:     m.AgencyID,
		MembershipId: m.ID,
		GrantedBy:    m.GrantedBy,
		GrantedAt:    ts(m.GrantedAt),
		RevokedAt:    tsPtr(m.RevokedAt),
	}
}

func toProtoInvitation(inv codevaldorg.Invitation) *pb.Invitation {
	status := pb.InvitationStatus_INVITATION_STATUS_UNSPECIFIED
	switch inv.Status {
	case "pending":
		status = pb.InvitationStatus_INVITATION_STATUS_PENDING
	case "accepted":
		status = pb.InvitationStatus_INVITATION_STATUS_ACCEPTED
	case "expired":
		status = pb.InvitationStatus_INVITATION_STATUS_EXPIRED
	case "revoked":
		status = pb.InvitationStatus_INVITATION_STATUS_REVOKED
	}
	return &pb.Invitation{
		AgencyId:     inv.AgencyID,
		InvitationId: inv.ID,
		Token:        inv.TokenHash,
		Status:       status,
		ExpiresAt:    ts(inv.ExpiresAt),
		CreatedAt:    ts(inv.CreatedAt),
		AcceptedAt:   tsPtr(inv.AcceptedAt),
		RevokedAt:    tsPtr(inv.RevokedAt),
	}
}

func toProtoOAuthClient(c codevaldorg.OAuthClient) *pb.OAuthClient {
	return &pb.OAuthClient{
		AgencyId:          c.AgencyID,
		ClientId:          c.ClientID,
		ClientType:        c.ClientType,
		Name:              c.Name,
		AllowedGrantTypes: c.AllowedGrantTypes,
		Description:       c.Description,
		CreatedAt:         ts(c.CreatedAt),
		UpdatedAt:         ts(c.UpdatedAt),
		DisabledAt:        tsPtr(c.DisabledAt),
		DeletedAt:         tsPtr(c.DeletedAt),
	}
}

func toProtoAuditEvent(e codevaldorg.AuditEvent) *pb.AuditEvent {
	return &pb.AuditEvent{
		AgencyId:  e.AgencyID,
		EventId:   e.ID,
		EventType: e.EventType,
		ActorId:   e.ActorID,
		SubjectId: e.SubjectID,
		Outcome:   e.Outcome,
		EventAt:   ts(e.EventAt),
		Payload:   e.Payload,
	}
}
