# Revocation & Cache Propagation

## Purpose

How revocation is performed, persisted, and propagated to downstream
caches (Cross, resource servers). Closes research-gap **Area 6 (Key
rotation & revocation)**.

Builds on:

- [data-model/oauth.md](data-model/oauth.md) — `TokenRevocation` shape
- [token-issuance.md](token-issuance.md) — strong-CP publish-or-roll-back
- [introspection.md](introspection.md) — read-side caching contract

---

## Revocation primitives

Three callsites produce revocations:

| Callsite | Reason | Triggered by |
|---|---|---|
| `OrgService.Revoke` | `user_logout` or `admin_revoke` | Explicit caller (RFC 7009) |
| Refresh-token rotation | `refresh_reuse` | `Token` flow detecting a consumed `RefreshToken` |
| User suspension | `suspend_user` | `SuspendUser` admin RPC |
| Organization disable | `disable_org` | `DisableOrganization` admin RPC |

All four converge on the same write path:

```
1. Compute token_hash = SHA-256(prefixed plaintext)  OR  use token_hash directly
   for bulk revocations (suspend / disable).
2. CreateEntity TokenRevocation {
     token_hash, agency_id, revoked_at: now,
     expires_at: <matches the underlying token's natural expiry>,
     reason
   }
3. Publish org.token.revoked.
4. Audit: AuditEvent {event_type: "token.revoked", outcome: "success",
                     subject_id: token_id, payload: {reason}}.
```

The strong-CP contract from token issuance applies here too: if Cross
publish fails, roll back the `TokenRevocation` row and return 503.
Revocation must be observable to downstream caches if Org claims it
succeeded.

---

## TokenRevocation lifetime

`TokenRevocation.expires_at` is set to **match the underlying token's
natural expiry** at the moment of revocation. The TTL index then
purges the revocation record exactly when it is no longer load-bearing
(i.e. when the token would have expired anyway).

This bounds the size of `org_oauth_artifacts` — long-lived revocations
do not accumulate forever.

---

## Pub/sub contract — `org.token.revoked`

Topic name and payload are the contract. Subscribers MUST treat this
topic as **at-least-once**; receivers must be idempotent on
`token_hash`.

### Payload

```json
{
  "event_id":   "<uuid>",
  "event_at":   "2026-04-27T12:34:56Z",
  "agency_id":  "<agency-id>",
  "token_hash": "<base64url SHA-256>",
  "token_id":   "<entity-_key — convenient for ID-keyed caches>",
  "token_kind": "access | refresh | authorization_code",
  "reason":     "user_logout | admin_revoke | refresh_reuse | disable_org | suspend_user",
  "expires_at": "2026-04-27T13:34:56Z"
}
```

Plaintext is **never** in the payload — caches keyed on plaintext
must hash before lookup.

### Subscriber expectations

| Subscriber | Action on receipt |
|---|---|
| Cross introspection cache | Evict by `token_hash` (or `token_id` if cache is ID-keyed) |
| Resource-server introspection cache | Same — evict any cached `{active: true}` matching this `token_hash` |
| Audit aggregator | Append to long-term audit-event store; useful for forensics |
| Anything else | Ignore — strict subset model |

Subscribers SHOULD process in single-digit-millisecond budgets; the
revocation propagation deadline is **the cache's `configured_max` TTL**
(see [introspection.md](introspection.md)). A subscriber that takes
longer simply leaves the cached `{active: true}` honoured for that
much longer.

---

## Bulk revocation flows

### `SuspendUser`

```
1. Query AccessToken + RefreshToken where issued_for == user_id AND
   expires_at > now AND no matching TokenRevocation exists.
2. For each token: write TokenRevocation (reason: suspend_user) +
   publish in a single transaction batch.
3. Set User.status = "suspended", User.updated_at = now.
4. Audit: AuditEvent {event_type: "user.suspended", outcome: "success", ...}.
```

If the transaction batch is large (suspended user with many active
sessions), the implementation MAY batch by N (e.g. 100) and publish in
chunks — each chunk is its own atomic publish-or-rollback unit. A
partial-suspend is acceptable because subsequent introspection will
still hit the User.status == "suspended" check at mint time.

### `DisableOrganization`

Same shape as `SuspendUser` but scoped to all tokens in the agency.
This is a high-blast-radius operation; the implementation SHOULD log
a "disable started" and "disable complete" pair and emit a single
aggregate `AuditEvent` for the operation as a whole rather than one
per token.

---

## Refresh-token reuse — chain revocation

Decided in [token-issuance.md](token-issuance.md). If a `RefreshToken`
arrives with `consumed_at != null`:

```
1. Walk parent edges to find the rotation chain root.
2. Walk forward from the root to find every descendant RefreshToken.
3. For each member of the chain (root, every descendant, the
   presented token): write TokenRevocation (reason: refresh_reuse).
4. For each AccessToken still alive that was minted alongside any
   member of the chain (via shared issued_for + issued_to + minted_at
   window): also write TokenRevocation.
5. Publish all revocations.
6. Return ErrInvalidGrant to the caller.
```

The shared-window heuristic for AccessToken correlation is
implementation-internal — there is no `parent` edge from AccessToken
to RefreshToken. (This is a v1 simplification; v2 may add explicit
linkage if forensics need it.)

---

## Race conditions

Two concurrent `/token` calls present the same already-consumed
`RefreshToken`:

```
Both detect reuse simultaneously.
Both attempt to write TokenRevocation rows for the chain.
ArangoDB's unique index on (agency_id, token_hash) makes the second
INSERT a no-op (returns "already exists" error → caught and treated
as success — revocation is idempotent on token_hash).
Both publish the org.token.revoked event.
At-least-once delivery means subscribers see the event twice; idempotent
eviction handles this cleanly.
Both return ErrInvalidGrant.
```

No locking required — the unique index is the synchronisation
primitive.

---

## Per-OAuthClient bulk revocation on client deletion

Resolved 2026-04-28. `DeleteOAuthClient` MUST revoke every still-alive
token issued by that client before the client entity itself is
deleted. The flow:

```
1. Verify client_id exists and is not already deleted.
2. Query AccessToken + RefreshToken where issued_to == client_id AND
   expires_at > now AND no matching TokenRevocation exists.
3. For each token (chunked by N=100 — same chunking pattern as
   SuspendUser): write TokenRevocation (reason: admin_revoke) and
   publish org.token.revoked.
4. Soft-delete the OAuthClient (set deleted_at = now). The
   client entity is NOT hard-deleted — its existence is needed for
   audit-log reachability of past events that referenced it.
5. Audit: AuditEvent {event_type: "oauth_client.deleted",
                      outcome: "success", subject_id: client_id,
                      payload: {revoked_token_count: N}}.
```

`reason: admin_revoke` is reused (not a new sentinel) because from
the holder's perspective the cause is identical — a privileged caller
revoked their access. The audit event's `payload` carries the count
so operators can correlate the bulk effect.

If the agency has a high-volume client (many active tokens), step 3
may take longer than the request deadline. The implementation MAY
return success after kicking off the revocation as a background
goroutine; the client soft-delete step blocks until the revocation
completes so a subsequent token-mint by the same `client_id` (in the
brief window) deterministically fails. v1 keeps it synchronous —
optimisation for high-volume clients is a v2 concern.

## Other resolved implementation items

- **Revocation event ordering** — multiple subscribers may process
  the same agency's events out of order if Cross uses parallel
  consumers. Subscribers must treat each event independently
  (idempotent on `token_hash`); ordering is not guaranteed. The
  `event_at` timestamp is informational only.
