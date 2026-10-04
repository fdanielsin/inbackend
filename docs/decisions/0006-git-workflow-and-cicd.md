# ADR 0006: Git Workflow, Commit Conventions, CI/CD Pipeline, and Governance

**Date:** 2026-09-20  
**Status:** accepted
**Scope:** Git branching, commit conventions, CI/CD pipeline, workflow file governance, documentation audit

---

## Problem

The backend project requires clear, enforced discipline across three domains:
1. **Git & Commits:** Branching strategy, commit message conventions, co-authored rules
2. **CI/CD Pipeline:** 7-stage automated gating, testing strategy (PostgreSQL via Testcontainers)
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
feature/<subject>   (e.g., feature/localized-api-messages)
chore/<subject>     (e.g., chore/replace-zap-with-slog)
```

**Branch names ([Conventional Branch](https://conventional-branch.github.io/),
user decision of 2026-10-01):** `feature/`, `bugfix/`, `hotfix/`, `release/` or
`chore/` (refactoring and tests are `chore/`), then a subject of 3 to 4
lowercase words separated by hyphens, with no batch or ticket number.

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

**Format ([Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/),
in English, user decision of 2026-10-01):**

```
<type>(<scope>): <description>

<body>

<footer>
```

The description is lowercase, imperative, without a final period, about 50
characters (72 at most). A breaking change adds `!` after the type or scope
and/or a `BREAKING CHANGE:` footer (e.g. `refactor(logger)!: replace zap with
slog`). The body is optional (what and why, at most 3 lines of 72 characters);
tests and measurements go in the pull request. One commit per logical change.

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
feat(user): validate email in create command

Return 400 on a malformed email instead of storing it.

Closes #123
```

**Critical Rule:**
- NO `Co-authored-by` trailers in ANY commits, for new commits going forward.
  No CI check enforces this. The existing trailers were removed by the history
  rewrite of 2026-10-01.
- Each commit represents work by the actual author only
- If pair programming: both developers have their own commits, or one takes attribution

**Why:**
- Maintains clear accountability and authorship
- Simplifies git history for audits
- Aligns with team trust model (lost trust is difficult to recover)
- Conventional commits make changelog generation trivial

---

### 3. CI/CD Pipeline: 7-Stage Gating

**Pipeline Stages (must all pass before merge) :**

| # | Stage | Command | Duration | Notes |
|---|-------|---------|----------|-------|
| 1 | Lint & Format | `make lint` (golangci-lint, gofumpt, goimports), `make lint-api` (Redocly), `make lint-ci` (actionlint), `go mod tidy` | ~30s | Enforces code style, static analysis, the OpenAPI contract and the workflow syntax |
| 2 | Build | `go build -v -o app .` | ~10s | Catches syntax/import errors |
| 3 | Unit Tests | `go test -v -race -short ./tests/units/...` | ~30s | No I/O, fast feedback |
| 4 | Security | `make vuln` (govulncheck) | ~20s | Bloquant : échoue si une vulnérabilité est atteignable par le code. Remplace `nancy` (`Q-05`). |
| 4b | Docker | `docker compose build`, `docker compose up -d --wait`, `curl -f /health`, `curl -f /ready`, `curl -f /openapi.yaml`, `docker compose down -v` | ~1-2 min | Bloquant : l'image se construit et la pile démarre avec PostgreSQL (`Q-03`) |
| 4c | Publish image | `docker/build-push-action` vers `ghcr.io/fdanielsin/inbackend` (tags sha complet, `develop`, `v*`) | ~1 min | Après `summary` vert, sur push `develop` ou tag `v*` seulement, `packages: write` limité à ce job ; `prune-image` (`scripts/prune-image-versions.sh`) garde `develop`, les `v*` et 10 versions sha, et ne supprime que si `PRUNE_IMAGE_VERSIONS=true` (sinon liste seulement) ; `concurrency` par ref avec annulation du run précédent (un push plus récent annule l'ancien, PR comprises), pour que `develop` ne repointe jamais sur une image plus ancienne |
| 5 | Integration Tests | `go test -v -race ./tests/integration/... -timeout 5m` | ~90s | PostgreSQL via Testcontainers |
| 6 | E2E Tests | `go test -v -race ./tests/e2e/... -timeout 5m` | ~60s | PostgreSQL via Testcontainers, HTTP contract coverage through `app.NewRouter` |
| 7 | Coverage | `make cover` puis envoi à Codecov | ~30s | Bloquant : échoue sous 85 % de couverture de `internal/` (même commande qu'en local) |

**Outillage de lint :** `golangci-lint` (v2.14.0, config `.golangci.yml`),
`govulncheck` (v1.8.0), `actionlint` (v1.7.7) et `@redocly/cli` (v2.55.0, config
`docs/specs/api/redocly.yaml` et `.redocly.lint-ignore.yaml`) sont épinglés dans le `Makefile` et lancés avec
`go run ...@version` / `npx`, à l'identique en local et en CI. Une dépendance
d'outil Go (`go get -tool`) a été écartée : elle faisait monter des
dépendances de production (`x/crypto`, `x/net`, `protobuf`, `otel`). Chaque
règle désactivée l'est avec sa justification dans le fichier de configuration.

**Version de Go (`Q-03`) :** `go.mod` porte `go 1.26.0` et `toolchain go1.26.8`,
pour la sécurité : govulncheck signale 25 vulnérabilités atteignables de la
stdlib en 1.25.0, 16 en 1.26.0 et 0 en 1.26.8. Tous les jobs
(`go-version-file: go.mod`) et le Dockerfile utilisent cette seule version, et
govulncheck analyse la stdlib réellement embarquée.

**Total time:** ~10-11 minutes, mesuré sur le run `develop` du 2026-09-29
(22:02–22:13 UTC, `gh run view` sur le commit `d2fede3`). L'estimation
précédente (~4-5 min) n'était plus à jour.

---

### 4. Local Development vs CI/CD Testing Strategy

#### Historique : SQLite (retiré, Q-06)

Les phases 1 à 5 utilisaient SQLite en mémoire pour les tests E2E (sans Docker,
~500 ms). Il a été retiré : il masquait les différences avec PostgreSQL. Les
dépendances `gorm.io/driver/sqlite` et `go-sqlite3` ne sont plus dans `go.mod`.

#### PostgreSQL integration validation (Current)

**Requirement:** Integration tests must validate real PostgreSQL behavior when a
Docker or Podman provider is available.

**Approach:** Use Testcontainers to spin up PostgreSQL container per test run.

**Rationale:**
- Validates PostgreSQL-specific features (constraints, JSONB, transactions)
- Tests schema migrations against real database
- CI/CD uses the same setup: Testcontainers, no runner service container

**Trade-offs:**
- ⚠️ Local tests need Docker or Podman and take longer (container start)
- ✅ Production-parity testing (PostgreSQL everywhere)
- ✅ CI/CD matches local dev (no surprise failures)

**Local Development:**
- Unit tests remain runnable without a container provider.
- PostgreSQL integration and E2E tests are skipped with an explicit reason when
  no provider is available locally. With `CI=true` (GitHub Actions) they fail
  instead of being skipped (Q-09).

**Local commands:**
```bash
make fmt              # Format & tidy modules
make test-unit        # Unit tests
make test-integration # PostgreSQL integration tests
make test-e2e         # HTTP E2E tests on PostgreSQL (requires Docker or Podman)
make test             # All packages; PostgreSQL tests require a provider
```

---

#### CI/CD Pipeline (GitHub Actions)

The integration, E2E and coverage jobs start their own PostgreSQL through
Testcontainers (`tests/testsupport/postgres.go`) and declare no `services:`
block nor `DATABASE_URL`.

**Historical note:**
- E2E tests no longer use SQLite (Q-06): they call `app.NewRouter` on PostgreSQL.
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
# Lint and formatting check (same command as CI)
make lint
make lint-api

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

### C. SQLite locally, PostgreSQL in CI
- **Considered:** Fast local feedback without Docker
- **Rejected (Q-06):** SQLite hides PostgreSQL differences
- **Decision:** Testcontainers/PostgreSQL everywhere, locally and in CI

### D. Co-authored Commits (multiple developers)
- **Considered:** Recognize pair programming
- **Rejected:** User explicitly demanded sole attribution; lost trust is hard to recover

---

## Implementation Checklist

- [x] Branching strategy documented
- [x] Commit convention (Conventional Commits, no Co-authored-by) documented ;
      not enforced by any check ; the trailers of existing commits were
      removed by the history rewrite of 2026-10-01
- [x] CI/CD pipeline contains seven jobs plus a summary
- [x] A successful remote run is independently verified for the current commit
- [ ] Branch protection rules are independently verified in GitHub
- [x] .gitignore updated (exclude `coverage.out` if needed)
- [x] Pre-commit/pre-push hooks exist (`.githooks/pre-commit`, `.githooks/pre-push`) ; activation locale non automatique, nécessite `git config core.hooksPath .githooks` ; le pre-push refuse aussi tout push vers `main` sauf `RELEASE_TAG` sur le commit poussé (voir `docs/deployment.md`)
- [ ] GitHub issue templates (optional, prompt for `Closes #NNN`)

---

## Consequences

### Positive
- ✅ Clear accountability (single author per commit)
- ✅ Reproducible builds and tests locally
- ✅ CI jobs provide lint, build, unit, integration and E2E validation
- ✅ Predictable git history (conventional commits)
- ✅ Team trust maintained (no spurious attribution)

### Negative
- ⚠️ Local PostgreSQL tests need Docker or Podman and are skipped without a
  provider (not a proof that they ran)
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
- **ADR 0005:** Dependency Injection (no framework, `internal/app` assembly)
- **ADR 0001:** Backend Architecture (domain → application → infrastructure)

---

## Questions & Answers

**Q: Why Testcontainers for every database test?**
A: One real PostgreSQL everywhere (Q-06). Locally, Docker or Podman is needed;
without a provider the tests are skipped with their cause, never counted as passed.

**Q: Why no merge commits?**
A: Rebase keeps git history linear, easier to bisect, simpler to understand cause of regressions.

**Q: What if I need to pair program?**  
A: Each developer creates their own commit with their own message. If one developer makes most changes, they take sole attribution; recognize pair programming in PR description instead.

**Q: Can I push to main directly?**  
A: No. All changes must flow through feature branch → `develop` → PR → CR → CI/CD gating → merge to `develop` → manual QA → merge to `main`.

---

**Approved By:** Architecture Review  
**Last Updated:** 2026-09-30  
**Version:** 1.0
