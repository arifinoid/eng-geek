# eng-geek

Personalized English learning tracker for a two-person household (software engineer + nurse),
from A1 to IELTS/OET B2-C1. Two profiles, shared core plus professional role packs.

## Status

Scaffold only. Design and domain rules live in `docs/`; `CLAUDE.md` is the entry point
for Claude Code.

## Stack

Bun workspaces monorepo: Go 1.24+ + Postgres (`pgx`, `sqlc`, `goose`) backend in
`apps/api`; TypeScript PWA in `apps/web` (Next.js static export, next-intl `id`/`en`,
Tailwind 4, Serwist); OpenAPI contract in `packages/api-contract` with a generated client
in `packages/api-client`. Nix flake dev shell; single Docker image for deploy.

## Quickstart

```bash
nix develop        # go, bun, sqlc, goose, postgres
bun install
bun run test:api   # rating + SRS unit tests (no DB needed)
bun run test:web
```

Local Postgres + migrations:

```bash
docker compose up -d db
export DATABASE_URL=postgres://enggreek:enggreek@localhost:5432/enggreek?sslmode=disable
goose -dir apps/api/db/migrations postgres "$DATABASE_URL" up
```

Run the app (Go server serves the API and the static export):

```bash
bun run build:web
bun run dev:api     # http://localhost:8080 -> /id
```

Or build the whole thing as one image:

```bash
docker compose up --build
```

## Documents

- `CLAUDE.md` — guidance for Claude Code
- `docs/architecture.md` — monorepo layout, stack, boundaries, data model
- `docs/domain.md` — rating, SRS, gates
- `docs/content.md` — A1 content plan and JSON shapes
- `docs/roadmap.md` — goals and build order
- `docs/decisions.md` — decisions and rationale
