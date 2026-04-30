# Audit Entity

A single entity, `AuditEvent`, in its own `org_audit_events`
collection. `Immutable: true` — `Update*` paths return
`ErrImmutableType`. Coverage required by FR-009.

---

## `AuditEvent`

The structured-details requirement in FR-009 is satisfied by a single
`payload` `PropertyTypeString` holding JSON, per the
entitygraph telemetry/event convention (CodeValdDT precedent — see the
`feedback_no_separate_telemetry_event_types` memory). This is *not* a
violation of the no-freeform-map rule because `payload` is a typed
string field with a documented convention, not an open `attributes`
map.

| Tier | Property | Type | Required | Index |
|---|---|---|---|---|
| Core | `agency_id` | string | yes | partition |
| Core | `event_type` | string (e.g. `user.invited`, `token.issued`, `oauth_client.rotated`) | yes | filter index |
| Core | `event_at` | string (RFC 3339) | yes | range index |
| Core | `actor_id` | string (user_id or client_id of caller) | yes | filter index |
| Core | `subject_id` | string (entity acted on — user_id, client_id, token_hash, etc.) | yes | filter index |
| Core | `outcome` | option (`success` / `failure` / `denied`) | yes | filter index |
| Long tail | `source_ip` | string | no | — |
| Long tail | `source_user_agent` | string | no | — |
| Long tail | `error_code` | string (sentinel error name when `outcome != success`) | no | — |
| Long tail | `payload` | string (JSON-encoded event-specific structured details) | no | — |

**Edges in:** `belongs_to_organization` (auto-inverse of
`Organization.has_audit_event`). **No edges out** — `actor_id` and
`subject_id` are denormalised IDs, not edges, so audit reads stay
single-collection-fast and meet the NFR-002 latency budget without a
graph traversal.

---

## Required `event_type` coverage

FR-009 names the events that must exist. Each maps to a single
`event_type` string:

| Lifecycle | `event_type` |
|---|---|
| User invited | `user.invited` |
| User activated | `user.activated` |
| User suspended | `user.suspended` |
| User deleted | `user.deleted` |
| Role created / updated / deleted | `role.created` / `role.updated` / `role.deleted` |
| Scope added / removed (on a role) | `role.scope_added` / `role.scope_removed` |
| Membership granted / revoked | `membership.granted` / `membership.revoked` |
| OAuth client created / rotated / revoked | `oauth_client.created` / `oauth_client.rotated` / `oauth_client.revoked` |
| Authorization code issued / consumed / expired | `auth_code.issued` / `auth_code.consumed` / `auth_code.expired` |
| Token issued / refreshed / revoked / introspect-denied | `token.issued` / `token.refreshed` / `token.revoked` / `token.introspect_denied` |

Resource services may register additional event types via the audit
write path; the schema does not constrain the string set.

---

## Implication for the gRPC API

The `ListAuditEvents` RPC declared in
[architecture.md §6](../../../2-SoftwareDesignAndArchitecture/architecture.md)
must be a thin wrapper over
`entitygraph.DataManager.ListEntities(typeID="audit_event", filter=…)` —
not a bespoke audit-storage path. This keeps the entity-graph mental
model uniform and avoids splitting the immutability and storage-routing
story across two systems.

This is tracked under **research-gaps.md Area 10 (gRPC proto
definition)**; the wrapper must accept time-range, actor, subject, and
event-type filters that translate directly to `EntityFilter` fields.

If `EntityFilter` does not yet support time-range filtering, the
extension must be made in `CodeValdSharedLib/entitygraph` rather than
re-implemented inside CodeValdOrg.
