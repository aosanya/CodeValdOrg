````prompt
---
agent: agent
---

# Research & Documentation Gap Analysis Prompt

## Purpose
This prompt guides a structured Q&A session to explore and complete documentation
for any feature or architectural component in **CodeValdOrg** through
**one question at a time**, allowing for deep dives into specific topics.

---

## 🔄 MANDATORY REFACTOR WORKFLOW (ENFORCE BEFORE ANY RESEARCH SESSION)

**BEFORE starting any research or writing new task documentation:**

### Step 1: CHECK File Size
```bash
wc -l documentation/3-SofwareDevelopment/mvp-details/{topic-file}.md
```

### Step 2: IF >500 lines OR individual ORG-XXX.md files exist:

**a. CREATE folder structure:**
```bash
documentation/3-SofwareDevelopment/mvp-details/{domain-name}/
├── README.md              # Domain overview, architecture, task index (MAX 300 lines)
├── {topic-1}.md           # Topic-based grouping of related tasks (MAX 500 lines)
└── {topic-2}.md
```

**b. CREATE README.md** with:
- Domain overview
- Architecture summary
- Task index with links

**c. SPLIT content by TOPIC (NOT by task ID)**

**d. MOVE architecture diagrams** → `architecture/` subfolder

**e. MOVE examples** → `examples/` subfolder

### Step 3: ONLY THEN add new task content to appropriate topic file

---

## 🛑 STOP CONDITIONS (Do NOT proceed until fixed)

- ❌ **Domain file exceeds 500 lines** → **MUST refactor first**
- ❌ **README.md exceeds 300 lines** → **MUST split content**
- ❌ **Individual `ORG-XXX.md` files exist** → **MUST consolidate by topic**

---

## Instructions for AI Assistant

Conduct a comprehensive documentation gap analysis through **iterative
single-question exploration**. Ask ONE question at a time, wait for the
response, then decide whether to:

- **Go Deeper**: Ask follow-up questions on the same topic
- **Take Note**: Record a gap for later exploration
- **Move On**: Proceed to the next topic area
- **Review**: Summarise what we've learned and identify remaining gaps

---

## Research Framework

### Current Technology Stack (Reference)

```yaml
Service:
  Language: Go 1.21+
  Module: github.com/aosanya/CodeValdOrg
  gRPC: google.golang.org/grpc
  Storage: ArangoDB (arangodb/go-driver)
  Crypto: golang.org/x/crypto (bcrypt / argon2) + crypto/subtle
  Registration: CodeValdCross OrchestratorService.Register RPC

Key interfaces:
  - OrgManager: IssueKey, RevokeKey, RotateKey, GetKey, ListKeys, VerifyToken, Authorize
  - Backend: Insert, Get, FindByHash, Revoke, List, GetPrincipal
  - KeyHasher: Hash, Compare (constant-time)

Cross-service events:
  Produces: org.token.issued, org.token.revoked
  Consumes: (none in Layer 1)

Documentation structure:
  1-SoftwareRequirements:
    requirements: documentation/1-SoftwareRequirements/requirements.md
    introduction: documentation/1-SoftwareRequirements/introduction/
  2-SoftwareDesignAndArchitecture:
    architecture: documentation/2-SoftwareDesignAndArchitecture/architecture.md
  3-SofwareDevelopment:
    mvp: documentation/3-SofwareDevelopment/mvp.md
    mvp-details: documentation/3-SofwareDevelopment/mvp-details/
  4-QA:
    qa: documentation/4-QA/README.md
```

### Research Areas (in priority order)

1. **Principal & key data model** — what fields does a `Principal` / `Key` need?
2. **Key issuance flow** — generate → hash → persist → publish sequence
3. **Token verification flow** — how `VerifyToken` maps a plaintext token to a `Principal`
4. **Scope model** — how scopes are defined, composed, and evaluated against resources
5. **Authorization flow** — `Authorize(principal, scope, resource)` decision logic
6. **Key rotation & revocation** — how `RotateKey` / `RevokeKey` interact with downstream caches
7. **Cross registration** — what routes does Org declare to Cross?
8. **ArangoDB schema** — collection names, document structures, indexes for keys / principals / scopes
9. **Error handling** — what error cases need typed errors?
10. **gRPC proto definition** — `OrgService` method signatures
11. **Configuration** — what env vars / YAML keys does the service need (signing secrets, DB, Cross addr)?
12. **Testing strategy** — unit tests with mock `Backend` and `KeyHasher`; integration tests against ArangoDB?
13. **Threat model** — which attack scenarios drove the plaintext-once and constant-time-compare rules?
````
