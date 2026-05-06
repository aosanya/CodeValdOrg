package codevaldorg

import (
	"context"
	"regexp"
	"strings"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

var scopeGrammar = regexp.MustCompile(`^[a-z0-9_]{1,20}:[a-z0-9_]{1,28}$`)

var reservedPrefixes = []string{"org", "audit"}

// CreateRole creates a custom (non-builtin) Role.
func (m *orgManager) CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error) {
	existing, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "Role",
		Properties: map[string]any{"name": req.Name},
	})
	if err != nil {
		return Role{}, ErrTemporarilyUnavailable
	}
	if len(existing) > 0 {
		return Role{}, ErrRoleAlreadyExists
	}

	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	e, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Role",
		Properties: map[string]any{
			"agency_id":    m.cfg.AgencyID,
			"name":         req.Name,
			"builtin":      false,
			"display_name": req.DisplayName,
			"description":  req.Description,
			"created_at":   now,
			"updated_at":   now,
		},
	})
	if err != nil {
		return Role{}, ErrTemporarilyUnavailable
	}
	return entityToRole(e), nil
}

// UpdateRole patches a custom Role's display_name and description.
func (m *orgManager) UpdateRole(ctx context.Context, roleID string, req UpdateRoleRequest) (Role, error) {
	existing, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, roleID)
	if err != nil {
		return Role{}, ErrRoleNotFound
	}
	if propBool(existing.Properties, "builtin") {
		return Role{}, ErrRoleBuiltinImmutable
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	props := map[string]any{"updated_at": now}
	if req.DisplayName != "" {
		props["display_name"] = req.DisplayName
	}
	if req.Description != "" {
		props["description"] = req.Description
	}
	e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, roleID, entitygraph.UpdateEntityRequest{
		Properties: props,
	})
	if err != nil {
		return Role{}, ErrTemporarilyUnavailable
	}
	return entityToRole(e), nil
}

// DeleteRole soft-deletes a custom Role. Returns ErrRoleBuiltinImmutable for built-ins.
func (m *orgManager) DeleteRole(ctx context.Context, roleID string) error {
	existing, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, roleID)
	if err != nil {
		return ErrRoleNotFound
	}
	if propBool(existing.Properties, "builtin") {
		return ErrRoleBuiltinImmutable
	}
	if err := m.dm.DeleteEntity(ctx, m.cfg.AgencyID, roleID); err != nil {
		return ErrTemporarilyUnavailable
	}
	return nil
}

// ListRoles returns all Roles for the agency.
func (m *orgManager) ListRoles(ctx context.Context, req ListRequest) ([]Role, error) {
	entities, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Role",
	})
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}
	roles := make([]Role, len(entities))
	for i, e := range entities {
		roles[i] = entityToRole(e)
	}
	return roles, nil
}

// RegisterScope registers or idempotently updates a Scope.
// Returns ErrScopeReserved for reserved prefixes from non-org callers,
// ErrScopeNameCollision when the scope exists with a different registered_by.
func (m *orgManager) RegisterScope(ctx context.Context, req RegisterScopeRequest) (Scope, error) {
	if !scopeGrammar.MatchString(req.Name) {
		return Scope{}, ErrInvalidScope
	}
	parts := strings.SplitN(req.Name, ":", 2)
	prefix := parts[0]
	for _, rp := range reservedPrefixes {
		if prefix == rp && req.RegisteredBy != "codevaldorg" {
			return Scope{}, ErrScopeReserved
		}
	}

	existing, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "Scope",
		Properties: map[string]any{"name": req.Name},
	})
	if err != nil {
		return Scope{}, ErrTemporarilyUnavailable
	}

	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")

	if len(existing) > 0 {
		s := entityToScope(existing[0])
		if s.RegisteredBy != req.RegisteredBy {
			return Scope{}, ErrScopeNameCollision
		}
		// Idempotent update: refresh description and updated_at; clear deprecated_at.
		e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, existing[0].ID, entitygraph.UpdateEntityRequest{
			Properties: map[string]any{
				"description":   req.Description,
				"updated_at":    now,
				"deprecated_at": "",
			},
		})
		if err != nil {
			return Scope{}, ErrTemporarilyUnavailable
		}
		return entityToScope(e), nil
	}

	e, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Scope",
		Properties: map[string]any{
			"agency_id":     m.cfg.AgencyID,
			"name":          req.Name,
			"registered_by": req.RegisteredBy,
			"description":   req.Description,
			"created_at":    now,
			"updated_at":    now,
		},
	})
	if err != nil {
		return Scope{}, ErrTemporarilyUnavailable
	}
	return entityToScope(e), nil
}

// DeprecateScope sets deprecated_at on the Scope.
func (m *orgManager) DeprecateScope(ctx context.Context, scopeID string) (Scope, error) {
	if _, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, scopeID); err != nil {
		return Scope{}, ErrScopeNotFound
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	e, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, scopeID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{
			"deprecated_at": now,
			"updated_at":    now,
		},
	})
	if err != nil {
		return Scope{}, ErrTemporarilyUnavailable
	}
	return entityToScope(e), nil
}

// ListScopes returns all Scopes for the agency.
func (m *orgManager) ListScopes(ctx context.Context, req ListRequest) ([]Scope, error) {
	entities, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "Scope",
	})
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}
	scopes := make([]Scope, len(entities))
	for i, e := range entities {
		scopes[i] = entityToScope(e)
	}
	return scopes, nil
}
