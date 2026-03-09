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
    privateKey = keyBagList[0].key;
  } else {
    const plainKeyBags = p12.getBags({ bagType: forge.pki.oids.keyBag });
    const plainKeyList = plainKeyBags[forge.pki.oids.keyBag] ?? [];
    if (plainKeyList.length > 0 && plainKeyList[0].key) {
      privateKey = plainKeyList[0].key;
    }
  }

  if (!privateKey) {
    throw new P12ParseError(
      'No private key found in file — only client certificates with keys are supported'
    );
  }

  // First cert is the client cert; subsequent certs are CA chain
  const clientCert = certBagList[0].cert;
  const certPem = forge.pki.certificateToPem(clientCert);
  const keyPem = forge.pki.privateKeyToPem(privateKey);

  // Use first CA cert if present (index 1+)
  let caPem = '';
  if (certBagList.length > 1 && certBagList[1].cert) {
    caPem = forge.pki.certificateToPem(certBagList[1].cert);
  }

  return { certPem, keyPem, caPem };
}
