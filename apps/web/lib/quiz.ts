/** A multiple-choice question as stored in `activities.content`. */
export type MultipleChoiceQuestion = {
  prompt: string;
  options: string[];
  answer: number;
  explain?: string;
};

/** The shared shape behind `vocab`, `quiz`, and `listening` activities. */
export type MultipleChoiceContent = {
  passage?: string | null;
  script?: string | null;
  audio?: string | null;
  questions: MultipleChoiceQuestion[];
};

function isQuestion(value: unknown): value is MultipleChoiceQuestion {
  if (typeof value !== "object" || value === null) return false;
  const { prompt, options, answer } = value as Record<string, unknown>;
  if (typeof prompt !== "string") return false;
  if (!Array.isArray(options) || !options.every((option) => typeof option === "string")) {
    return false;
  }
  return typeof answer === "number" && answer >= 0 && answer < options.length;
}

/** Type guard for the content of a playable activity. */
export function isMultipleChoiceContent(
  value: unknown,
): value is MultipleChoiceContent {
  if (typeof value !== "object" || value === null) return false;
  const { questions } = value as Record<string, unknown>;
  return Array.isArray(questions) && questions.every(isQuestion);
}

/** Score is the share of correctly answered questions. */
export function scoreQuestions(
  content: MultipleChoiceContent,
  answers: Readonly<Record<number, number>>,
): number {
  if (content.questions.length === 0) return 0;
  const correct = content.questions.reduce(
    (total, question, index) =>
      answers[index] === question.answer ? total + 1 : total,
    0,
  );
  return correct / content.questions.length;
}

/** Renders a score as a whole percentage. */
export function scorePercent(score: number): number {
  return Math.round(score * 100);
}
