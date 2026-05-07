import { resolve } from "node:path";

import { defineConfig } from "vite";

export default defineConfig({
  publicDir: "../public",
  build: {
    outDir: "../../dist/theme",
    emptyOutDir: true,
    manifest: true,
    assetsDir: "assets",
    rollupOptions: {
      input: {
        theme: resolve(__dirname, "src/theme.ts"),
      },
      output: {
        entryFileNames: "assets/[name].[hash].js",
        chunkFileNames: "assets/[name].[hash].js",
        assetFileNames: "assets/[name].[hash][extname]",
      },
    },
  },
});
