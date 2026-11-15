# eng-geek

Personalized English learning tracker for a two-person household (software engineer + nurse),
from A1 to IELTS/OET B2-C1. Two profiles, shared core plus professional role packs.

## Status

Scaffold only. The domain specs live in the three `english-*.md` documents at the repository
root; `CLAUDE.md` holds the guidance for Claude Code.

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

- `english-ielts-tracker-plan.md`
- `english-tracker-db-and-rating.md`
- `english-a1-seed-content-plan.md`
- `CLAUDE.md` — guidance for Claude Code
