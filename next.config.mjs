import { createMDX } from "fumadocs-mdx/next";

const withMDX = createMDX();

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  basePath: "/website",
  output: "export",
  images: {
    unoptimized: true,
  },
};

export default withMDX(config);
