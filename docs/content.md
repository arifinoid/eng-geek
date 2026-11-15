# Content (A1)

Konten disimpan sebagai YAML di `apps/api/content/a1/`, lalu di-seed ke Postgres oleh
`apps/api/cmd/seed`. Schema activity mengikuti enum `activity_type` di
`apps/api/db/migrations/00001_init.sql`.
Contoh lengkap: `apps/api/content/a1/02-greetings-introductions.yaml`.

## Scope and numbers

- A1 mencakup 16 minggu; menyelesaikan A1 berarti mencapai A2.
- 14 unit **shared** (minggu 1-13 dan 16) + 2 unit **role pack** per track (minggu 14-15).
- Tiap profil melihat 16 unit; total activity yang ditulis: `14*9 + 4*9 = 162`.
- Tiap unit 9 activity; tiap profil: `16*9 = 144` activity.
- Vocab ~14 kata per unit; ~200 shared + ~30 per track.
- Estimasi usaha: 144 activity * ~15 menit = ~36 jam, plus SRS dan AI conversation,
  cocok dengan ~88 jam belajar dalam 16 minggu.

Yang **bukan** A1: future tenses, comparatives, present perfect. Kalimat A1 pendek
(~8-10 kata), vocab umum.

## Unit map (16 minggu)

| Wk | Pos | Track | Unit | Grammar focus | Vocabulary theme |
| --- | --- | --- | --- | --- | --- |
| 1 | 1 | shared | Alphabet and Sounds | spelling, basic sounds | letters, classroom words |
| 2 | 2 | shared | Greetings and Introductions | *to be* (am/is/are), my/your | greetings, countries |
| 3 | 3 | shared | Numbers, Time and Dates | numbers 0-100, "What time is it?" | days, months, time words |
| 4 | 4 | shared | Present Simple | I/you/we/they, he/she/it + s | common verbs |
| 5 | 5 | shared | Daily Routine | time expressions, always/usually | morning to night actions |
| 6 | 6 | shared | Family and People | have/has, my/his/her | family, jobs, adjectives |
| 7 | 7 | shared | Food and Drink | a/an/some, countable vs uncountable | food, drink, meals |
| 8 | 8 | shared | Can/Can't and Questions | can/can't, wh-questions | abilities, question words |
| 9 | 9 | shared | Past Simple (regular) | -ed endings, yesterday | weekend, hobbies |
| 10 | 10 | shared | Past Simple (irregular) | went/had/saw/did, negatives | travel, past events |
| 11 | 11 | shared | Places and Directions | there is/are, prepositions, go straight/turn | town, buildings |
| 12 | 12 | shared | Shopping and Simple Emails | How much?, would like, email openers/closers | shops, money, clothes |
| 13 | 13 | shared | Review 1 and Mistake Clinic | weeks 1-12 mix | mixed review |
| 14 | 14 | engineer | Introducing Your Job | I work/am + job, tools | team, project, computer |
| 14 | 14 | nurse | Basic Patient Questions | polite questions, please | patient, pain, medicine |
| 15 | 15 | engineer | Meetings and Simple Messages | present simple at work, short chat messages | meeting, task, deadline |
| 15 | 15 | nurse | Simple Instructions and Handover | commands, short status sentences | bed, water, temperature |
| 16 | 16 | shared | Review 2 and Checkpoint | all A1 grammar | mixed review |

Unique key: `units` punya `UNIQUE (level_code, track, position)`. Di Postgres, NULL
`track` dianggap berbeda, jadi unit shared tidak bentrok, dan unit track di posisi
14-15 unik per track.

## Role pack vocabulary (minggu 14-15)

Tetap level A1: kata pendek, makna sehari-hari.

**Engineer**
- W14 *Introducing Your Job*: engineer, team, project, computer, code, work, office,
  company, manager, language (programming language), tool, user, remote, laptop.
- W15 *Meetings and Simple Messages*: meeting, task, deadline, bug, fix, update,
  message, today, tomorrow, ready, late, help, question, problem.
- Contoh: "I am a software engineer." · "I work with a team." · "The bug is fixed." ·
  "Can you help me, please?" · "The meeting is at three o'clock."

**Nurse**
- W14 *Basic Patient Questions*: patient, nurse, doctor, pain, medicine, hospital,
  name, age, allergy, sleep, eat, drink, hurt, help.
- W15 *Simple Instructions and Handover*: bed, water, temperature, breathe, sit down,
  stand up, wait, blood, ward, bag, call, check, better, same.
- Contoh: "What is your name?" · "Where does it hurt?" · "Do you have pain?" ·
  "Please sit down." · "Take this medicine." · "I will call the doctor."

Paruh pertama pack nurse = pertanyaan yang ditanyakan perawat; paruh kedua = instruksi
pendek + handover status sederhana ("The patient is sleeping. The temperature is
normal."). SBAR penuh baru di B1.

## Test

### Checkpoint (akhir A1)

- Disimpan di `assessments`, `kind = 'checkpoint'`, `level_code = 'A1'`.
- 20 pertanyaan: 5 per skill. Semua item difficulty **1020**.
- Listening: 5 MC dari audio pendek. Reading: 5 MC dari 2 passage pendek.
- Writing: 1 task 30-40 kata, AI score 0-1, dihitung 5 poin.
- Speaking: 1 ronde 3 pertanyaan, AI score 0-1, dihitung 5 poin.
- `scores` JSONB contoh: `{"listening": 0.8, "reading": 0.8, "writing": 0.6, "speaking": 0.7}`.
- `overall` = rata-rata 4 skill. Gate butuh `overall >= 0.70`.
- Re-take boleh setelah 7 hari, pakai set item lain (simpan 2 set).

### Placement (awal)

Format sama, difficulty 700-1100, untuk learner yang di atas A1.

## Daily plan mapping

| Day | Time | Pick |
| --- | --- | --- |
| Mon | 30 min | SRS cards (10) + activities 2-3 (grammar) |
| Tue | 30 min | activities 4-5 (listening) |
| Wed | 30 min | SRS cards + activity 6 (reading) |
| Thu | 30 min | activities 8-9 (speaking) |
| Fri | 30 min | mistake review + activity 1 (vocab quiz) |
| Sat | 1 h | new unit lesson + activity 7 (writing) |
| Sun | 1-2 h | AI conversation, weekly quiz, catch up missed activities |

Satu unit per minggu, jadi 9 activity muat dalam satu minggu.

## Content JSON shapes

Disimpan di `activities.content` (JSONB). Satu bentuk per `type` -> satu renderer di
frontend.

**`vocab`**
```json
{
  "items": [
    {"term": "hello", "meaning": "a greeting", "example": "Hello, my name is Ana."}
  ],
  "questions": [
    {"prompt": "Choose the word: 'a greeting'", "options": ["hello", "name", "from"], "answer": 0}
  ]
}
```

**`quiz`** (grammar atau reading)
```json
{
  "passage": null,
  "questions": [
    {"prompt": "I ___ a student.", "options": ["am", "is", "are"], "answer": 0,
     "explain": "Use 'am' with I."}
  ]
}
```
Untuk reading, isi `passage` dengan teks, lalu tanyakan isinya.

**`listening`**
```json
{
  "script": "Hello. My name is Tom. I am from Canada.",
  "audio": "tts:en-US:slow",
  "questions": [
    {"prompt": "Where is Tom from?", "options": ["Canada", "Japan", "Peru"], "answer": 0}
  ]
}
```
`audio` dibangkitkan dari `script` via TTS saat build dan disimpan sebagai file; field
ini menyimpan voice + speed sampai itu selesai.

**`writing`**
```json
{
  "prompt": "Write 4 sentences to introduce yourself.",
  "min_words": 20,
  "rubric": ["task", "grammar", "vocabulary", "spelling"],
  "model_answer": "Hello. My name is ..."
}
```

**`speaking`**
```json
{"mode": "shadow", "lines": ["Hello. My name is Tom.", "I am from Canada."]}
```
atau
```json
{"mode": "answer", "questions": ["What is your name?", "Where are you from?"],
 "expected_keywords": ["name", "from"]}
```

## Seeder (`cmd/seed`)

```
apps/api/content/a1/*.yaml
apps/api/cmd/seed/main.go
```

Perilaku:

1. Baca semua YAML di `content/a1/`.
2. Upsert unit by `(level_code, track, position)`.
3. Activity: delete & re-insert per unit **hanya kalau unit belum punya attempt**;
   kalau sudah, insert versi baru dan tandai lama retired (tambah kolom `retired_at`
   di migrasi berikutnya) supaya riwayat attempt tetap valid.
4. Upsert vocab ke `vocab_items` by `(term, level_code)`.
5. Validasi sebelum tulis: 4 skill ada, semua `answer` valid, difficulty sesuai
   formula, `track` cocok dengan filename.
6. Idempotent, bisa dijalankan ulang setelah konten diedit.

TTS: `apps/api/cmd/tts` membangkitkan audio dari `script`/`lines`, cache by hash teks,
simpan path di activity.

## Authoring prompt

Dipakai untuk menulis draf tiap unit, lalu review manual.

```
Write the content for unit {position} ("{title}") of an A1 English course,
for adult learners, as one YAML file matching this schema: {paste the Unit 2 YAML}.

Constraints:
- A1 level only: sentences of at most 10 words, common words, present simple
  and the grammar focus: {grammar focus}.
- Exactly 14 vocab items about: {vocabulary theme}.
- Exactly 9 activities in this order and with these difficulties: {list from section 3
  with the computed numbers for week {week}}.
- Every multiple-choice question has exactly one correct option; set "answer" to its index.
- Listening scripts under 40 words. Reading passage under 60 words.
- Neutral, adult topics. No slang or idioms.
- Track: {shared | engineer | nurse}. If a track is given, use that profession in examples.
Return only YAML.
```

Review tiap unit hasil generate: jawaban salah, phrasing tidak natural, kata di atas A1.

## Quality checklist per unit

- [ ] 14 vocab items, semua level A1 (cek ke Cambridge English Vocabulary Profile).
- [ ] 9 activity sesuai urutan template; difficulty sesuai formula.
- [ ] Setiap pertanyaan tepat satu jawaban benar; penjelasan singkat.
- [ ] Listening dan speaking lines natural (bahasa lisan).
- [ ] Tidak mengasumsikan grammar unit berikutnya.
- [ ] Role pack tetap A1 (tidak ada istilah medis/teknis di luar daftar).

## Open decisions (content)

- Tambah `reading` ke enum `activity_type`, atau tetap `quiz` + `passage`.
- Provider TTS & voice (US vs UK; IELTS memakai banyak aksen, jadi campur mulai A2).
- Unlock unit berikutnya lebih awal kalau 80% unit saat ini selesai.
- Review week shared: bersamaan atau jadwal masing-masing profil (saran: masing-masing).
