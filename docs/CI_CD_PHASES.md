# CI/CD Pipeline: Phased Evolution

**Status:** Phase 1-5 active (SQLite-based)  
**Last Updated:** 2026-09-20

---

## Phase 1-5: SQLite-Based CI/CD (Current)

### Active Workflow: `ci.yml`

```yaml
1. Lint & Format        (gofmt, go vet, go mod tidy)      ~5s
2. Build                (go build)                          ~10s
3. Unit Tests           (./tests/units/...)                 ~30s
4. Security Check       (nancy sleuth)                      ~20s
5. Integration Tests    (./tests/integration/... + SQLite)  ~60s
6. E2E Tests            (./tests/e2e/... + SQLite)          ~30s
7. Coverage Report      (80% threshold, blocking)           ~20s
───────────────────────────────────────────────────────────────
Total: ~3-4 minutes
```

### Key Characteristics
- ✅ **Database:** SQLite in-memory (no Docker required)
- ✅ **Local Parity:** Identical tests run locally (`make test`) and CI/CD
- ✅ **Coverage:** Blocking gate (must ≥80% to merge)
- ✅ **Security:** Nancy dependency scan (non-blocking for now)
- ✅ **Branches:** Push to `main`, `develop`, `feature/**`, `bugfix/**`

### Why SQLite?
- **Local:** Zero friction (developers don't need Podman/Docker for basic validation)
- **Speed:** Tests complete in ~100-200ms per test
- **Catch Rate:** Catches 95%+ of logic bugs (database logic, HTTP marshaling, routing)
- **Trade-off:** May miss PostgreSQL-specific issues (constraints, JSONB, isolation levels)

---

## Phase 6+: PostgreSQL + Testcontainers (Future)

### Planned Date
After:
- Main features stabilized
- Performance profiling complete
- Database schema frozen
- Team readiness assessed

### Workflow: `ci-phase6-postgresql.yml` (template, not active)

```yaml
Same 7 stages, but:

5. Integration Tests    (./tests/integration/... + PostgreSQL) ~90s
6. E2E Tests            (./tests/e2e/... + PostgreSQL)         ~60s
+ 8. DB Migration Tests (schema validation)                     ~40s (NEW)
───────────────────────────────────────────────────────────────
Total: ~5-6 minutes
```

### Key Changes
- **Database:** PostgreSQL 16 (via Docker service)
- **Tests:** No code changes; database adapter switched via environment
- **Local Development:** Still SQLite (developers choose per workflow)
- **CI/CD:** PostgreSQL (production-like validation)

### Migration Checklist
- [ ] Add `github.com/testcontainers/testcontainers-go` to `go.mod`
- [ ] Create PostgreSQL database setup fixture
- [ ] Refactor integration/E2E tests to detect `DATABASE_URL` and use PostgreSQL
- [ ] Add database migration validation tests
- [ ] Rename `ci.yml` → `ci-legacy.yml` (Phase 1-5 backup)
- [ ] Rename `ci-phase6-postgresql.yml` → `ci.yml` (activate)
- [ ] Test full CI/CD run with PostgreSQL
- [ ] Verify coverage still ≥80%
- [ ] Update this document with completion date

---

## Testing Strategy by Phase

| Tier | Phase 1-5 | Phase 6+ | Notes |
|------|-----------|---------|-------|
| Unit | SQLite (in-memory) | SQLite (in-memory) | No DB needed; logic-only tests |
| Integration | SQLite (in-memory) | PostgreSQL (Docker service) | Database behavior validation |
| E2E | SQLite (in-memory) | PostgreSQL (Docker service) | Full stack flow validation |
| Coverage | Local + CI (SQLite) | Local SQLite, CI PostgreSQL | Mixed for flexibility |

---

## How to Switch (Phase 6 Activation)

```bash
# Step 1: Backup Phase 1-5
mv .github/workflows/ci.yml .github/workflows/ci-legacy-sqlite.yml

# Step 2: Activate Phase 6+
mv .github/workflows/ci-phase6-postgresql.yml .github/workflows/ci.yml

# Step 3: Commit
git add .github/workflows/
git commit -m "ci: activate Phase 6 PostgreSQL-based pipeline"

# Step 4: Verify
# Push to develop and observe 7-stage run with PostgreSQL
```

---

## Related Documentation

- **ADR 0006:** Git Workflow & CI/CD Pipeline (decisions, conventions, gating rules)
- **ADR 0004:** Test Strategy (Given/When/Then, unit/integration/E2E classification)
- **CI_CD_PIPELINE.md:** Operational guide (7 stages in detail, troubleshooting)
- **handoff.md:** Current Phase 1-5 status and testing patterns

---

**Next Decision Point:** When Phase 6 is ready to begin (estimated after Q4 feature freeze)
