````prompt
---
agent: agent
---

# Debug a CodeValdOrg Issue

## How to Use This Prompt

When you encounter a bug in CodeValdOrg, describe the failing behaviour and use
the guidelines below to add targeted debug logging, isolate the cause, and
clean up before merging.

> ⚠️ **Never log plaintext tokens, keys, passwords, or key hashes.** When a
> debug print needs to reference a credential, log its `KeyID` (and optionally
> the first/last 4 characters of the hash) — never the full secret.

## Common Failure Scenarios

### Scenario 1: `IssueKey` Succeeds but Downstream Services Don't See the New Key
**Symptom**: Key is stored in ArangoDB but other services reject requests using
it because their cache doesn't have the new key metadata
**Cause**: `org.token.issued` publish is missing or the Cross client is nil
**Check**: Confirm `m.crossClient.Publish(...)` is called after `backend.Insert`;
check Cross client wiring in `cmd/main.go`

### Scenario 2: `Register` Always Fails with `DeadlineExceeded`
**Symptom**: Heartbeat loop logs `ping CodeValdCross at :50052: rpc error: code = DeadlineExceeded`
**Cause**: CodeValdCross is not running, or wrong address configured
**Check**: Verify `CROSS_ADDR` env var; confirm CodeValdCross is up before starting CodeValdOrg

### Scenario 3: `VerifyToken` Returns `ErrKeyNotFound` for a Key That Was Just Issued
**Symptom**: Client got a token from `IssueKey`, but the very next `VerifyToken`
call fails
**Cause**: The hash stored and the hash computed at verify time don't match —
typically because `IssueKey` stored the plaintext by accident, or
`VerifyToken` is hashing a different input (e.g., including a prefix)
**Check**: Confirm `backend.Insert` receives the hashed value, not the plaintext;
confirm `VerifyToken` hashes the exact token the client sent

### Scenario 4: `Authorize` Allows a Call That Should Be Denied
**Symptom**: A caller without the required scope receives `Decision{Allowed: true}`
**Cause**: Scope comparison is case-sensitive / prefix-matching incorrectly, or
the principal's scopes were not reloaded after an update
**Check**: Print the principal's scopes (IDs only, not secrets) and the
requested `(scope, resource)` tuple; confirm `Scopes.Allows` logic

### Scenario 5: Context Cancellation Not Respected in Heartbeat Loop
**Symptom**: Service does not shut down cleanly; heartbeat goroutine leaks
**Cause**: Missing `ctx.Done()` select case in the registration loop
**Check**: Ensure heartbeat loop has `case <-ctx.Done(): return` in the select

### Scenario 6: Backend Not Injected — Nil Pointer Panic
**Symptom**: `nil pointer dereference` in `internal/manager/manager.go`
**Cause**: `cmd/main.go` did not construct and inject the `Backend` (or
`KeyHasher`) before calling `NewOrgManager`
**Check**: Trace wiring in `cmd/main.go`; ensure `arangodb.NewBackend(cfg)` and
`crypto.NewHasher(cfg)` are called first

### Scenario 7: Revoked Key Still Accepted
**Symptom**: A key that was revoked via `RevokeKey` continues to pass `VerifyToken`
**Cause**: `org.token.revoked` was not published, so downstream caches
still hold the old `Allowed: true` decision — or `VerifyToken` is not
checking the `Revoked` flag on the stored key
**Check**: Confirm `RevokeKey` publishes the event AND sets the key's `Revoked`
flag in storage; confirm `VerifyToken` returns `ErrKeyRevoked` when the flag is set

## Debug Print Guidelines

### Prefix Format
All debug prints MUST be prefixed with: `[ORG-XXX]`

### Go
```go
log.Printf("[ORG-XXX] Function called: %s with args: %+v", functionName, safeArgs)
log.Printf("[ORG-XXX] State before: %+v", safeState)
log.Printf("[ORG-XXX] Error in operation: %v", err)
```

`safeArgs` / `safeState` means values with plaintext secrets redacted. Never
print a full token, key, hash, or password.

### Strategic Placement

Add debug prints at:

1. **Function Entry Points**
   - `log.Printf("[ORG-XXX] IssueKey called: principalID=%s scopes=%v", req.PrincipalID, req.Scopes)`

2. **After Storage Operations**
   - `log.Printf("[ORG-XXX] Key inserted: keyID=%s", key.ID)`

3. **Before and After Pub/Sub Publish**
   - `log.Printf("[ORG-XXX] Publishing org.token.issued: keyID=%s", key.ID)`

4. **Authorization Decisions**
   - `log.Printf("[ORG-XXX] Authorize: principalID=%s scope=%s resource=%s allowed=%v", req.PrincipalID, req.Scope, req.Resource, decision.Allowed)`

5. **Heartbeat Loop**
   - `log.Printf("[ORG-XXX] Register: attempt addr=%s err=%v", addr, err)`

6. **Error Handling**
   - `log.Printf("[ORG-XXX] IssueKey failed: %v", err)`

### What NOT to Debug

- Simple getters
- Trivial utility functions
- Already well-instrumented production logs
- **Anything containing a plaintext secret or full key hash**

### Debug Print Structure

Use descriptive messages that answer:
1. **WHERE**: Which function/block is executing
2. **WHAT**: What operation is happening
3. **VALUES**: Relevant variable values (redacted for secrets)

**Good Example:**
```go
log.Printf("[ORG-XXX] IssueKey: principalID=%s keyID=%s published=%v", req.PrincipalID, key.ID, published)
```

**Bad Example:**
```go
log.Printf("here: token=%s", plaintext) // ❌ leaks plaintext
```
````
