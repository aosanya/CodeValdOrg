````prompt
---
agent: agent
---

# CodeValdOrg — Status Update Prompt

## Purpose
Record status updates, findings, and progress notes for **CodeValdOrg** into
topic files under:

```
CodeValdOrg/documentation/3-SofwareDevelopment/status/
```

---

## 📊 CodeValdOrg — Current Capabilities

> For full architecture details see
> `documentation/2-SoftwareDesignAndArchitecture/architecture.md` (produced in a
> later step).

### Role in the Platform
CodeValdOrg is the **identity, authentication, and authorization service** —
it owns the authoritative record of every principal, the keys they hold, and
the scopes attached to those keys. Every other CodeVald service asks
CodeValdOrg whether an inbound caller is authenticated and authorized.

### gRPC Endpoints (Inbound)

| Service | Method | Description |
|---|---|---|
| `OrgService` | `IssueKey` | Issues a new API key for a principal; returns the plaintext exactly once; publishes `org.token.issued` |
| `OrgService` | `RotateKey` | Rotates an existing key; returns a new plaintext exactly once; publishes `org.token.issued` |
| `OrgService` | `RevokeKey` | Revokes a key; publishes `org.token.revoked` |
| `OrgService` | `GetKey` | Returns key metadata (no plaintext, no hash) |
| `OrgService` | `ListKeys` | Returns a filtered list of key metadata |
| `OrgService` | `VerifyToken` | Validates a plaintext token and returns the authenticated `Principal` |
| `OrgService` | `Authorize` | Answers `(principalID, scope, resource) → Decision` — pure, side-effect-free |

### HTTP Routes (proxied via CodeValdCross)

| Method | Pattern |
|---|---|
| `POST`   | `/org/keys` |
| `POST`   | `/org/keys/{keyId}/rotate` |
| `POST`   | `/org/keys/{keyId}/revoke` |
| `GET`    | `/org/keys/{keyId}` |
| `GET`    | `/org/keys` |
| `POST`   | `/org/verify` |
| `POST`   | `/org/authorize` |

### Pub/Sub

| Topic | Direction | Description |
|---|---|---|
| `org.token.issued`  | **produces** | Published after every successful `IssueKey` / `RotateKey` |
| `org.token.revoked` | **produces** | Published after every successful `RevokeKey` |

### Key Design Properties
- **Single interface** — `OrgManager` is the only business-logic entry point
- **Backend-agnostic** — `Backend` interface injected; ArangoDB is the production impl
- **Crypto-isolated** — `KeyHasher` interface injected; bcrypt/argon2 stays behind it
- **Plaintext-once** — issued/rotated keys return plaintext exactly once; storage holds only the hash
- **Constant-time comparisons** — all secret comparisons use `crypto/subtle`
- **Heartbeat** — `Register` called every 20 s; Cross treats repeat calls as liveness
- **No domain logic from other services** — this service manages principals, keys, and scopes only

---

## 🗂️ Status File Rules

### Target directory
```
CodeValdOrg/documentation/3-SofwareDevelopment/status/
```

### File size limit
- **≤ 400 lines** → write/append to a single topic file: `status/{topic}.md`
- **> 400 lines** → escalate to a subfolder with a `README.md` index

### Workflow (enforce every session)

```bash
# Step 1 — Check existing file size
wc -l documentation/3-SofwareDevelopment/status/{topic}.md

# Step 2 — Choose write target
# If file doesn't exist → create status/{topic}.md
# If file ≤ 400 lines  → append to status/{topic}.md
# If file > 400 lines  → create status/{topic}/ subfolder

# Step 3 — Write the status entry
```

---

### Status Entry Format

```markdown
## {YYYY-MM-DD} — {Short title}

**Status**: {In Progress | Blocked | Done | Investigating}
**Topic**: {key-lifecycle | authentication | authorization | storage | grpc-service | cross-registration | general}

### What changed / was found
- ...

### Gaps / open questions
- ...

### Next actions
- [ ] ...
```

---

### Topic → File Mapping

| Topic | File |
|---|---|
| Key lifecycle (issue, rotate, revoke, list) | `status/key-lifecycle.md` |
| Authentication (`VerifyToken`) | `status/authentication.md` |
| Authorization (`Authorize`, scopes, principals) | `status/authorization.md` |
| Storage / ArangoDB backend | `status/storage.md` |
| Crypto / hashing / RNG | `status/crypto.md` |
| gRPC service and proto | `status/grpc-service.md` |
| Cross registration & heartbeat | `status/cross-registration.md` |
| General / cross-cutting | `status/general.md` |
| Recommendations | `status/recommendations/codevaldorg.md` |
````
