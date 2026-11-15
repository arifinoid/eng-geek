# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Personalized English learning tracker for one household with two profiles: a software
engineer and a nurse. Starts at A1, targets IELTS/OET B2-C1 for working abroad. Progress
is tracked per CEFR level and per skill (listening, reading, writing, speaking).

## Stack

Go 1.24 + Postgres (`pgx`, `sqlc`, `goose`) · `River` jobs · Anthropic SDK + Whisper ·
TypeScript PWA (React + Vite) in `web/` · Nix flake · single Docker image.

## Gate

Run from the repository root; all must pass before a change is done:

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
```

Database tooling (install with Nix or `go install`; not vendored):

```bash
sqlc generate
goose -dir db/migrations postgres "$DATABASE_URL" up
```

## Docs

Read the relevant doc before changing that area:

- `docs/architecture.md` — stack, layout, boundaries, data model.
- `docs/domain.md` — Elo rating, SRS, attempt transaction, level gates, difficulty.
- `docs/content.md` — A1 unit map, activity template, content JSON shapes, seeder.
- `docs/roadmap.md` — goals, features, build order, status, open questions.
- `docs/decisions.md` — decisions and rationale.

## Non-negotiables

- `skill` lives on `activities`, not `units`.
- Recording an attempt is one transaction; lock the `skill_ratings` row `FOR UPDATE`.
- An activity is done at `score >= 0.7`.
- Vocab SRS reviews never change skill ratings.
- A1 -> A2 gate: >=80% activities, checkpoint `overall >= 0.70`, all ratings `>= 1000`.
- `db/migrations` is the schema source of truth; `internal/store` is generated.

## Conventions

- Code MUST be English: identifiers, comments, API/CLI messages, UI copy, test titles.
  Prose docs may be Indonesian.
- Do not commit secrets, `.env`, or personal data.
- Workflow: work on a branch, push, and open a PR. Do not merge; the user merges.
- Commit only when the user asks.
