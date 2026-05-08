/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "export",
  basePath: "/admin",
  assetPrefix: "/admin",
  images: { unoptimized: true },
};

module.exports = nextConfig;
