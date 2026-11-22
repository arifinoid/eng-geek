# Architecture

## Sistem

Aplikasi belajar bahasa Inggris personalized untuk satu rumah tangga dengan dua profil:
software engineer dan perawat (nurse). Mulai dari A1, target IELTS/OET B2-C1 untuk
bekerja di luar negeri. Progres dicatat per level CEFR dan per skill
(listening, reading, writing, speaking).

## Stack

| Layer | Pilihan |
| --- | --- |
| Backend | Go 1.24+, stdlib `net/http` (routing Go 1.22+) di `apps/api` |
| DB | Postgres, `pgx/v5`, `sqlc` (typed queries), `goose` (migrations) |
| Background jobs | `River` (Postgres-backed) untuk AI feedback dan transkripsi |
| AI | Anthropic Go SDK (writing/speaking feedback), Whisper (speech-to-text) |
| Frontend | TypeScript PWA: Next.js 16 App Router, static export (`output: export`), next-intl (`id`/`en`), Tailwind 4, Serwist — di `apps/web` |
| API contract | OpenAPI 3.1 di `packages/api-contract`; Go digenerate (oapi-codegen), TS client digenerate (orval) ke `packages/api-client` |
| Monorepo | Bun workspaces (`apps/*`, `packages/*`), root `package.json` |
| Dev env | Nix flake (`nix develop`) |
| Deploy | Satu Docker image: binary Go sajikan `/api/v1` + static export |

## Layout

```
apps/api/            Go module: github.com/arifinoid/eng-geek/apps/api
  cmd/server/        HTTP server (/api/v1 + healthz + static export)
  cmd/seed/          seeder YAML -> Postgres (idempotent)
  cmd/tts/           script/lines -> audio (rencana)
  db/migrations/     migrations goose (source of truth schema)
  db/queries/        query sqlc per domain
  internal/api/      handler HTTP /api/v1 (mapping request/JSON)
  internal/attempt/  attempt + rating update dalam satu transaksi
  internal/dashboard/ streak, waktu belajar, aktivitas berikutnya
  internal/content/  parser + validator konten YAML (pure)
  internal/rating/   Elo per skill (pure, no DB)
  internal/srs/      SM-2 SRS (pure)
  internal/store/    output sqlc (generated, jangan edit)
  internal/config/   env config (PORT, DATABASE_URL, STATIC_DIR)
  content/a1/        konten A1 sebagai YAML
apps/web/            Next.js static export PWA
  app/[locale]/      routing i18n (id default, en) + halaman
  components/        client components (header, dashboard, unit, activity runner)
  lib/               API hooks, profil aktif, parser konten aktivitas (pure)
  i18n/              next-intl routing + request config
  messages/          id.json, en.json
packages/api-contract/  OpenAPI 3.1
packages/api-client/    TS client generated (orval + react-query)
packages/tsconfig/      tsconfig together (base, next)
flake.nix            dev shell: go, bun, sqlc, goose, postgresql
docker-compose.yml   Postgres dev (+ api)
Dockerfile           multi-stage: web export + Go binary -> satu image
```

## Batas (boundaries)

- `apps/api/db/migrations/` adalah source of truth schema. `internal/store` digenerate
  oleh `sqlc generate` (dijalankan dari `apps/api`); jangan edit manual.
- `internal/rating` dan `internal/srs` murni (tanpa DB/HTTP) supaya mudah ditest.
- Logika domain (rating, gate, SRS) tidak boleh berada di handler HTTP.
- Konten ada di YAML (`apps/api/content/`), bukan hard-code di Go. Seeder mengubahnya
  menjadi baris `units`, `activities`, dan `vocab_items`.
- Frontend hanya lewat API; jangan query DB langsung dari `apps/web`. Kontrak API
  ditulis di `packages/api-contract/openapi.yaml`, lalu client TS digenerate ke
  `packages/api-client` (orval) dan server Go memakai tipe dari `oapi-codegen`.
- Satu Docker image: server Go melayani `/api/v1` dan file statis dari `apps/web/out`
  (hasil `bun run build:web`). `STATIC_DIR` menunjuk ke direktori export (default
  `../web/out`, dihitung dari `apps/api` tempat server dijalankan).
- Halaman web memakai query string (`/practice/units?id=`, `/practice/activities?id=`)
  bukan segment `[id]`, karena `output: export` menuntut `generateStaticParams` untuk
  segment dinamis dan isinya berasal dari DB. Profil aktif disimpan di `localStorage`
  (`eng-geek.profileId`), bukan di URL. `next dev` mem-proxy `/api/v1/*` ke server Go
  lewat `rewrites` di `next.config.ts` karena base URL client relatif.

## Data model (ringkas)

Source of truth: `apps/api/db/migrations/00001_init.sql`. Seed level:
`apps/api/db/migrations/00002_seed_levels.sql`.

| Tabel | Isi |
| --- | --- |
| `households`, `profiles` | profil: `track` (engineer/nurse), `target_exam` (ielts/oet), `exam_date`, `timezone` |
| `levels` | A1..C2, `rating_min`/`rating_max`, estimasi IELTS band |
| `units` | `level_code`, `track` (NULL = shared), `position`; UNIQUE `(level_code, track, position)` |
| `activities` | `unit_id`, `type`, `skill`, `difficulty`, `content` JSONB, `position` |
| `attempts` | `profile_id`, `activity_id`, `score` 0..1, `duration_sec`, `feedback` JSONB |
| `skill_ratings` | `profile_id`, `skill`, `rating`, `attempts_count` |
| `rating_events` | riwayat perubahan rating (untuk grafik) |
| `level_progress` | status `locked`/`in_progress`/`completed` per level |
| `assessments` | `placement`/`checkpoint`, `scores` JSONB, `overall` |
| `vocab_items` | `term`, `meaning`, `example`, `level_code`, `track` |
| `srs_cards` | `due_at`, `interval_days`, `ease`, `reps`, `lapses` |
| `mistakes` | `category`, `example`, `count`, `last_seen` |

Aturan penting:

- `skill` ada di `activities`, bukan `units` — satu unit bisa campur skill.
- `track` NULL di `units`/`vocab_items` = shared untuk semua profil.
- Pemisahan per level: `level_progress` + `units` per level memberi persentase
  penyelesaian, waktu belajar, dan tren skor sendiri-sendiri.
- Sebuah activity dianggap selesai kalau profil punya attempt dengan `score >= 0.7`.

## Development workflows

```bash
nix develop                     # toolchain: go, bun, sqlc, goose, postgres
bun install                     # workspace deps
bun run dev:api                 # Go server (:8080)
bun run dev:web                 # Next dev (:3000)
bun run build:web               # static export -> apps/web/out
docker compose up -d db         # Postgres lokal
goose -dir apps/api/db/migrations postgres "$DATABASE_URL" up
bun run seed:api                # content YAML + demo household/profiles
```

API Contract: `GET /api/v1/profiles?householdId=`, `GET /api/v1/profiles/{profileId}`,
`GET /api/v1/profiles/{profileId}/dashboard?weeks=`, `GET /api/v1/units?profileId=`,
`GET /api/v1/units/{unitId}/activities`, `GET /api/v1/activities/{activityId}`,
`POST /api/v1/attempts`. Response memakai DTO di `internal/api` supaya tipe enum sqlc
(mis. `NullTrack`) serialisasi sesuai kontrak (`null`/string). Mount `/api/v1/` di
`cmd/server` tanpa method supaya routing method (GET/POST) diatur muxe API.

Catatan sqlc: `NUMERIC` dipetakan ke `pgtype.Numeric` dan override `db_type` tidak
berlaku di sqlc v1.31, jadi konversi angka ada di pemanggil query.
