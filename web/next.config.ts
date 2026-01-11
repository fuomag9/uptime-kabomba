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
};

export default nextConfig;
