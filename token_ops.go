package codevaldorg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// Authorize validates PKCE params, creates an AuthorizationCode entity, and
// returns the plaintext code and echoed state. The caller must have already
// authenticated the user; user_id links the code to the user.
func (m *orgManager) Authorize(ctx context.Context, req AuthorizeRequest) (string, string, error) {
	// 1. Look up the OAuthClient by client_id.
	clients, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "OAuthClient",
		Properties: map[string]any{"client_id": req.ClientID},
	})
	if err != nil || len(clients) == 0 {
		return "", "", ErrInvalidClient
	}
	clientEnt := clients[0]
	client := entityToOAuthClient(clientEnt)
	if client.DeletedAt != nil || client.DisabledAt != nil {
		return "", "", ErrInvalidClient
	}

	// 2. Validate redirect_uri against registered RedirectURI entities.
	redirectRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_redirect_uri",
		FromID:   clientEnt.ID,
	})
	if err != nil {
		return "", "", ErrTemporarilyUnavailable
	}
	var validRedirect bool
	for _, rel := range redirectRels {
		uriEnt, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, rel.ToID)
		if err != nil {
			continue
		}
		if propStr(uriEnt.Properties, "uri") == req.RedirectURI && propStr(uriEnt.Properties, "revoked_at") == "" {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		return "", "", ErrRedirectURIMismatch
	}

	// 3. PKCE requirements.
	if client.ClientType == "public" && req.CodeChallenge == "" {
		return "", "", ErrPKCERequired
	}
	if req.CodeChallenge != "" && req.CodeChallengeMethod != "S256" {
		return "", "", ErrPKCEMethodInvalid
	}

	// 4. Resolve requested scope names to entity IDs.
	scopeIDs, err := m.resolveScopeNames(ctx, req.Scopes)
	if err != nil {
		return "", "", err
	}

	// 5. Generate auth code.
	code, codeHash, err := generateToken("cv_ac_")
	if err != nil {
		return "", "", ErrTemporarilyUnavailable
	}

	now := m.clock.Now().UTC()
	ttl := m.cfg.AuthCodeTTL
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	expiresAt := now.Add(ttl).UTC().Format(time.RFC3339)
	nowStr := now.Format(time.RFC3339)

	// 6. Persist AuthorizationCode entity.
	codeEnt, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "AuthorizationCode",
		Properties: map[string]any{
			"agency_id":      m.cfg.AgencyID,
			"code_hash":      codeHash,
			"code_challenge": req.CodeChallenge,
			"redirect_uri":   req.RedirectURI,
			"expires_at":     expiresAt,
			"state":          req.State,
			"created_at":     nowStr,
		},
	})
	if err != nil {
		return "", "", ErrTemporarilyUnavailable
	}

	// 7. issued_to → OAuthClient
	if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_to",
		FromID:   codeEnt.ID,
		ToID:     clientEnt.ID,
	}); err != nil {
		return "", "", ErrTemporarilyUnavailable
	}

	// 8. issued_for → User (if user_id provided)
	if req.UserID != "" {
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "issued_for",
			FromID:   codeEnt.ID,
			ToID:     req.UserID,
		}); err != nil {
			return "", "", ErrTemporarilyUnavailable
		}
	}

	// 9. has_requested_scope → Scope (for each resolved scope)
	for _, sid := range scopeIDs {
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "has_requested_scope",
			FromID:   codeEnt.ID,
			ToID:     sid,
		}); err != nil {
			return "", "", ErrTemporarilyUnavailable
		}
	}

	return code, req.State, nil
}

// Token handles authorization_code, client_credentials, and refresh_token grants.
func (m *orgManager) Token(ctx context.Context, req TokenRequest) (TokenResult, error) {
	switch req.GrantType {
	case "authorization_code", "GRANT_TYPE_AUTHORIZATION_CODE":
		return m.tokenAuthCode(ctx, req)
	case "client_credentials", "GRANT_TYPE_CLIENT_CREDENTIALS":
		return m.tokenClientCreds(ctx, req)
	case "refresh_token", "GRANT_TYPE_REFRESH_TOKEN":
		return m.tokenRefresh(ctx, req)
	default:
		return TokenResult{}, ErrUnsupportedGrantType
	}
}

// tokenAuthCode implements the Authorization Code + PKCE token exchange.
func (m *orgManager) tokenAuthCode(ctx context.Context, req TokenRequest) (TokenResult, error) {
	// 1. Validate the auth code format.
	if !strings.HasPrefix(req.Code, "cv_ac_") {
		return TokenResult{}, ErrInvalidGrant
	}

	codeHash := hashSHA256(req.Code)
	codes, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "AuthorizationCode",
		Properties: map[string]any{"code_hash": codeHash},
	})
	if err != nil || len(codes) == 0 {
		return TokenResult{}, ErrInvalidGrant
	}
	codeEnt := codes[0]
	p := codeEnt.Properties

	// 2. Check consumed (replay).
	if propStr(p, "consumed_at") != "" {
		return TokenResult{}, ErrInvalidGrant
	}

	// 3. Check expiry.
	expiresAt := propTime(p, "expires_at")
	if !expiresAt.IsZero() && m.clock.Now().After(expiresAt) {
		return TokenResult{}, ErrInvalidGrant
	}

	// 4. Validate redirect_uri.
	if propStr(p, "redirect_uri") != req.RedirectURI {
		return TokenResult{}, ErrRedirectURIMismatch
	}

	// 5. Validate PKCE.
	challenge := propStr(p, "code_challenge")
	if challenge != "" {
		if req.CodeVerifier == "" || !verifyPKCE(req.CodeVerifier, challenge) {
			return TokenResult{}, ErrPKCEMismatch
		}
	}

	// 6. Get the OAuthClient via issued_to edge.
	issuedToRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_to",
		FromID:   codeEnt.ID,
	})
	if err != nil || len(issuedToRels) == 0 {
		return TokenResult{}, ErrInvalidGrant
	}
	clientEnt, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, issuedToRels[0].ToID)
	if err != nil {
		return TokenResult{}, ErrInvalidGrant
	}
	client := entityToOAuthClient(clientEnt)
	if client.ClientID != req.ClientID {
		return TokenResult{}, ErrInvalidGrant
	}

	// 7. Get user_id via issued_for edge.
	var userID string
	issuedForRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_for",
		FromID:   codeEnt.ID,
	})
	if err == nil && len(issuedForRels) > 0 {
		userID = issuedForRels[0].ToID
	}

	// 8. Compute effective scopes.
	requestedScopeIDs, err := m.getAuthCodeScopes(ctx, codeEnt.ID)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	clientScopeIDs, err := m.getClientScopeIDs(ctx, clientEnt.ID)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	var userScopeIDs []string
	if userID != "" {
		userScopeIDs, err = m.getUserScopeIDs(ctx, userID)
		if err != nil {
			return TokenResult{}, ErrTemporarilyUnavailable
		}
	}
	effectiveScopeIDs := m.intersectScopes(ctx, requestedScopeIDs, userScopeIDs, clientScopeIDs)
	if len(effectiveScopeIDs) == 0 {
		return TokenResult{}, ErrInvalidScope
	}
	effectiveScopeNames, err := m.scopeIDsToNames(ctx, effectiveScopeIDs)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// 9. Mint tokens.
	now := m.clock.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	accessPlain, accessHash, err := generateToken("cv_at_")
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	atExpires := now.Add(m.cfg.AccessTokenTTL).UTC().Format(time.RFC3339)

	atEnt, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "AccessToken",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"token_hash": accessHash,
			"expires_at": atExpires,
			"created_at": nowStr,
		},
	})
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	if err := m.attachTokenEdges(ctx, atEnt.ID, clientEnt.ID, userID, effectiveScopeIDs); err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	var refreshPlain string
	var rtEnt entitygraph.Entity
	wantsRefresh := containsStr(client.AllowedGrantTypes, "refresh_token")
	if wantsRefresh {
		refreshPlain, _, err = generateToken("cv_rt_")
		if err != nil {
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
			return TokenResult{}, ErrTemporarilyUnavailable
		}
		rtHash := hashSHA256(refreshPlain)
		rtExpires := now.Add(m.cfg.RefreshTokenTTL).UTC().Format(time.RFC3339)
		rtEnt, err = m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
			AgencyID: m.cfg.AgencyID,
			TypeID:   "RefreshToken",
			Properties: map[string]any{
				"agency_id":  m.cfg.AgencyID,
				"token_hash": rtHash,
				"expires_at": rtExpires,
				"created_at": nowStr,
			},
		})
		if err != nil {
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
			return TokenResult{}, ErrTemporarilyUnavailable
		}
		if err := m.attachTokenEdges(ctx, rtEnt.ID, clientEnt.ID, userID, effectiveScopeIDs); err != nil {
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, rtEnt.ID)
			return TokenResult{}, ErrTemporarilyUnavailable
		}
	}

	// 10. Mark AuthorizationCode consumed.
	if _, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, codeEnt.ID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{"consumed_at": nowStr},
	}); err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
		if wantsRefresh {
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, rtEnt.ID)
		}
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// 11. Publish (if publisher configured).
	if m.pub != nil {
		payload := m.tokenIssuedPayload(atEnt.ID, client.ClientID, userID, effectiveScopeNames, atExpires)
		if pubErr := m.pub.Publish(ctx, m.cfg.AgencyID, "cross.org."+m.cfg.AgencyID+".token.issued", "codevaldorg", payload); pubErr != nil {
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
			if wantsRefresh {
				_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, rtEnt.ID)
			}
			return TokenResult{}, ErrTemporarilyUnavailable
		}
	}

	m.writeAuditEvent(ctx, "token.issued", client.ClientID, userID, "success",
		fmt.Sprintf(`{"scopes":%s}`, jsonStrSlice(effectiveScopeNames)))

	expiresIn := int32(m.cfg.AccessTokenTTL.Seconds())
	return TokenResult{
		AccessToken:  accessPlain,
		TokenType:    "Bearer",
		RefreshToken: refreshPlain,
		ExpiresIn:    expiresIn,
		Scopes:       effectiveScopeNames,
	}, nil
}

// tokenClientCreds implements the Client Credentials grant.
func (m *orgManager) tokenClientCreds(ctx context.Context, req TokenRequest) (TokenResult, error) {
	// 1. Look up client.
	clients, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "OAuthClient",
		Properties: map[string]any{"client_id": req.ClientID},
	})
	if err != nil || len(clients) == 0 {
		return TokenResult{}, ErrInvalidClient
	}
	clientEnt := clients[0]
	client := entityToOAuthClient(clientEnt)
	if client.DeletedAt != nil || client.DisabledAt != nil {
		return TokenResult{}, ErrInvalidClient
	}
	if client.ClientType != "confidential" {
		return TokenResult{}, ErrInvalidClient
	}
	if !containsStr(client.AllowedGrantTypes, "client_credentials") {
		return TokenResult{}, ErrUnauthorizedClient
	}

	// 2. Verify client_secret.
	if err := m.verifyClientSecret(ctx, clientEnt.ID, req.ClientSecret); err != nil {
		return TokenResult{}, err
	}

	// 3. Resolve requested scopes.
	clientScopeIDs, err := m.getClientScopeIDs(ctx, clientEnt.ID)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	var requestedScopeIDs []string
	if len(req.Scopes) > 0 {
		requestedScopeIDs, err = m.resolveScopeNames(ctx, req.Scopes)
		if err != nil {
			return TokenResult{}, err
		}
	} else {
		requestedScopeIDs = clientScopeIDs
	}
	effectiveScopeIDs := m.intersectScopes(ctx, requestedScopeIDs, nil, clientScopeIDs)
	if len(effectiveScopeIDs) == 0 {
		return TokenResult{}, ErrInvalidScope
	}
	effectiveScopeNames, err := m.scopeIDsToNames(ctx, effectiveScopeIDs)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// 4. Mint AccessToken (no refresh for client_credentials).
	now := m.clock.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	accessPlain, accessHash, err := generateToken("cv_at_")
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	atExpires := now.Add(m.cfg.AccessTokenTTL).UTC().Format(time.RFC3339)

	atEnt, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "AccessToken",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"token_hash": accessHash,
			"expires_at": atExpires,
			"created_at": nowStr,
		},
	})
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// No user for client_credentials.
	if err := m.attachTokenEdges(ctx, atEnt.ID, clientEnt.ID, "", effectiveScopeIDs); err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// 5. Publish.
	if m.pub != nil {
		payload := m.tokenIssuedPayload(atEnt.ID, client.ClientID, "", effectiveScopeNames, atExpires)
		if pubErr := m.pub.Publish(ctx, m.cfg.AgencyID, "cross.org."+m.cfg.AgencyID+".token.issued", "codevaldorg", payload); pubErr != nil {
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, atEnt.ID)
			return TokenResult{}, ErrTemporarilyUnavailable
		}
	}

	m.writeAuditEvent(ctx, "token.issued", client.ClientID, client.ClientID, "success",
		fmt.Sprintf(`{"scopes":%s}`, jsonStrSlice(effectiveScopeNames)))

	return TokenResult{
		AccessToken: accessPlain,
		TokenType:   "Bearer",
		ExpiresIn:   int32(m.cfg.AccessTokenTTL.Seconds()),
		Scopes:      effectiveScopeNames,
	}, nil
}

// tokenRefresh implements the Refresh Token rotation flow.
func (m *orgManager) tokenRefresh(ctx context.Context, req TokenRequest) (TokenResult, error) {
	if !strings.HasPrefix(req.RefreshToken, "cv_rt_") {
		return TokenResult{}, ErrInvalidGrant
	}

	rtHash := hashSHA256(req.RefreshToken)
	rts, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		AgencyID:   m.cfg.AgencyID,
		TypeID:     "RefreshToken",
		Properties: map[string]any{"token_hash": rtHash},
	})
	if err != nil || len(rts) == 0 {
		return TokenResult{}, ErrInvalidGrant
	}
	rtEnt := rts[0]
	p := rtEnt.Properties

	// Check expiry.
	if exp := propTime(p, "expires_at"); !exp.IsZero() && m.clock.Now().After(exp) {
		return TokenResult{}, ErrInvalidGrant
	}

	// Verify client match.
	issuedToRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_to",
		FromID:   rtEnt.ID,
	})
	if err != nil || len(issuedToRels) == 0 {
		return TokenResult{}, ErrInvalidGrant
	}
	clientEnt, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, issuedToRels[0].ToID)
	if err != nil {
		return TokenResult{}, ErrInvalidGrant
	}
	client := entityToOAuthClient(clientEnt)
	if client.ClientID != req.ClientID {
		return TokenResult{}, ErrInvalidGrant
	}

	// Reuse detection: if already consumed, revoke the chain.
	if propStr(p, "consumed_at") != "" {
		_ = m.revokeRefreshChain(ctx, rtEnt.ID)
		return TokenResult{}, ErrInvalidGrant
	}

	// Get user.
	var userID string
	issuedForRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_for",
		FromID:   rtEnt.ID,
	})
	if err == nil && len(issuedForRels) > 0 {
		userID = issuedForRels[0].ToID
	}

	// Effective scope: existing scopes (optionally narrowed by request).
	existingScopeIDs, err := m.getTokenScopeIDs(ctx, rtEnt.ID)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	effectiveScopeIDs := existingScopeIDs
	if len(req.Scopes) > 0 {
		requested, err := m.resolveScopeNames(ctx, req.Scopes)
		if err != nil {
			return TokenResult{}, err
		}
		effectiveScopeIDs = intersectStringSlices(requested, existingScopeIDs)
	}
	// Filter out deprecated.
	effectiveScopeIDs, err = m.filterNonDeprecated(ctx, effectiveScopeIDs)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	if len(effectiveScopeIDs) == 0 {
		return TokenResult{}, ErrInvalidScope
	}
	effectiveScopeNames, err := m.scopeIDsToNames(ctx, effectiveScopeIDs)
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// Mint new pair.
	now := m.clock.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	accessPlain, accessHash, err := generateToken("cv_at_")
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	atExpires := now.Add(m.cfg.AccessTokenTTL).UTC().Format(time.RFC3339)

	newATEnt, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "AccessToken",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"token_hash": accessHash,
			"expires_at": atExpires,
			"created_at": nowStr,
		},
	})
	if err != nil {
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	if err := m.attachTokenEdges(ctx, newATEnt.ID, clientEnt.ID, userID, effectiveScopeIDs); err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newATEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	refreshPlain, rtHash2, err := generateToken("cv_rt_")
	if err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newATEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	rtExpires := now.Add(m.cfg.RefreshTokenTTL).UTC().Format(time.RFC3339)
	newRTEnt, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		AgencyID: m.cfg.AgencyID,
		TypeID:   "RefreshToken",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"token_hash": rtHash2,
			"expires_at": rtExpires,
			"created_at": nowStr,
		},
	})
	if err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newATEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}
	if err := m.attachTokenEdges(ctx, newRTEnt.ID, clientEnt.ID, userID, effectiveScopeIDs); err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newATEnt.ID)
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newRTEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// parent edge: newRT → presented RT
	if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		AgencyID: m.cfg.AgencyID,
		Name:     "parent",
		FromID:   newRTEnt.ID,
		ToID:     rtEnt.ID,
	}); err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newATEnt.ID)
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newRTEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// Mark presented RT consumed.
	if _, err := m.dm.UpdateEntity(ctx, m.cfg.AgencyID, rtEnt.ID, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{"consumed_at": nowStr},
	}); err != nil {
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newATEnt.ID)
		_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newRTEnt.ID)
		return TokenResult{}, ErrTemporarilyUnavailable
	}

	// Publish.
	if m.pub != nil {
		payload := m.tokenIssuedPayload(newATEnt.ID, client.ClientID, userID, effectiveScopeNames, atExpires)
		if pubErr := m.pub.Publish(ctx, m.cfg.AgencyID, "cross.org."+m.cfg.AgencyID+".token.issued", "codevaldorg", payload); pubErr != nil {
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newATEnt.ID)
			_ = m.dm.DeleteEntity(ctx, m.cfg.AgencyID, newRTEnt.ID)
			return TokenResult{}, ErrTemporarilyUnavailable
		}
	}

	m.writeAuditEvent(ctx, "token.refreshed", client.ClientID, userID, "success",
		fmt.Sprintf(`{"scopes":%s}`, jsonStrSlice(effectiveScopeNames)))

	return TokenResult{
		AccessToken:  accessPlain,
		TokenType:    "Bearer",
		RefreshToken: refreshPlain,
		ExpiresIn:    int32(m.cfg.AccessTokenTTL.Seconds()),
		Scopes:       effectiveScopeNames,
	}, nil
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

	// Resolve scopes.
	scopeIDs, _ := m.getTokenScopeIDs(ctx, t.ID)
	scopeNames, _ := m.scopeIDsToNames(ctx, scopeIDs)

	// Resolve user (sub).
	var sub string
	issuedForRels, _ := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_for",
		FromID:   t.ID,
	})
	if len(issuedForRels) > 0 {
		sub = issuedForRels[0].ToID
	}

	// Resolve client_id.
	var clientID string
	issuedToRels, _ := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_to",
		FromID:   t.ID,
	})
	if len(issuedToRels) > 0 {
		if clientEnt, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, issuedToRels[0].ToID); err == nil {
			clientID = propStr(clientEnt.Properties, "client_id")
		}
	}

	return IntrospectResult{
		Active:    true,
		TokenType: "Bearer",
		Scopes:    scopeNames,
		Sub:       sub,
		ClientID:  clientID,
		Exp:       expiresAt,
		Iat:       propTime(t.Properties, "created_at"),
	}, nil
}

// Revoke creates a TokenRevocation entity for the given token.
func (m *orgManager) Revoke(ctx context.Context, token, reason string) error {
	tokenHash := hashSHA256(token)
	now := m.clock.Now().UTC().Format(time.RFC3339)

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

	if m.pub != nil {
		payload := fmt.Sprintf(`{"token_hash":%q,"revoked_at":%q,"reason":%q}`, tokenHash, now, reason)
		_ = m.pub.Publish(ctx, m.cfg.AgencyID, "cross.org."+m.cfg.AgencyID+".token.revoked", "codevaldorg", payload)
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

// ── Token helper methods ──────────────────────────────────────────────────────

// resolveScopeNames resolves scope name strings to Scope entity IDs.
// Unknown scope names return ErrInvalidScope.
func (m *orgManager) resolveScopeNames(ctx context.Context, names []string) ([]string, error) {
	ids := make([]string, 0, len(names))
	for _, name := range names {
		scopes, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
			AgencyID:   m.cfg.AgencyID,
			TypeID:     "Scope",
			Properties: map[string]any{"name": name},
		})
		if err != nil {
			return nil, ErrTemporarilyUnavailable
		}
		if len(scopes) == 0 {
			return nil, ErrInvalidScope
		}
		ids = append(ids, scopes[0].ID)
	}
	return ids, nil
}

// getAuthCodeScopes returns the Scope entity IDs from has_requested_scope edges.
func (m *orgManager) getAuthCodeScopes(ctx context.Context, codeEntID string) ([]string, error) {
	return m.getRelatedIDs(ctx, "has_requested_scope", codeEntID)
}

// getClientScopeIDs returns the Scope entity IDs from allows_scope edges on a client.
func (m *orgManager) getClientScopeIDs(ctx context.Context, clientEntID string) ([]string, error) {
	return m.getRelatedIDs(ctx, "allows_scope", clientEntID)
}

// getUserScopeIDs walks User → has_membership → Membership → grants_role → Role → has_scope → Scope.
func (m *orgManager) getUserScopeIDs(ctx context.Context, userEntID string) ([]string, error) {
	membershipRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_membership",
		FromID:   userEntID,
	})
	if err != nil {
		return nil, err
	}
	scopeSet := map[string]struct{}{}
	for _, memRel := range membershipRels {
		memEnt, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, memRel.ToID)
		if err != nil || propStr(memEnt.Properties, "revoked_at") != "" {
			continue
		}
		roleRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
			AgencyID: m.cfg.AgencyID,
			Name:     "grants_role",
			FromID:   memRel.ToID,
		})
		if err != nil {
			continue
		}
		for _, roleRel := range roleRels {
			scopeRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
				AgencyID: m.cfg.AgencyID,
				Name:     "has_scope",
				FromID:   roleRel.ToID,
			})
			if err != nil {
				continue
			}
			for _, sr := range scopeRels {
				scopeSet[sr.ToID] = struct{}{}
			}
		}
	}
	if len(scopeSet) == 0 {
		// No has_scope edges on any role — no restriction from the user side.
		return nil, nil
	}
	ids := make([]string, 0, len(scopeSet))
	for id := range scopeSet {
		ids = append(ids, id)
	}
	return ids, nil
}

// getTokenScopeIDs returns the Scope entity IDs from has_scope edges on an access/refresh token.
func (m *orgManager) getTokenScopeIDs(ctx context.Context, tokenEntID string) ([]string, error) {
	return m.getRelatedIDs(ctx, "has_scope", tokenEntID)
}

// getRelatedIDs returns all ToIDs for a named relationship from a given entity.
func (m *orgManager) getRelatedIDs(ctx context.Context, relName, fromID string) ([]string, error) {
	rels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     relName,
		FromID:   fromID,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rels))
	for i, r := range rels {
		ids[i] = r.ToID
	}
	return ids, nil
}

// intersectScopes computes the effective scope intersection.
// If userScopeIDs is nil (client_credentials), the user dimension is skipped.
func (m *orgManager) intersectScopes(ctx context.Context, requested, userScopes, clientScopes []string) []string {
	clientSet := toSet(clientScopes)
	var baseSet map[string]struct{}
	if userScopes != nil {
		userSet := toSet(userScopes)
		baseSet = intersectSets(userSet, clientSet)
	} else {
		baseSet = clientSet
	}

	// If no specific scopes requested, default to max allowed.
	if len(requested) == 0 {
		ids := make([]string, 0, len(baseSet))
		for id := range baseSet {
			ids = append(ids, id)
		}
		nonDepr, _ := m.filterNonDeprecated(ctx, ids)
		return nonDepr
	}

	result := make([]string, 0, len(requested))
	for _, id := range requested {
		if _, ok := baseSet[id]; ok {
			result = append(result, id)
		}
	}
	nonDepr, _ := m.filterNonDeprecated(ctx, result)
	return nonDepr
}

// filterNonDeprecated removes deprecated scope IDs from the list.
func (m *orgManager) filterNonDeprecated(ctx context.Context, ids []string) ([]string, error) {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		ent, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, id)
		if err != nil {
			continue
		}
		if propStr(ent.Properties, "deprecated_at") == "" {
			result = append(result, id)
		}
	}
	return result, nil
}

// scopeIDsToNames resolves Scope entity IDs to their name strings.
func (m *orgManager) scopeIDsToNames(ctx context.Context, ids []string) ([]string, error) {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		ent, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, id)
		if err != nil {
			continue
		}
		if name := propStr(ent.Properties, "name"); name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// attachTokenEdges creates issued_to, issued_for (if userID != ""), and has_scope edges
// for an AccessToken or RefreshToken entity.
func (m *orgManager) attachTokenEdges(ctx context.Context, tokenEntID, clientEntID, userID string, scopeIDs []string) error {
	if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		AgencyID: m.cfg.AgencyID,
		Name:     "issued_to",
		FromID:   tokenEntID,
		ToID:     clientEntID,
	}); err != nil {
		return err
	}
	if userID != "" {
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "issued_for",
			FromID:   tokenEntID,
			ToID:     userID,
		}); err != nil {
			return err
		}
	}
	for _, sid := range scopeIDs {
		if _, err := m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
			AgencyID: m.cfg.AgencyID,
			Name:     "has_scope",
			FromID:   tokenEntID,
			ToID:     sid,
		}); err != nil {
			return err
		}
	}
	return nil
}

// verifyClientSecret checks the plaintext secret against all active ClientSecret
// entities for the given client (including those in grace period).
func (m *orgManager) verifyClientSecret(ctx context.Context, clientEntID, plaintext string) error {
	secretRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		AgencyID: m.cfg.AgencyID,
		Name:     "has_client_secret",
		FromID:   clientEntID,
	})
	if err != nil {
		return ErrTemporarilyUnavailable
	}
	now := m.clock.Now()
	for _, rel := range secretRels {
		secretEnt, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, rel.ToID)
		if err != nil {
			continue
		}
		p := secretEnt.Properties
		// Skip revoked secrets.
		if propStr(p, "revoked_at") != "" {
			continue
		}
		// Accept if not past grace expiry.
		graceExp := propTime(p, "grace_expires_at")
		if !graceExp.IsZero() && now.After(graceExp) {
			continue
		}
		phc := propStr(p, "secret_hash")
		ok, err := verifyArgon2id(plaintext, phc)
		if err == nil && ok {
			return nil
		}
	}
	return ErrInvalidClient
}

// revokeRefreshChain revokes all tokens in the refresh token chain (reuse detection).
func (m *orgManager) revokeRefreshChain(ctx context.Context, rtEntID string) error {
	now := m.clock.Now().UTC().Format(time.RFC3339)
	// Walk up to root via parent edges.
	current := rtEntID
	seen := map[string]struct{}{}
	for current != "" {
		if _, ok := seen[current]; ok {
			break
		}
		seen[current] = struct{}{}
		rtEnt, err := m.dm.GetEntity(ctx, m.cfg.AgencyID, current)
		if err != nil {
			break
		}
		exp := propStr(rtEnt.Properties, "expires_at")
		m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{ //nolint:errcheck
			AgencyID: m.cfg.AgencyID,
			TypeID:   "TokenRevocation",
			Properties: map[string]any{
				"agency_id":  m.cfg.AgencyID,
				"token_hash": propStr(rtEnt.Properties, "token_hash"),
				"revoked_at": now,
				"expires_at": exp,
				"reason":     "refresh_reuse",
			},
		})
		parentRels, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
			AgencyID: m.cfg.AgencyID,
			Name:     "parent",
			FromID:   current,
		})
		if err != nil || len(parentRels) == 0 {
			break
		}
		current = parentRels[0].ToID
	}
	return nil
}

// tokenIssuedPayload builds the JSON payload for the cross.org.*.token.issued event.
func (m *orgManager) tokenIssuedPayload(tokenID, clientID, userID string, scopes []string, expiresAt string) string {
	ev := map[string]any{
		"event_id":   uuid.New().String(),
		"event_at":   m.clock.Now().UTC().Format(time.RFC3339),
		"agency_id":  m.cfg.AgencyID,
		"client_id":  clientID,
		"token_id":   tokenID,
		"token_kind": "access",
		"scopes":     scopes,
		"expires_at": expiresAt,
	}
	if userID != "" {
		ev["user_id"] = userID
	}
	b, _ := json.Marshal(ev)
	return string(b)
}

// writeAuditEvent creates an AuditEvent entity (best-effort; errors are swallowed).
func (m *orgManager) writeAuditEvent(ctx context.Context, eventType, actorID, subjectID, outcome, payload string) {
	now := m.clock.Now().UTC().Format(time.RFC3339)
	m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{ //nolint:errcheck
		AgencyID: m.cfg.AgencyID,
		TypeID:   "AuditEvent",
		Properties: map[string]any{
			"agency_id":  m.cfg.AgencyID,
			"event_type": eventType,
			"actor_id":   actorID,
			"subject_id": subjectID,
			"outcome":    outcome,
			"event_at":   now,
			"payload":    payload,
		},
	})
}

// ── Pure utility helpers ──────────────────────────────────────────────────────

func toSet(ids []string) map[string]struct{} {
	s := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		s[id] = struct{}{}
	}
	return s
}

func intersectSets(a, b map[string]struct{}) map[string]struct{} {
	result := map[string]struct{}{}
	for k := range a {
		if _, ok := b[k]; ok {
			result[k] = struct{}{}
		}
	}
	return result
}

func intersectStringSlices(a, b []string) []string {
	bSet := toSet(b)
	result := make([]string, 0, len(a))
	for _, id := range a {
		if _, ok := bSet[id]; ok {
			result = append(result, id)
		}
	}
	return result
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func jsonStrSlice(s []string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
