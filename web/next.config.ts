import type { NextConfig } from "next";

// Get backend URL from environment variable or default to internal Docker network
const INTERNAL_API_URL = process.env.INTERNAL_API_URL || "http://backend:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  compress: true,
  reactStrictMode: true,
  typescript: {
    ignoreBuildErrors: false,
  },
  experimental: {
    optimizePackageImports: ["lucide-react", "recharts"],
  },
  // Proxy API requests to backend (only accessible via frontend)
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${INTERNAL_API_URL}/api/:path*`,
      },
      {
        source: "/ws",
        destination: `${INTERNAL_API_URL}/ws`,
      },
      {
        source: "/health",
        destination: `${INTERNAL_API_URL}/health`,
      },
    ];
  },
  // Security headers for HTML documents. The backend's SecurityHeadersMiddleware
  // only ever applies to its own /api/* JSON responses - browsers take CSP/
  // clickjacking protection from the HTML document's own response headers, so
  // without this the frontend served plain HTML with none of that protection.
  // Content-Security-Policy is set separately in middleware.ts, not here:
  // it needs a per-request nonce for Next.js's own inline hydration scripts,
  // which a static header value (all this function can produce) can't express.
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "geolocation=(), microphone=(), camera=()" },
        ],
      },
    ];
  },
};

export default nextConfig;
