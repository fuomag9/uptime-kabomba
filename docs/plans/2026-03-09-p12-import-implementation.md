# .p12 Certificate Import Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a client-side .p12 import button to the certificates page that parses the file in the browser using node-forge and saves the extracted PEMs via the existing API.

**Architecture:** Three tasks — install node-forge, create a pure parser utility, then add the import UI to the existing certificates page. No backend changes. The parser utility is extracted so it can be reasoned about independently of the UI.

**Tech Stack:** Next.js 16 (App Router), TypeScript, node-forge, Tailwind CSS

---

### Task 1: Install node-forge

**Files:**
- Modify: `web/package.json` (via npm)

**Step 1: Install the package**

```bash
cd web && npm install node-forge
```

node-forge v1.x ships its own TypeScript definitions, so no `@types/node-forge` is needed.

**Step 2: Verify TypeScript compiles**

```bash
cd web && npx tsc --noEmit
```

Expected: no errors. If you see "Could not find a declaration file for module 'node-forge'", install `@types/node-forge` as a dev dependency:
```bash
npm install --save-dev @types/node-forge
```

**Step 3: Commit**

```bash
git add web/package.json web/package-lock.json
git commit --no-gpg-sign -m "feat: add node-forge dependency for p12 parsing"
```

---

### Task 2: Create p12 parser utility

**Files:**
- Create: `web/lib/parsep12.ts`

**Step 1: Create the utility file**

```ts
// web/lib/parsep12.ts
// Parses a PKCS#12 (.p12/.pfx) binary string using node-forge.
// Returns PEM-encoded cert, private key, and optional CA certificate.

import * as forge from 'node-forge';

export interface ParsedP12 {
  certPem: string;
  keyPem: string;
  caPem: string; // empty string if no CA cert found
}

export class P12ParseError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'P12ParseError';
  }
}

/**
 * Parse a PKCS#12 binary string (from FileReader.readAsBinaryString) with a password.
 * Throws P12ParseError for all user-facing error conditions.
 */
export function parseP12(binaryString: string, password: string): ParsedP12 {
  let p12: forge.pkcs12.Pkcs12Pfx;
  try {
    const asn1 = forge.asn1.fromDer(binaryString);
    p12 = forge.pkcs12.pkcs12FromAsn1(asn1, password);
  } catch {
    throw new P12ParseError('Failed to parse certificate: incorrect password or invalid file');
  }

  // Extract certificate
  const certBags = p12.getBags({ bagType: forge.pki.oids.certBag });
  const certBagList = certBags[forge.pki.oids.certBag] ?? [];
  if (certBagList.length === 0 || !certBagList[0].cert) {
    throw new P12ParseError('No certificate found in file');
  }

  // Extract private key (pkcs8ShroudedKeyBag or keyBag)
  const keyBags = p12.getBags({
    bagType: forge.pki.oids.pkcs8ShroudedKeyBag,
  });
  const keyBagList = keyBags[forge.pki.oids.pkcs8ShroudedKeyBag] ?? [];

  // Fallback: unencrypted keyBag
  let privateKey: forge.pki.PrivateKey | null = null;
  if (keyBagList.length > 0 && keyBagList[0].key) {
    privateKey = keyBagList[0].key!;
  } else {
    const plainKeyBags = p12.getBags({ bagType: forge.pki.oids.keyBag });
    const plainKeyList = plainKeyBags[forge.pki.oids.keyBag] ?? [];
    if (plainKeyList.length > 0 && plainKeyList[0].key) {
      privateKey = plainKeyList[0].key!;
    }
  }

  if (!privateKey) {
    throw new P12ParseError(
      'No private key found in file — only client certificates with keys are supported'
    );
  }

  // First cert is the client cert; subsequent certs are CA chain
  const clientCert = certBagList[0].cert!;
  const certPem = forge.pki.certificateToPem(clientCert);
  const keyPem = forge.pki.privateKeyToPem(privateKey);

  // Use first CA cert if present (index 1+)
  let caPem = '';
  if (certBagList.length > 1 && certBagList[1].cert) {
    caPem = forge.pki.certificateToPem(certBagList[1].cert);
  }

  return { certPem, keyPem, caPem };
}
```

**Step 2: Verify TypeScript compiles**

```bash
cd web && npx tsc --noEmit
```

Expected: no errors

**Step 3: Commit**

```bash
git add web/lib/parsep12.ts
git commit --no-gpg-sign -m "feat: add p12 parser utility using node-forge"
```

---

### Task 3: Add import UI to the certificates page

**Files:**
- Modify: `web/app/(dashboard)/certificates/page.tsx`

The existing page (`web/app/(dashboard)/certificates/page.tsx`) has:
- `showForm` state for the create/edit inline form
- `openCreate()` / `openEdit()` that set `showForm = true`
- A header row with the "Add Certificate" button

We need to add a parallel `showImport` state and section.

**Step 1: Add the import to the page**

Read the current file first, then apply the following changes:

#### 1a. Update the import line at the top

Old:
```tsx
import { apiClient, Certificate, CreateCertificateRequest, UpdateCertificateRequest } from '@/lib/api';
```

New:
```tsx
import { apiClient, Certificate, CreateCertificateRequest, UpdateCertificateRequest } from '@/lib/api';
import { parseP12, P12ParseError } from '@/lib/parsep12';
```

#### 1b. Add import UI state after the existing `saving` state (line 20)

After:
```tsx
  const [saving, setSaving] = useState(false);
```

Add:
```tsx
  // Import .p12 state
  const [showImport, setShowImport] = useState(false);
  const [importName, setImportName] = useState('');
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importPassword, setImportPassword] = useState('');
  const [importError, setImportError] = useState<string | null>(null);
  const [importing, setImporting] = useState(false);
```

#### 1c. Update `openCreate` to also close the import section

Old:
```tsx
  function openCreate() {
    setEditing(null);
    setName(''); setCertPem(''); setKeyPem(''); setCaPem('');
    setShowForm(true);
  }
```

New:
```tsx
  function openCreate() {
    setEditing(null);
    setName(''); setCertPem(''); setKeyPem(''); setCaPem('');
    setShowImport(false);
    setShowForm(true);
  }
```

#### 1d. Update `openEdit` to also close the import section

Old:
```tsx
  function openEdit(cert: Certificate) {
    setEditing(cert);
    setName(cert.name); setCertPem(cert.cert_pem); setKeyPem(''); setCaPem(cert.ca_pem || '');
    setShowForm(true);
  }
```

New:
```tsx
  function openEdit(cert: Certificate) {
    setEditing(cert);
    setName(cert.name); setCertPem(cert.cert_pem); setKeyPem(''); setCaPem(cert.ca_pem || '');
    setShowImport(false);
    setShowForm(true);
  }
```

#### 1e. Add `openImport` and `handleImport` functions after `handleDelete`

After the `handleDelete` function, add:

```tsx
  function openImport() {
    setShowForm(false);
    setEditing(null);
    setImportName('');
    setImportFile(null);
    setImportPassword('');
    setImportError(null);
    setShowImport(true);
  }

  async function handleImport(e: React.FormEvent) {
    e.preventDefault();
    if (!importFile) return;
    setImporting(true);
    setImportError(null);

    try {
      const binary = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result as string);
        reader.onerror = () => reject(new Error('Failed to read file'));
        reader.readAsBinaryString(importFile);
      });

      const parsed = parseP12(binary, importPassword);
      const req: CreateCertificateRequest = {
        name: importName || importFile.name.replace(/\.(p12|pfx)$/i, ''),
        cert_pem: parsed.certPem,
        key_pem: parsed.keyPem,
        ca_pem: parsed.caPem,
      };
      await apiClient.createCertificate(req);
      setShowImport(false);
      await load();
    } catch (err: any) {
      if (err instanceof P12ParseError) {
        setImportError(err.message);
      } else {
        alert('Failed to import: ' + (err.message || 'Unknown error'));
      }
    } finally {
      setImporting(false);
    }
  }
```

#### 1f. Update the header row to add the "Import .p12" button

Old:
```tsx
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Client Certificates</h1>
        <button
          onClick={openCreate}
          className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary/90"
        >
          Add Certificate
        </button>
      </div>
```

New:
```tsx
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Client Certificates</h1>
        <div className="flex gap-3">
          <button
            onClick={openImport}
            className="rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2 text-sm font-semibold text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Import .p12
          </button>
          <button
            onClick={openCreate}
            className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary/90"
          >
            Add Certificate
          </button>
        </div>
      </div>
```

#### 1g. Add the import section after the `{error && ...}` line and before `{showForm && ...}`

After:
```tsx
      {error && <p className="mb-4 text-red-500">{error}</p>}
```

Add:
```tsx
      {showImport && (
        <div className="mb-6 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6">
          <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-white">Import .p12 Certificate</h2>
          <form onSubmit={handleImport} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name</label>
              <input
                type="text"
                value={importName}
                onChange={(e) => setImportName(e.target.value)}
                placeholder="Leave blank to use filename"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Certificate file (.p12 / .pfx)</label>
              <input
                type="file"
                accept=".p12,.pfx"
                required
                onChange={(e) => setImportFile(e.target.files?.[0] ?? null)}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</label>
              <input
                type="password"
                value={importPassword}
                onChange={(e) => setImportPassword(e.target.value)}
                placeholder="Certificate password"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white"
              />
            </div>
            {importError && (
              <p className="text-sm text-red-500">{importError}</p>
            )}
            <div className="flex gap-3">
              <button
                type="submit"
                disabled={importing}
                className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary/90 disabled:opacity-50"
              >
                {importing ? 'Importing...' : 'Import'}
              </button>
              <button
                type="button"
                onClick={() => setShowImport(false)}
                className="rounded-md bg-gray-200 dark:bg-gray-700 px-4 py-2 text-sm font-semibold text-gray-700 dark:text-gray-300"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}
```

**Step 2: Verify TypeScript compiles**

```bash
cd web && npx tsc --noEmit
```

Expected: no errors

**Step 3: Verify full build passes**

```bash
cd web && npm run build
```

Expected: build succeeds

**Step 4: Commit**

```bash
git add web/app/\(dashboard\)/certificates/page.tsx
git commit --no-gpg-sign -m "feat: add p12 import UI to certificates page"
```
