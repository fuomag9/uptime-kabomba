# mTLS Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add per-user certificate store and mTLS support for HTTP monitors.

**Architecture:** A new `certificates` table stores client certs (PEM text) scoped per user. A `CertLoader` interface is injected into `HTTPMonitor` at startup (via `main.go`, replacing the `init()` auto-registration). HTTP monitors read `certificate_id` from their config JSON and load the cert at check time.

**Tech Stack:** Go 1.26, GORM, chi router, Next.js 15 (App Router), TypeScript

---

### Task 1: DB migration — create certificates table

**Files:**
- Create: `migrations/postgres/000016_add_certificates.up.sql`
- Create: `migrations/postgres/000016_add_certificates.down.sql`

**Step 1: Write the up migration**

```sql
-- migrations/postgres/000016_add_certificates.up.sql
CREATE TABLE certificates (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    cert_pem   TEXT NOT NULL,
    key_pem    TEXT NOT NULL,
    ca_pem     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_certificates_user_id ON certificates(user_id);
```

**Step 2: Write the down migration**

```sql
-- migrations/postgres/000016_add_certificates.down.sql
DROP TABLE IF EXISTS certificates;
```

**Step 3: Verify migration files exist**

Run: `ls migrations/postgres/000016*`
Expected: two files listed

**Step 4: Commit**

```bash
git add migrations/postgres/000016_add_certificates.up.sql migrations/postgres/000016_add_certificates.down.sql
git commit -m "feat: add certificates table migration"
```

---

### Task 2: Go model — Certificate

**Files:**
- Create: `internal/models/certificate.go`

**Step 1: Write the model**

```go
package models

import "time"

// Certificate stores a client TLS certificate for mTLS connections.
// key_pem is never serialised back to API callers — use the json:"-" tag.
type Certificate struct {
	ID        int       `json:"id"         gorm:"primaryKey;autoIncrement"`
	UserID    int       `json:"user_id"    gorm:"not null;index"`
	Name      string    `json:"name"       gorm:"not null"`
	CertPEM   string    `json:"cert_pem"   gorm:"not null"`
	KeyPEM    string    `json:"-"          gorm:"not null"`  // never returned by API
	CAPEM     string    `json:"ca_pem"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Certificate) TableName() string { return "certificates" }
```

**Step 2: Verify it compiles**

Run: `go build ./internal/models/...`
Expected: no output (success)

**Step 3: Commit**

```bash
git add internal/models/certificate.go
git commit -m "feat: add Certificate model"
```

---

### Task 3: CertLoader interface + DB implementation

**Files:**
- Create: `internal/monitor/certloader.go`

**Step 1: Write the interface and DB implementation**

```go
package monitor

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// CertLoader loads a certificate for a given user.
type CertLoader interface {
	LoadCertificate(ctx context.Context, certID int, userID int) (*models.Certificate, error)
}

// DBCertLoader is the production implementation backed by GORM.
type DBCertLoader struct {
	db *gorm.DB
}

// NewDBCertLoader creates a new DBCertLoader.
func NewDBCertLoader(db *gorm.DB) *DBCertLoader {
	return &DBCertLoader{db: db}
}

// LoadCertificate loads a certificate scoped to the given user.
// Returns an error if the cert does not exist or belongs to a different user.
func (l *DBCertLoader) LoadCertificate(ctx context.Context, certID int, userID int) (*models.Certificate, error) {
	var cert models.Certificate
	err := l.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", certID, userID).
		First(&cert).Error
	if err != nil {
		return nil, fmt.Errorf("certificate %d not found for user %d: %w", certID, userID, err)
	}
	return &cert, nil
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/monitor/...`
Expected: no output

**Step 3: Commit**

```bash
git add internal/monitor/certloader.go
git commit -m "feat: add CertLoader interface and DBCertLoader"
```

---

### Task 4: Update HTTPMonitor to accept CertLoader + apply mTLS

**Files:**
- Modify: `internal/monitor/http.go`

The `HTTPMonitor` currently has no fields and registers itself via `init()`. We need to:
1. Add a `certLoader CertLoader` field
2. Export a constructor `NewHTTPMonitor(loader CertLoader) *HTTPMonitor`
3. Remove the `init()` auto-registration (registration will move to `main.go`)
4. Add cert loading logic in `Check`

**Step 1: Replace the struct, remove init(), add constructor**

Find and replace in `internal/monitor/http.go`:

Old:
```go
// HTTPMonitor implements HTTP/HTTPS monitoring
type HTTPMonitor struct{}

func init() {
	RegisterMonitorType(&HTTPMonitor{})
}
```

New:
```go
// HTTPMonitor implements HTTP/HTTPS monitoring.
// certLoader may be nil; if nil, mTLS is unavailable.
type HTTPMonitor struct {
	certLoader CertLoader
}

// NewHTTPMonitor creates an HTTPMonitor with the given CertLoader.
func NewHTTPMonitor(loader CertLoader) *HTTPMonitor {
	return &HTTPMonitor{certLoader: loader}
}
```

**Step 2: Add cert loading in Check, just before the http.Client is created**

The existing `Check` function builds an `http.Client` starting around line 74. After reading `ignoreTLS` (line 70) add:

```go
	// Load client certificate if configured
	var tlsCerts []tls.Certificate
	var rootCAs *x509.CertPool

	if certIDRaw, ok := monitor.Config["certificate_id"]; ok && h.certLoader != nil {
		certID := 0
		switch v := certIDRaw.(type) {
		case float64:
			certID = int(v)
		case int:
			certID = v
		}
		if certID > 0 {
			certRecord, err := h.certLoader.LoadCertificate(ctx, certID, monitor.UserID)
			if err != nil {
				heartbeat.Message = fmt.Sprintf("Failed to load certificate: %v", err)
				return heartbeat, nil
			}
			tlsCert, err := tls.X509KeyPair([]byte(certRecord.CertPEM), []byte(certRecord.KeyPEM))
			if err != nil {
				heartbeat.Message = fmt.Sprintf("Invalid certificate: %v", err)
				return heartbeat, nil
			}
			tlsCerts = append(tlsCerts, tlsCert)

			if certRecord.CAPEM != "" {
				rootCAs = x509.NewCertPool()
				if !rootCAs.AppendCertsFromPEM([]byte(certRecord.CAPEM)) {
					heartbeat.Message = "Failed to parse CA certificate"
					return heartbeat, nil
				}
			}
		}
	}
```

**Step 3: Update the TLSClientConfig to use the loaded certs**

Replace the existing `TLSClientConfig` block in the http.Client:

Old:
```go
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ignoreTLS,
			},
```

New:
```go
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ignoreTLS,
				Certificates:       tlsCerts,
				RootCAs:            rootCAs,
			},
```

**Step 4: Add missing import `crypto/x509`**

The imports block already has `crypto/tls`. Add `crypto/x509` next to it:

Old:
```go
import (
	"context"
	"crypto/tls"
	"fmt"
```

New:
```go
import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
```

**Step 5: Verify it compiles**

Run: `go build ./internal/monitor/...`
Expected: no output

**Step 6: Commit**

```bash
git add internal/monitor/http.go
git commit -m "feat: add mTLS cert loading to HTTPMonitor"
```

---

### Task 5: Wire HTTPMonitor registration into main.go

Currently `http.go` uses `init()` to self-register. Since we removed that, main.go must register it after building the DB cert loader.

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: Read main.go first**

Run: read `cmd/server/main.go` to find where `monitor.NewExecutor` is called.

**Step 2: Add registration after DB is ready**

Find where `monitor.NewExecutor(db, hub, dispatcher)` is called. Just before that line, add:

```go
// Register HTTP monitor with mTLS cert loader
monitor.RegisterMonitorType(monitor.NewHTTPMonitor(monitor.NewDBCertLoader(db)))
```

**Step 3: Verify it compiles**

Run: `go build ./cmd/server/...`
Expected: no output

**Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire HTTPMonitor registration with CertLoader in main"
```

---

### Task 6: Certificate API handlers

**Files:**
- Create: `internal/api/certificate_handlers.go`

**Step 1: Write the handlers file**

```go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

func HandleGetCertificates(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var certs []models.Certificate
		if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&certs).Error; err != nil {
			http.Error(w, "Failed to fetch certificates", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(certs)
	}
}

func HandleGetCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		var cert models.Certificate
		if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&cert).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Certificate not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch certificate", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cert)
	}
}

func HandleCreateCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var req struct {
			Name    string `json:"name"`
			CertPEM string `json:"cert_pem"`
			KeyPEM  string `json:"key_pem"`
			CAPEM   string `json:"ca_pem"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.CertPEM == "" || req.KeyPEM == "" {
			http.Error(w, "name, cert_pem, and key_pem are required", http.StatusBadRequest)
			return
		}

		cert := models.Certificate{
			UserID:    user.ID,
			Name:      req.Name,
			CertPEM:   req.CertPEM,
			KeyPEM:    req.KeyPEM,
			CAPEM:     req.CAPEM,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(&cert).Error; err != nil {
			http.Error(w, "Failed to create certificate", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(cert)
	}
}

func HandleUpdateCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		var req struct {
			Name    string `json:"name"`
			CertPEM string `json:"cert_pem"`
			KeyPEM  string `json:"key_pem"`
			CAPEM   string `json:"ca_pem"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		updates := map[string]interface{}{
			"name":       req.Name,
			"cert_pem":   req.CertPEM,
			"ca_pem":     req.CAPEM,
			"updated_at": time.Now(),
		}
		// Only update key_pem if a new one is provided
		if req.KeyPEM != "" {
			updates["key_pem"] = req.KeyPEM
		}

		result := db.Model(&models.Certificate{}).
			Where("id = ? AND user_id = ?", id, user.ID).
			Updates(updates)
		if result.Error != nil {
			http.Error(w, "Failed to update certificate", http.StatusInternalServerError)
			return
		}
		if result.RowsAffected == 0 {
			http.Error(w, "Certificate not found", http.StatusNotFound)
			return
		}

		var cert models.Certificate
		db.Where("id = ?", id).First(&cert)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cert)
	}
}

func HandleDeleteCertificate(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		// Check if any monitor references this certificate
		var count int64
		// Config is stored as JSON text; search for the certificate_id value
		db.Raw(
			`SELECT COUNT(*) FROM monitors WHERE user_id = ? AND config::text LIKE ?`,
			user.ID,
			`%"certificate_id":`+strconv.Itoa(id)+`%`,
		).Scan(&count)
		if count > 0 {
			http.Error(w, "Certificate is in use by one or more monitors", http.StatusConflict)
			return
		}

		result := db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Certificate{})
		if result.Error != nil {
			http.Error(w, "Failed to delete certificate", http.StatusInternalServerError)
			return
		}
		if result.RowsAffected == 0 {
			http.Error(w, "Certificate not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/api/...`
Expected: no output

**Step 3: Commit**

```bash
git add internal/api/certificate_handlers.go
git commit -m "feat: add certificate CRUD API handlers"
```

---

### Task 7: Register certificate routes in router.go

**Files:**
- Modify: `internal/api/router.go`

**Step 1: Add routes inside the protected `r.Group` block, after the API Key routes (around line 141)**

```go
			// Certificate routes
			r.Get("/certificates", HandleGetCertificates(db))
			r.Post("/certificates", HandleCreateCertificate(db))
			r.Get("/certificates/{id}", HandleGetCertificate(db))
			r.Put("/certificates/{id}", HandleUpdateCertificate(db))
			r.Delete("/certificates/{id}", HandleDeleteCertificate(db))
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no output

**Step 3: Commit**

```bash
git add internal/api/router.go
git commit -m "feat: register certificate routes"
```

---

### Task 8: Frontend — API client types and methods

**Files:**
- Modify: `web/lib/api.ts`

**Step 1: Add the Certificate interface (after the Monitor interfaces, around line 415)**

```typescript
export interface Certificate {
  id: number;
  user_id: number;
  name: string;
  cert_pem: string;
  ca_pem: string;
  created_at: string;
  updated_at: string;
  // key_pem is intentionally absent — never returned by the API
}

export interface CreateCertificateRequest {
  name: string;
  cert_pem: string;
  key_pem: string;
  ca_pem?: string;
}

export interface UpdateCertificateRequest {
  name: string;
  cert_pem: string;
  key_pem?: string; // omit to keep existing key
  ca_pem?: string;
}
```

**Step 2: Add API methods to the ApiClient class (before the closing `}` of the class)**

```typescript
  // Certificate endpoints
  async getCertificates(): Promise<Certificate[]> {
    const result = await this.request<Certificate[] | null>('/api/certificates');
    return result || [];
  }

  async getCertificate(id: number): Promise<Certificate> {
    return this.request<Certificate>(`/api/certificates/${id}`);
  }

  async createCertificate(data: CreateCertificateRequest): Promise<Certificate> {
    return this.request<Certificate>('/api/certificates', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateCertificate(id: number, data: UpdateCertificateRequest): Promise<Certificate> {
    return this.request<Certificate>(`/api/certificates/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteCertificate(id: number): Promise<void> {
    return this.request<void>(`/api/certificates/${id}`, {
      method: 'DELETE',
    });
  }
```

**Step 3: Verify TypeScript compiles**

Run: `cd web && npx tsc --noEmit`
Expected: no errors

**Step 4: Commit**

```bash
git add web/lib/api.ts
git commit -m "feat: add Certificate types and API client methods"
```

---

### Task 9: Frontend — Certificates page

**Files:**
- Create: `web/app/(dashboard)/certificates/page.tsx`

**Step 1: Write the page**

```tsx
'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, Certificate, CreateCertificateRequest, UpdateCertificateRequest } from '@/lib/api';

export default function CertificatesPage() {
  const router = useRouter();
  const [certs, setCerts] = useState<Certificate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Certificate | null>(null);

  // Form state
  const [name, setName] = useState('');
  const [certPem, setCertPem] = useState('');
  const [keyPem, setKeyPem] = useState('');
  const [caPem, setCaPem] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => { load(); }, []);

  async function load() {
    try {
      setLoading(true);
      setCerts(await apiClient.getCertificates());
    } catch (err: any) {
      setError(err.message || 'Failed to load certificates');
      if (err.status === 401) router.push('/login');
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    setName(''); setCertPem(''); setKeyPem(''); setCaPem('');
    setShowForm(true);
  }

  function openEdit(cert: Certificate) {
    setEditing(cert);
    setName(cert.name); setCertPem(cert.cert_pem); setKeyPem(''); setCaPem(cert.ca_pem || '');
    setShowForm(true);
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      if (editing) {
        const req: UpdateCertificateRequest = { name, cert_pem: certPem, ca_pem: caPem };
        if (keyPem) req.key_pem = keyPem;
        await apiClient.updateCertificate(editing.id, req);
      } else {
        const req: CreateCertificateRequest = { name, cert_pem: certPem, key_pem: keyPem, ca_pem: caPem };
        await apiClient.createCertificate(req);
      }
      setShowForm(false);
      await load();
    } catch (err: any) {
      alert('Failed to save: ' + (err.message || 'Unknown error'));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: number) {
    if (!confirm('Delete this certificate?')) return;
    try {
      await apiClient.deleteCertificate(id);
      setCerts(certs.filter((c) => c.id !== id));
    } catch (err: any) {
      alert('Failed to delete: ' + (err.message || 'Unknown error'));
    }
  }

  if (loading) return <div className="p-8 text-center text-gray-500">Loading...</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Client Certificates</h1>
        <button
          onClick={openCreate}
          className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary/90"
        >
          Add Certificate
        </button>
      </div>

      {error && <p className="mb-4 text-red-500">{error}</p>}

      {showForm && (
        <div className="mb-6 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6">
          <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-white">
            {editing ? 'Edit Certificate' : 'New Certificate'}
          </h2>
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Client Certificate (PEM)</label>
              <textarea
                value={certPem}
                onChange={(e) => setCertPem(e.target.value)}
                required
                rows={6}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm font-mono text-gray-900 dark:text-white"
                placeholder="-----BEGIN CERTIFICATE-----"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Private Key (PEM){editing && ' — leave blank to keep existing'}
              </label>
              <textarea
                value={keyPem}
                onChange={(e) => setKeyPem(e.target.value)}
                required={!editing}
                rows={6}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm font-mono text-gray-900 dark:text-white"
                placeholder={editing ? '••••••••' : '-----BEGIN PRIVATE KEY-----'}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">CA Certificate (PEM, optional)</label>
              <textarea
                value={caPem}
                onChange={(e) => setCaPem(e.target.value)}
                rows={6}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm font-mono text-gray-900 dark:text-white"
                placeholder="-----BEGIN CERTIFICATE----- (optional)"
              />
            </div>
            <div className="flex gap-3">
              <button
                type="submit"
                disabled={saving}
                className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary/90 disabled:opacity-50"
              >
                {saving ? 'Saving...' : 'Save'}
              </button>
              <button
                type="button"
                onClick={() => setShowForm(false)}
                className="rounded-md bg-gray-200 dark:bg-gray-700 px-4 py-2 text-sm font-semibold text-gray-700 dark:text-gray-300"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {certs.length === 0 && !showForm ? (
        <div className="rounded-lg border border-dashed border-gray-300 dark:border-gray-700 p-12 text-center">
          <p className="text-gray-500 dark:text-gray-400">No certificates yet. Add one to use mTLS with HTTP monitors.</p>
        </div>
      ) : (
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">CA</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Created</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
              {certs.map((cert) => (
                <tr key={cert.id}>
                  <td className="px-6 py-4 text-sm font-medium text-gray-900 dark:text-white">{cert.name}</td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{cert.ca_pem ? 'Yes' : 'No'}</td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">{new Date(cert.created_at).toLocaleDateString()}</td>
                  <td className="px-6 py-4 text-right text-sm space-x-3">
                    <button onClick={() => openEdit(cert)} className="text-primary hover:underline">Edit</button>
                    <button onClick={() => handleDelete(cert.id)} className="text-red-500 hover:underline">Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
```

**Step 2: Verify TypeScript compiles**

Run: `cd web && npx tsc --noEmit`
Expected: no errors

**Step 3: Commit**

```bash
git add web/app/\(dashboard\)/certificates/page.tsx
git commit -m "feat: add certificates management page"
```

---

### Task 10: Frontend — Add Certificates link to nav + HTTP monitor cert selector

**Files:**
- Modify: `web/app/(dashboard)/layout.tsx`
- Modify: `web/app/(dashboard)/monitors/new/page.tsx` (and `monitors/[id]/edit/page.tsx`)

**Step 1: Add Certificates link to layout.tsx nav**

In `web/app/(dashboard)/layout.tsx`, find the `Notifications` nav link block and add a Certificates link after it:

```tsx
<Link
  href="/certificates"
  className={`text-sm font-medium ${pathname === '/certificates'
      ? 'text-primary'
      : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
    }`}
>
  Certificates
</Link>
```

**Step 2: Read the monitor new/edit pages to understand the HTTP config form**

Read `web/app/(dashboard)/monitors/new/page.tsx` to find where `type === 'http'` config fields are rendered.

**Step 3: Add certificate selector to the HTTP monitor config section**

In the HTTP monitor config section of the monitor form, load certs once on component mount and add a select dropdown. The pattern:

In the component, add state:
```tsx
const [certificates, setCertificates] = useState<Certificate[]>([]);
```

In `useEffect` / data loading, fetch certs (can be done alongside existing data fetches):
```tsx
apiClient.getCertificates().then(setCertificates).catch(() => {});
```

In the HTTP-specific config fields, add:
```tsx
{monitor.type === 'http' && (
  <div>
    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
      Client Certificate (mTLS)
    </label>
    <select
      value={monitor.config?.certificate_id || ''}
      onChange={(e) =>
        setMonitor({
          ...monitor,
          config: {
            ...monitor.config,
            certificate_id: e.target.value ? Number(e.target.value) : undefined,
          },
        })
      }
      className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white"
    >
      <option value="">None</option>
      {certificates.map((c) => (
        <option key={c.id} value={c.id}>{c.name}</option>
      ))}
    </select>
  </div>
)}
```

Apply the same change to both the new and edit monitor pages.

**Step 4: Verify TypeScript compiles**

Run: `cd web && npx tsc --noEmit`
Expected: no errors

**Step 5: Commit**

```bash
git add web/app/\(dashboard\)/layout.tsx web/app/\(dashboard\)/monitors/new/page.tsx web/app/\(dashboard\)/monitors/\[id\]/edit/page.tsx
git commit -m "feat: add certificates nav link and mTLS selector to HTTP monitor form"
```

---

### Task 11: Smoke test end-to-end

**Step 1: Build the backend**

Run: `go build ./cmd/server/...`
Expected: no output

**Step 2: Build the frontend**

Run: `cd web && npm run build`
Expected: build succeeds, no TypeScript errors

**Step 3: Run backend unit tests**

Run: `go test ./...`
Expected: all tests pass (PASS lines, no FAIL)

**Step 4: Final commit if anything was missed**

If any changes weren't committed in prior tasks, commit them now. Otherwise, this task is complete.
