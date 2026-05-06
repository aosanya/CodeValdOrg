package codevaldorg

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// CreateOAuthClient registers a new OAuth client. Returns the client and
// (for confidential clients) the plaintext secret.
func (m *orgManager) CreateOAuthClient(ctx context.Context, req CreateOAuthClientRequest) (OAuthClient, string, error) {
	clientID := uuid.New().String()
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")

	e, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "OAuthClient",
		Properties: map[string]any{
			"agency_id":           m.cfg.AgencyID,
			"client_id":           clientID,
			"client_type":         req.ClientType,
			"name":                req.Name,
			"allowed_grant_types": req.AllowedGrantTypes,
			"description":         req.Description,
			"created_at":          now,
			"updated_at":          now,
		},
	})
	if err != nil {
		return OAuthClient{}, "", ErrTemporarilyUnavailable
	}

	// RedirectURI entities
	for _, uri := range req.RedirectURIs {
		uriE, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
			AgencyID: m.cfg.AgencyID,
			TypeID:   "RedirectURI",
			Properties: map[string]any{
				"agency_id":  m.cfg.AgencyID,
				"uri":        uri,
				"created_at": now,
			},
		})
		if err != nil {
			return OAuthClient{}, "", ErrTemporarilyUnavailable
		}
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "has_redirect_uri",
			FromID:   e.ID,
			ToID:     uriE.ID,
		}); err != nil {
			return OAuthClient{}, "", ErrTemporarilyUnavailable
		}
	}

	// allows_scope edges
	for _, scopeID := range req.AllowedScopeIDs {
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "allows_scope",
			FromID:   e.ID,
			ToID:     scopeID,
		}); err != nil {
			return OAuthClient{}, "", ErrTemporarilyUnavailable
		}
	}

	client := entityToOAuthClient(e)
	var plaintextSecret string

	if req.ClientType == "confidential" {
		pt, secretHash, err := m.createClientSecret(ctx, e.ID, now)
		if err != nil {
			return OAuthClient{}, "", err
		}
		plaintextSecret = pt
		_ = secretHash
	}

	return client, plaintextSecret, nil
}

func (m *orgManager) createClientSecret(ctx context.Context, clientEntityID, now string) (plaintext, secretHash string, err error) {
	plaintext = fmt.Sprintf("cv_cs_%s", uuid.New().String())
	var phc string
	phc, err = hashArgon2id(plaintext, m.cfg.Argon2Time, m.cfg.Argon2MemoryKiB, m.cfg.Argon2Threads)
	if err != nil {
		return "", "", ErrTemporarilyUnavailable
	}
	secretE, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "ClientSecret",
		Properties: map[string]any{
			"agency_id":   m.cfg.AgencyID,
			"secret_hash": phc,
			"created_at":  now,
		},
	})
	if err != nil {
		return "", "", ErrTemporarilyUnavailable
	}
	if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_client_secret",
		FromID:   clientEntityID,
		ToID:     secretE.ID,
	}); err != nil {
		return "", "", ErrTemporarilyUnavailable
	}
	return plaintext, phc, nil
}

// RotateClientSecret creates a new ClientSecret and returns the plaintext.
func (m *orgManager) RotateClientSecret(ctx context.Context, clientID string) (string, error) {
	clients, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "OAuthClient",
		Properties: map[string]any{"client_id": clientID},
	})
	if err != nil || len(clients) == 0 {
		return "", ErrOAuthClientNotFound
	}
	now := m.clock.Now().UTC().Format("2006-01-02T15:04:05Z")
	plaintext, _, err := m.createClientSecret(ctx, clients[0].ID, now)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

// ListOAuthClients returns all OAuthClient entities for the agency.
func (m *orgManager) ListOAuthClients(ctx context.Context, req ListRequest) ([]OAuthClient, error) {
	entities, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "OAuthClient",
	})
	if err != nil {
		return nil, ErrTemporarilyUnavailable
	}
	clients := make([]OAuthClient, len(entities))
	for i, e := range entities {
		clients[i] = entityToOAuthClient(e)
	}
	return clients, nil
}

// DeleteOAuthClient soft-deletes the OAuthClient entity.
func (m *orgManager) DeleteOAuthClient(ctx context.Context, clientID string) error {
	clients, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "OAuthClient",
		Properties: map[string]any{"client_id": clientID},
	})
	if err != nil || len(clients) == 0 {
		return ErrOAuthClientNotFound
	}
	if err := m.dm.DeleteEntity(ctx, m.cfg.AgencyID, clients[0].ID); err != nil {
		return ErrTemporarilyUnavailable
	}
	return nil
}
