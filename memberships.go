package codevaldorg

import (
	"context"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// GrantMembership creates a Membership entity and edges to User and Role.
func (m *orgManager) GrantMembership(ctx context.Context, req GrantMembershipRequest) (Membership, error) {
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	memE, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Membership",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"granted_at": now,
			"granted_by": req.GrantedBy,
		},
	})
	if err != nil {
		return Membership{}, ErrTemporarilyUnavailable
	}

	// User → has_membership → Membership
	if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_membership",
		FromID:   req.UserID,
		ToID:     memE.ID,
	}); err != nil {
		return Membership{}, ErrTemporarilyUnavailable
	}

	// Membership → grants_role → Role
	if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		AgencyID: m.cfg.AgencyID,
		Name:     "grants_role",
		FromID:   memE.ID,
		ToID:     req.RoleID,
	}); err != nil {
		return Membership{}, ErrTemporarilyUnavailable
	}

	mem := entityToMembership(memE)
	mem.UserID = req.UserID
	mem.RoleID = req.RoleID
	return mem, nil
}

// RevokeMembership sets revoked_at on the Membership entity.
func (m *orgManager) RevokeMembership(ctx context.Context, membershipID string) (Membership, error) {
	if _, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, membershipID); err != nil {
		return Membership{}, ErrMembershipNotFound
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, membershipID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"revoked_at": now,
		},
	})
	if err != nil {
		return Membership{}, ErrTemporarilyUnavailable
	}
	return entityToMembership(e), nil
}

// ListMemberships returns Memberships, optionally filtered by user_id.
func (m *orgManager) ListMemberships(ctx context.Context, req ListMembershipsRequest) ([]Membership, error) {
	filter := entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Membership",
	}
	entities, err := m.dm.ListEntities(ctx, filter)
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}

	// If UserID filter is set, keep only memberships reachable from that user.
	if req.UserID != "" {
		rels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
			AgencyID: m.cfg.AgencyID,
			Name:     "has_membership",
			FromID:   req.UserID,
		})
		if err != nil {
			return nil, ErrTemporarilyUnavailable
		}
		membershipIDs := make(map[string]struct{}, len(rels))
		for _, r := range rels {
			membershipIDs[r.ToID] = struct{}{}
		}
		filtered := entities[:0]
		for _, e := range entities {
			if _, ok := membershipIDs[e.ID]; ok {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}

	// Build user→membership and membership→role maps from relationships.
	userRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_membership",
	})
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}
	roleRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "grants_role",
	})
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}
	membershipUser := make(map[string]string, len(userRels))
	for _, r := range userRels {
		membershipUser[r.ToID] = r.FromID
	}
	membershipRole := make(map[string]string, len(roleRels))
	for _, r := range roleRels {
		membershipRole[r.FromID] = r.ToID
	}

	memberships := make([]Membership, len(entities))
	for i, e := range entities {
		mem := entityToMembership(e)
		mem.UserID = membershipUser[e.ID]
		mem.RoleID = membershipRole[e.ID]
		memberships[i] = mem
	}
	return memberships, nil
}
