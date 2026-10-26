import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  cacheComponents: true,
  partialPrefetching: true,
  webpack(config, { isServer }) {
    config.resolve.alias = {
      ...config.resolve.alias,
      "msw/browser": isServer ? false : `${process.cwd()}/node_modules/msw/lib/browser/index.js`,
      "@/mocks/browser": isServer || process.env.NODE_ENV === "production" ? false : `${process.cwd()}/src/mocks/browser.ts`,
      [`${process.cwd()}/src/mocks/browser.ts`]: isServer || process.env.NODE_ENV === "production" ? false : `${process.cwd()}/src/mocks/browser.ts`,
    };
    return config;
  },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
