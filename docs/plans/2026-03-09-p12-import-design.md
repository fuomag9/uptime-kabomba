# .p12 Certificate Import Design

**Date:** 2026-03-09
**Status:** Approved

## Summary

Add a client-side .p12 (PKCS#12) certificate import button to the certificates management page. Users can upload a `.p12`/`.pfx` file with its password; `node-forge` parses it in the browser and the extracted PEM components are sent to the existing `createCertificate` API endpoint — no backend changes required.

## Scope

- Frontend only — no new backend endpoints
- Library: `node-forge` for PKCS#12 parsing
- Target file: `web/app/(dashboard)/certificates/page.tsx`

## Data Flow

1. User clicks "Import .p12" → inline import section appears
2. User selects a `.p12`/`.pfx` file and enters password (+ optional display name, pre-filled from filename)
3. On submit: file read via `FileReader` as binary string
4. `forge.pkcs12.pkcs12FromAsn1(forge.asn1.fromDer(binary), password)` extracts cert + private key + optional CA chain
5. Extracted PEMs passed to `apiClient.createCertificate()` — reuses existing API
6. On success: import section closes, cert list reloads

## UI

On `web/app/(dashboard)/certificates/page.tsx`:

- Header row: **"Import .p12"** button (outline/secondary style) alongside existing "Add Certificate" button
- Clicking it shows an inline card section (same style as the create form):
  - **Name** — text input, pre-filled with filename (minus `.p12`/`.pfx` extension)
  - **Certificate file** — `<input type="file" accept=".p12,.pfx">`
  - **Password** — `<input type="password">`, placeholder "Certificate password"
  - **Import** + **Cancel** buttons
- Opening import section closes the create/edit form, and vice versa
- Errors shown inline above the buttons (not via `alert()`)

## Error Handling

| Scenario | Inline error message |
|----------|---------------------|
| Wrong password / corrupted file | "Failed to parse certificate: incorrect password or invalid file" |
| No certificate in file | "No certificate found in file" |
| No private key in file | "No private key found in file — only client certificates with keys are supported" |
| File read error | "Failed to read file" |

API errors (server/network): `alert()` consistent with existing create/delete handlers.

## Out of Scope

- Backend .p12 parsing endpoint
- Support for certificate chains beyond extracting the first CA cert
- .p12 export functionality
