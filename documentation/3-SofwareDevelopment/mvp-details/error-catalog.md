# Error Catalog

## Purpose

Single source of truth for every sentinel error CodeValdOrg can
return. Each row is the contract between four layers:

- **Sentinel** — Go `error` value declared in `errors.go`
- **gRPC code** — `codes.Code` returned by the gRPC handler
- **HTTP status** — used by the OAuth HTTP endpoints (`/authorize`,
  `/token`, `/introspect`, `/revoke`, `.well-known/...`) and the
  Cross-proxied admin surface
- **OAuth error code** — RFC 6749 §5.2 / §4.1.2.1 `error` body string
  (only relevant on OAuth endpoints)
- **Handling** — what the audit / telemetry pipeline does on this
  outcome (decided in Q22)

Closes research-gap **Area 9 (Error handling)**. Decided 2026-04-27 in
Q22–Q23 of the research Q&A. Drives implementation of `errors.go` and
`internal/server/mappers.go`.

---

## Handling labels

| Label | Meaning |
|---|---|
| `audit-per-occurrence` | Write an `AuditEvent` with `outcome = denied` or `failure` |
| `metric-count` | Increment an in-process counter (one per OAuth `error` code) — no DB write |
| `audit + metric` | Both — used for auth-relevant denials that are also worth rate-trending |
| — | Neither — pure domain error, surfaced to caller, no observability side-effect |

The metric counter taxonomy is one counter per OAuth error code
(`invalid_request_total`, `invalid_client_total`, etc.) plus
`server_error_total`. Exposed via the platform's existing telemetry
path; no new endpoint.

---

## Internal-error sanitisation contract

Decided in Q23. Every error not in this catalog (panics, ArangoDB
unreachable, Cross publish failure, hash-library failure, …) is
wrapped before it leaves the service:

| Layer | Returned value |
|---|---|
| gRPC | `codes.Internal` + message `"internal error"` + no details |
| HTTP / OAuth | `500` + `{"error": "server_error"}` |
| Trailer / header | `request-id: <correlation_id>` (for log grep) |
| Logs | `error` level with the **real** error string + stack trace + `correlation_id` |

The correlation ID is the only signal the caller gets that ties their
request to a server-side log entry. Real error strings never cross
the trust boundary.

---

## OAuth 2.0 endpoint errors

Returned from `/authorize`, `/token`, `/introspect`, `/revoke`. The
gRPC mappings apply to the corresponding RPCs (`Authorize`, `Token`,
`Introspect`, `Revoke`).

| Sentinel | gRPC code | HTTP | OAuth `error` | Handling |
|---|---|---|---|---|
| `ErrInvalidRequest` | `InvalidArgument` | 400 | `invalid_request` | `metric-count` |
| `ErrInvalidClient` | `Unauthenticated` | 401 | `invalid_client` | `audit + metric` |
| `ErrInvalidGrant` | `PermissionDenied` | 400 | `invalid_grant` | `audit + metric` |
| `ErrUnauthorizedClient` | `PermissionDenied` | 400 | `unauthorized_client` | `audit + metric` |
| `ErrUnsupportedGrantType` | `InvalidArgument` | 400 | `unsupported_grant_type` | `metric-count` |
| `ErrInvalidScope` | `InvalidArgument` | 400 | `invalid_scope` | `audit + metric` |
| `ErrAccessDenied` | `PermissionDenied` | 403 | `access_denied` | `audit-per-occurrence` |
| `ErrTemporarilyUnavailable` | `Unavailable` | 503 | `temporarily_unavailable` | `metric-count` |
| `ErrRateLimitExceeded` | `ResourceExhausted` | 429 | `temporarily_unavailable` (with `error_description: "rate limit exceeded"`) | `metric-count` |
| `ErrServerError` (sanitisation catch-all) | `Internal` | 500 | `server_error` | `metric-count` |

Notes:

- `ErrInvalidGrant` covers: unknown / expired / consumed authorization
  code, unknown / expired / consumed refresh token, refresh-token
  reuse detected, wrong password on resource-owner password grant
  (not in v1 but reserved). RFC 6749 §5.2 says **400** for all of
  these; we follow that rather than 401.
- `ErrInvalidClient` is the only OAuth error that returns **401**, per
  RFC 6749 §5.2 — and only when the client used `Authorization: Basic`.
- `Introspect` denying a token (returning `{active: false}`) is **not**
  an error in this catalog — it's a normal response per RFC 7662. The
  audit-per-occurrence label there belongs to a separate
  `token.introspect_denied` event type.

---

## Admin surface errors (gRPC + Cross-proxied HTTP)

Returned from the admin RPCs (`InitOrganization`, `InviteUser`,
`SuspendUser`, `CreateRole`, `DeleteOAuthClient`, …). No OAuth `error`
column — these are not OAuth endpoints.

| Sentinel | gRPC code | HTTP | Used when | Handling |
|---|---|---|---|---|
| `ErrOrgNotFound` | `NotFound` | 404 | Organization with given `agency_id` doesn't exist | — |
| `ErrOrgAlreadyExists` | `AlreadyExists` | 409 | `InitOrganization` called twice for same agency | — |
| `ErrOrgDisabled` | `FailedPrecondition` | 409 | Mutating call on a disabled Organization | `audit-per-occurrence` |
| `ErrUserNotFound` | `NotFound` | 404 | — | — |
| `ErrUserAlreadyExists` | `AlreadyExists` | 409 | `InviteUser` for an email that already has a User | — |
| `ErrUserSuspended` | `FailedPrecondition` | 409 | `Token` flow attempts mint for a suspended user | `audit-per-occurrence` |
| `ErrRoleNotFound` | `NotFound` | 404 | — | — |
| `ErrRoleAlreadyExists` | `AlreadyExists` | 409 | `CreateRole` with an existing `(agency_id, name)` | — |
| `ErrRoleBuiltinImmutable` | `FailedPrecondition` | 409 | `UpdateRole` / `DeleteRole` against a `builtin: true` role attempting to rename or delete (FR-003) | `audit-per-occurrence` |
| `ErrScopeNotFound` | `NotFound` | 404 | — | — |
| `ErrScopeNameCollision` | `AlreadyExists` | 409 | `RegisterScope` for a name owned by a different `registered_by` (see [scope-model.md](scope-model.md)) | `audit-per-occurrence` |
| `ErrScopeReserved` | `PermissionDenied` | 403 | `RegisterScope` for a reserved-prefix scope (`org:`, `audit:`) by a non-CodeValdOrg caller | `audit-per-occurrence` |
| `ErrMembershipNotFound` | `NotFound` | 404 | — | — |
| `ErrInvitationNotFound` | `NotFound` | 404 | `AcceptInvitation` with unknown token hash | `metric-count` |
| `ErrInvitationExpired` | `FailedPrecondition` | 410 | `AcceptInvitation` after `expires_at` | `audit-per-occurrence` |
| `ErrInvitationAlreadyAccepted` | `FailedPrecondition` | 409 | `AcceptInvitation` on an already-accepted invitation | `audit-per-occurrence` |
| `ErrOAuthClientNotFound` | `NotFound` | 404 | — | — |
| `ErrRedirectURIMismatch` | `InvalidArgument` | 400 | `/authorize` or `/token` redirect_uri doesn't exact-match a `RedirectURI` (architecture §5.1) | `audit + metric` |
| `ErrPKCERequired` | `InvalidArgument` | 400 | `/authorize` from a public client without `code_challenge` | `audit + metric` |
| `ErrPKCEMethodInvalid` | `InvalidArgument` | 400 | `code_challenge_method` other than `S256` | `audit + metric` |
| `ErrPKCEMismatch` | `PermissionDenied` | 400 | `/token` `code_verifier` doesn't hash to the stored `code_challenge` | `audit + metric` |
| `ErrTokenRevoked` | `PermissionDenied` | 401 | Surface only on admin paths; OAuth introspection returns `{active:false}` instead | `audit-per-occurrence` |
| `ErrTokenExpired` | `PermissionDenied` | 401 | Same as above — admin-path surface only | `metric-count` |
| `ErrImmutableType` | `FailedPrecondition` | 409 | Any `Update*` against an immutable entity type (audit, OAuth artifacts) | — |
| `ErrSuperAdminRequired` | `FailedPrecondition` | 409 | Demoting / removing the last `super_admin` membership for the agency (role-taxonomy invariant) | `audit-per-occurrence` |

---

## Sentinel-naming convention

- All sentinels are **package-level `var`** declarations in `errors.go`,
  using the standard `errors.New` pattern (no error wrapping at the
  declaration site).
- Wrap with `fmt.Errorf("...: %w", ErrXxx, …)` when adding context;
  callers use `errors.Is(err, ErrXxx)` to dispatch.
- `internal/server/mappers.go` is the **only** translator from
  sentinel to gRPC code / HTTP status / OAuth error body. All other
  layers return raw sentinels.

---

## Open implementation questions

These are tracked as follow-ups, not unresolved decisions:

- **Audit `payload` shape per `outcome != success` row** — what
  structured details go in `AuditEvent.payload`? Likely the
  `correlation_id`, the offending field name, and (for OAuth errors)
  the `error_description` text. Spec lives in
  [data-model/audit.md](data-model/audit.md) when written; not blocking.
- **Metric label cardinality** — counters by OAuth error code are safe
  (small fixed set). Avoid labelling by `client_id` or `agency_id` —
  unbounded cardinality.
