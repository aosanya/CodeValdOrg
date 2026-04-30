# Testing Strategy

## Purpose

How CodeValdOrg is tested at every layer. Closes research-gap
**Area 12 (Testing strategy)**.

Three layers, each with a clear charter and tooling. Negative-test
checklist at the bottom.

---

## Layer 1 — Unit tests

In-process, no I/O. Cover domain logic, error mapping, scope
intersection, PKCE verification, Argon2id encoding.

### Doubles

- **`entitygraph.DataManager`** — in-memory implementation in
  `internal/testdouble/datamanager.go`. Implements the full
  `DataManager` interface against a `map[string]Entity` +
  `map[string]Relationship`. Indexes are simulated by linear scans
  (test data sets are small).
- **`CrossPublisher`** — mock that records publish calls for
  assertion. A `failingPublisher` variant returns an error to test the
  Q26 strong-CP rollback path.
- **`KeyHasher`** (Argon2id wrapper) — real implementation; Argon2id
  is fast enough at test parameters (`t=1, m=8192, p=1`). Per-test
  override via `ORG_ARGON2_*` env vars.

### Coverage targets

| Package | Target |
|---|---|
| `internal/server` (handler logic, mappers) | ≥ 90 % |
| Top-level `org_impl_*.go` files | ≥ 85 % |
| `internal/crypto` (PKCE, hash, token gen) | ≥ 95 % — exhaustive table tests |
| `internal/config` | ≥ 80 % — required-set + format-error tables |

`make test-unit` runs the layer; CI gates merges on the targets.

---

## Layer 2 — Integration tests

Real ArangoDB via [`testcontainers-go`](https://golang.testcontainers.org/).
A single container is started per test package; databases inside the
container are per-test for isolation.

### Setup pattern

```go
func TestMain(m *testing.M) {
    ctx := context.Background()
    container, _ := arangodb.Run(ctx, "arangodb/arangodb:3.11", arangodb.WithRootPassword("test"))
    defer container.Terminate(ctx)
    arangoEndpoint = container.Endpoint(ctx)
    os.Exit(m.Run())
}

func setupSvc(t *testing.T) (codevaldorg.OrgService, func()) {
    dbName := fmt.Sprintf("agency-test-%s", uuid.New().String())
    dm, _ := arangoutil.Connect(ctx, arangoutil.Config{...})
    sm, _ := entitygraph.NewSchemaManager(dm)
    sm.SetSchema(codevaldorg.DefaultOrgSchema())
    pub := &recordingPublisher{}
    svc := codevaldorg.NewOrgService(dm, sm, pub, dbName)
    return svc, func() { dm.DropDatabase(ctx, dbName) }
}
```

### Coverage scope

- Full Authorization Code + PKCE flow end-to-end
- Full Client Credentials flow
- Refresh-token rotation including reuse-detection chain revocation
- Suspend / disable bulk revocation
- Audit-event persistence (assert each mutating call writes an
  AuditEvent row)
- Schema seeding idempotency (re-run `SetSchema` against a
  pre-seeded DB)
- TTL-purge behaviour for AuthorizationCode (60 s; test uses a 2 s
  override and time.Sleep)

`make test-integration` runs the layer; ~30 s budget, gated on the
`integration` build tag so unit-test runs stay fast.

---

## Layer 3 — Conformance suite

OAuth 2.0 / RFC compliance, run as black-box tests against a running
service.

### v1 must-pass

- **RFC 6749 §5.2** — every error response uses the correct `error`
  code and HTTP status (driven by [error-catalog.md](error-catalog.md))
- **RFC 7636** — PKCE `S256` accepted; `plain` rejected; missing
  `code_verifier` rejected
- **RFC 7662** — introspection response shape; `{active: false}`
  parity for unknown/expired/revoked
- **RFC 7009** — revocation accepts both access and refresh tokens;
  returns 200 even for unknown tokens (RFC §2.2)
- **RFC 8414** — `/.well-known/oauth-authorization-server` document
  shape; required fields present

### v1 want-to-pass

- OAuth 2.1 draft tightenings already in scope (mandatory PKCE,
  exact-match redirect URI, no implicit flow)

### Tooling

[OIDF conformance fixtures](https://gitlab.com/openid/conformance-suite)
where licensing permits. Otherwise hand-rolled black-box scripts in
`test/conformance/` driven by a Go test runner; the runner spins up
the service via the same `testcontainers` ArangoDB used in Layer 2.

`make test-conformance` runs the layer; gated on `conformance` build
tag (slow — minute scale).

---

## Negative-test checklist

Every flow gets explicit negative coverage. This list is the floor,
not the ceiling.

### Authorization Code + PKCE

- [ ] `code_challenge_method = plain` → 400 `invalid_request`
- [ ] Missing `code_challenge` from public client → 400 `invalid_request`
- [ ] Wildcard or fuzzy redirect_uri match → 400 `invalid_request`
- [ ] Reused authorization code → 400 `invalid_grant`
- [ ] Expired authorization code → 400 `invalid_grant`
- [ ] Wrong `code_verifier` → 400 `invalid_grant`
- [ ] `redirect_uri` mismatch between `/authorize` and `/token` → 400 `invalid_grant`

### Client Credentials

- [ ] Public client attempting client-credentials → 400 `unauthorized_client`
- [ ] Wrong client_secret → 401 `invalid_client`
- [ ] Expired client_secret (post-grace) → 401 `invalid_client`
- [ ] Within grace window → 200 with token (verifies grace honour)
- [ ] Requested scope not in `client.allowedScopes` → 400 `invalid_scope`

### Refresh Token

- [ ] Reused refresh token → chain revocation + 400 `invalid_grant`
- [ ] Expired refresh token → 400 `invalid_grant`
- [ ] Refresh from a different `client_id` than the original → 400 `invalid_grant`
- [ ] Scope upgrade attempt → 400 `invalid_scope`

### Introspection

- [ ] Unknown token → `{active: false}`
- [ ] Expired token → `{active: false}`
- [ ] Revoked token → `{active: false}`
- [ ] Malformed token plaintext → `{active: false}`
- [ ] Anonymous caller → 401 `invalid_client`

### Admin

- [ ] Calling admin RPC without `org:admin` scope → 403 `access_denied`
- [ ] Demoting the last `super_admin` → 409 `failed_precondition`
- [ ] Renaming a built-in role → 409 `failed_precondition`
- [ ] Cross-agency call (request `agency_id` ≠ baked-in `AGENCY_ID`) → 403

### Schema invariants

- [ ] `Update*` against an immutable type → `ErrImmutableType`
- [ ] Re-`SetSchema` against existing DB is a no-op
- [ ] TTL-purge actually removes expired AuthorizationCode after `expires_at`

---

## Performance & latency benchmarks

NFR-002 demands p99 introspection ≤ 10 ms in-cluster over gRPC.
Benchmark via `go test -bench` against the `testcontainers` ArangoDB:

```go
func BenchmarkIntrospect(b *testing.B) {
    svc, cleanup := setupSvc(b)
    defer cleanup()
    // Pre-mint 1000 tokens, store plaintexts
    ...
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = svc.Introspect(ctx, IntrospectRequest{Token: tokens[i % len(tokens)]})
    }
}
```

Result interpretation: any benchmark exceeding 1 ms p99 wall-clock per
introspection is a regression. The gap between bench wall-clock and
NFR-002 (10 ms p99) is the network-hop budget for in-cluster calls.

`make test-bench` runs the layer; not gated on every PR — run on
release branches and when touching the introspection hot path.

---

## CI matrix

| Job | Runs on | Gates merge? |
|---|---|---|
| `make test-unit` | every push | yes |
| `make test-integration` | every push | yes |
| `make test-conformance` | every push to `main`; nightly on feature branches | merging to `main` only |
| `make test-bench` | release branches, manual dispatch | no |
| `make proto-lint` and `make proto-breaking` | every PR touching `proto/` | yes |

---

## Time injection — `Clock` interface

Resolved 2026-04-28 in Q33. CodeValdOrg uses a `Clock` interface
plumbed through every issuance and lookup function rather than calling
`time.Now()` directly:

```go
// internal/clock/clock.go
type Clock interface {
    Now() time.Time
}

type RealClock struct{}
func (RealClock) Now() time.Time { return time.Now() }

// internal/testdouble/clock.go (test-only)
type FakeClock struct {
    mu  sync.Mutex
    now time.Time
}
func (f *FakeClock) Now() time.Time { f.mu.Lock(); defer f.mu.Unlock(); return f.now }
func (f *FakeClock) Advance(d time.Duration) { f.mu.Lock(); defer f.mu.Unlock(); f.now = f.now.Add(d) }
```

`OrgService` takes a `Clock` in its constructor:

```go
func NewOrgService(dm entitygraph.DataManager, sm OrgSchemaManager,
                   pub CrossPublisher, clock Clock, agencyID string) OrgService
```

In production, `cmd/main.go` injects `clock.RealClock{}`. In tests, a
`FakeClock` lets TTL behaviour be exercised deterministically without
sleeps:

```go
fc := &FakeClock{now: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)}
svc := codevaldorg.NewOrgService(dm, sm, pub, fc, dbName)
// ... mint a token with ORG_AUTH_CODE_TTL=60s
fc.Advance(61 * time.Second)
// token is now expired without any time.Sleep
```

The trade-off accepted: ~12 issuance and lookup functions take an
extra parameter. This is mechanical and the test-determinism win
justifies it.

**Note on TTL indexes.** ArangoDB's TTL purge is real-clock driven;
tests that need to exercise *purge* behaviour (not just expiry-check
behaviour) still need real sleeps with short `expires_at` overrides.
The `Clock` interface doesn't and shouldn't influence the database
clock.

## Other open implementation items

- **Conformance-fixture licensing** — confirm the OIDF suite is
  re-usable for non-certified self-checking; otherwise build the
  hand-rolled scripts from scratch in `test/conformance/`.
