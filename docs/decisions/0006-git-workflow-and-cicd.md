# ADR 0006: Git Workflow, Commit Conventions, CI/CD Pipeline, and Governance

**Date:** 2026-09-20  
**Status:** accepted
**Scope:** Git branching, commit conventions, CI/CD pipeline, workflow file governance, documentation audit

---

## Problem

The backend project requires clear, enforced discipline across three domains:
1. **Git & Commits:** Branching strategy, commit message conventions, co-authored rules
2. **CI/CD Pipeline:** 7-stage automated gating, testing strategy (SQLite vs PostgreSQL phases)
3. **Governance:** Workflow file hygiene (single active file), documentation deduplication

Without explicit decisions:
- Developers create unclear commits with spurious metadata
- Untested changes bypass local validation
- Multiple workflow files create confusion ("which one runs?")
- Documentation drift (duplicate files with diverging content)

---

## Constraints

1. **Trustworthiness:** Lost trust is difficult to recover. Required quality
   checks must be enforced consistently.
2. **Local Efficiency:** Developers must validate locally without Docker/PostgreSQL for fast feedback.
3. **CI Completeness:** CI/CD must expose all quality checks (lint, build,
   tests, security, coverage) and clearly distinguish blocking from
   informational jobs.
4. **Documentation Integrity:** One source of truth per topic; no templates or duplicate files.

---

## Decision

### 1. Git Branching Strategy

**Feature-branch workflow with `develop` and `main`:**

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
  - ✅ All required CI jobs pass
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
- The repository currently uses feature branches with separate integration and
  release branches. The label "Gitflow" is not used here to avoid ambiguity
  with the rejected full Gitflow model.

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
- NO `Co-authored-by` trailers in ANY commits, for new commits going forward.
  No CI check enforces this today ; 7 existing commits on `origin/develop`
  (e.g. `66335a8`) contain the trailer. A history rewrite to remove them is
  planned.
- Each commit represents work by the actual author only
- If pair programming: both developers have their own commits, or one takes attribution

**Why:**
- Maintains clear accountability and authorship
- Simplifies git history for audits
- Aligns with team trust model (lost trust is difficult to recover)
- Conventional commits make changelog generation trivial

---

### 3. CI/CD Pipeline: 7-Stage Gating

**Pipeline Stages (must all pass before merge, sauf l'étape 4 — voir note) :**

| # | Stage | Command | Duration | Notes |
|---|-------|---------|----------|-------|
| 1 | Lint & Format | `gofmt`, `go vet`, `go mod tidy` | ~5s | Enforces code style |
| 2 | Build | `go build -v -o app .` | ~10s | Catches syntax/import errors |
| 3 | Unit Tests | `go test -v -race -short ./tests/units/...` | ~30s | No I/O, fast feedback |
| 4 | Security | `nancy sleuth` | ~20s | Échoue actuellement (`401 Unauthorized` sur OSS Index) à chaque run ; l'erreur est masquée par `\|\| true` dans `ci.yml` et ne bloque jamais le job. Ne détecte donc aucune dépendance vulnérable en pratique. Suivi : `Q-05`. |
| 5 | Integration Tests | `go test -v -race ./tests/integration/... -timeout 60s` | ~90s | PostgreSQL via Testcontainers |
| 6 | E2E Tests | `go test -v -race ./tests/e2e/... -timeout 60s` | ~60s | SQLite in-memory, HTTP contract coverage |
| 7 | Coverage Report | `go test -coverprofile=coverage.out ./...` | ~30s | Threshold reported, non-blocking |

**Total time:** ~10-11 minutes, mesuré sur le run `develop` du 2026-09-29
(22:02–22:13 UTC, `gh run view` sur le commit `d2fede3`). L'estimation
précédente (~4-5 min) n'était plus à jour.

---

### 4. Local Development vs CI/CD Testing Strategy

#### Phase 1-5: SQLite (Legacy)

**Approach:** Use SQLite in-memory for unit, integration, and E2E tests.

**Rationale:**
- Zero setup friction (no Docker/Podman required)
- Fast feedback loop (tests finish in ~500ms)
- Sufficient for 95% of logic bugs

#### PostgreSQL integration validation (Current)

**Requirement:** Integration tests must validate real PostgreSQL behavior when a
Docker or Podman provider is available.

**Approach:** Use Testcontainers to spin up PostgreSQL container per test run.

**Rationale:**
- Validates PostgreSQL-specific features (constraints, JSONB, transactions)
- Tests schema migrations against real database
- CI/CD uses same setup: PostgreSQL via service container
- Catches edge cases that SQLite misses (e.g., prepared statement differences)

**Trade-offs:**
- ⚠️ Slower local tests (~10-15 seconds per test run vs ~500ms with SQLite)
- ✅ Production-parity testing (PostgreSQL everywhere)
- ✅ CI/CD matches local dev (no surprise failures)

**Local Development:**
- Unit and E2E tests remain runnable without a container provider.
- PostgreSQL integration tests are skipped with an explicit reason when no
  provider is available and remain blocking when a provider is available.

**Local commands:**
```bash
make fmt              # Format & tidy modules
make test-unit        # Unit tests
make test-integration # PostgreSQL integration tests
make test-e2e         # HTTP E2E tests with SQLite
make test             # All packages; PostgreSQL tests require a provider
```

---

#### CI/CD Pipeline (GitHub Actions)

The integration job's workflow file also declares a `services: postgres` block
and a `DATABASE_URL` env var, but `tests/integration/persistence/setup_test.go`
never reads `DATABASE_URL`: it starts its own PostgreSQL container via
Testcontainers instead, ignoring the runner's service container. The E2E job
executes `tests/e2e` directly; those tests use SQLite in-memory by design and
do not require PostgreSQL at all.

**Historical rationale for the earlier SQLite transition:**
- **Local:** SQLite eliminates setup complexity, enables fast iteration
- **CI/CD:** PostgreSQL (via Testcontainers or service container) validates real database behavior
  - Tests schema migrations
  - Tests PostgreSQL-specific features (JSONB, transactions, constraints)
  - Catches prepared statement bugs that SQLite doesn't catch
- **Trade-off:** Accept that local SQLite tests may miss some PostgreSQL edge cases; CI/CD catches them before merge

**Historical note:**
- Earlier phases used SQLite more broadly.
- The current repository uses PostgreSQL for repository integration tests when a
  provider is available, and SQLite in memory for E2E tests.
- A skipped local PostgreSQL test is not evidence that the integration test ran.

---

### 5. Pull Request Gating

**PR can only merge if the hosting-service branch protection confirms:**
1. ✅ The required CI jobs pass
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
      required_status_checks: [lint, build, unit, integration, e2e]
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
- [ ] Commit convention (no Co-authored-by) documented for new commits ;
      non enforced by any check ; 7 existing commits on `origin/develop`
      contain the trailer, history rewrite planned
- [x] CI/CD pipeline contains seven jobs plus a summary
- [x] A successful remote run is independently verified for the current commit
- [ ] Branch protection rules are independently verified in GitHub
- [x] .gitignore updated (exclude `coverage.out` if needed)
- [x] Pre-commit/pre-push hooks exist (`.githooks/pre-commit`, `.githooks/pre-push`) ; activation locale non automatique, nécessite `git config core.hooksPath .githooks`
- [ ] GitHub issue templates (optional, prompt for `Closes #NNN`)

---

## Consequences

### Positive
- ✅ Clear accountability (single author per commit)
- ✅ Reproducible builds and tests locally
- ✅ Fast feedback (SQLite local tests < 1 second)
- ✅ CI jobs provide lint, build, unit, integration and E2E validation
- ✅ Predictable git history (conventional commits)
- ✅ Team trust maintained (no spurious attribution)

### Negative
- ⚠️ SQLite E2E tests may miss PostgreSQL edge cases; integration tests cover
  those cases only when PostgreSQL actually runs
- ⚠️ ~10-11 min CI/CD runtime mesuré (voir section 3) ; optimisation future
  possible : parallélisation des étapes

### Risks
- **Risk:** Developer ignores local validation, pushes broken code
  - **Mitigation:** PR merge protection enforces CI/CD pass
- **Risk:** Accidental commits with `Co-authored-by`
  - **Mitigation:** Pre-commit hook can validate (add in future phase)

---

## Governance Rules

### Single Active Workflow File

**Rule:** Only ONE workflow file exists at any time.

- ✅ `.github/workflows/ci.yml` — The active, current workflow
- ❌ NO template files (e.g., `ci-phase6-postgresql.yml`)
- ❌ NO future-state files (e.g., `ci-alternative.yml`)

**Rationale:**
- One file = no ambiguity about which workflow runs
- Phase transitions are deliberate commits, not speculative code
- No maintenance drift (single source of truth)

The current workflow is updated in place. A future database-strategy change
must update this file and the workflow together; no template workflow should be
created.

---

### Documentation Audit & Deduplication

**Rule:** One source of truth per topic (per `documentation-guide.md`).

**Process:**
1. Identify all documents
2. Compare content for overlaps
3. Determine canonical source per topic
4. Remove duplicates, keep owner document
5. Verify no docs have diverged

**Example Applied (Session 2026-09-20):**
- Identified: `CI_CD_PIPELINE.md` + `ADR 0006` both covered 7-stage pipeline
- Decision: Remove `CI_CD_PIPELINE.md` (ADR is canonical for decisions)
- Result: One document per topic, no maintenance divergence

**Audit Cadence:**
- At each major phase transition (estimated Phase 6+)
- Check: no v1/v2/final files, all links work, ADRs coherent, handoff matches code

---

## Related Decisions

- **ADR 0004:** Test Strategy (Given/When/Then patterns, test tiers)
- **ADR 0005:** Dependency Injection (no framework, main.go assembly)
- **ADR 0001:** Backend Architecture (domain → application → infrastructure)

---

## Questions & Answers

**Q: Why not use Testcontainers for every local test?**
A: SQLite is faster for E2E feedback. PostgreSQL integration tests remain
available locally when Docker or Podman is configured and are expected in CI.

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
