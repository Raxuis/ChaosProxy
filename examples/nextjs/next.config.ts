import type { NextConfig } from "next";

const apiOrigin = process.env.CHAOSPROXY_URL ?? process.env.API_ORIGIN ?? "http://localhost:9000";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiOrigin}/api/:path*` }];
  },
};

export default nextConfig;
