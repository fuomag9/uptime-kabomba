'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient, Certificate, CreateCertificateRequest, UpdateCertificateRequest } from '@/lib/api';
import { parseP12, P12ParseError } from '@/lib/parsep12';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Card, CardContent } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { toast } from 'sonner';

export default function CertificatesPage() {
  const router = useRouter();
  const [certs, setCerts] = useState<Certificate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Certificate | null>(null);

  const [name, setName] = useState('');
  const [certPem, setCertPem] = useState('');
  const [keyPem, setKeyPem] = useState('');
  const [caPem, setCaPem] = useState('');
  const [saving, setSaving] = useState(false);

  const [showImport, setShowImport] = useState(false);
  const [importName, setImportName] = useState('');
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importPassword, setImportPassword] = useState('');
  const [importError, setImportError] = useState<string | null>(null);
  const [importing, setImporting] = useState(false);

  const [deleteId, setDeleteId] = useState<number | null>(null);

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
      toast.error('Failed to save: ' + (err.message || 'Unknown error'));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await apiClient.deleteCertificate(id);
      setCerts(certs.filter((c) => c.id !== id));
    } catch (err: any) {
      toast.error('Failed to delete: ' + (err.message || 'Unknown error'));
    }
  }

  function handleDeleteRequest(id: number) {
    setDeleteId(id);
  }

  function handleDeleteConfirm() {
    if (deleteId !== null) {
      handleDelete(deleteId);
      setDeleteId(null);
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
        toast.error('Failed to import: ' + (err.message || 'Unknown error'));
      }
    } finally {
      setImporting(false);
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-48" />
          <div className="flex gap-3">
            <Skeleton className="h-8 w-28" />
            <Skeleton className="h-8 w-36" />
          </div>
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Client Certificates</h1>
        <div className="flex gap-3">
          <Button variant="outline" onClick={openImport}>
            Import .p12
          </Button>
          <Button onClick={openCreate}>
            Add Certificate
          </Button>
        </div>
      </div>

      {error && (
        <Alert variant="destructive" className="mb-4">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Dialog open={showImport} onOpenChange={(open) => { if (!open) setShowImport(false); }}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Import .p12 Certificate</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleImport} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="import-name">Name</Label>
              <Input
                id="import-name"
                type="text"
                value={importName}
                onChange={(e) => setImportName(e.target.value)}
                placeholder="Leave blank to use filename"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="import-file">Certificate file (.p12 / .pfx)</Label>
              <Input
                id="import-file"
                type="file"
                accept=".p12,.pfx"
                required
                onChange={(e) => setImportFile(e.target.files?.[0] ?? null)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="import-password">Password</Label>
              <Input
                id="import-password"
                type="password"
                value={importPassword}
                onChange={(e) => setImportPassword(e.target.value)}
                placeholder="Certificate password"
              />
            </div>
            {importError && (
              <Alert variant="destructive">
                <AlertDescription>{importError}</AlertDescription>
              </Alert>
            )}
            <div className="flex gap-3 justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={() => setShowImport(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={importing}
              >
                {importing ? 'Importing...' : 'Import'}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={showForm} onOpenChange={(open) => { if (!open) setShowForm(false); }}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? 'Edit Certificate' : 'New Certificate'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSave} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="cert-name">Name</Label>
              <Input
                id="cert-name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cert-pem">Client Certificate (PEM)</Label>
              <Textarea
                id="cert-pem"
                value={certPem}
                onChange={(e) => setCertPem(e.target.value)}
                required
                rows={6}
                className="font-mono text-sm"
                placeholder="-----BEGIN CERTIFICATE-----"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="key-pem">
                Private Key (PEM){editing && ' \u2014 leave blank to keep existing'}
              </Label>
              <Textarea
                id="key-pem"
                value={keyPem}
                onChange={(e) => setKeyPem(e.target.value)}
                required={!editing}
                rows={6}
                className="font-mono text-sm"
                placeholder={editing ? '\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022' : '-----BEGIN PRIVATE KEY-----'}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ca-pem">CA Certificate (PEM, optional)</Label>
              <Textarea
                id="ca-pem"
                value={caPem}
                onChange={(e) => setCaPem(e.target.value)}
                rows={6}
                className="font-mono text-sm"
                placeholder="-----BEGIN CERTIFICATE----- (optional)"
              />
            </div>
            <div className="flex gap-3 justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={() => setShowForm(false)}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={saving}
              >
                {saving ? 'Saving...' : 'Save'}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      {certs.length === 0 && !showForm && !showImport ? (
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">No certificates yet. Add one to use mTLS with HTTP monitors.</p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>CA</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {certs.map((cert) => (
                  <TableRow key={cert.id}>
                    <TableCell className="font-medium">{cert.name}</TableCell>
                    <TableCell>{cert.ca_pem ? 'Yes' : 'No'}</TableCell>
                    <TableCell>{new Date(cert.created_at).toLocaleDateString()}</TableCell>
                    <TableCell className="text-right space-x-2">
                      <Button variant="ghost" size="sm" onClick={() => openEdit(cert)}>
                        Edit
                      </Button>
                      <Button variant="destructive" size="sm" onClick={() => handleDeleteRequest(cert.id)}>
                        Delete
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <AlertDialog open={deleteId !== null} onOpenChange={(open) => { if (!open) setDeleteId(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete certificate?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the certificate.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={handleDeleteConfirm}>
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
