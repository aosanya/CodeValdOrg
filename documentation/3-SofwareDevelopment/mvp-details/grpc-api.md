# gRPC API

## Purpose

Spec for `proto/codevaldorg/v1/org.proto`. Closes research-gap
**Area 10 (gRPC proto definition)**.

Builds on:

- [architecture.md §6](../../2-SoftwareDesignAndArchitecture/architecture.md) — Go interface signatures
- [error-catalog.md](error-catalog.md) — gRPC code mapping
- [data-model/](data-model/) — message field shapes

---

## Generation toolchain

Match the CodeValdGit setup — `buf` for linting, breaking-change
detection, and code generation.

| File | Purpose |
|---|---|
| `buf.yaml` | Module config: import path, lint rules (`DEFAULT`), breaking-change rules (`FILE`) |
| `buf.gen.yaml` | Plugin pipeline — `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-validate` |
| `proto/codevaldorg/v1/org.proto` | The single proto file for the `OrgService` |
| `gen/go/codevaldorg/v1/` | Generated stubs — committed to the repo (CodeValdGit precedent) |

`make proto` regenerates; `make proto-lint` and `make proto-breaking`
run via CI on PR.

---

## File layout — `proto/codevaldorg/v1/org.proto`

```protobuf
syntax = "proto3";
package codevaldorg.v1;
option go_package = "github.com/aosanya/CodeValdOrg/gen/go/codevaldorg/v1;codevaldorgv1";

import "google/protobuf/timestamp.proto";
import "validate/validate.proto";

service OrgService {
  // ── Organization Lifecycle ─────────────────────────────────────────
  rpc InitOrganization (InitOrganizationRequest) returns (Organization);
  rpc GetOrganization (GetOrganizationRequest) returns (Organization);
  rpc UpdateOrganization (UpdateOrganizationRequest) returns (Organization);
  rpc DisableOrganization (DisableOrganizationRequest) returns (Organization);
  rpc DeleteOrganization (DeleteOrganizationRequest) returns (DeleteOrganizationResponse);

  // ── User & Invitation ──────────────────────────────────────────────
  rpc InviteUser (InviteUserRequest) returns (Invitation);
  rpc AcceptInvitation (AcceptInvitationRequest) returns (User);
  rpc GetUser (GetUserRequest) returns (User);
  rpc ListUsers (ListUsersRequest) returns (ListUsersResponse);
  rpc SuspendUser (SuspendUserRequest) returns (User);
  rpc ReactivateUser (ReactivateUserRequest) returns (User);
  rpc DeleteUser (DeleteUserRequest) returns (DeleteUserResponse);

  // ── Roles & Scopes ─────────────────────────────────────────────────
  rpc CreateRole (CreateRoleRequest) returns (Role);
  rpc UpdateRole (UpdateRoleRequest) returns (Role);
  rpc DeleteRole (DeleteRoleRequest) returns (DeleteRoleResponse);
  rpc ListRoles (ListRolesRequest) returns (ListRolesResponse);

  rpc RegisterScope (RegisterScopeRequest) returns (Scope);
  rpc DeprecateScope (DeprecateScopeRequest) returns (Scope);
  rpc ListScopes (ListScopesRequest) returns (ListScopesResponse);

  // ── Membership ─────────────────────────────────────────────────────
  rpc GrantMembership (GrantMembershipRequest) returns (Membership);
  rpc RevokeMembership (RevokeMembershipRequest) returns (Membership);
  rpc ListMemberships (ListMembershipsRequest) returns (ListMembershipsResponse);

  // ── OAuth Clients ──────────────────────────────────────────────────
  rpc CreateOAuthClient (CreateOAuthClientRequest) returns (CreateOAuthClientResponse);
  rpc RotateClientSecret (RotateClientSecretRequest) returns (RotateClientSecretResponse);
  rpc ListOAuthClients (ListOAuthClientsRequest) returns (ListOAuthClientsResponse);
  rpc DeleteOAuthClient (DeleteOAuthClientRequest) returns (DeleteOAuthClientResponse);

  // ── OAuth 2.0 Protocol ─────────────────────────────────────────────
  rpc Authorize (AuthorizeRequest) returns (AuthorizeResponse);
  rpc Token (TokenRequest) returns (TokenResponse);
  rpc Introspect (IntrospectRequest) returns (IntrospectResponse);
  rpc Revoke (RevokeRequest) returns (RevokeResponse);

  // ── Audit ──────────────────────────────────────────────────────────
  rpc ListAuditEvents (ListAuditEventsRequest) returns (ListAuditEventsResponse);
}
```

All RPCs are unary. **No streaming RPCs in v1** — even the
list endpoints use cursor-based pagination via `page_token` rather
than server-streaming, matching the Cross HTTP-proxy contract.

---

## Message shape conventions

- **Plurals over filter sub-messages.** `ListUsersRequest { string status_filter = 2; }` not `ListUsersRequest { UserFilter filter = 2; }`. Easier to evolve.
- **Pagination.** Every list RPC takes `int32 page_size = …` and `string page_token = …`; returns `string next_page_token = …`. Server enforces a max page size (1000).
- **`agency_id` is always field 1** on every request that accepts it. The server cross-checks against the baked-in `AGENCY_ID` env var (Q24) and rejects mismatches with `codes.PermissionDenied`.
- **Timestamps as `google.protobuf.Timestamp`** in proto, not strings. Proto serialisation of timestamps is well-defined; the data-model RFC 3339 strings are the at-rest representation.
- **Validation** via `protoc-gen-validate` annotations (`(validate.rules)`):
  - `string email = N [(validate.rules).string.email = true];`
  - `string scope_name = N [(validate.rules).string = {pattern: "^[a-z0-9_]{1,20}:[a-z0-9_]{1,28}$"}];` (per [scope-model.md](scope-model.md))
  - `string redirect_uri = N [(validate.rules).string.uri = true];`

---

## Selected message definitions

The full set is large; representative shapes that pin format decisions:

```protobuf
message Organization {
  string agency_id = 1;
  string name = 2;
  bool   enabled = 3;
  string description = 4;
  string contact_email = 5;
  string logo_url = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
  google.protobuf.Timestamp disabled_at = 9;
  google.protobuf.Timestamp deleted_at = 10;
}

message User {
  string agency_id = 1;
  string user_id = 2;            // entity _key
  string email = 3 [(validate.rules).string.email = true];
  enum Status { STATUS_UNSPECIFIED = 0; INVITED = 1; ACTIVE = 2; SUSPENDED = 3; DELETED = 4; }
  Status status = 4;
  string display_name = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
  google.protobuf.Timestamp deleted_at = 8;
}

message TokenRequest {
  string agency_id = 1;
  enum GrantType { GRANT_TYPE_UNSPECIFIED = 0;
                   AUTHORIZATION_CODE = 1; CLIENT_CREDENTIALS = 2; REFRESH_TOKEN = 3; }
  GrantType grant_type = 2;
  string client_id = 3;
  string client_secret = 4;      // present for client_secret_post
  string code = 5;               // for AUTHORIZATION_CODE
  string redirect_uri = 6;       // for AUTHORIZATION_CODE
  string code_verifier = 7;      // for AUTHORIZATION_CODE (PKCE)
  string refresh_token = 8;      // for REFRESH_TOKEN
  repeated string scopes = 9;
}

message TokenResponse {
  string access_token = 1;       // plaintext, prefixed (cv_at_…)
  string token_type = 2;         // always "Bearer"
  int32  expires_in = 3;         // seconds until expiry
  string refresh_token = 4;      // plaintext, prefixed (cv_rt_…); empty for client_credentials
  repeated string scopes = 5;    // effective scopes bound to the token
}

message IntrospectRequest {
  string agency_id = 1;
  string token = 2;              // plaintext (any cv_*_ prefix)
  // No token_type_hint — the prefix carries that information.
}

message IntrospectResponse {
  bool   active = 1;
  // The remaining fields are absent when active=false (RFC 7662).
  repeated string scopes = 2;
  string sub = 3;
  string client_id = 4;
  google.protobuf.Timestamp exp = 5;
  google.protobuf.Timestamp iat = 6;
  string token_type = 7;
}

message ListAuditEventsRequest {
  string agency_id = 1;
  string event_type_filter = 2;
  string actor_id_filter = 3;
  string subject_id_filter = 4;
  google.protobuf.Timestamp from = 5;
  google.protobuf.Timestamp to = 6;
  int32 page_size = 7;
  string page_token = 8;
}
```

The full enumeration of every request/response message is the next
mechanical task and lives directly in the `.proto` file.

---

## Generated-stub location

`gen/go/codevaldorg/v1/org.pb.go` and `org_grpc.pb.go` — committed.

Other languages (TypeScript for CodeValdWorkOrg, Swift for any future
mobile client) generate from the same `.proto` via the consumer's own
toolchain; we do not commit non-Go stubs in this repo.

---

## `ListAuditEvents` is a thin wrapper

Per the memory rule about telemetry/event types
(`feedback_no_separate_telemetry_event_types`) and confirmed in
[data-model/audit.md](data-model/audit.md), `ListAuditEvents` MUST
internally translate to
`entitygraph.DataManager.ListEntities(typeID="audit_event", filter=…)`.

If `EntityFilter` does not yet support time-range filtering (the `from`
/ `to` fields above), the extension is made in
`CodeValdSharedLib/entitygraph` rather than re-implementing inside
CodeValdOrg.

---

## Status code reference

The generated handler maps domain errors to gRPC `codes.Code` per
[error-catalog.md](error-catalog.md). The mapping lives in
`internal/server/mappers.go` — the proto layer carries no error
metadata, only the bare `codes.Code` and a sanitised message.
