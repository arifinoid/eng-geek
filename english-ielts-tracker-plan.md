# English & IELTS Progress Tracker: Plan

A personalized app to learn English from A1, track progress per CEFR level, and prepare for IELTS (B2/C1) to land international jobs. Built first for two users: a software engineer and a nurse.

## 1. Goals

- Start from A1 and reach B2 (IELTS ~6.0-6.5), then C1 (IELTS ~7-8).
- Know the current level at any time (A1, A2, B1, ...) per skill.
- Track progress separately for each level.
- Support two profiles with different professional tracks.

**IELTS vs CEFR:** IELTS reports bands (0-9), not CEFR levels. Rough mapping: B2 is about band 5.5-6.5, C1 is about band 7-8. Most international tech jobs want ~6.5-7 overall.

## 2. Study Time and Realistic Timeline

- Weekdays: 30 minutes per day.
- Weekends: 1-2 hours per day.
- About 5-6 hours per week in total.
- A1 to B2 needs roughly 400-600 study hours, so expect **~18-24 months** at this pace.
- Milestones: A2 in ~4 months, B1 in ~10 months.
- Consistency matters more than session length, so the tracker should protect streaks.

## 3. Users and Tracks

One household, two profiles, each with its own level, plan and history.

- **Shared core:** grammar, general vocabulary, IELTS skills.
- **Software engineer pack:** stand-ups, code reviews, job interviews, tech vocabulary.
- **Nurse pack:** patient handover (SBAR), medication and symptom vocabulary, patient conversations.
- **Open question:** if the nurse targets working abroad, check whether the destination accepts **OET** instead of IELTS. It is healthcare-specific and may fit better. The tracker should support both (`target_exam`).

## 4. Features

### MVP
- Profiles (household with two users).
- Placement test mapping to CEFR plus an estimated IELTS band.
- Four skills tracked separately: Listening, Reading, Writing, Speaking.
- Daily plan generated from the weakest skill and the deadline.
- Progress dashboard: level over time, streaks, weekly study hours.
- Vocabulary spaced repetition (SRS), seeded with workplace and professional vocabulary.
- Mistake log to spot recurring grammar and vocabulary errors.

### AI-powered
- Writing feedback scored on IELTS criteria (task response, coherence, lexical range, grammar).
- Speaking practice: record, transcribe, feedback on fluency and pronunciation.
- Mock job interviews (engineer) and patient-conversation roleplay (nurse).
- Adaptive exercises generated at the current level.

## 5. Tech Stack

| Layer | Choice |
| --- | --- |
| Backend language | Go |
| API | stdlib `net/http` (Go 1.22+ routing) or `chi` |
| Database | Postgres with `pgx` and `sqlc` (typed queries); SQLite is fine for a self-hosted MVP |
| Migrations | `goose` |
| Background jobs | `River` (Postgres-backed queue) for AI feedback and transcription |
| AI | Anthropic Go SDK for writing and speaking feedback |
| Speech-to-text | Whisper or a similar speech API |
| API contract | OpenAPI spec with `oapi-codegen`, generating a typed TypeScript client |
| Frontend | TypeScript PWA (React/Vite), usable on phone and desktop; Effect.ts is an option |
| Dev environment | Nix flake |
| Deploy | Single Docker image |

## 6. Data Model (Postgres)

- `profiles`: id, household_id, name, track (`engineer` | `nurse`), target_exam (`ielts` | `oet`), exam_date
- `levels`: code (A1, A2, B1, ...), ielts_band_min, ielts_band_max (estimate)
- `units`: level, skill, track (nullable means shared), title, position
- `activities`: unit_id, type (`quiz`, `vocab`, `writing`, `speaking`, `listening`), content (JSONB)
- `attempts`: profile_id, activity_id, score, duration_sec, created_at
- `skill_ratings`: profile_id, skill, rating, cefr (current)
- `rating_events`: profile_id, skill, delta, attempt_id (history for charts)
- `level_progress`: profile_id, level, status, started_at, completed_at
- `assessments`: profile_id, level, kind (`placement` | `checkpoint`), scores
- `srs_cards`: profile_id, vocab_id, due_at, interval, ease
- `mistakes`: profile_id, category, example, count

**Per-level separation:** `level_progress` plus units grouped by level gives each level its own completion percentage, time spent and score trend.

## 7. Scoring Design

- A per-user, per-skill Elo-like rating. Each attempt updates the rating.
- Rating thresholds map to CEFR levels and an estimated IELTS band.
- Placement test at the start (even from A1, to set a baseline), then re-test about every 6 weeks.

### Gate to advance from A1 to A2 (all required)
- At least 80% of A1 units completed.
- Checkpoint test score of at least 70%.
- All four skill ratings above the A1 threshold.

## 8. Study Plan: A1 to A2 (~16 weeks)

### Weekdays (30 min)
- **Mon:** vocab SRS (10 min) + grammar (20 min)
- **Tue:** listening, short slow dialogues
- **Wed:** vocab SRS + reading
- **Thu:** speaking, shadowing and repeating
- **Fri:** mistake review + vocab

### Weekends
- **Sat (1h):** new lesson + write 5-8 simple sentences (AI feedback)
- **Sun (1-2h):** AI conversation practice, weekly quiz, catch up on missed days

### Content by phase
- **Weeks 1-4:** alphabet and pronunciation, greetings, numbers, *to be*, present simple
- **Weeks 5-8:** daily routine, family, food, *can*, questions
- **Weeks 9-12:** past simple, directions, shopping, simple emails
- **Weeks 13-16:** review, role packs (engineer: introducing your job; nurse: basic patient questions), checkpoint test

## 9. Build Order

1. Profiles, placement quiz, dashboard.
2. Daily 30-minute plan (vocab SRS plus one skill).
3. Writing feedback (AI).
4. Speaking practice.

A simple first version (CLI or small web app with writing feedback and score history) gives value in week one.

## 10. Decisions So Far

- Start from A1 for both users.
- Backend in Go; frontend as a TypeScript PWA.
- Two profiles in one household with shared core plus professional content packs.
- Progress tracked separately per level with explicit gates.
- Postgres, `sqlc`, `River` for jobs, Anthropic API for feedback.

## 11. Open Questions

- Does the nurse target IELTS or OET, and in which country?
- Exam target dates for each profile.
- Personal use only, or a product to sell later?
- Exact Elo parameters and level thresholds.

## 12. Next Steps

- Write the SQL migration and `sqlc` queries.
- Detail the Elo rating math and thresholds.
- Draft A2 to B1 plan once A1 to A2 is validated.
