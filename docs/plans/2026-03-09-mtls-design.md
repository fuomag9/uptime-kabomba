# mTLS Support Design

**Date:** 2026-03-09
**Status:** Approved

## Summary

Add mutual TLS (mTLS) support for HTTP monitors. Users can store client certificates in a per-user certificate store and reference them from HTTP monitor configurations.

## Scope

- Monitor types: HTTP only
- Certificate storage: new `certificates` DB table (plaintext PEM in DB)
- Scoping: per-user (each user manages their own certificates)

## Data Model

### New `certificates` table (migration `000016_add_certificates`)

| Column | Type | Notes |
|--------|------|-------|
| `id` | serial PK | |
| `user_id` | int FK | scoped per-user |
| `name` | text | human-readable label |
| `cert_pem` | text | client certificate (PEM) |
| `key_pem` | text | private key (PEM) — never returned in API responses |
| `ca_pem` | text | optional CA cert for server verification (PEM) |
| `created_at` | timestamptz | |
| `updated_at` | timestamptz | |

### Monitor config change

HTTP monitors gain an optional `certificate_id` int field in their JSON config blob. No DB schema migration needed for monitors.

## API

New REST endpoints under `/api/certificates`, auth-gated, same pattern as `/api/notifications`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/certificates` | List user's certificates |
| `POST` | `/api/certificates` | Create a certificate |
| `GET` | `/api/certificates/{id}` | Get certificate (key_pem redacted) |
| `PUT` | `/api/certificates/{id}` | Update a certificate |
| `DELETE` | `/api/certificates/{id}` | Delete a certificate |

**Constraints:**
- `key_pem` is never returned in API responses (write-only)
- Deleting a cert referenced by one or more monitors returns HTTP 409

## HTTP Monitor Execution

When `HTTPMonitor.Check` runs with `certificate_id` set:

1. Load `Certificate` record from DB (scoped by user_id to prevent cross-user access)
2. Parse `cert_pem` + `key_pem` via `tls.X509KeyPair`
3. If `ca_pem` is set, build a `*x509.CertPool` and set as `RootCAs` on `tls.Config`
4. Add the client cert to `tls.Config.Certificates`
5. Proceed with existing HTTP check logic

### Dependency injection

A `CertLoader` interface is injected into `HTTPMonitor` at registration time (startup):

```go
type CertLoader interface {
    LoadCertificate(ctx context.Context, certID int, userID int) (*Certificate, error)
}
```

`HTTPMonitor` stores a reference to `CertLoader`. At startup, a DB-backed implementation is wired in. This keeps the monitor package testable without a real DB.

The monitor executor already passes the full `Monitor` struct (including `UserID`) so no changes are needed to the executor.

## Frontend

New **Certificates** section (same pattern as Notifications):

- **List page** (`/certificates`): table with name, created date, delete button
- **Create/Edit form**: Name, Certificate PEM textarea, Private Key PEM (write-only, placeholder `••••••••`), CA Certificate PEM (optional textarea)
- **HTTP monitor form**: optional "Client Certificate" dropdown listing user's certs by name, defaulting to "None"

## Out of Scope

- Encryption of private keys at rest (future enhancement)
- TCP monitor mTLS support
- Certificate expiry monitoring/alerting
