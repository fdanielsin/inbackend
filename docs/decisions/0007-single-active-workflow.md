# ADR 0007: Single Active Workflow File

**Date:** 2026-09-20  
**Status:** ACCEPTED  
**Scope:** GitHub Actions workflow governance, avoiding template/future state files

---

## Problem

Multiple workflow files create ambiguity and confusion:
- Which workflow is currently active?
- Should developers reference the "phase 1-5" or "phase 6+" version?
- Template files that are never used waste maintenance overhead
- Future-state files in the codebase encourage speculation over clear decisions

**Decision Made After Mistake:**
Initially created both `ci.yml` (SQLite, Phase 1-5) and `ci-phase6-postgresql.yml` (template, Phase 6+).  
This violated the "one source of truth" principle and confused git history.

---

## Decision

### 1. Single Workflow File, No Templates

**Only ONE workflow file exists at any time:**
- `.github/workflows/ci.yml` — The active, current, production workflow

**No template files, no "future state" files.**

### 2. Workflow Updates Happen In-Place

When transitioning to a new phase (e.g., Phase 6 with PostgreSQL):
1. Update the single `ci.yml` file with new configuration
2. Commit with clear message: `ci: migrate from SQLite to PostgreSQL (Phase 6)`
3. Delete the commit message which should clearly state what changed and why

**Example commit message (NOT what we do now, but the pattern for Phase 6):**
```
ci: migrate Phase 1-5 to Phase 6 PostgreSQL pipeline

- Replace SQLite with PostgreSQL 16 service
- Add database migration tests
- Increase coverage timeout to 60s (Testcontainers slower)
- Local development unchanged (still SQLite)

Phase 1-5 SQLite workflow replaced. Rollback: git revert [commit hash]
```

### 3. Phase Documentation Lives in `docs/`, Not `.github/`

**Not:** `.github/WORKFLOW_MIGRATION.md`, `.github/workflows/ci-phase6-postgresql.yml`

**Yes:** `docs/CI_CD_PIPELINE.md` explains operational details, phase transitions are discussed in:
- **ADR 0006:** Git Workflow & CI/CD Pipeline (high-level strategy)
- **CI_CD_PIPELINE.md:** Operational guide (7 stages, how to run locally)

When ready for Phase 6, update these docs. No template workflows.

---

## Rationale

### Why One Active File?

| Approach | Pros | Cons |
|----------|------|------|
| **Single active file** | Clear, no ambiguity, easy to debug, minimal maintenance | Requires planning before phase transition |
| **Template files** | Can show "future state" | Confusing (which one runs?), clutter, false documentation |
| **Branch per phase** | Isolates changes | Fragmented history, harder to compare |

**Decision:** Single active file. Phase transitions are planned, documented, and executed deliberately (not speculative).

### Why Not In `.github/`?

- `.github/` is for automation code only (workflows, issue templates)
- Documentation belongs in `docs/` (centralized, versioned, auditable)
- Principle: *one source of truth per domain*

---

## Implementation

### Current State (Phase 1-5)
- ✅ `.github/workflows/ci.yml` — SQLite-based pipeline, 7 stages
- ✅ `docs/CI_CD_PIPELINE.md` — Operational guide
- ✅ `ADR 0006` — Why this pipeline design
- ✅ No templates, no future-state files

### Phase 6+ Transition (When Ready)
1. Update `docs/CI_CD_PIPELINE.md` with Phase 6 design
2. Modify `ci.yml` in place: swap SQLite → PostgreSQL service
3. Commit clearly with rationale
4. Verify all 7 stages pass with PostgreSQL
5. No new files created

---

## Related Decisions

- **ADR 0006:** Git Workflow & CI/CD Pipeline (branching, commit conventions, 7-stage gating)
- **ADR 0001:** Backend Architecture (clean layers, testing strategy)

---

## Consequences

### Positive
- ✅ Clear: One workflow file = one active pipeline
- ✅ No confusion: developers always look at `ci.yml` for current state
- ✅ Lean: no dead template files
- ✅ Explicit: phase transitions are deliberate commits, not hidden in alternatives

### Negative
- ⚠️ Requires upfront planning before phase transitions
- ⚠️ Git history doesn't show "alternative" approaches (they're not in git until implemented)

### Risk Mitigation
- Document the planned Phase 6+ design in ADRs/docs (not as code)
- When Phase 6 is ready, commit the change explicitly with full rationale
- If Phase 6 needs rollback: `git revert [commit hash]` restores Phase 1-5 workflow

---

## Lesson Learned

**Do NOT create future-state files, templates, or "for later" code in the repository.**

This includes:
- ❌ Workflow templates (e.g., `ci-phase6-postgresql.yml`)
- ❌ Migration guides as code (e.g., `.github/WORKFLOW_MIGRATION.md`)
- ❌ "Placeholder" implementations
- ❌ Alternative architectures checked into main

**Instead:**
- ✅ Document decisions in ADRs (ADR 0006, ADR 0007)
- ✅ Document operational details in `docs/` (CI_CD_PIPELINE.md, README.md)
- ✅ When ready to implement a phase, commit it with clear rationale
- ✅ Keep the repo clean: one source of truth per concern

---

**Approved By:** Architecture Review  
**Last Updated:** 2026-09-20  
**Version:** 1.0
