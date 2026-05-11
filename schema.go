package codevaldorg

import "github.com/aosanya/CodeValdSharedLib/types"

// DefaultOrgSchema returns the fixed schema seeded on startup via SchemaManager.SetSchema.
// It declares 15 TypeDefinitions across four storage collections:
//   - org_entities       — identity graph (7 types)
//   - org_oauth_clients  — OAuth client registry (3 types)
//   - org_oauth_artifacts — OAuth artifacts (4 types, all immutable)
//   - org_audit_events   — append-only audit log (1 type, immutable)
//
// No Inverse fields are set on any RelationshipDefinition to avoid ValidateSchema
// requiring both sides of every edge to be declared.
func DefaultOrgSchema() types.Schema {
	return types.Schema{
		ID:      "org-schema-v1",
		Version: 1,
		Tag:     "v1",
		Types: []types.TypeDefinition{
			// ── Identity entities (org_entities) ──────────────────────────────────
			{
				Name:              "Organization",
				DisplayName:       "Organization",
				PathSegment:       "organizations",
				EntityIDParam:     "organizationId",
				StorageCollection: "org_entities",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "name", Type: types.PropertyTypeString, Required: true},
					{Name: "enabled", Type: types.PropertyTypeBoolean, Required: true},
					{Name: "description", Type: types.PropertyTypeString},
					{Name: "contact_email", Type: types.PropertyTypeString},
					{Name: "logo_url", Type: types.PropertyTypeString},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "updated_at", Type: types.PropertyTypeString},
					{Name: "disabled_at", Type: types.PropertyTypeString},
					{Name: "deleted_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "has_user", Label: "Users", ToType: "User", ToMany: true},
					{Name: "has_role", Label: "Roles", ToType: "Role", ToMany: true},
					{Name: "has_scope", Label: "Scopes", ToType: "Scope", ToMany: true},
					{Name: "has_oauth_client", Label: "OAuth Clients", ToType: "OAuthClient", ToMany: true},
					{Name: "has_audit_event", Label: "Audit Events", ToType: "AuditEvent", ToMany: true},
				},
			},
			{
				Name:              "User",
				DisplayName:       "User",
				PathSegment:       "users",
				EntityIDParam:     "userId",
				StorageCollection: "org_entities",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "email", Type: types.PropertyTypeString, Required: true},
					{Name: "status", Type: types.PropertyTypeOption, Required: true,
						Options: []string{"invited", "active", "suspended", "deleted"}},
					{Name: "display_name", Type: types.PropertyTypeString},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "updated_at", Type: types.PropertyTypeString},
					{Name: "deleted_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "has_password_credential", Label: "Password Credential", ToType: "PasswordCredential", ToMany: false},
					{Name: "has_membership", Label: "Memberships", ToType: "Membership", ToMany: true},
					{Name: "has_invitation", Label: "Invitations", ToType: "Invitation", ToMany: true},
				},
			},
			{
				Name:              "PasswordCredential",
				DisplayName:       "Password Credential",
				PathSegment:       "password-credentials",
				EntityIDParam:     "passwordCredentialId",
				StorageCollection: "org_entities",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "password_hash", Type: types.PropertyTypeString, Required: true},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "last_used_at", Type: types.PropertyTypeString},
					{Name: "revoked_at", Type: types.PropertyTypeString},
					{Name: "expires_at", Type: types.PropertyTypeString},
				},
			},
			{
				Name:              "Role",
				DisplayName:       "Role",
				PathSegment:       "roles",
				EntityIDParam:     "roleId",
				StorageCollection: "org_entities",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "name", Type: types.PropertyTypeString, Required: true},
					{Name: "builtin", Type: types.PropertyTypeBoolean, Required: true},
					{Name: "display_name", Type: types.PropertyTypeString},
					{Name: "description", Type: types.PropertyTypeString},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "updated_at", Type: types.PropertyTypeString},
					{Name: "deleted_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "has_scope", Label: "Scopes", ToType: "Scope", ToMany: true},
				},
			},
			{
				Name:              "Scope",
				DisplayName:       "Scope",
				PathSegment:       "scopes",
				EntityIDParam:     "scopeId",
				StorageCollection: "org_entities",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "name", Type: types.PropertyTypeString, Required: true},
					{Name: "registered_by", Type: types.PropertyTypeString, Required: true},
					{Name: "description", Type: types.PropertyTypeString},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "updated_at", Type: types.PropertyTypeString},
					{Name: "deprecated_at", Type: types.PropertyTypeString},
				},
			},
			{
				Name:              "Membership",
				DisplayName:       "Membership",
				PathSegment:       "memberships",
				EntityIDParam:     "membershipId",
				StorageCollection: "org_entities",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "granted_at", Type: types.PropertyTypeString},
					{Name: "granted_by", Type: types.PropertyTypeString},
					{Name: "revoked_at", Type: types.PropertyTypeString},
					{Name: "revoked_by", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "grants_role", Label: "Role", ToType: "Role", ToMany: false},
				},
			},
			{
				Name:              "Invitation",
				DisplayName:       "Invitation",
				PathSegment:       "invitations",
				EntityIDParam:     "invitationId",
				StorageCollection: "org_entities",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "token_hash", Type: types.PropertyTypeString, Required: true},
					{Name: "status", Type: types.PropertyTypeOption, Required: true,
						Options: []string{"pending", "accepted", "expired", "revoked"}},
					{Name: "expires_at", Type: types.PropertyTypeString, Required: true},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "accepted_at", Type: types.PropertyTypeString},
					{Name: "revoked_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "invited_by", Label: "Invited By", ToType: "User", ToMany: false},
					{Name: "will_grant_role", Label: "Will Grant Roles", ToType: "Role", ToMany: true},
				},
			},

			// ── OAuth client entities (org_oauth_clients) ─────────────────────────
			{
				Name:              "OAuthClient",
				DisplayName:       "OAuth Client",
				PathSegment:       "oauth-clients",
				EntityIDParam:     "clientId",
				StorageCollection: "org_oauth_clients",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "client_id", Type: types.PropertyTypeString, Required: true},
					{Name: "client_type", Type: types.PropertyTypeOption, Required: true,
						Options: []string{"public", "confidential"}},
					{Name: "name", Type: types.PropertyTypeString, Required: true},
					{Name: "allowed_grant_types", Type: types.PropertyTypeArray, ElementType: types.PropertyTypeString},
					{Name: "description", Type: types.PropertyTypeString},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "updated_at", Type: types.PropertyTypeString},
					{Name: "disabled_at", Type: types.PropertyTypeString},
					{Name: "deleted_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "has_redirect_uri", Label: "Redirect URIs", ToType: "RedirectURI", ToMany: true},
					{Name: "has_client_secret", Label: "Client Secrets", ToType: "ClientSecret", ToMany: true},
					{Name: "allows_scope", Label: "Allowed Scopes", ToType: "Scope", ToMany: true},
				},
			},
			{
				Name:              "ClientSecret",
				DisplayName:       "Client Secret",
				PathSegment:       "client-secrets",
				EntityIDParam:     "clientSecretId",
				StorageCollection: "org_oauth_clients",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "secret_hash", Type: types.PropertyTypeString, Required: true},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "last_used_at", Type: types.PropertyTypeString},
					{Name: "revoked_at", Type: types.PropertyTypeString},
					{Name: "grace_expires_at", Type: types.PropertyTypeString},
				},
			},
			{
				Name:              "RedirectURI",
				DisplayName:       "Redirect URI",
				PathSegment:       "redirect-uris",
				EntityIDParam:     "redirectUriId",
				StorageCollection: "org_oauth_clients",
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "uri", Type: types.PropertyTypeString, Required: true},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "last_used_at", Type: types.PropertyTypeString},
					{Name: "revoked_at", Type: types.PropertyTypeString},
				},
			},

			// ── OAuth artifacts (org_oauth_artifacts, all immutable) ──────────────
			{
				Name:              "AuthorizationCode",
				DisplayName:       "Authorization Code",
				PathSegment:       "authorization-codes",
				EntityIDParam:     "authorizationCodeId",
				StorageCollection: "org_oauth_artifacts",
				Immutable:         true,
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "code_hash", Type: types.PropertyTypeString, Required: true},
					{Name: "code_challenge", Type: types.PropertyTypeString, Required: true},
					{Name: "redirect_uri", Type: types.PropertyTypeString, Required: true},
					{Name: "expires_at", Type: types.PropertyTypeString, Required: true},
					{Name: "state", Type: types.PropertyTypeString},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "consumed_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "issued_to", Label: "Client", ToType: "OAuthClient", ToMany: false},
					{Name: "issued_for", Label: "User", ToType: "User", ToMany: false},
					{Name: "has_requested_scope", Label: "Requested Scopes", ToType: "Scope", ToMany: true},
				},
			},
			{
				Name:              "AccessToken",
				DisplayName:       "Access Token",
				PathSegment:       "access-tokens",
				EntityIDParam:     "accessTokenId",
				StorageCollection: "org_oauth_artifacts",
				Immutable:         true,
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "token_hash", Type: types.PropertyTypeString, Required: true},
					{Name: "expires_at", Type: types.PropertyTypeString, Required: true},
					{Name: "created_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "issued_to", Label: "Client", ToType: "OAuthClient", ToMany: false},
					{Name: "issued_for", Label: "User", ToType: "User", ToMany: true},
					{Name: "has_scope", Label: "Scopes", ToType: "Scope", ToMany: true},
				},
			},
			{
				Name:              "RefreshToken",
				DisplayName:       "Refresh Token",
				PathSegment:       "refresh-tokens",
				EntityIDParam:     "refreshTokenId",
				StorageCollection: "org_oauth_artifacts",
				Immutable:         true,
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "token_hash", Type: types.PropertyTypeString, Required: true},
					{Name: "expires_at", Type: types.PropertyTypeString, Required: true},
					{Name: "created_at", Type: types.PropertyTypeString},
					{Name: "consumed_at", Type: types.PropertyTypeString},
				},
				Relationships: []types.RelationshipDefinition{
					{Name: "issued_to", Label: "Client", ToType: "OAuthClient", ToMany: false},
					{Name: "issued_for", Label: "User", ToType: "User", ToMany: true},
					{Name: "has_scope", Label: "Scopes", ToType: "Scope", ToMany: true},
					{Name: "parent", Label: "Parent Token", ToType: "RefreshToken", ToMany: false},
				},
			},
			{
				Name:              "TokenRevocation",
				DisplayName:       "Token Revocation",
				PathSegment:       "token-revocations",
				EntityIDParam:     "tokenRevocationId",
				StorageCollection: "org_oauth_artifacts",
				Immutable:         true,
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "token_hash", Type: types.PropertyTypeString, Required: true},
					{Name: "revoked_at", Type: types.PropertyTypeString, Required: true},
					{Name: "expires_at", Type: types.PropertyTypeString, Required: true},
					{Name: "reason", Type: types.PropertyTypeOption,
						Options: []string{"user_logout", "admin_revoke", "refresh_reuse", "disable_org", "suspend_user"}},
				},
			},

			// ── Audit events (org_audit_events, immutable) ────────────────────────
			{
				Name:              "AuditEvent",
				DisplayName:       "Audit Event",
				PathSegment:       "audit-events",
				EntityIDParam:     "auditEventId",
				StorageCollection: "org_audit_events",
				Immutable:         true,
				Properties: []types.PropertyDefinition{
					{Name: "agency_id", Type: types.PropertyTypeString, Required: true},
					{Name: "event_type", Type: types.PropertyTypeString, Required: true},
					{Name: "actor_id", Type: types.PropertyTypeString},
					{Name: "subject_id", Type: types.PropertyTypeString},
					{Name: "outcome", Type: types.PropertyTypeString},
					{Name: "event_at", Type: types.PropertyTypeString, Required: true},
					{Name: "payload", Type: types.PropertyTypeString},
				},
			},
		},
	}
}
