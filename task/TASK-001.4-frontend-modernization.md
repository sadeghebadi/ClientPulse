# TASK-001.4: Frontend Framework Modernization

**Status:** COMPLETED  
**Assignee:** Antigravity  
**Created:** 2026-10-31  
**Target:** Phase 0 Frontend & Runtime Modernization  

---

## 1. Goal and Scope

Modernize the frontend stack and resolve remaining runtime deprecations prior to Phase 1 authorization:
1. **Node.js Runtime Update:**
   - Upgrade Node.js from 24.22.1 to 24.22.2 (released October 28, 2026).
   - Docker builder & runner: `node:24.22.2-alpine3.24`.
   - CI workflow pinned to exact Node.js version `24.22.2`.
   - Verify official image tag and `linux/amd64` digest.
2. **Next.js & React Upgrade:**
   - Upgrade Next.js from 14.2.35 to latest stable `next@16.4.3` (Turbopack enabled).
   - Upgrade React & React DOM to stable `react@19.2.9` and `react-dom@19.2.9`.
   - Upgrade `@types/react` (19.2.18), `@types/react-dom` (19.2.7), and `@types/node` (24.19.2).
   - Upgrade `lucide-react` to `1.66.0` for native React 19 compatibility without peer dependency overrides.
3. **ESLint Migration:**
   - Next.js 16 deprecated/removed `next lint` as an independent CLI command.
   - Migrate to ESLint 9+ Flat Config format (`apps/web/eslint.config.mjs`) importing `eslint-config-next@16.4.3`.
   - Update `apps/web/package.json` `"lint": "eslint ."` to run standalone linting.
   - Remove legacy `.eslintrc.json`.
   - Ensure React 19 hook lint rules pass cleanly with 0 errors.
4. **Compatibility & Reproducibility:**
   - Synchronize `package-lock.json` and verify `npm ci` runs reproducibly.
   - Preserve existing App Router architecture, English interface, Persian interface, and RTL support.
   - No UI redesign, no fake features, no Phase 1 features.
5. **Testing & Quality Checks:**
   - Run `npm ci`, `npm run lint`, `npm run build`.
   - Rebuild and test Docker container (`clientpulse_web`).
   - Run backend regression tests (`go test -count=1 -race ./...`, `go vet ./...`).
   - Verify all health probes (`/health/live`, `/health/ready`, `/health/worker`, web `:3000`).
6. **Documentation:**
   - Create `task-report/TASK-001.4-report.md`.

---

## 2. Acceptance Criteria

- [x] Node.js updated to `24.22.2` across Dockerfiles, CI, and documentation.
- [x] Docker image `node:24.22.2-alpine3.24` verified via registry inspection.
- [x] Next.js upgraded to `16.4.3` (latest stable).
- [x] React & React DOM upgraded to `19.2.9`.
- [x] ESLint migrated to Flat Config (`eslint.config.mjs`) running standalone `eslint .`.
- [x] Zero ESLint warnings or errors (`npm run lint`).
- [x] `npm ci` installs cleanly without `--force` or `--legacy-peer-deps`.
- [x] Next.js production build (`npm run build`) succeeds with Turbopack.
- [x] Docker production image `clientpulse-web` built and running.
- [x] Running container verified: Node `v24.22.2`, Next.js `16.4.3`, React `19.2.9`.
- [x] English and Persian RTL interfaces verified operational.
- [x] Backend Go tests pass with race detector (`go test -count=1 -race ./...`).
- [x] All service health probes respond with HTTP 200.
- [x] Completion report created in `task-report/TASK-001.4-report.md`.
