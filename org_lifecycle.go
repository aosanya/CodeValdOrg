package codevaldorg

import (
	"context"
	"fmt"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// InitOrganization creates the Organization entity and seeds four built-in roles.
// Returns ErrOrgAlreadyExists if an organization already exists for this agency.
func (m *orgManager) InitOrganization(ctx context.Context, req InitOrganizationRequest) (Organization, error) {
	existing, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Organization",
	})
	if err != nil {
		return Organization{}, ErrTemporarilyUnavailable
	}
	if len(existing) > 0 {
		return Organization{}, ErrOrgAlreadyExists
	}

	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	e, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Organization",
		Properties: map[string]any{
			"agency_id":     m.cfg.AgencyID,
			"name":          req.Name,
			"enabled":       true,
			"description":   req.Description,
			"contact_email": req.ContactEmail,
			"logo_url":      req.LogoURL,
			"created_at":    now,
			"updated_at":    now,
		},
	})
	if err != nil {
		return Organization{}, ErrTemporarilyUnavailable
	}

	if seedErr := m.seedBuiltinRoles(ctx, now); seedErr != nil {
		return Organization{}, seedErr
	}

	return entityToOrg(e), nil
}

// seedBuiltinRoles creates the four built-in roles idempotently.
func (m *orgManager) seedBuiltinRoles(ctx context.Context, now string) error {
	builtins := []struct {
		name        string
		displayName string
	}{
		{"super_admin", "Super Admin"},
		{"admin", "Admin"},
		{"member", "Member"},
		{"viewer", "Viewer"},
	}
	for _, b := range builtins {
		existing, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
			AgencyID: m.cfg.AgencyID,
			TypeID:   "Role",
			Properties: map[string]any{
				"name": b.name,
			},
		})
		if err != nil {
			return ErrTemporarilyUnavailable
		}
		if len(existing) > 0 {
			continue
		}
		_, err = m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
			AgencyID: m.cfg.AgencyID,
			TypeID:   "Role",
			Properties: map[string]any{
				"agency_id":    m.cfg.AgencyID,
				"name":         b.name,
				"builtin":      true,
				"display_name": b.displayName,
				"created_at":   now,
				"updated_at":   now,
			},
		})
		if err != nil {
			return ErrTemporarilyUnavailable
		}
	}
	return nil
}

// GetOrganization returns the single Organization for this agency.
func (m *orgManager) GetOrganization(ctx context.Context) (Organization, error) {
	entities, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Organization",
	})
	if err != nil {
		return Organization{}, ErrTemporarilyUnavailable
	}
	if len(entities) == 0 {
		return Organization{}, ErrOrgNotFound
	}
	return entityToOrg(entities[0]), nil
}

// UpdateOrganization patches mutable Organization fields.
func (m *orgManager) UpdateOrganization(ctx context.Context, req UpdateOrganizationRequest) (Organization, error) {
	org, err := m.GetOrganization(ctx)
	if err != nil {
		return Organization{}, err
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	props := map[string]any{"updated_at": now}
	if req.Name != "" {
		props["name"] = req.Name
	}
	if req.Description != "" {
		props["description"] = req.Description
	}
	if req.ContactEmail != "" {
		props["contact_email"] = req.ContactEmail
	}
	if req.LogoURL != "" {
		props["logo_url"] = req.LogoURL
	}
	e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, org.ID, entitygraph.UpdateEntityRequest{
		Properties: props,
	})
	if err != nil {
		return Organization{}, ErrTemporarilyUnavailable
	}
	return entityToOrg(e), nil
}

// DisableOrganization sets enabled=false on the Organization.
func (m *orgManager) DisableOrganization(ctx context.Context) (Organization, error) {
	org, err := m.GetOrganization(ctx)
	if err != nil {
		return Organization{}, err
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, org.ID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"enabled":     false,
			"disabled_at": now,
			"updated_at":  now,
		},
	})
	if err != nil {
		return Organization{}, ErrTemporarilyUnavailable
	}
	return entityToOrg(e), nil
}

// DeleteOrganization soft-deletes the Organization.
func (m *orgManager) DeleteOrganization(ctx context.Context) error {
	org, err := m.GetOrganization(ctx)
	if err != nil {
		return err
	}
	if err := m.dm.DeleteEntity(ctx, m.cfg.AgencyID, org.ID); err != nil {
		return ErrTemporarilyUnavailable
	}
	return nil
}

// InviteUser creates a User in "invited" status and an Invitation entity.
func (m *orgManager) InviteUser(ctx context.Context, req InviteUserRequest) (Invitation, error) {
	// Check for duplicate email.
	existing, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "User",
		Properties: map[string]any{"email": req.Email},
	})
	if err != nil {
		return Invitation{}, ErrTemporarilyUnavailable
	}
	if len(existing) > 0 {
		return Invitation{}, ErrUserAlreadyExists
	}

	now := m.clock.Now().UTC()
	nowStr := now.Format("2006-01-02T15:04:05Z")

	userE, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "User",
		Properties: map[string]any{
			"agency_id":    m.cfg.AgencyID,
			"email":        req.Email,
			"status":       "invited",
			"display_name": req.DisplayName,
			"created_at":   nowStr,
			"updated_at":   nowStr,
		},
	})
	if err != nil {
		return Invitation{}, ErrTemporarilyUnavailable
	}

	plaintoken, tokenHash, err := generateToken("cv_iv_")
	if err != nil {
		return Invitation{}, ErrTemporarilyUnavailable
	}

	ttl := req.ExpiresIn
	if ttl <= 0 {
		ttl = 72 * 60 * 60 * 1e9 // 72h default
	}
	expiresAt := now.Add(ttl).Format("2006-01-02T15:04:05Z")

	invE, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Invitation",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"token_hash": tokenHash,
			"status":     "pending",
			"expires_at": expiresAt,
			"created_at": nowStr,
		},
	})
	if err != nil {
		return Invitation{}, ErrTemporarilyUnavailable
	}

	// Edge: User → has_invitation → Invitation
	if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_invitation",
		FromID:   userE.ID,
		ToID:     invE.ID,
	}); err != nil {
		return Invitation{}, ErrTemporarilyUnavailable
	}

	// Edge: Invitation → invited_by → inviting User (if provided)
	if req.InvitedByUserID != "" {
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "invited_by",
			FromID:   invE.ID,
			ToID:     req.InvitedByUserID,
		}); err != nil {
			return Invitation{}, ErrTemporarilyUnavailable
		}
	}

	// Edges: Invitation → will_grant_role → Role (for each role ID)
	for _, roleID := range req.RoleIDs {
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "will_grant_role",
			FromID:   invE.ID,
			ToID:     roleID,
		}); err != nil {
			return Invitation{}, ErrTemporarilyUnavailable
		}
	}

	inv := entityToInvitation(invE)
	// Surface plaintext token to caller (not stored).
	inv.TokenHash = plaintoken
	return inv, nil
}

// AcceptInvitation validates and accepts the invitation token, activating the User.
func (m *orgManager) AcceptInvitation(ctx context.Context, token string) (User, error) {
	tokenHash := hashSHA256(token)
	invitations, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "Invitation",
		Properties: map[string]any{"token_hash": tokenHash},
	})
	if err != nil {
		return User{}, ErrTemporarilyUnavailable
	}
	if len(invitations) == 0 {
		return User{}, ErrInvitationNotFound
	}
	inv := entityToInvitation(invitations[0])

	if inv.Status == "accepted" {
		return User{}, ErrInvitationAlreadyAccepted
	}
	if m.clock.Now().After(inv.ExpiresAt) || inv.Status == "expired" {
		return User{}, ErrInvitationExpired
	}

	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Find the User linked to this Invitation.
	// The Invitation is linked FROM the User via has_invitation.
	rels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_invitation",
		ToID:     invitations[0].ID,
	})
	if err != nil || len(rels) == 0 {
		return User{}, ErrTemporarilyUnavailable
	}
	userID := rels[0].FromID

	userE, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, userID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"status":     "active",
			"updated_at": now,
		},
	})
	if err != nil {
		return User{}, ErrTemporarilyUnavailable
	}

	if _, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, invitations[0].ID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"status":      "accepted",
			"accepted_at": now,
		},
	}); err != nil {
		return User{}, ErrTemporarilyUnavailable
	}

	return entityToUser(userE), nil
}

// GetUser returns a User by ID.
func (m *orgManager) GetUser(ctx context.Context, userID string) (User, error) {
	e, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, userID)
	if err != nil {
		return User{}, fmt.Errorf("%w: %w", ErrUserNotFound, err)
	}
	if e.TypeID != "User" {
		return User{}, ErrUserNotFound
	}
	return entityToUser(e), nil
}

// ListUsers returns paginated Users, optionally filtered by status.
func (m *orgManager) ListUsers(ctx context.Context, req ListUsersRequest) ([]User, error) {
	filter := entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "User",
	}
	if req.StatusFilter != "" {
		filter.Properties = map[string]any{"status": req.StatusFilter}
	}
	entities, err := m.dm.ListEntities(ctx, filter)
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}
	users := make([]User, len(entities))
	for i, e := range entities {
		users[i] = entityToUser(e)
	}
	return users, nil
}

// SuspendUser sets User.status = "suspended".
func (m *orgManager) SuspendUser(ctx context.Context, userID string) (User, error) {
	if _, err := m.GetUser(ctx, userID); err != nil {
		return User{}, err
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, userID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"status":     "suspended",
			"updated_at": now,
		},
	})
	if err != nil {
		return User{}, ErrTemporarilyUnavailable
	}
	return entityToUser(e), nil
}

// ReactivateUser sets User.status = "active".
func (m *orgManager) ReactivateUser(ctx context.Context, userID string) (User, error) {
	if _, err := m.GetUser(ctx, userID); err != nil {
		return User{}, err
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, userID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"status":     "active",
			"updated_at": now,
		},
	})
	if err != nil {
		return User{}, ErrTemporarilyUnavailable
	}
	return entityToUser(e), nil
}

// DeleteUser soft-deletes a User (sets status="deleted", deleted_at=now).
func (m *orgManager) DeleteUser(ctx context.Context, userID string) error {
	if _, err := m.GetUser(ctx, userID); err != nil {
		return err
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	if _, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, userID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"status":     "deleted",
			"deleted_at": now,
			"updated_at": now,
		},
	}); err != nil {
		return ErrTemporarilyUnavailable
	}
	return nil
}
