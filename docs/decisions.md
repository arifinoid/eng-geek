# Decisions

Keputusan desain + alasannya. Yang belum diputuskan ada di Open decisions.

## D-001: Mulai dari A1 untuk kedua profil

Dasar pemula absolut; placement test tetap dijalankan untuk baseline, bukan asumsi.
Alasan: kedua user mulai dari nol, dan konten A1 sudah terpetakan penuh.

## D-002: Backend Go + Postgres

Go 1.24, stdlib `net/http` (routing 1.22+). Postgres dengan `pgx/v5`. SQLite dipakai
hanya kalau self-hosted MVP, tapi Postgres dipilih karena `River` butuh Postgres.

## D-003: `sqlc` + `goose`

Query ditulis sebagai SQL dengan annotation `sqlc`, digenerate jadi tipe Go. Migration
pakai `goose` dengan anotasi `-- +goose Up/Down`. Alasan: schema dan query remain sumber
kebenaran, hindari ORM magic.

## D-004: Frontend TypeScript PWA (Next.js static export)

Next.js 16 App Router dengan `output: export`, next-intl (`id` default, `en`), Tailwind 4,
Serwist untuk service worker. Dipakai di phone dan desktop, installable, offline-ready.
Static export dipilih supaya tidak perlu runtime Node di produksi: server Go yang sajikan
file statis.

## D-005: Dua profil, shared core + role pack

Satu household, dua profil. `units.track` NULL = shared; `engineer`/`nurse` khusus.
Alasan: hemat penulisan konten, tapi tetap relevan dengan profesi.

## D-006: Progres dipisah per level dengan gate eksplisit

`level_progress` + `units` per level. Gate A1->A2: >=80% activity, checkpoint >=0.70,
keempat rating >=1000. Level baru pindah setelah gate lolos, bukan otomatis.

## D-007: Elo-like rating per skill

`E = 1/(1+10^((d-r)/400))`, `delta = K*(s-E)`, start 800, clamp 400-2200, K 32/24/16.
Alasan: sudah dikenal, mudah dijelaskan, dan bisa di-tune. Parameter pertama masih
tebakan; lihat Open decisions.

## D-008: Difficulty formula

`unit_base = 840 + (week-1)*10`, `difficulty = unit_base + offset` (offset -10..+20).
Checkpoint di 1020. Alasan: activity A1 berada di 830-1010, jadi learner yang konsisten
bisa mendorong rating melampaui 1000 meski konten dibatasi ~1000.

## D-009: Konten sebagai YAML + seeder

Konten tidak hard-code di Go. `cmd/seed` membaca `content/a1/*.yaml` dan upsert ke
Postgres, idempotent. Alasan: penulisan konten bisa paralel dan di-review via git.

## D-010: SRS SM-2 style

Implementasi di `internal/srs`. Review vocab **tidak** mengubah rating skill; hanya
unit activity yang mengubah. Alasan: memisahkan pengukuran kemampuan dari hapalan.

## D-011: `River` untuk background jobs

AI feedback dan transkripsi berjalan async via antrean Postgres. Alasan: API tetap
cepat, retry alami, dan tidak butuh broker terpisah.

## D-012: AI scoring 0-1 dengan rubrik IELTS, feedback tersimpan

Writing/speaking dinilai 0-1 memakai rubrik IELTS; feedback lengkap disimpan di
`attempts.feedback` agar skor bisa diaudit dan prompt bisa di-tune.

## D-013: `skill` di `activities`, bukan `units`

Satu unit bisa campur skill (vocab, listening, writing). Alasan: fleksibel dan cocok
dengan data nyata.

## D-014: Kode English, docs boleh Indonesia

Identifier, komentar, pesan API/CLI, UI copy, dan judul test wajib English. Prose docs
boleh Indonesia. Alasan: konsisten di codebase, tapi tetap terbaca oleh pemilik.

## D-015: Workflow branch + PR

Agent bekerja di branch, push, buka PR; user yang merge. Commit hanya kalau diminta.
Alasan: preferensi user.

## D-016: Struktur docs

`CLAUDE.md` singkat sebagai indeks + gate; detail per topik di `docs/` (architecture,
domain, content, roadmap, decisions). Alasan: CLAUDE.md yang gemuk anti-pattern; docs
bisa tumbuh tanpa membebani konteks Claude Code.

## D-017: Monorepo Bun workspaces

Satu repo: `apps/api` (Go module), `apps/web` (Next.js), `packages/*` (api-contract,
api-client, tsconfig). Root `package.json` punya workspace scripts (`dev:*`, `build:web`,
`test:*`). Alasan: satu lockfile TypeScript, kontrak API dipakai bersama, dan riwayat git
untuk backend + frontend menyatu.

## D-018: API contract dulu (OpenAPI), client digenerate

Kontrak di `packages/api-contract/openapi.yaml`. TS client digenerate via orval ke
`packages/api-client`; server Go memakai tipe dari `oapi-codegen`. Alasan: frontend dan
backend tidak drift, dan URL/field punya satu sumber kebenaran.

## D-019: Satu Docker image

Image produksi berisi binary Go + static export (`apps/web/out`). Server Go melayani
`/api/v1` dan file statis, dengan `STATIC_DIR` menunjuk ke hasil export. Alasan: deploy
satu container, tanpa Node runtime, dan PWA tetap offline.

## D-020: UI bahasa Indonesia (default) + Inggris

`next-intl` dengan locale `id` (default) dan `en`. Alasan: pemilik aplikasi berbahasa
Indonesia, tapi konten belajar dan kode tetap Inggris.

## D-021: Seeder idempoten + validator konten murni

`apps/api/cmd/seed` membaca `apps/api/content/a1/*.yaml`, memvalidasi (template 9
activity, formula difficulty, jawaban, skill lengkap, prefix filename), lalu upsert.
Validator di `internal/content` murni (tanpa DB) supaya bisa dites. Activity unit
tidak diubah lagi setelah unit punya attempt, agar riwayat attempt tetap valid.
Alasan: konten bisa diedit dan di-review via git, tapi data attempt tidak boleh
rusak.

## D-022: Response API pakai DTO, bukan struct sqlc

sqlc memetakan enum Postgres ke tipe Go (mis. `NullTrack`) yang serialisasinya
tidak cocok dengan kontrak (`{"track":"","valid":false}`). Handler `internal/api`
memetakan ke DTO sendiri (`track: null` atau string) supaya JSON mengikuti
`packages/api-contract`. Alasan: kontrak API adalah sumber kebenaran frontend.

## D-023: Attempt + rating satu transaksi, level & SRS aktif di attempt pertama

`internal/attempt` membungkus seluruh langkah domain: insert `attempts`, lock
`skill_ratings` `FOR UPDATE`, hitung Elo, update rating + `attempts_count`, tulis
`rating_events`, lalu `StartLevel` dan `SeedCards` untuk level activity tersebut.
`StartLevel` bersifat insert-only (`ON CONFLICT DO NOTHING`) supaya level yang sudah
`completed` tidak dihidupkan lagi oleh attempt berikutnya. Kartu SRS dibuat pada
attempt pertama di level itu, jadi review vocab langsung punya isi. Alasan: attempt
dan rating tidak boleh pernah berbeda, dan lock mencegah race antar device.

## D-024: Dashboard composition + aturan streak

`internal/dashboard` menyusun streak, menit belajar mingguan, rating keempat skill,
dan tiga aktivitas berikutnya per skill (skill terlemah lebih dulu). Streak dihitung
dari `attempts` per hari di timezone profil; streak masih hidup kalau hari belajar
terakhir adalah kemarin, jadi belum belajar hari ini tidak menghapus streak.
`week_minutes` adalah menit pada minggu berjalan, `weekly_minutes` adalah bucket
mingguan (default 8, maks 52). Alasan: aturan streak harus eksplisit dan bisa dites
tanpa DB, jadi `Streak` murni dan `weekly_minutes` dihitung dari bucket.

## Open decisions

- Gate A2->B1 dan seterusnya: sementara reuse pola 80% / 70% / next-level-min.
- Checkpoint test manual atau auto-scheduled tiap ~6 minggu.
- Repeat attempt: hitung hanya attempt pertama per activity per hari, atau skala
  `delta * 0.5` untuk pengulangan.
- Threshold hysteresis (~25 poin) perlu diverifikasi dengan data nyata.
- `activity_type`: tambah `reading`, atau tetap `quiz` + `passage`.
- Provider TTS & voice (US vs UK; campur aksen mulai A2).
- Unlock unit awal kalau 80% unit selesai.
- Review week shared: jadwal bersama atau masing-masing profil.
- Nurse: IELTS atau OET, negara mana, tanggal exam.
