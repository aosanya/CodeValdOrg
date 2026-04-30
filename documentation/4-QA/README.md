# 4 — QA

## Overview

This section covers testing strategy, acceptance criteria, and quality assurance for CodeValdOrg.

---

## Index

| Document | Description |
|---|---|
| _(none yet)_ | Test plans and QA artifacts will be added as tasks are implemented |

---

## Testing Standards

All contributions must satisfy:

| Check | Command | Requirement |
|---|---|---|
| Build | `go build ./...` | Must succeed — no compilation errors |
| Unit tests | `go test -v -race ./...` | All tests green; no data races |
| Static analysis | `go vet ./...` | 0 issues |
| Linting | `golangci-lint run ./...` | Must pass |
| Coverage | `go test -coverprofile=coverage.out ./...` | Target ≥ 85% on exported functions |

---

## Test Structure Convention

Tests live alongside source files using Go's standard `_test.go` convention:

```
org_manager_test.go            ← OrgManager domain logic tests (in-memory double)
internal/
  token/
    issue_test.go              ← Token issuance unit tests
    introspect_test.go         ← Introspection unit tests
  scope/
    scope_test.go              ← Scope registration unit tests
  server/
    server_test.go             ← gRPC handler tests via bufconn
storage/
  arangodb/
    backend_test.go            ← ArangoDB integration tests (testcontainers)
```

Integration tests that require ArangoDB must use `t.Skip()` when `ORG_ARANGO_ENDPOINT` is not set.

---

## Three Test Layers

| Layer | Scope | Gate |
|---|---|---|
| Unit | In-memory `DataManager` double + `Clock` mock; all domain logic | Every PR |
| Integration | Real ArangoDB via `testcontainers-go`; per-test database isolation | Every PR |
| Conformance | RFC 6749 / 7636 / 7662 / 7009 / 8414 end-to-end flows | `main` merges only |

---

## Acceptance Criteria per Task

See the `### Tests` section of each task file in [../3-SofwareDevelopment/mvp-details/](../3-SofwareDevelopment/mvp-details/) for the full test matrix per MVP task.

Full testing strategy — including coverage targets, NFR-002 benchmark recipe, and the complete
negative-test checklist — is in [../3-SofwareDevelopment/mvp-details/testing-strategy.md](../3-SofwareDevelopment/mvp-details/testing-strategy.md).
