package codevaldorg

import (
	"time"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// propStr reads a string property from an entity's Properties map.
func propStr(props map[string]any, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// propBool reads a boolean property from an entity's Properties map.
func propBool(props map[string]any, key string) bool {
	if v, ok := props[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// propTime parses a RFC 3339 string property; returns zero value on failure.
func propTime(props map[string]any, key string) time.Time {
	s := propStr(props, key)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// propTimePtr parses a RFC 3339 string property; returns nil on absence or failure.
func propTimePtr(props map[string]any, key string) *time.Time {
	s := propStr(props, key)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// propStrSlice reads a []string array property.
func propStrSlice(props map[string]any, key string) []string {
	v, ok := props[key]
	if !ok {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// ── Entity → domain object converters ────────────────────────────────────────

func entityToOrg(e entitygraph.Entity) Organization {
	p := e.Properties
	return Organization{
		ID:           e.ID,
		AgencyID:     propStr(p, "agency_id"),
		Name:         propStr(p, "name"),
		Enabled:      propBool(p, "enabled"),
		Description:  propStr(p, "description"),
		ContactEmail: propStr(p, "contact_email"),
		LogoURL:      propStr(p, "logo_url"),
		CreatedAt:    propTime(p, "created_at"),
		UpdatedAt:    propTime(p, "updated_at"),
		DisabledAt:   propTimePtr(p, "disabled_at"),
		DeletedAt:    propTimePtr(p, "deleted_at"),
	}
}

func entityToUser(e entitygraph.Entity) User {
	p := e.Properties
	return User{
		ID:          e.ID,
		AgencyID:    propStr(p, "agency_id"),
		Email:       propStr(p, "email"),
		Status:      propStr(p, "status"),
		DisplayName: propStr(p, "display_name"),
		CreatedAt:   propTime(p, "created_at"),
		UpdatedAt:   propTime(p, "updated_at"),
		DeletedAt:   propTimePtr(p, "deleted_at"),
	}
}

func entityToRole(e entitygraph.Entity) Role {
	p := e.Properties
	return Role{
		ID:          e.ID,
		AgencyID:    propStr(p, "agency_id"),
		Name:        propStr(p, "name"),
		Builtin:     propBool(p, "builtin"),
		DisplayName: propStr(p, "display_name"),
		Description: propStr(p, "description"),
		CreatedAt:   propTime(p, "created_at"),
		UpdatedAt:   propTime(p, "updated_at"),
		DeletedAt:   propTimePtr(p, "deleted_at"),
	}
}

func entityToScope(e entitygraph.Entity) Scope {
	p := e.Properties
	return Scope{
		ID:           e.ID,
		AgencyID:     propStr(p, "agency_id"),
		Name:         propStr(p, "name"),
		RegisteredBy: propStr(p, "registered_by"),
		Description:  propStr(p, "description"),
		CreatedAt:    propTime(p, "created_at"),
		UpdatedAt:    propTime(p, "updated_at"),
		DeprecatedAt: propTimePtr(p, "deprecated_at"),
	}
}

func entityToMembership(e entitygraph.Entity) Membership {
	p := e.Properties
	return Membership{
		ID:        e.ID,
		AgencyID:  propStr(p, "agency_id"),
		GrantedAt: propTime(p, "granted_at"),
		GrantedBy: propStr(p, "granted_by"),
		RevokedAt: propTimePtr(p, "revoked_at"),
		RevokedBy: propStr(p, "revoked_by"),
	}
}

func entityToInvitation(e entitygraph.Entity) Invitation {
	p := e.Properties
	return Invitation{
		ID:         e.ID,
		AgencyID:   propStr(p, "agency_id"),
		TokenHash:  propStr(p, "token_hash"),
		Status:     propStr(p, "status"),
		ExpiresAt:  propTime(p, "expires_at"),
		CreatedAt:  propTime(p, "created_at"),
		AcceptedAt: propTimePtr(p, "accepted_at"),
		RevokedAt:  propTimePtr(p, "revoked_at"),
	}
}

func entityToOAuthClient(e entitygraph.Entity) OAuthClient {
	p := e.Properties
	return OAuthClient{
		ID:                e.ID,
		AgencyID:          propStr(p, "agency_id"),
		ClientID:          propStr(p, "client_id"),
		ClientType:        propStr(p, "client_type"),
		Name:              propStr(p, "name"),
		AllowedGrantTypes: propStrSlice(p, "allowed_grant_types"),
		Description:       propStr(p, "description"),
		CreatedAt:         propTime(p, "created_at"),
		UpdatedAt:         propTime(p, "updated_at"),
		DisabledAt:        propTimePtr(p, "disabled_at"),
		DeletedAt:         propTimePtr(p, "deleted_at"),
	}
}

func entityToAuditEvent(e entitygraph.Entity) AuditEvent {
	p := e.Properties
	return AuditEvent{
		ID:        e.ID,
		AgencyID:  propStr(p, "agency_id"),
		EventType: propStr(p, "event_type"),
		ActorID:   propStr(p, "actor_id"),
		SubjectID: propStr(p, "subject_id"),
		Outcome:   propStr(p, "outcome"),
		EventAt:   propTime(p, "event_at"),
		Payload:   propStr(p, "payload"),
	}
}
