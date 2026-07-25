import { NextRequest, NextResponse } from "next/server";

// Content-Security-Policy needs to live in middleware rather than
// next.config.ts's static headers(): script-src must allow Next.js's own
// per-request inline hydration/bootstrap scripts, which requires a nonce
// that's unique per request - something a static header value can't express.
//
// 'strict-dynamic' lets scripts loaded by an already-nonce'd script run too
// (Next's chunk loader), while browsers that honor it ignore the 'self'
// fallback for script-src entirely - the nonce is what actually gates
// execution. Everything else stays as a static allowlist.
export function middleware(request: NextRequest) {
  const nonce = Buffer.from(crypto.randomUUID()).toString("base64");

  const cspHeader = [
    "default-src 'self'",
    // 'sha256-n46v...' allows next-themes' fixed, framework-generated
    // FOUC-prevention script (it sets the theme class before hydration and
    // isn't nonce-tagged by Next.js). Its content is static/deterministic,
    // not user-influenced; if next-themes' config changes, this hash will
    // need updating too (the browser console reports the expected hash).
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic' 'sha256-n46vPwSWuMC0W703pBofImv82Z26xo4LXymv0E9caPk='`,
    // Admin-supplied custom CSS on the public status page is rendered via
    // <style dangerouslySetInnerHTML>; it's sanitized server-side before
    // storage (see internal/api/status_page_handlers.go), so this is a
    // deliberately scoped exception rather than a blanket one.
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data:",
    "font-src 'self' data:",
    "connect-src 'self' ws: wss:",
    "frame-ancestors 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ].join("; ");

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("Content-Security-Policy", cspHeader);

  const response = NextResponse.next({
    request: { headers: requestHeaders },
  });
  response.headers.set("Content-Security-Policy", cspHeader);
  return response;
}

export const config = {
  matcher: [
    // Skip static assets and image optimization - CSP only matters for
    // documents/scripts the browser executes.
    "/((?!_next/static|_next/image|favicon.ico|icon.svg).*)",
  ],
};
