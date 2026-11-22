import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
import withSerwist from "@serwist/next";

const withNextIntl = createNextIntlPlugin("./i18n/request.ts");

const withPWA = withSerwist({
  swSrc: "app/sw.ts",
  swDest: "public/sw.js",
  register: true,
  reloadOnOnline: true,
});

const nextConfig: NextConfig = {
  output: "export",
  trailingSlash: true,
  images: { unoptimized: true },
  // The API client uses a relative base URL, so `next dev` proxies to the Go
  // server. The production build is served by that same server.
  async rewrites() {
    return [
      {
        source: "/api/v1/:path*",
        destination: `${process.env.API_BASE_URL ?? "http://localhost:8080"}/api/v1/:path*`,
      },
    ];
  },
};

export default withNextIntl(withPWA(nextConfig));
