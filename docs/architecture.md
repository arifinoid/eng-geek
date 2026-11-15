# Architecture

## Sistem

Aplikasi belajar bahasa Inggris personalized untuk satu rumah tangga dengan dua profil:
software engineer dan perawat (nurse). Mulai dari A1, target IELTS/OET B2-C1 untuk
bekerja di luar negeri. Progres dicatat per level CEFR dan per skill
(listening, reading, writing, speaking).

## Stack

| Layer | Pilihan |
| --- | --- |
| Backend | Go 1.24, stdlib `net/http` (routing Go 1.22+) |
| DB | Postgres, `pgx/v5`, `sqlc` (typed queries), `goose` (migrations) |
| Background jobs | `River` (Postgres-backed) untuk AI feedback dan transkripsi |
| AI | Anthropic Go SDK (writing/speaking feedback), Whisper (speech-to-text) |
| Frontend | TypeScript PWA (React + Vite) di `web/` |
| Dev env | Nix flake |
| Deploy | Satu Docker image |

## Layout

```
db/migrations/   migrations goose (source of truth schema)
db/queries/      query sqlc per domain
internal/rating/ Elo per skill (pure, no DB)
internal/srs/    SM-2 SRS (pure)
internal/store/  output sqlc (generated, jangan edit)
cmd/seed/        seeder YAML -> Postgres
cmd/tts/         script/lines -> file audio
content/a1/      konten A1 sebagai YAML
web/             TypeScript PWA
```

## Batas (boundaries)

- `db/migrations/` adalah source of truth schema. `internal/store` digenerate oleh
  `sqlc generate`; jangan edit manual.
- `internal/rating` dan `internal/srs` murni (tanpa DB/HTTP) supaya mudah ditest.
- Logika domain (rating, gate, SRS) tidak boleh berada di handler HTTP.
- Konten ada di YAML (`content/`), bukan hard-code di Go. Seeder mengubahnya menjadi
  baris `units`, `activities`, dan `vocab_items`.
- Frontend hanya lewat API; jangan query DB langsung dari `web/`.

## Data model (ringkas)

Source of truth: `db/migrations/00001_init.sql`. Seed level: `00002_seed_levels.sql`.

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
