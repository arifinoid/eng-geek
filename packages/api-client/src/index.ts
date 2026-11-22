// Generated client lives in `generated.ts` (run `bun run generate:api-client`).
// This entry point re-exports it so `@eng-geek/api-client` has one stable import.
export * from "./generated";
// Types live in a separate file; re-export them so consumers only need one import.
export * from "./generated.schemas";
