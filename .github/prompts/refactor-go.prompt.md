````prompt
---
agent: agent
---

# Refactor Go Code

Guides a safe, incremental Go refactoring for **CodeValdOrg**.

---

## When to Refactor

- File exceeds **500 lines** (hard limit)
- Function exceeds **50 lines**
- Multiple concerns in one file (e.g., manager + storage in the same file)
- Business logic leaked into `cmd/main.go` or gRPC handler
- Agency / task / git / AI / comms / frontend logic crept into the service — remove it
- Plaintext secret handling spread across multiple files — consolidate into
  `internal/crypto/` and `internal/manager/`

---

## Refactoring Workflow

### Step 1: Understand the File

```bash
wc -l internal/manager/manager.go
grep -n "^func " internal/manager/manager.go
```

### Step 2: Plan the Split

Identify distinct responsibilities. For CodeValdOrg, typical splits:

```
internal/manager/manager.go      # OrgManager concrete implementation
internal/server/server.go        # gRPC handler + server lifecycle
internal/config/config.go        # Configuration loading
internal/crypto/hasher.go        # KeyHasher (bcrypt / argon2) + constant-time compare
internal/crypto/rng.go           # Random token generator
storage/arangodb/storage.go      # ArangoDB Backend implementation
errors.go                        # All exported error types
models.go                        # Key, IssueKeyRequest, Principal, Scope, AuthorizeRequest, Decision
cmd/main.go                      # Wiring only — no logic
```

### Step 3: Extract — One File at a Time

1. Create the new file with its package declaration
2. Move types / functions
3. Update imports
4. Run `go build ./...` — must succeed after each file move
5. Run `go test -v -race ./...`

### Step 4: Handle Shared Dependencies

If a type is used across multiple files, move it to `models.go`.
If an error type is referenced by multiple packages, keep it in `errors.go` at module root.

### Step 5: Validate

```bash
go build ./...           # must succeed
go vet ./...             # must show 0 issues
go test -v -race ./...   # must pass
golangci-lint run ./...  # must pass
```

---

## Specific Concerns for CodeValdOrg

### Remove domain logic that doesn't belong here
- Agency/task/git/AI/comms/frontend code does NOT belong in this service
- Move to the appropriate service or delete if no longer needed
- After removal: `go build ./...` must still succeed

### Keep Cross registration separate from business logic
- Cross registration/heartbeat lives in `cmd/main.go` or a dedicated
  `internal/registrar/` package
- Never mix registration retry logic with `IssueKey` / `VerifyToken` / `Authorize`
  business logic

### Isolate cryptographic code
- All hashing, constant-time comparison, and token generation lives in
  `internal/crypto/`
- `internal/manager/` depends on the `KeyHasher` interface, never on `bcrypt` /
  `argon2` directly
- This keeps the surface area that handles plaintext secrets small and
  reviewable

### Never break the plaintext-once contract during a refactor
- `IssueKey` / `RotateKey` must still return the plaintext exactly once
- `GetKey` / `ListKeys` must still return metadata only
- If a refactor changes these signatures, update the architecture doc in the
  same PR
````
