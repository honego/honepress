import { resolve } from "node:path";

import { defineConfig } from "vite";

export default defineConfig({
  build: {
    emptyOutDir: true,
    lib: {
      entry: resolve(__dirname, "src/theme.ts"),
      name: "BlogTheme",
      formats: ["iife"],
      fileName: () => "theme.js",
    },
  },
});
