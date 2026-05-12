````prompt
---
agent: agent
---

# Start New Task

> ⚠️ **Before starting a new task**, run `CodeValdOrg/.github/prompts/finish-task.prompt.md` to ensure any in-progress task is properly completed and merged first.

Follow the **mandatory task startup process** for CodeValdOrg tasks:

## Task Startup Process (MANDATORY)

1. **Select the next task**
   - Check `documentation/3-SofwareDevelopment/mvp.md` for the task list and current status
   - Check `documentation/3-SofwareDevelopment/mvp-details/` for detailed specs per topic
   - Check `documentation/1-SoftwareRequirements/requirements.md` for unimplemented functional requirements
   - Follow the onion approach — Layer 1 (raw core) before Layer 2 (integration)
   - Layer 1 priority: `IssueKey` → `VerifyToken` → `Authorize` → `Register with Cross` → `org.token.issued`

2. **Read the specification**
   - Re-read the relevant FRs in `documentation/1-SoftwareRequirements/requirements.md`
   - Re-read the corresponding section in `documentation/2-SoftwareDesignAndArchitecture/architecture.md`
   - Read the task spec in `documentation/3-SofwareDevelopment/mvp-details/{topic-file}.md`
   - Understand how the task fits into the single-interface design (`OrgManager`)
   - Note the mandatory pub/sub requirements (`org.token.issued` on issue,
     `org.token.revoked` on revoke) and the plaintext-once rule for issued keys

3. **Create feature branch from `main`**
   ```bash
   cd /workspaces/CodeVald-AIProject/CodeValdOrg
   git checkout main
   git pull origin main
   git checkout -b feature/ORG-XXX_description
   ```
   Branch naming: `feature/ORG-XXX_description` (lowercase with underscores)

4. **Read project guidelines**
   - Review `.github/instructions/rules.instructions.md`
   - Key rules: interface-first, inject Backend, hash keys before persistence,
     plaintext returned once, publish `org.token.issued` / `org.token.revoked`,
     no agency/task/git/AI/comms logic, context propagation, godoc on all exports

5. **Create a todo list**
   - Break the task into actionable steps
   - Use the manage_todo_list tool to track progress
   - Mark items in-progress and completed as you go

## Pre-Implementation Checklist

Before starting:
- [ ] Relevant FRs and architecture sections re-read
- [ ] Feature branch created: `feature/ORG-XXX_description`
- [ ] Existing files checked — no duplicate types in `models.go` or `errors.go`
- [ ] Understood which file(s) to modify (`internal/manager/`, `internal/server/`,
      `internal/crypto/`, `storage/arangodb/`, `cmd/`, `proto/`)
- [ ] Todo list created for this task

## Development Standards

- **No agency/task/git/AI/comms logic, no frontend serving** — this service
  manages principals, keys, and scopes only
- **`OrgManager` is the only entry point** — gRPC handlers delegate to it
- **Pub/sub is mandatory** — publish `org.token.issued` on every successful
  `IssueKey`, `org.token.revoked` on every successful `RevokeKey`
- **Plaintext secrets** — never logged, never persisted, returned from
  `Issue` / `Rotate` exactly once
- **Backend is injected** — never hardcode ArangoDB connection in manager
- **Every exported symbol** must have a godoc comment
- **Every exported method** takes `context.Context` as the first argument
- **Registration heartbeat** — call `Register` on Cross every 20 seconds

## Git Workflow

```bash
# Create feature branch
git checkout -b feature/ORG-XXX_description

# Regular commits during development
git add .
git commit -m "ORG-XXX: Descriptive message"

# Build validation before merge
go build ./...           # must succeed
go test -v -race ./...   # must pass
go vet ./...             # must show 0 issues
golangci-lint run ./...  # must pass

# Run status update before merging
# Run CodeValdOrg/.github/prompts/statusUpdate.md

# Merge when complete
git checkout main
git merge feature/ORG-XXX_description --no-ff
git branch -d feature/ORG-XXX_description
```

## Success Criteria

- ✅ Relevant FR(s) and architecture doc reviewed
- ✅ Feature branch created from `main`
- ✅ Todo list created with implementation steps
- ✅ Ready to implement following service design rules
````
