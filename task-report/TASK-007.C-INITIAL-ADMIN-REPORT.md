# TASK-007.C: Initial ClientPulse Admin Account Creation Report

**Project:** ClientPulse  
**Target Domain:** https://mindu.ir  
**Target VPS:** 37.152.191.170:2222 (`ubuntu-g1-small5-foroogh-1`)  
**Execution Date:** 2026-11-02  
**Status:** ✅ Successfully Created, Verified & Fully Operational  

---

## 1. Account & Organization Summary

| Attribute | Value |
|---|---|
| **Account Email** | `admin@mindu.ir` |
| **Display Name** | `Mindu Admin` |
| **Organization Name** | `Mindu` |
| **Organization Slug** | `mindu` |
| **Organization Role** | `Owner` (`owner`) |
| **Account Status** | `active` |
| **User ID** | `abc00ee1-cdf2-47d1-8ba2-790e0cf5d957` |
| **Organization ID** | `1f21de76-603e-4ea1-8254-400a3300d210` |
| **Locale / Timezone** | `fa` / `Asia/Tehran` |

> [!NOTE] Plaintext Password Redaction Policy
> In accordance with strict production security requirements, the plaintext password is **NOT** stored or included in this report or any repository file. It is revealed exclusively to the authorized user in the final interactive response.

---

## 2. Password Hashing Algorithm & Entropy

- **Algorithm:** Argon2id (`argon2id`)
- **Version:** Argon2 v19
- **OWASP Recommended Parameters:**
  - Memory: `64 MB` (`65536 KB`)
  - Iterations: `1`
  - Parallelism: `4` threads
  - Salt Length: `16` bytes (CSPRNG generated)
  - Key Length: `32` bytes
- **Entropy Source:** Cryptographically Secure Pseudorandom Number Generator (`crypto/rand`) generating 256 bits of entropy, represented as a 43-character URL-safe string.
- **Storage:** Stored strictly as an Argon2id formatted hash string in PostgreSQL:  
  `$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>`

---

## 3. Account Creation & Domain Logic

1. **Transactional Signup Execution:**
   - The account was created through the established application domain logic (`AuthService.Signup`), validating password complexity and email format.
   - The organization `Mindu` was automatically provisioned with slug `mindu`.
   - The user `Mindu Admin` was linked to the organization with role `owner` in `organization_memberships`.
   - Initial tenant isolation and PostgreSQL Row Level Security (RLS) were preserved without bypass.
   - Database operations were executed via restricted application role permissions. No PostgreSQL superuser account was used for runtime operations.

---

## 4. Administrative Bootstrap Email Verification

Because production Email is intentionally deferred (`EMAIL_PROVIDER=disabled`), external SMTP delivery was not attempted. To permit normal application operation and sensitive tenant features without weakening global security controls:

1. **Targeted Administrative Verification:**
   - The user record was updated directly in PostgreSQL: `UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = 'abc00ee1-cdf2-47d1-8ba2-790e0cf5d957';`.
   - Global email verification enforcement was NOT modified or disabled; unverified accounts continue to require verification.
2. **Audit Trail Recording:**
   - An immutable audit log was committed to `audit_logs`:
     ```json
     {
       "action": "auth.administrative_bootstrap_verification",
       "entity_type": "user",
       "entity_id": "abc00ee1-cdf2-47d1-8ba2-790e0cf5d957",
       "organization_id": "1f21de76-603e-4ea1-8254-400a3300d210",
       "metadata": {
         "reason": "administrative bootstrap verification for initial production admin (production email delivery deferred)",
         "task": "TASK-007.C",
         "approved_by_user": true
       }
     }
     ```
3. **No External SMTP Transmission:**
   - Verified that zero external email packets or SMTP connections were dispatched.

---

## 5. Production Login & Session Verification

Full end-to-end authentication was verified against `https://mindu.ir`:

### Authentication Probe (`POST /api/v1/auth/login`)

- **Status Code:** `200 OK`
- **Session Cookie:** `clientpulse_session` issued with attributes:
  - `HttpOnly: true`
  - `Secure: true`
  - `SameSite: Lax`
  - `Path: /`
  - `Max-Age: 604800` (7 days)
- **CSRF Cookie & Token:** `clientpulse_csrf` issued and double-cookie verification verified.
- **Identity Payload Returned:**
  - `user.email`: `admin@mindu.ir`
  - `user.display_name`: `Mindu Admin`
  - `user.email_verified_at`: `2026-11-02T17:07:57.656347Z`
  - `organization.name`: `Mindu`
  - `role`: `owner`

### Identity Probe (`GET /api/v1/auth/me`)

- **Status Code:** `200 OK`
- Successfully resolved active tenant context to `Mindu` with role `owner` and confirmed `email_verified_at` is active.

---

## 6. Dashboard & Protected Route Access Verification

All primary application domain endpoints were tested using the authenticated session:

| Endpoint | Resource | HTTP Status | Access Result |
|---|---|---|---|
| `GET /api/v1/clients` | Clients List | `200 OK` | Accessible, tenant-scoped |
| `GET /api/v1/assets` | Assets List | `200 OK` | Accessible, tenant-scoped |
| `GET /api/v1/monitors` | Website Monitors | `200 OK` | Accessible, tenant-scoped |
| `GET /api/v1/incidents` | Incident Tracker | `200 OK` | Accessible, tenant-scoped |
| `GET /api/v1/reports` | Care Reports | `200 OK` | Accessible, tenant-scoped |
| `GET /api/v1/settings/branding` | Agency Branding | `200 OK` | Accessible, tenant-scoped |
| `POST /api/v1/clients` | Sensitive Write Test | `201 Created` | Verified write permission + CSRF |
| `DELETE /api/v1/clients/{id}` | Sensitive Delete Test | `200 OK` | Test client removed cleanly |
| `POST /api/v1/auth/logout` | Session Cleanup | `200 OK` | Test session cookies invalidated |

---

## 7. Data Integrity & Protected Services Preservation

1. **Preservation of Existing Accounts:**
   - Verified that prior smoke-test account `deploy-test@mindu.ir` and organization `Deploy Org` remain intact and completely unmodified.
   - Total users in database: 2 (`deploy-test@mindu.ir`, `admin@mindu.ir`).
2. **Protected Containers Non-Disruption:**
   - Verified all 8 protected containers (`rocketchat-rocketchat-1`, `rocketchat-mongodb-1`, `rustdesk-hbbs`, `rustdesk-hbbr`, `fchat-mattermost-1`, `fchat-postgres-1`, `docker-main-mattermost-1`, `docker-main-postgres-1`):
   - `RestartCount: 0` across all 8 services.
   - Uptime completely uninterrupted.

---

*Report prepared and certified on 2026-11-02.*
