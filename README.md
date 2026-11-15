# eng-geek

Personalized English learning tracker for a two-person household (software engineer + nurse),
from A1 to IELTS/OET B2-C1. Two profiles, shared core plus professional role packs.

## Status

Scaffold only. Design and domain rules live in `docs/`; `CLAUDE.md` is the entry point
for Claude Code.

## Stack

Go 1.24 + Postgres (`pgx`, `sqlc`, `goose`) backend; TypeScript PWA (React + Vite) frontend;
Nix flake dev shell; single Docker image for deploy.

## Quickstart

```bash
go test ./...   # rating + SRS unit tests (no DB needed)
```

Database tooling is not vendored; install `sqlc` and `goose` with Nix or `go install`, then:

```bash
sqlc generate
goose -dir db/migrations postgres "$DATABASE_URL" up
```

## Documents

- `CLAUDE.md` — guidance for Claude Code
- `docs/architecture.md` — stack, layout, data model
- `docs/domain.md` — rating, SRS, gates
- `docs/content.md` — A1 content plan and JSON shapes
- `docs/roadmap.md` — goals and build order
- `docs/decisions.md` — decisions and rationale
