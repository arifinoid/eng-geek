# A1 Seed Content Plan

Companion to `english-ielts-tracker-plan.md` and `english-tracker-db-and-rating.md`. This is the content to load into `units`, `activities` and `vocab_items` so the first attempts have something to run against.

Contents:
1. Scope and numbers
2. Unit map (16 weeks)
3. Activity template per unit
4. Difficulty formula
5. How activities map to the daily plan
6. Content JSON shapes
7. Full example: Unit 2 (YAML)
8. Role pack vocabulary (engineer and nurse)
9. Checkpoint test (end of A1)
10. Seeding pipeline and file layout
11. Authoring prompt for Claude
12. Quality checklist and open decisions

---

## 1. Scope and numbers

- A1 covers the whole 16-week phase in the plan, so finishing A1 means reaching A2.
- **Shared units:** 14 (weeks 1-13 and 16), seen by both profiles.
- **Role pack units:** 2 per track (weeks 14-15), so each profile sees 16 units.
- **Activities per unit:** 9. Each profile sees 16 × 9 = **144 activities**. Authored total: 14 × 9 + 4 × 9 = **162**.
- **Vocabulary:** about 14 words per unit, about 200 shared words plus about 30 per track.
- Rough effort: 144 activities × ~15 min ≈ 36 hours, plus SRS and AI conversation practice, which fits ~88 study hours over 16 weeks.

Note: vocab SRS reviews do **not** change skill ratings. Only unit activities (section 3) update ratings. The "vocab quiz" activity in each unit counts as **reading**.

---

## 2. Unit map (16 weeks)

| Wk | Position | Track | Unit | Grammar focus | Vocabulary theme |
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

**Unique key check:** `units` has `UNIQUE (level_code, track, position)`. In Postgres, NULL track values are treated as distinct, so shared units never clash, and each track's units at positions 14 and 15 are unique per track.

**Content that belongs at A2, not here:** future tenses, comparatives, present perfect. Keep A1 sentences short (about 8-10 words) and the vocabulary common.

---

## 3. Activity template per unit

Every unit has the same 9 activities, so the daily plan generator can rely on them:

| # | Type | Skill | Offset vs unit base | Purpose |
| --- | --- | --- | --- | --- |
| 1 | `vocab` | reading | 0 | unit word set, recognition quiz |
| 2 | `quiz` | reading | 0 | grammar practice 1 |
| 3 | `quiz` | reading | +10 | grammar practice 2 |
| 4 | `listening` | listening | -10 | short audio, easier |
| 5 | `listening` | listening | +10 | short dialogue |
| 6 | `quiz` | reading | 0 | short reading text with questions |
| 7 | `writing` | writing | +20 | 3-6 sentences, AI feedback |
| 8 | `speaking` | speaking | -10 | repeat and shadow |
| 9 | `speaking` | speaking | +20 | answer 2-3 questions aloud |

Activity 6 uses type `quiz` with a `passage` field (the `activity_type` enum has no `reading` value). If you prefer, add `reading` to the enum in a later migration.

---

## 4. Difficulty formula

```
unit_base = 840 + (week - 1) * 10        # week 1 = 840 ... week 16 = 990
difficulty = unit_base + offset          # offset from section 3
```

So A1 activities span about 830-1010. The checkpoint test sits at **1020**, just above the A2 line, so a learner can only pass the gate by showing slightly more than A1 ability.

Why this works with the Elo math: a learner who keeps scoring above expectation keeps gaining rating even when the rating is above the activity's difficulty, so reaching 1000 is possible with content capped near 1000. If ratings stall just below 1000, raise the weeks 13-16 offsets by +20.

---

## 5. How activities map to the daily plan

The daily plan picks from the current level's unit for the week, using `NextActivities` (weakest skill first, difficulty nearest to rating).

| Day | Time | Pick |
| --- | --- | --- |
| Mon | 30 min | SRS cards (10) + activities 2-3 (grammar) |
| Tue | 30 min | activities 4-5 (listening) |
| Wed | 30 min | SRS cards + activity 6 (reading) |
| Thu | 30 min | activities 8-9 (speaking) |
| Fri | 30 min | mistake review + activity 1 (vocab quiz) |
| Sat | 1 h | new unit lesson + activity 7 (writing) |
| Sun | 1-2 h | AI conversation, weekly quiz, catch up on missed activities |

One unit per week, so a unit's 9 activities fit the week with some room.

---

## 6. Content JSON shapes

Stored in `activities.content` (JSONB). Keep one shape per type so the frontend has one renderer each.

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

**`quiz`** (grammar or reading)
```json
{
  "passage": null,
  "questions": [
    {"prompt": "I ___ a student.", "options": ["am", "is", "are"], "answer": 0,
     "explain": "Use 'am' with I."}
  ]
}
```
For reading, set `passage` to the text and ask questions about it.

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
`audio` is generated from `script` with text-to-speech at build time and stored as a file; the field holds the voice and speed setting until then.

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
{
  "mode": "shadow",
  "lines": ["Hello. My name is Tom.", "I am from Canada."]
}
```
or
```json
{
  "mode": "answer",
  "questions": ["What is your name?", "Where are you from?"],
  "expected_keywords": ["name", "from"]
}
```

**Scoring:** quizzes score as correct ÷ total. Writing and speaking get an AI score from 0 to 1 using the rubric; keep the full feedback in `attempts.feedback`.

---

## 7. Full example: Unit 2 (YAML)

`content/a1/02-greetings-introductions.yaml` (week 2, unit base = 850)

```yaml
unit:
  level: A1
  week: 2
  position: 2
  track: null
  title: Greetings and Introductions

vocab:
  - {term: hello, meaning: a greeting, example: "Hello, my name is Ana."}
  - {term: goodbye, meaning: words you say when you leave, example: "Goodbye, see you tomorrow."}
  - {term: name, meaning: what people call you, example: "My name is Tom."}
  - {term: please, meaning: a polite word, example: "Sit down, please."}
  - {term: thank you, meaning: words to show you are grateful, example: "Thank you for your help."}
  - {term: sorry, meaning: a word for a mistake, example: "Sorry, I am late."}
  - {term: yes, meaning: the opposite of no, example: "Yes, I am a student."}
  - {term: no, meaning: the opposite of yes, example: "No, I am not a teacher."}
  - {term: from, meaning: the place you come from, example: "I am from Japan."}
  - {term: country, meaning: a nation like Japan or Peru, example: "Peru is a country."}
  - {term: nice to meet you, meaning: words when you meet someone, example: "Nice to meet you, Sam."}
  - {term: friend, meaning: a person you like, example: "She is my friend."}
  - {term: teacher, meaning: a person who teaches, example: "He is a teacher."}
  - {term: student, meaning: a person who studies, example: "I am a student."}

activities:
  - type: vocab
    skill: reading
    difficulty: 850
    content:
      questions:
        - {prompt: "Which word means 'a polite word'?", options: [please, name, from], answer: 0}
        - {prompt: "Which word means 'the place you come from'?", options: [sorry, from, no], answer: 1}
        - {prompt: "Which words do you say when you meet someone?", options: ["nice to meet you", "goodbye", "sorry"], answer: 0}

  - type: quiz
    skill: reading
    difficulty: 850
    content:
      questions:
        - {prompt: "I ___ a student.", options: [am, is, are], answer: 0, explain: "Use 'am' with I."}
        - {prompt: "She ___ a nurse.", options: [am, is, are], answer: 1, explain: "Use 'is' with he, she, it."}
        - {prompt: "You ___ from Japan.", options: [am, is, are], answer: 2, explain: "Use 'are' with you, we, they."}

  - type: quiz
    skill: reading
    difficulty: 860
    content:
      questions:
        - {prompt: "Is she a teacher? Yes, she ___.", options: [am, is, are], answer: 1}
        - {prompt: "They ___ not from Peru.", options: [am, is, are], answer: 2}
        - {prompt: "What ___ your name?", options: [am, is, are], answer: 1}

  - type: listening
    skill: listening
    difficulty: 840
    content:
      script: "Hello. My name is Tom. I am from Canada. Nice to meet you."
      audio: "tts:en-US:slow"
      questions:
        - {prompt: "What is his name?", options: [Tom, Sam, Tim], answer: 0}
        - {prompt: "Where is he from?", options: [Canada, Japan, Peru], answer: 0}

  - type: listening
    skill: listening
    difficulty: 860
    content:
      script: "Anna: Hello, I am Anna. What is your name? Ken: Hi, Anna. I am Ken. Nice to meet you. Anna: Nice to meet you, too. Are you from Japan? Ken: Yes, I am."
      audio: "tts:en-GB:slow"
      questions:
        - {prompt: "Is Ken from Japan?", options: ["Yes", "No"], answer: 0}
        - {prompt: "Who says 'What is your name?'", options: [Anna, Ken], answer: 0}

  - type: quiz
    skill: reading
    difficulty: 850
    content:
      passage: "My name is Maria. I am a teacher. I am from Brazil. My friend is Paulo. He is a student. He is from Peru."
      questions:
        - {prompt: "What is Maria's job?", options: [teacher, student, nurse], answer: 0}
        - {prompt: "Where is Paulo from?", options: [Brazil, Peru, Japan], answer: 1}
        - {prompt: "Paulo is a ___.", options: [teacher, student, nurse], answer: 1}

  - type: writing
    skill: writing
    difficulty: 870
    content:
      prompt: "Write 4 sentences to introduce yourself. Say your name, your job, and your country."
      min_words: 20
      rubric: [task, grammar, vocabulary, spelling]
      model_answer: "Hello. My name is Ana. I am a nurse. I am from Brazil."

  - type: speaking
    skill: speaking
    difficulty: 840
    content:
      mode: shadow
      lines:
        - "Hello. My name is Tom."
        - "I am from Canada."
        - "Nice to meet you."

  - type: speaking
    skill: speaking
    difficulty: 870
    content:
      mode: answer
      questions: ["What is your name?", "Where are you from?", "What is your job?"]
      expected_keywords: [name, from, job]
```

Check the example against the formula: base 850, offsets 0, 0, +10, -10, +10, 0, +20, -10, +20 give 850, 850, 860, 840, 860, 850, 870, 840, 870. Matches.

---

## 8. Role pack vocabulary (weeks 14-15)

Stay at A1 level: short words, everyday meaning.

**Engineer pack**
- Week 14, *Introducing Your Job:* engineer, team, project, computer, code, work, office, company, manager, language (as in programming language), tool, user, remote, laptop
- Week 15, *Meetings and Simple Messages:* meeting, task, deadline, bug, fix, update, message, today, tomorrow, ready, late, help, question, problem

Example sentences:
- "I am a software engineer."
- "I work with a team."
- "The bug is fixed."
- "Can you help me, please?"
- "The meeting is at three o'clock."

**Nurse pack**
- Week 14, *Basic Patient Questions:* patient, nurse, doctor, pain, medicine, hospital, name, age, allergy, sleep, eat, drink, hurt, help
- Week 15, *Simple Instructions and Handover:* bed, water, temperature, breathe, sit down, stand up, wait, blood, ward, bag, call, check, better, same

Example sentences:
- "What is your name?"
- "Where does it hurt?"
- "Do you have pain?"
- "Please sit down."
- "Take this medicine."
- "I will call the doctor."

The nurse pack's first half is questions the nurse asks; the second half is short instructions and a simple status handover ("The patient is sleeping. The temperature is normal."). Full SBAR comes at B1.

---

## 9. Checkpoint test (end of A1)

Stored in `assessments` with `kind = 'checkpoint'` and `level_code = 'A1'`. Not part of the 144 activities.

- **20 questions:** 5 per skill (listening, reading, writing, speaking).
- **Difficulty:** all items at **1020**.
- **Format:**
  - Listening: 5 multiple-choice from short audio.
  - Reading: 5 multiple-choice from two short passages.
  - Writing: one 30-40 word task, AI-scored 0-1, counted as 5 points.
  - Speaking: one 3-question answer round, AI-scored 0-1, counted as 5 points.
- `scores` JSONB example: `{"listening": 0.8, "reading": 0.8, "writing": 0.6, "speaking": 0.7}`
- `overall` = average of the four. Gate needs `overall ≥ 0.70`.
- Allow a re-take after 7 days. Use a different item set on a re-take (keep 2 sets).

The placement test at the start uses the same format but at difficulties 700-1100, so it works for learners who are above A1.

---

## 10. Seeding pipeline and file layout

```
content/
└── a1/
    ├── 01-alphabet-sounds.yaml
    ├── 02-greetings-introductions.yaml
    ├── ...
    ├── 13-review-1.yaml
    ├── 14-engineer-introducing-your-job.yaml
    ├── 14-nurse-basic-patient-questions.yaml
    ├── 15-engineer-meetings-messages.yaml
    ├── 15-nurse-instructions-handover.yaml
    ├── 16-review-2.yaml
    └── checkpoint-a.yaml, checkpoint-b.yaml
cmd/seed/main.go
```

**Seeder behavior** (`cmd/seed`):
1. Read every YAML file under `content/a1/`.
2. Upsert the unit by `(level_code, track, position)`.
3. For activities, delete and re-insert per unit **only when the unit has no attempts**; otherwise insert new versions and mark old ones retired (add a `retired_at` column in a later migration), so attempt history stays valid.
4. Upsert vocab into `vocab_items` by `(term, level_code)`.
5. Validate before writing: all four skills present, every `answer` index is within `options`, difficulty equals the formula, `track` matches the filename.

The seeder should be idempotent so you can re-run it after editing content.

Audio: generate TTS files from `script`/`lines` in a separate step (`cmd/tts`), store paths in the activity, and cache by a hash of the script text.

---

## 11. Authoring prompt for Claude

Use this to draft each unit, then review by hand.

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

Review each generated unit for wrong answers, unnatural phrasing and words above A1 before seeding.

---

## 12. Quality checklist and open decisions

**Checklist per unit**
- [ ] 14 vocab items, all A1-level (check against an A1 word list such as the Cambridge English Vocabulary Profile).
- [ ] 9 activities in the template order; difficulties match the formula.
- [ ] Every question has exactly one correct answer; explanations are short.
- [ ] Listening and speaking lines are natural spoken English.
- [ ] Nothing assumes the learner knows a later unit's grammar.
- [ ] Role pack content stays at A1 (no medical or technical terms beyond the lists in section 8).

**Open decisions**
- Add a `reading` value to `activity_type`, or keep reading as `quiz` with a passage.
- TTS provider and voices (US vs UK accents; IELTS uses several accents, so mix both from A2 onward).
- How to handle learners who finish units faster than weekly: unlock the next unit early once 80% of the current unit is done.
- Whether both profiles get the shared review week at the same time or each on their own schedule (suggest own schedule, per profile).

**Next steps**
1. Write units 1-3 by hand to validate the schema and the seeder.
2. Generate units 4-16 with the prompt in section 11 and review.
3. Build `cmd/seed` and the validator.
4. Write the two checkpoint sets and the placement test.
