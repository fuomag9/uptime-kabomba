'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, Certificate, CreateCertificateRequest, UpdateCertificateRequest } from '@/lib/api';
import { parseP12, P12ParseError } from '@/lib/parsep12';

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

  // Import .p12 state
  const [showImport, setShowImport] = useState(false);
  const [importName, setImportName] = useState('');
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importPassword, setImportPassword] = useState('');
  const [importError, setImportError] = useState<string | null>(null);
  const [importing, setImporting] = useState(false);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      setCerts(await apiClient.getCertificates());
    } catch (err: any) {
      setError(err.message || 'Failed to load certificates');
      if (err.status === 401) router.push('/login');
    } finally {
      setLoading(false);
    }
  }, [router]);

  useEffect(() => { load(); }, [load]);

  function openCreate() {
    setEditing(null);
    setName(''); setCertPem(''); setKeyPem(''); setCaPem('');
    setShowImport(false);
    setShowForm(true);
  }

  function openEdit(cert: Certificate) {
    setEditing(cert);
    setName(cert.name); setCertPem(cert.cert_pem); setKeyPem(''); setCaPem(cert.ca_pem || '');
    setShowImport(false);
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
        reader.onload = () => {
          const buf = reader.result as ArrayBuffer;
          const bytes = new Uint8Array(buf);
          let str = '';
          for (let i = 0; i < bytes.length; i++) str += String.fromCharCode(bytes[i]);
          resolve(str);
        };
        reader.onerror = () => reject(new Error('Failed to read file'));
        reader.readAsArrayBuffer(importFile);
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

  if (loading) return <div className="p-8 text-center text-gray-500">Loading...</div>;

  return (
    <div>
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

      {error && <p className="mb-4 text-red-500">{error}</p>}

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
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Client Certificate (PEM, optional for mTLS)</label>
              <textarea
                value={certPem}
                onChange={(e) => setCertPem(e.target.value)}
                rows={6}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm font-mono text-gray-900 dark:text-white"
                placeholder="-----BEGIN CERTIFICATE----- (optional)"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Private Key (PEM, optional for mTLS){editing && ' — leave blank to keep existing'}
              </label>
              <textarea
                value={keyPem}
                onChange={(e) => setKeyPem(e.target.value)}
                rows={6}
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm font-mono text-gray-900 dark:text-white"
                placeholder={editing ? '••••••••' : '-----BEGIN PRIVATE KEY----- (optional)'}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">CA Certificate (PEM) — required for self-signed server certificates</label>
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

      {certs.length === 0 && !showForm && !showImport ? (
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
