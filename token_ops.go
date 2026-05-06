package codevaldorg

import (
	"context"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// Authorize is a stub for the Authorization Code + PKCE flow.
// Full implementation is ORG-010 (blocked on ORG-012/013).
func (m *orgManager) Authorize(ctx context.Context, req AuthorizeRequest) (string, string, error) {
	return "", "", ErrTemporarilyUnavailable
}

// Token is a stub for the multi-grant token flow.
// Full implementation is ORG-010 (blocked on ORG-012/013).
func (m *orgManager) Token(ctx context.Context, req TokenRequest) (TokenResult, error) {
	return TokenResult{}, ErrTemporarilyUnavailable
}

// Introspect looks up a token hash in org_oauth_artifacts, checks for
// revocation, and returns active=false for unknown/expired/revoked tokens.
func (m *orgManager) Introspect(ctx context.Context, token string) (IntrospectResult, error) {
	tokenHash := hashSHA256(token)

	// Check revocation list first.
	revocations, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "TokenRevocation",
		Properties: map[string]any{"token_hash": tokenHash},
	})
	if err != nil {
		return IntrospectResult{Active: false}, nil
	}
	if len(revocations) > 0 {
		return IntrospectResult{Active: false}, nil
	}

	// Look up AccessToken.
	tokens, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "AccessToken",
		Properties: map[string]any{"token_hash": tokenHash},
	})
	if err != nil || len(tokens) == 0 {
		return IntrospectResult{Active: false}, nil
	}

	t := tokens[0]
	expiresAt := propTime(t.Properties, "expires_at")
	if !expiresAt.IsZero() && m.clock.Now().After(expiresAt) {
		return IntrospectResult{Active: false}, nil
	}

	result := IntrospectResult{
		Active:    true,
		TokenType: "Bearer",
		Exp:       expiresAt,
		Iat:       propTime(t.Properties, "created_at"),
	}
	return result, nil
}

// Revoke creates a TokenRevocation entity for the given token.
func (m *orgManager) Revoke(ctx context.Context, token, reason string) error {
	tokenHash := hashSHA256(token)
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Try to find the token to get its expires_at.
	var expiresAt string
	for _, typeID := range []string{"AccessToken", "RefreshToken"} {
		tokens, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
			AgencyID:   m.cfg.AgencyID,
			TypeID:     typeID,
			Properties: map[string]any{"token_hash": tokenHash},
		})
		if err == nil && len(tokens) > 0 {
			expiresAt = propStr(tokens[0].Properties, "expires_at")
			break
		}
	}
	if expiresAt == "" {
		expiresAt = now
	}

	if reason == "" {
		reason = "user_logout"
	}

	_, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "TokenRevocation",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"token_hash": tokenHash,
			"revoked_at": now,
			"expires_at": expiresAt,
			"reason":     reason,
		},
	})
	if err != nil {
		return ErrTemporarilyUnavailable
	}
	return nil
}

// ListAuditEvents returns AuditEvent entities with optional filters.
func (m *orgManager) ListAuditEvents(ctx context.Context, filter AuditEventFilter) ([]AuditEvent, error) {
	ef := entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "AuditEvent",
	}
	if filter.EventTypeFilter != "" || filter.ActorIDFilter != "" || filter.SubjectIDFilter != "" {
		ef.Properties = make(map[string]any)
		if filter.EventTypeFilter != "" {
			ef.Properties["event_type"] = filter.EventTypeFilter
		}
		if filter.ActorIDFilter != "" {
			ef.Properties["actor_id"] = filter.ActorIDFilter
		}
		if filter.SubjectIDFilter != "" {
			ef.Properties["subject_id"] = filter.SubjectIDFilter
		}
	}
	entities, err := m.dm.ListEntities(ctx, ef)
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}
	events := make([]AuditEvent, len(entities))
	for i, e := range entities {
		events[i] = entityToAuditEvent(e)
	}
	return events, nil
}
