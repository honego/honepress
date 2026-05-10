/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "export",
  basePath: "/admin",
  assetPrefix: "/admin",
  images: { unoptimized: true },
};

export default nextConfig;
