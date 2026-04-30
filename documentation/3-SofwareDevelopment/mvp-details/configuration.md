# Configuration

## Purpose

Single source of truth for every environment variable CodeValdOrg
reads at startup. Drives `internal/config/config.go`. Closes
research-gap **Area 11 (Configuration)**. Decided 2026-04-27 in
Q24–Q25 of the research Q&A.

---

## Multi-tenancy mode

**One process per agency.** `AGENCY_ID` is baked in at startup; the
process holds a connection to exactly one ArangoDB database and rejects
any RPC whose `agencyId` path parameter doesn't match.

This matches CodeValdGit's per-repo model and gives strong isolation:
a bug in one agency's process cannot touch another agency's data.
The trade-off is more processes to operate, accepted in v1.

The architecture's existing wording — *"agency ID is fixed at
service-handler construction time"* — resolves cleanly with this
choice: handler construction is process startup.

---

## Required environment variables

Process refuses to start (fail-fast) if any of these are unset or
empty.

| Env var | Type | Used for |
|---|---|---|
| `AGENCY_ID` | string | The single agency this process serves; baked into every handler |
| `ARANGO_ENDPOINTS` | comma-separated URLs | ArangoDB cluster endpoints |
| `ARANGO_USER` | string | ArangoDB auth |
| `ARANGO_PASSWORD` | string | ArangoDB auth |
| `CROSS_ENDPOINT` | URL (gRPC) | `OrchestratorService` address for the `Register` heartbeat |
| `ORG_ISSUER_URL` | URL | Public URL of *this* process; appears in `iss` claims and `/.well-known/oauth-authorization-server` |

---

## Optional environment variables

All defaulted; override only when the default is unsuitable.

| Env var | Default | Used for |
|---|---|---|
| `ARANGO_DB_NAME` | `agency-${AGENCY_ID}` | Per-agency database name (override only for non-standard naming) |
| `BIND_ADDR` | `:9090` | cmux listener — gRPC and HTTP/1.1 multiplexed on one port (matches CodeValdGit Smart HTTP pattern) |
| `METRICS_ADDR` | `:9091` | Telemetry endpoint serving the counters defined in [error-catalog.md](error-catalog.md) |
| `ORG_ACCESS_TOKEN_TTL` | `1h` | `AccessToken.expires_at = now + this` |
| `ORG_REFRESH_TOKEN_TTL` | `720h` (30 days) | `RefreshToken.expires_at = now + this` |
| `ORG_AUTH_CODE_TTL` | `60s` | `AuthorizationCode.expires_at = now + this` (architecture §5.1) |
| `ORG_CLIENT_SECRET_GRACE` | `5m` | After `RotateClientSecret`, the old secret is still honoured for this window (`ClientSecret.grace_expires_at`) |
| `ORG_ARGON2_TIME` | `3` | Argon2id time cost — number of iterations |
| `ORG_ARGON2_MEMORY_KIB` | `65536` | Argon2id memory cost in KiB |
| `ORG_ARGON2_THREADS` | `4` | Argon2id parallelism |
| `ORG_REGISTRAR_INTERVAL` | `20s` | Cross heartbeat cadence (architecture §3 / §8) |
| `ORG_RATELIMIT_TOKEN_PER_CLIENT` | `50:100` (sustained:burst, req/s) | `/oauth/token` per-`client_id` token bucket — see [threat-model.md](threat-model.md) |
| `ORG_RATELIMIT_AUTHORIZE_PER_IP` | `20:40` | `/oauth/authorize` per-source-IP token bucket |
| `ORG_RATELIMIT_INTROSPECT_PER_CLIENT` | `1000:2000` | `/oauth/introspect` per-`client_id` (hot path; high burst expected) |
| `ORG_RATELIMIT_ADMIN_PER_CLIENT` | `100:200` | Cross-proxied admin RPCs per-`client_id` |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

---

## Secret loading

**Plain environment variables only.** Production deployments project
Kubernetes `Secret` resources as env vars in the deployment manifest.

- No Vault integration in v1.
- No file-based secret mounting (no `_FILE` suffix variants).
- No on-disk YAML/JSON config overlay.

This is twelve-factor by choice — the operations surface stays small
and the same env-var set works identically in dev, CI, staging, prod.
Vault and dynamic secret rotation are v2 concerns.

---

## Startup validation

`internal/config/config.go` performs validation in two passes:

1. **Required-set check** — every variable in the *Required* table
   above must be non-empty. Missing variables cause `os.Exit(2)` with
   a single-line error naming all missing variables (no partial start).
2. **Range / format check** — `URL` types parse as `url.Parse`;
   duration types parse as `time.ParseDuration`; integer types parse
   with bounds (`ORG_ARGON2_TIME >= 1`, `ORG_ARGON2_MEMORY_KIB >= 8192`,
   `ORG_ARGON2_THREADS >= 1`, all TTLs positive).

Validation errors are surfaced before any storage or network
connection attempt — a misconfigured process never reaches a
half-started state.

---

## Items deliberately not in v1

These are noted so future contributors know they were considered and
rejected, not forgotten.

- **TLS cert paths** — terminated at the platform ingress (Cross or
  the cluster's ingress controller), not in CodeValdOrg.
- **`HEALTH_ADDR`** — health endpoint shares `BIND_ADDR` (HTTP path
  `/healthz`).
- ~~**`RATE_LIMIT_*` knobs**~~ — added 2026-04-27; in-process per-`client_id` limiter is now in v1 ([threat-model.md](threat-model.md)).
- **YAML overlay** — twelve-factor; env only.
- **Per-agency runtime config** — every agency's process is
  configured the same way; cross-agency policy lives in the schema /
  data, not in process config.
