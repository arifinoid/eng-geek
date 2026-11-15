# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Personalized English learning tracker for one household with two profiles: a software
engineer and a nurse. Starts at A1, targets IELTS/OET B2-C1 for working abroad. Progress
is tracked per CEFR level and per skill (listening, reading, writing, speaking).

The three root documents are the source of truth for the design. Read them before changing
domain logic:

- `english-ielts-tracker-plan.md` — goals, users, features, data model, build order.
- `english-tracker-db-and-rating.md` — migration, sqlc queries, Elo math, SRS, attempt flow.
- `english-a1-seed-content-plan.md` — A1 unit map, activity template, content JSON shapes, seeding.

## Stack

| Layer | Choice |
| --- | --- |
| Backend | Go 1.24, stdlib `net/http` (Go 1.22+ routing) |
| DB | Postgres, `pgx/v5`, `sqlc` (typed queries), `goose` (migrations) |
| Jobs | `River` (Postgres-backed) for AI feedback and transcription |
| AI | Anthropic Go SDK; Whisper for speech-to-text |
| Frontend | TypeScript PWA (React + Vite) in `web/` |
| Dev env | Nix flake |
| Deploy | Single Docker image |

## Layout

```
db/migrations/   goose SQL migrations
db/queries/      sqlc query files (one per domain)
internal/rating/ Elo-like per-skill rating (pure, no DB)
internal/srs/    SM-2 style spaced repetition (pure)
internal/store/  sqlc-generated code (generated, do not edit)
cmd/seed/        content YAML -> Postgres seeder
cmd/tts/         script/lines -> audio files
content/a1/      authored A1 content as YAML
web/             TypeScript PWA
```

## Gate

Run from the repository root; all must pass before a change is done:

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
```

Database tooling (install with Nix or `go install`; not yet in this repo's flake):

```bash
sqlc generate
goose -dir db/migrations postgres "$DATABASE_URL" up
```

## Domain rules

These are load-bearing; the docs explain the why.

- Rating is per profile per skill. `E = 1/(1+10^((d-r)/400))`, `delta = K*(s-E)`,
  `K = 32 / 24 / 16` for `<20 / <100 / 100+` attempts. Start rating 800, clamp 400-2200.
- CEFR bands: A1 `<1000`, A2 `1000-1199`, B1 `1200-1399`, B2 `1400-1599`,
  C1 `1600-1799`, C2 `1800+`. Display uses ~25 points of hysteresis.
- An activity counts as done when the profile has an attempt with `score >= 0.7`.
- Recording an attempt is one transaction: insert attempt, lock the `skill_ratings` row
  `FOR UPDATE`, update the rating, insert a `rating_events` row. Never split these.
- A1 -> A2 gate (all required): >=80% of A1 activities done, latest checkpoint
  `overall >= 0.70`, all four skill ratings `>= 1000`.
- Vocab SRS reviews never change skill ratings. Only unit activities do. The vocab
  quiz activity counts as **reading**.
- `skill` lives on `activities`, not `units`; a unit can mix skills.
- Difficulty: `unit_base = 840 + (week-1)*10`, `difficulty = unit_base + offset`.
- Every unit has 9 activities in a fixed order (vocab, quiz, quiz, listening,
  listening, quiz+passage, writing, speaking, speaking). Content JSONB has one
  shape per `activity_type`; the frontend gets one renderer per type.

## Conventions

- Code MUST be English: identifiers, comments, API/CLI messages, UI copy, test titles.
  Prose docs may be Indonesian.
- Do not commit secrets, `.env`, or personal data.
- Workflow: work on a branch, push, and open a PR. Do not merge; the user merges.
- Commit only when the user asks.
