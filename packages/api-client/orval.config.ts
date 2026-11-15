import { defineConfig } from "orval";

export default defineConfig({
  api: {
    input: "../api-contract/openapi.yaml",
    output: {
      target: "./src/generated.ts",
      client: "react-query",
      mode: "split",
      baseUrl: process.env.NEXT_PUBLIC_API_BASE_URL ?? "",
      // Keep `src/index.ts`; orval's clean would wipe the whole `src/` directory.
      clean: false,
    },
  },
});
