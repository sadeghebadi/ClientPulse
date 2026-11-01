# TASK-005: Maintenance Activity, Reports, PDF & White Label

**Phase**: Phase 4  
**Previous Phase**: Phase 3 — Incident Management & Heartbeats  
**Status**: Completed  
**Branch**: `task/005-maintenance-reports`  
**Single Source of Truth**: `PRD.md`

---

## 1. Objectives

1. **Pre-Implementation Verification**:
   - Verify Heartbeat token redaction in application, HTTP access, reverse proxy, and Docker logs.
   - Enforce a first deadline on newly created, unpinged heartbeats (`created_at + interval + grace`).
   - Separate monitoring coverage percentage and uptime percentage; prevent paused checks or missing data from being counted as uptime.
2. **Maintenance Activity CRUD**:
   - Agency members can log maintenance work (`update`, `backup`, `bug_fix`, `performance`, `security_review`, `content_change`, `configuration`, `other`).
   - Audit trail triggered when modifying an activity that falls within an existing immutable report snapshot.
3. **Agency White-Label Branding**:
   - Custom display name, primary & accent hex colors with WCAG contrast verification, custom report footer, contact email, website URL, and default locale (`en` or `fa`).
   - Tenant-isolated storage for agency logos with MIME-type sniffing, dimension checking, 2MB size limit, and explicit SVG upload prevention.
4. **Monthly Care Report Generation & Snapshot Versioning**:
   - Deterministic calculations across identical timezone-adjusted UTC boundaries.
   - Strictly immutable snapshots persisted in `reports.snapshot_json` with monotonically incrementing version numbers (`v1`, `v2`, etc.).
   - Generation of distinct uptime vs. coverage percentages, incident history, maintenance activities, heartbeat status, and deterministic recommendations.
5. **Production Headless Chromium PDF Generator**:
   - Vector selectable-text PDFs with embedded Noto Sans Arabic and DejaVu Sans fonts.
   - 100% extractable English and Persian RTL text, custom headers, page numbering, zero clipped text.
6. **Cryptographically Secure Report Sharing**:
   - 32-byte cryptographic token material (`crypto/rand`), storing only SHA-256 hashes in PostgreSQL (`report_share_tokens`).
   - Read-only unauthenticated public viewer with optional expiration and immediate revocation.
   - Strict rate limiting on public endpoints (`/public/reports/:token` and `/public/reports/:token/pdf`).

