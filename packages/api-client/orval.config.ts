import { defineConfig } from "orval";

export default defineConfig({
  api: {
    input: "../api-contract/openapi.yaml",
    output: {
      target: "./src/generated.ts",
      client: "react-query",
      mode: "split",
      // Root-absolute so a page at /id/ still calls /api/v1/* (the API mount).
      baseUrl: process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api/v1",
      // Keep `src/index.ts`; orval's clean would wipe the whole `src/` directory.
      clean: false,
    },
  },
});
