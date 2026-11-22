# Domain Rules

Aturan di sini load-bearing. Alasan dan riwayat ada di `docs/decisions.md`.

## Elo-like rating (per profile, per skill)

```
E = 1 / (1 + 10^((d - r) / 400))
delta = K * (s - E)
r_new = clamp(r + delta, 400, 2200)
```

- `r` = rating skill, `d` = difficulty activity, `s` = score 0..1.
- Start rating **800** (pemula total), clamp 400-2200.
- K-factor: **32** (<20 attempt), **24** (20-99), **16** (100+).
- Implementasi: `apps/api/internal/rating/rating.go` + `rating_test.go`.

Contoh: `r=800`, `d=900`, `s=0.8`, 5 attempt -> `E≈0.36`, `delta≈+14.1`, `r≈814.1`.

## Band CEFR

| Level | Rating | IELTS kira-kira |
| --- | --- | --- |
| A1 | < 1000 | <= 2.5 |
| A2 | 1000-1199 | 3.0-4.0 |
| B1 | 1200-1399 | 4.5-5.5 |
| B2 | 1400-1599 | 5.5-6.5 |
| C1 | 1600-1799 | 7.0-8.0 |
| C2 | 1800+ | 8.5-9.0 |

Band ini sama dengan tabel `levels` (seed `00002_seed_levels.sql`). Hysteresis:
tampilkan level lebih tinggi hanya setelah rating lewat threshold ~25 poin, supaya
level tidak flip-flop.

## Attempt flow (satu transaksi)

Satu attempt = satu transaksi:

1. Insert `attempts`.
2. Lock baris `skill_ratings` dengan `FOR UPDATE`.
3. Hitung rating baru (`rating.Update`).
4. Update `skill_ratings` (+ `attempts_count`).
5. Insert `rating_events` (before/after/delta).
6. `StartLevel` level activity (insert-only) + `SeedCards` vocab level itu.
7. Opsional: insert `mistakes` dari feedback AI (belum ada; menyusul saat feedback AI).

Jangan pecah jadi beberapa request/transaksi. Lock mencegah race antar device.
Implementasi: `apps/api/internal/attempt`.

## Gate A1 -> A2 (semua wajib)

1. >= 80% activity A1 selesai (`LevelCompletion`).
2. Checkpoint terakhir `overall >= 0.70`.
3. Keempat skill rating >= 1000.

Kalau terpenuhi: `CompleteLevel(A1)` lalu `StartLevel(A2)`.

## SRS (SM-2 style)

Implementasi: `apps/api/internal/srs/srs.go` + `srs_test.go`. Grade: 0 forgot, 1 hard, 2 good, 3 easy.

- **Forgot**: `lapses++`, `reps=0`, `interval=1`, `ease -= 0.2` (min 1.3), due +10 menit.
- **Good**: interval 1 (reps 0), 3 (reps 1), lalu `round(interval * ease)`.
- **Hard**: `ease -= 0.15`, `interval = round(interval * 0.8)`.
- **Easy**: `ease += 0.15`, `interval = round(interval * 1.3)`.

**Review vocab tidak mengubah skill rating.** Hanya unit activity yang mengubah rating.

Kartu dibuat oleh `SeedCards` saat attempt pertama di level/track tersebut, tidak saat
registrasi profil. `apps/api/internal/dashboard` memakai aturan streak D-024.

## Difficulty activity

```
unit_base = 840 + (week - 1) * 10        # week 1 = 840 ... week 16 = 990
difficulty = unit_base + offset
```

Offset per posisi activity (9 per unit, urutan tetap):

| # | Type | Skill | Offset |
| --- | --- | --- | --- |
| 1 | `vocab` | reading | 0 |
| 2 | `quiz` | reading | 0 |
| 3 | `quiz` | reading | +10 |
| 4 | `listening` | listening | -10 |
| 5 | `listening` | listening | +10 |
| 6 | `quiz` (ada `passage`) | reading | 0 |
| 7 | `writing` | writing | +20 |
| 8 | `speaking` | speaking | -10 |
| 9 | `speaking` | speaking | +20 |

Checkpoint A1: semua item di difficulty **1020**, sedikit di atas garis A2.

## Scoring

- Quiz: `correct / total`.
- Writing & speaking: skor AI 0..1 dengan rubrik IELTS; feedback lengkap disimpan di
  `attempts.feedback` agar bisa diaudit.
- Activity "vocab quiz" dihitung sebagai **reading**.

## Mistakes

`mistakes` mencatat `category` + `example`; `count` bertambah kalau berulang. Dipakai
untuk "mistake clinic" mingguan dan review hari Jumat.
