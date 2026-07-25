import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// decodeURIComponent throws on a malformed percent-encoding (e.g. a bare "%"
// from a query param like ?error=%). Query values in this app can come from
// an OAuth provider or another untrusted redirect, so callers must not let
// that throw go unhandled - falls back to the raw string instead.
export function safeDecodeURIComponent(value: string | null): string {
  if (!value) return "";
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}
