# ClientPulse Security Foundations

This document establishes mandatory security controls defined in PRD.md.

## 1. Multi-Tenant Data Isolation
- Every database query touching tenant-scoped records must enforce `organization_id = $context_org_id`.
- Tenant context is derived strictly on the server from authenticated session/membership tokens.
- Cross-tenant references are prevented using composite keys or database constraints where appropriate.
- Automated tests must verify that Tenant A cannot read, mutate, or delete Tenant B's data under any condition.

## 2. SSRF (Server-Side Request Forgery) Guard
ClientPulse executes automated network checks against arbitrary target URLs. The SSRF guard enforces:
- Permitted protocols: only `http` and `https`.
- Blocked targets:
  - `localhost` and its subdomains.
  - IPv4 loopback (`127.0.0.0/8`).
  - RFC 1918 private subnets (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`).
  - Link-local and cloud metadata addresses (`169.254.169.254`, etc.).
  - IPv6 loopback (`::1`), link-local (`fe80::/10`), unique local (`fc00::/7`).
  - Internal hostnames (`.local`, `.internal`, `.lan`).
  - Userinfo in URLs (`user:pass@host`).
  - Non-standard ports (only ports 80 and 443 are allowed in MVP).
- DNS rebinding mitigation: Target hostnames must be resolved at check time, all resolved IPs verified against the blocklist, and HTTP connections pinned to the verified IP. Redirects must be re-evaluated from the first step.

## 3. Secret and Token Handling
- Tokens (Heartbeat tokens, report share tokens, password reset tokens) are stored in the database exclusively as cryptographic hashes (e.g., SHA-256 with pepper).
- Raw tokens are generated with CSPRNG (`crypto/rand`) and presented to users exactly once.
- Tokens in URLs or query strings are redacted from server access logs, error logs, and metrics.
- Passwords must be hashed with Argon2id or equivalent state-of-the-art password hashing algorithms.

## 4. HTTP Security Headers
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy`: Restricts scripts, styles, and frames to trusted sources.
- Cookies: `HttpOnly`, `SameSite=Lax`, and `Secure` (in production).

