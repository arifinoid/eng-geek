import { describe, expect, it } from "vitest";
import {
  isMultipleChoiceContent,
  scorePercent,
  scoreQuestions,
  type MultipleChoiceContent,
} from "./quiz";

describe("isMultipleChoiceContent", () => {
  it("accepts a quiz with valid questions", () => {
    const content = {
      questions: [{ prompt: "I ___ a student.", options: ["am", "is"], answer: 0 }],
    };
    expect(isMultipleChoiceContent(content)).toBe(true);
  });

  it("rejects null and non-objects", () => {
    expect(isMultipleChoiceContent(null)).toBe(false);
    expect(isMultipleChoiceContent("quiz")).toBe(false);
  });

  it("rejects a question without options", () => {
    expect(isMultipleChoiceContent({ questions: [{ prompt: "x", answer: 0 }] })).toBe(
      false,
    );
  });

  it("rejects an answer outside the options", () => {
    const content = { questions: [{ prompt: "x", options: ["a", "b"], answer: 2 }] };
    expect(isMultipleChoiceContent(content)).toBe(false);
  });

  it("rejects an activity without questions", () => {
    expect(isMultipleChoiceContent({ passage: "text" })).toBe(false);
  });
});

describe("scoreQuestions", () => {
  const content: MultipleChoiceContent = {
    questions: [
      { prompt: "one", options: ["a", "b"], answer: 0 },
      { prompt: "two", options: ["a", "b"], answer: 1 },
      { prompt: "three", options: ["a", "b"], answer: 1 },
    ],
  };

  it("scores every correct answer", () => {
    expect(scoreQuestions(content, { 0: 0, 1: 1, 2: 1 })).toBe(1);
  });

  it("scores a partial result", () => {
    expect(scoreQuestions(content, { 0: 0, 1: 0, 2: 1 })).toBeCloseTo(2 / 3);
  });

  it("scores zero when nothing is answered", () => {
    expect(scoreQuestions(content, {})).toBe(0);
  });

  it("scores zero for a question with no answers", () => {
    expect(scoreQuestions({ questions: [] }, {})).toBe(0);
  });
});

describe("scorePercent", () => {
  it("rounds a share to a whole percentage", () => {
    expect(scorePercent(0.5)).toBe(50);
    expect(scorePercent(2 / 3)).toBe(67);
  });
});
