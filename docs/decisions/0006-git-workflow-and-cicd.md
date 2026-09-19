# ADR 0006: Git Workflow, Commit Conventions, and CI/CD Pipeline

**Date:** 2026-09-20  
**Status:** ACCEPTED  
**Scope:** Git branching, commit conventions, CI/CD stages, test execution in CI/CD

---

## Problem

The backend project requires clear, enforced discipline for:
- Git branching strategy
- Commit message conventions
- CI/CD pipeline stages and gating
- Testing strategy in CI/CD vs local development

Without explicit decisions, developers may:
- Create commits with unclear metadata (e.g., spurious co-authored-by trailers)
- Push untested changes
- Skip local validation before CI/CD
- Misunderstand which tests run where (local vs CI, SQLite vs PostgreSQL)

---

## Constraints

1. **Trustworthiness:** Lost trust is difficult to recover. CI/CD must gate all quality checks.
2. **Local Efficiency:** Developers must be able to validate locally without PostgreSQL in simple cases.
3. **CI Completeness:** CI/CD must catch real database issues (PostgreSQL in integration/E2E).
4. **No Co-authored Attribution:** All commits must be sole attribution to actual developer.

---

## Decision

### 1. Git Branching Strategy

**Trunk-Based Development with Feature Branches:**

```
main (production-ready)
  ↓ (merge only after CR + all CI checks pass)
develop (integration branch)
  ↓ (merge feature branches here first)
feature/USER-[issue-id]-[short-name]  (e.g., feature/USER-123-add-task-endpoint)
bugfix/USER-[issue-id]-[short-name]   (e.g., bugfix/USER-456-fix-user-validation)
```

**Rules:**
- Feature branches branch off `develop`
- Merge to `develop` requires:
  - ✅ All CI/CD stages pass
  - ✅ Code review approved
  - ✅ No merge commits (rebase + fast-forward)
- Merge to `main` requires:
  - ✅ `develop` branch is stable (all tests pass)
  - ✅ Manual QA sign-off in staging
  - ✅ Release tag created

**Why this pattern:**
- Clear separation: `main` = production, `develop` = integration, feature branches = isolated work
- Enables continuous integration (frequent, small merges)
- Prevents "broken main" scenarios
- Fits small team workflow (no GitHub Flow complexity needed yet)

---

### 2. Commit Message Convention

**Format (conventional commits):**

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code reorganization (no behavior change)
- `test`: Test additions/improvements
- `docs`: Documentation changes
- `chore`: Dependency updates, tooling
- `ci`: CI/CD pipeline changes

**Example:**
```
feat(user): add email validation to CreateUser command

- Validate email format using standard regex
- Return 400 Bad Request on invalid email
- Add unit tests for valid/invalid cases

Closes #123
```

**Critical Rule:**
- ✅ NO `Co-authored-by` trailers in ANY commits
- Each commit represents work by the actual author only
- If pair programming: both developers have their own commits, or one takes attribution

**Why:**
- Maintains clear accountability and authorship
- Simplifies git history for audits
- Aligns with team trust model (lost trust is difficult to recover)
- Conventional commits make changelog generation trivial

---

### 3. CI/CD Pipeline: 7-Stage Gating

**Pipeline Stages (must all pass before merge):**

| # | Stage | Command | Duration | Notes |
|---|-------|---------|----------|-------|
| 1 | Lint & Format | `gofmt`, `go vet`, `go mod tidy` | ~5s | Enforces code style |
| 2 | Build | `go build -v -o app .` | ~10s | Catches syntax/import errors |
| 3 | Unit Tests | `go test -v -race -short ./tests/units/...` | ~30s | No I/O, fast feedback |
| 4 | Security | `nancy sleuth` (or `go list -json -m all \| nancy sleuth -stdin`)  | ~20s | Detects vulnerable dependencies |
| 5 | Integration Tests | `go test -v -race ./tests/integration/... -timeout 60s` | ~60s | SQLite (or PostgreSQL in future) |
| 6 | E2E Tests | `go test -v -race ./tests/e2e/... -timeout 60s` | ~30s | SQLite (or PostgreSQL in future) |
| 7 | Coverage Report | `go test -coverprofile=coverage.out ./tests/...` | ~20s | Threshold ≥80% |

**Total time:** ~3-4 minutes

---

### 4. Local Development vs CI/CD Testing Strategy

#### Local Development (Developer's Machine)

**Requirement:** Must work without PostgreSQL.

**Approach:** Use SQLite in-memory for unit, integration, and E2E tests.

**Why:**
- Zero setup friction (no Docker/Podman required for basic validation)
- Fast feedback loop (tests finish in seconds)
- Identical test patterns and assertions across all tiers
- Catches 95%+ of logic errors before CI/CD

**Local commands:**
```bash
make fmt          # Format & vet
make test-unit    # Unit tests (4/4 pass)
make test-int     # Integration tests with SQLite (8/8 pass)
make test-e2e     # E2E tests with SQLite (8/8 pass)
make test         # All tests (20/20 pass)
```

---

#### CI/CD Pipeline (GitHub Actions)

**Future Phase:** PostgreSQL in integration/E2E tests.

**Current Phase:** SQLite (mirrors local development).

**Why SQLite locally, then PostgreSQL in CI:**
- **Local:** SQLite eliminates setup complexity, enables fast iteration
- **CI/CD:** PostgreSQL (via Testcontainers or service container) validates real database behavior
  - Tests schema migrations
  - Tests PostgreSQL-specific features (JSONB, transactions, constraints)
  - Catches prepared statement bugs that SQLite doesn't catch
- **Trade-off:** Accept that local SQLite tests may miss some PostgreSQL edge cases; CI/CD catches them before merge

**Timeline:**
- Phase 1-5 (current): SQLite everywhere (local + CI/CD)
- Phase 6 (future): Testcontainers in CI/CD for PostgreSQL validation
- Local development: remains SQLite (no Podman required)

---

### 5. Pull Request Gating

**PR can only merge if:**
1. ✅ All 7 CI/CD stages pass
2. ✅ Code review approved (≥1 reviewer)
3. ✅ Branch is up-to-date with `develop`
4. ✅ No merge commits (rebase + fast-forward only)
5. ✅ Commit messages follow convention

**GitHub Settings:**
```yaml
# .github/settings.yml (branch protection)
branches:
  - name: develop
    protection:
      required_status_checks: [lint, build, unit, security, integration, e2e, coverage]
      required_reviews: 1
      require_branches_up_to_date: true
      allow_force_pushes: false
      dismiss_stale_reviews: false
```

---

### 6. Local Pre-Commit Validation

**Before pushing, run:**

```bash
# Format
go fmt ./...

# Lint
go vet ./...

# Tidy modules
go mod tidy

# Run all tests locally
make test

# Verify commit message format
git log -1 --oneline | grep -E '^[a-z]+(\([a-z-]+\))?:'
```

**Rationale:** Catch issues early, reduce CI/CD run count, respect team time.

---

## Alternatives Considered

### A. GitHub Flow (single develop = main)
- **Rejected:** No clear production vs integration distinction
- **Reason:** Backend needs stable release cycle; GitHub Flow better for continuous deployment

### B. Gitflow (main + develop + release branches)
- **Considered:** Clear separation, release management
- **Rejected:** Overkill for small team; Trunk-Based Development simpler

### C. Always Testcontainers/PostgreSQL locally
- **Considered:** Identical test environments
- **Rejected:** High friction (Podman setup required); SQLite fast feedback sufficient
- **Decision:** Phase 1-5 local = SQLite; Phase 6+ CI/CD = PostgreSQL via Testcontainers

### D. Co-authored Commits (multiple developers)
- **Considered:** Recognize pair programming
- **Rejected:** User explicitly demanded sole attribution; lost trust is hard to recover

---

## Implementation Checklist

- [x] Branching strategy documented
- [x] Commit convention enforced (no Co-authored-by)
- [x] CI/CD pipeline: 7 stages, all passing
- [x] Local validation: `make test` passes (20/20)
- [x] Branch protection rules configured (GitHub)
- [x] .gitignore updated (exclude `coverage.out` if needed)
- [ ] Pre-commit hook (optional, `.git/hooks/pre-commit`)
- [ ] GitHub issue templates (optional, prompt for `Closes #NNN`)

---

## Consequences

### Positive
- ✅ Clear accountability (single author per commit)
- ✅ Reproducible builds and tests locally
- ✅ Fast feedback (SQLite local tests < 1 second)
- ✅ Strong CI gating (7 stages, cannot merge if any fail)
- ✅ Predictable git history (conventional commits)
- ✅ Team trust maintained (no spurious attribution)

### Negative
- ⚠️ SQLite-only local tests may miss PostgreSQL edge cases (mitigated: CI/CD will catch)
- ⚠️ 3-4 min CI/CD runtime (acceptable for team size; future optimization: parallel stages)

### Risks
- **Risk:** Developer ignores local validation, pushes broken code
  - **Mitigation:** PR merge protection enforces CI/CD pass
- **Risk:** Accidental commits with `Co-authored-by`
  - **Mitigation:** Pre-commit hook can validate (add in future phase)

---

## Related Decisions

- **ADR 0004:** Test Strategy (Given/When/Then patterns, test tiers)
- **ADR 0005:** Dependency Injection (no framework, main.go assembly)
- **ADR 0001:** Backend Architecture (domain → application → infrastructure)

---

## Questions & Answers

**Q: Why not use Testcontainers locally?**  
A: SQLite is faster, requires no Docker/Podman setup, and enables rapid iteration. PostgreSQL validation happens in CI/CD (future phase).

**Q: Why no merge commits?**  
A: Rebase keeps git history linear, easier to bisect, simpler to understand cause of regressions.

**Q: What if I need to pair program?**  
A: Each developer creates their own commit with their own message. If one developer makes most changes, they take sole attribution; recognize pair programming in PR description instead.

**Q: Can I push to main directly?**  
A: No. All changes must flow through feature branch → `develop` → PR → CR → CI/CD gating → merge to `develop` → manual QA → merge to `main`.

---

**Approved By:** Architecture Review  
**Last Updated:** 2026-09-20  
**Version:** 1.0
