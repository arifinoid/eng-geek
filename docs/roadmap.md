# Roadmap

## Tujuan

- A1 -> B2 (IELTS ~6.0-6.5), lalu C1 (IELTS ~7-8).
- Level saat ini selalu diketahui per skill.
- Progres dipisah per level.
- Dua profil dengan track profesional berbeda.

## Timeline realistis

- Weekday 30 menit/hari; weekend 1-2 jam/hari; ~5-6 jam/minggu.
- A1 -> B2 butuh ~400-600 jam belajar -> **~18-24 bulan** dengan pace ini.
- Milestone: A2 ~4 bulan, B1 ~10 bulan.
- Konsistensi lebih penting daripada durasi sesi; tracker harus melindungi streak.

## Users & tracks

- Satu household, dua profil, masing-masing punya level, rencana, dan riwayat.
- **Shared core:** grammar, vocabulary umum, skill IELTS.
- **Engineer pack:** stand-up, code review, job interview, vocab tech.
- **Nurse pack:** handover SBAR, vocab medis/gejala, percakapan pasien.
- `target_exam`: `ielts` | `oet` (nurse mungkin OET, cek negara tujuan).

## Fitur

### MVP

- Profiles (household dengan 2 user).
- Placement test -> CEFR + estimasi IELTS band.
- Empat skill terpisah: listening, reading, writing, speaking.
- Daily plan dari skill terlemah + deadline.
- Dashboard: level over time, streak, jam belajar mingguan.
- SRS vocab, seeded dengan vocab workplace/profesional.
- Mistake log untuk melihat error grammar/vocab yang berulang.

### AI-powered

- Writing feedback kriteria IELTS (task response, coherence, lexical range, grammar).
- Speaking: record, transcribe, feedback fluency & pronunciation.
- Mock job interview (engineer) & roleplay patient conversation (nurse).
- Adaptive exercises di level saat ini.

## Build order

1. Profiles, placement quiz, dashboard.
2. Daily 30-minute plan (SRS + satu skill).
3. Writing feedback (AI).
4. Speaking practice.

Versi pertama yang sederhana (CLI atau web kecil dengan writing feedback + riwayat skor)
sudah memberi nilai di minggu pertama.

## Status

- [x] Scaffold: migrations, sqlc queries, rating, SRS, contoh konten A1, docs.
- [ ] `cmd/seed` + validator + `cmd/tts`.
- [ ] HTTP API + handler.
- [ ] Frontend PWA (`web/`).
- [ ] Nix flake.
- [ ] River + integrasi AI.
- [ ] Konten A1 lengkap (unit 1, 3-16 + 2 checkpoint set + placement).

## Open questions

- Nurse target IELTS atau OET, di negara mana?
- Tanggal exam tiap profil.
- Personal use atau produk yang dijual?
- Parameter Elo & threshold level (tuning setelah ada data nyata).
