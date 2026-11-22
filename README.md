# eng-geek

Personalized English learning tracker for a two-person household (software engineer + nurse),
from A1 to IELTS/OET B2-C1. Two profiles, shared core plus professional role packs.

## Status

API v1 serves profiles, units, activities, attempt recording (one transaction with
the skill rating update), the profile dashboard, and the level gate
(`GET /profiles/{id}/gate`, `POST /assessments`, `GET /profiles/{id}/assessments`).
Content is seeded from `apps/api/content/a1`. The web app covers the core loop: pick a
profile, see the dashboard, open a unit, answer vocab/quiz/listening activities (writing
and speaking are placeholders until AI scoring lands), and track the A1 -> A2 gate with
manual checkpoint/placement scores on `/levels`. Design and domain rules live in `docs/`;
`CLAUDE.md` is the entry point for Claude Code.

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
bun run seed:api        # content YAML + demo household/profiles
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

## Deploy (VPS)

The production deploy is native NixOS (no Docker) at
`https://enggeek.arifinoid.web.id`: a Go systemd service on `127.0.0.1:8083`
plus nginx serving the static export and proxying `/api/v1`. See
[`deploy/vps/README.md`](deploy/vps/README.md) and run:

```bash
bash deploy/vps/deploy.sh
```

## Documents

- `CLAUDE.md` — guidance for Claude Code
- `docs/architecture.md` — monorepo layout, stack, boundaries, data model
- `docs/domain.md` — rating, SRS, gates
- `docs/content.md` — A1 content plan and JSON shapes
- `docs/roadmap.md` — goals and build order
- `docs/decisions.md` — decisions and rationale
