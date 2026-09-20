# ADR 0008: Documentation Audit & Deduplication (Phase 1-5)

**Date:** 2026-09-20  
**Status:** ACCEPTED  
**Scope:** Backend documentation structure, ownership, and redundancy removal

---

## Problem

Documentation audit performed at session start (ADR reading requirement):
- Discovered multiple documents covering CI/CD pipeline concepts
- Unclear which document was canonical source of truth for each topic
- Risk of divergence over time (docs updating independently)
- Needed systematic deduplication process

---

## Audit Findings

### Documentation Ownership (Before Deduplication)

| Document | Topic | Purpose | Status |
|----------|-------|---------|--------|
| `docs/README.md` | Index | Cross-references, navigation | ✅ Canonical |
| `docs/architecture.md` | Architecture | Layer design (domain, application, infrastructure) | ✅ Canonical |
| `docs/handoff.md` | Backend State | Phase completion, status, onboarding | ✅ Canonical |
| `docs/roadmap.md` | Roadmap | Phase progress, future phases | ✅ Canonical |
| `docs/specs/README.md` | Functional Specs | Feature requirements, API contracts | ✅ Canonical |
| `docs/decisions/0001-*.md` | Architecture Decision | Clean Architecture layers | ✅ Canonical |
| `docs/decisions/0002-*.md` | Code Cleanup | Legacy removal, conventions | ✅ Canonical |
| `docs/decisions/0004-*.md` | Test Strategy | Unit/Integration/E2E classification | ✅ Canonical |
| `docs/decisions/0005-*.md` | Dependency Injection | Why main.go assembly, not frameworks | ✅ Canonical |
| `docs/decisions/0006-*.md` | Git Workflow & CI/CD | Branching, commits, 7-stage pipeline | ✅ Canonical |
| `docs/decisions/0007-*.md` | Workflow Governance | Single active workflow file rule | ✅ Canonical |
| `docs/CI_CD_PIPELINE.md` | CI/CD Operations | 7 stages, commands, troubleshooting | ❓ DUPLICATE |

### Deduplication Decision: CI_CD_PIPELINE.md

**Verdict:** REMOVE (redundant with ADR 0006)

**Rationale:**
- **ADR 0006 covers:**
  - 7-stage pipeline breakdown
  - Configuration parameters (Go version, coverage threshold)
  - Branch triggers (main, develop, feature/**, bugfix/**)
  - Testing strategy (SQLite local vs PostgreSQL Phase 6+)
  - Coverage gating rules
  
- **CI_CD_PIPELINE.md covered SAME topics:**
  - Same 7-stage descriptions
  - Same configuration table
  - Same troubleshooting advice
  - Only difference: markdown formatting

- **Single Source of Truth:**
  - ADR 0006 is the canonical decision document
  - Operational guides should reference ADR, not duplicate it
  - CI/CD strategy changes should update ADR, not parallel file

**Action Taken:** Deleted `CI_CD_PIPELINE.md` (commit f6a2edf)

---

## Audit: Other Documents

### No Duplicates Found

Remaining documents have **distinct, non-overlapping purposes:**

| Document | Unique Responsibility |
|----------|----------------------|
| `architecture.md` | Layer organization, patterns, dependencies |
| `handoff.md` | Phases 1-5 status, testing patterns, next steps |
| `roadmap.md` | Feature list, completion status per phase |
| `specs/README.md` | User story requirements, API contracts |
| `decisions/*` | Decision rationale, alternatives, consequences |

**Conclusion:** No other redundancy detected.

---

## Documentation Ownership Matrix (Post-Audit)

| Topic | Canonical Source | Audience |
|-------|------------------|----------|
| Overall project identity | `README.md` (at root) | New developers, stakeholders |
| Backend state & phases | `docs/handoff.md` | Onboarding developers |
| Architecture layers | `docs/architecture.md` | Developers, architects |
| Feature requirements | `docs/specs/README.md` | Developers, QA |
| Clean Architecture decision | `docs/decisions/0001-*.md` | Architects, code reviewers |
| Test strategy | `docs/decisions/0004-*.md` | QA, test developers |
| Git workflow & CI/CD | `docs/decisions/0006-*.md` | All developers, CI maintainers |
| Workflow governance | `docs/decisions/0007-*.md` | CI maintainers, DevOps |

---

## Audit Process Applied

Per `docs/documentation-guide.md`:

1. ✅ Listed all documents
2. ✅ Identified overlapping topics
3. ✅ Compared content for redundancy
4. ✅ Determined canonical source for each topic
5. ✅ Removed duplicates
6. ✅ Verified remaining documents have distinct purposes
7. ✅ Updated links/index (README.md)

---

## Consequences

### Positive
- ✅ Single source of truth for CI/CD (ADR 0006)
- ✅ No maintenance divergence (one file to update)
- ✅ Clear ownership matrix (above)
- ✅ Lean documentation structure

### Negative
- ⚠️ ADR 0006 is longer (combines strategy + operations)
  - Mitigation: ADR remains focused on decisions, not step-by-step tutorial

---

## Future Audit Cadence

**Per documentation-guide.md §Revue de santé documentaire:**

At each major milestone:
- [ ] Check for new v1/v2/final files
- [ ] Verify each document has one owner
- [ ] Test that all links work
- [ ] Ensure active ADRs are coherent
- [ ] Verify handoff matches actual code state
- [ ] Mark obsolete decisions as `superseded`

**Scheduled for:** When frontend refactoring begins (Phase 6+)

---

## Related Documents

- **documentation-guide.md:** Global governance rules
- **ADR 0006:** CI/CD strategy (canonical source for pipeline decisions)
- **ADR 0007:** Single active workflow file (governance rule)

---

**Approved By:** Documentation Governance  
**Last Updated:** 2026-09-20  
**Version:** 1.0
