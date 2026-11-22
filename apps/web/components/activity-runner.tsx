"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import type { Activity, Attempt } from "@eng-geek/api-client";
import { useActivity, useRecordAttempt } from "@/lib/api";
import { useSelectedProfileId } from "@/lib/use-selected-profile";
import {
  isMultipleChoiceContent,
  scorePercent,
  scoreQuestions,
  type MultipleChoiceContent,
} from "@/lib/quiz";
import { ProfilePrompt } from "./profile-prompt";

function parseId(value: string | null): number | undefined {
  if (value === null) return undefined;
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : undefined;
}

function stringField(value: unknown, key: string): string | null {
  if (typeof value !== "object" || value === null) return null;
  const raw = (value as Record<string, unknown>)[key];
  return typeof raw === "string" ? raw : null;
}

function numberField(value: unknown, key: string): number | null {
  if (typeof value !== "object" || value === null) return null;
  const raw = (value as Record<string, unknown>)[key];
  return typeof raw === "number" ? raw : null;
}

function stringList(value: unknown, key: string): string[] {
  if (typeof value !== "object" || value === null) return [];
  const raw = (value as Record<string, unknown>)[key];
  return Array.isArray(raw) ? raw.filter((item): item is string => typeof item === "string") : [];
}

export function ActivityRunner() {
  const t = useTranslations("activity");
  const params = useSearchParams();
  const activityId = parseId(params.get("id"));
  const profileId = useSelectedProfileId();
  const activity = useActivity(activityId);

  if (profileId === undefined) return <ProfilePrompt />;
  if (activityId === undefined) {
    return <p className="text-zinc-500">{t("invalidActivity")}</p>;
  }
  if (activity.isPending) return <p className="text-zinc-500">{t("loading")}</p>;
  if (activity.isError || !activity.data) {
    return <p className="text-red-600">{t("loadFailed")}</p>;
  }

  const item = activity.data;

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-col gap-1">
        <Link
          href={`/practice/units?id=${item.unit_id}`}
          className="text-sm text-zinc-500 transition-colors hover:text-foreground"
        >
          {t("backToUnit")}
        </Link>
        <p className="text-sm text-zinc-500 capitalize">
          {item.skill} · {item.type}
        </p>
      </header>

      {item.type === "writing" ? (
        <WritingActivity activity={item} />
      ) : item.type === "speaking" ? (
        <SpeakingActivity activity={item} />
      ) : (
        <MultipleChoiceActivity activity={item} profileId={profileId} />
      )}
    </div>
  );
}

function MultipleChoiceActivity({
  activity,
  profileId,
}: {
  activity: Activity;
  profileId: number;
}) {
  const t = useTranslations("activity");
  const queryClient = useQueryClient();
  const recordAttempt = useRecordAttempt();
  const startedAt = useRef(0);
  const [answers, setAnswers] = useState<Record<number, number>>({});
  const [result, setResult] = useState<Attempt | null>(null);

  useEffect(() => {
    startedAt.current = Date.now();
  }, []);

  const content: MultipleChoiceContent | null = isMultipleChoiceContent(activity.content)
    ? activity.content
    : null;
  if (content === null) {
    return <p className="text-zinc-500">{t("unsupported")}</p>;
  }

  const answered = content.questions.filter((_, index) => answers[index] !== undefined).length;

  function check(questions: MultipleChoiceContent) {
    const score = scoreQuestions(questions, answers);
    const durationSec = Math.max(0, Math.round((Date.now() - startedAt.current) / 1000));
    recordAttempt.mutate(
      { profileId, activityId: activity.id, score, durationSec },
      {
        onSuccess: (attempt) => {
          setResult(attempt);
          void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
        },
      },
    );
  }

  if (result !== null) {
    return (
      <div className="flex flex-col gap-4 rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
        <h3 className="text-lg font-medium">{t("result")}</h3>
        <p>
          {t("score")}: {scorePercent(result.score)}%
        </p>
        <p>
          {t("ratingDelta")}: {result.rating_delta >= 0 ? "+" : ""}
          {result.rating_delta.toFixed(1)}
        </p>
        <Link
          href={`/practice/units?id=${activity.unit_id}`}
          className="text-sm text-zinc-500 transition-colors hover:text-foreground"
        >
          {t("backToUnit")}
        </Link>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {content.passage ? (
        <p className="whitespace-pre-wrap rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
          {content.passage}
        </p>
      ) : null}

      {content.script ? (
        <div className="flex flex-col gap-1">
          <h3 className="text-sm font-medium">{t("script")}</h3>
          <p className="whitespace-pre-wrap">{content.script}</p>
        </div>
      ) : content.audio ? (
        <p className="text-sm text-zinc-500">{t("audioPending")}</p>
      ) : null}

      <ol className="flex flex-col gap-5">
        {content.questions.map((question, index) => (
          <li key={index} className="flex flex-col gap-2">
            <p className="font-medium">
              {index + 1}. {question.prompt}
            </p>
            <div className="flex flex-col gap-1">
              {question.options.map((option, optionIndex) => (
                <label key={optionIndex} className="flex items-center gap-2">
                  <input
                    type="radio"
                    name={`question-${index}`}
                    checked={answers[index] === optionIndex}
                    onChange={() =>
                      setAnswers((previous) => ({ ...previous, [index]: optionIndex }))
                    }
                  />
                  <span>{option}</span>
                </label>
              ))}
            </div>
            {question.explain && answers[index] !== undefined ? (
              <p className="text-sm text-zinc-500">{question.explain}</p>
            ) : null}
          </li>
        ))}
      </ol>

      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => check(content)}
          disabled={answered < content.questions.length || recordAttempt.isPending}
          className="rounded-full bg-foreground px-5 py-2 text-sm text-background transition-colors hover:bg-[#383838] disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-[#ccc]"
        >
          {recordAttempt.isPending ? t("loading") : t("check")}
        </button>
        {recordAttempt.isError ? (
          <span className="text-sm text-red-600">{t("loadFailed")}</span>
        ) : null}
      </div>
    </div>
  );
}

function WritingActivity({ activity }: { activity: Activity }) {
  const t = useTranslations("activity");
  const prompt = stringField(activity.content, "prompt");
  const minWords = numberField(activity.content, "min_words");
  const rubric = stringList(activity.content, "rubric");
  const modelAnswer = stringField(activity.content, "model_answer");

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-lg font-medium">{t("writingTitle")}</h3>
      {prompt ? <p className="font-medium">{prompt}</p> : null}
      {minWords !== null ? (
        <p className="text-sm text-zinc-500">
          {t("minWords")}: {minWords}
        </p>
      ) : null}
      {rubric.length > 0 ? (
        <ul className="flex list-disc flex-col gap-1 pl-5 text-sm text-zinc-600 dark:text-zinc-400">
          {rubric.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      ) : null}
      <textarea
        rows={8}
        className="w-full rounded-xl border border-zinc-200 p-3 dark:border-zinc-800"
      />
      {modelAnswer ? (
        <p className="text-sm text-zinc-500">
          {t("modelAnswer")}: {modelAnswer}
        </p>
      ) : null}
      <p className="text-sm text-zinc-500">{t("writingNote")}</p>
    </div>
  );
}

function SpeakingActivity({ activity }: { activity: Activity }) {
  const t = useTranslations("activity");
  const mode = stringField(activity.content, "mode");
  const lines = stringList(activity.content, "lines");
  const questions = stringList(activity.content, "questions");

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-lg font-medium">{t("speakingTitle")}</h3>
      {mode === "shadow" && lines.length > 0 ? (
        <div className="flex flex-col gap-2">
          <h4 className="text-sm font-medium">{t("lines")}</h4>
          <ul className="flex list-disc flex-col gap-1 pl-5">
            {lines.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        </div>
      ) : null}
      {mode === "answer" && questions.length > 0 ? (
        <div className="flex flex-col gap-2">
          <h4 className="text-sm font-medium">{t("questions")}</h4>
          <ul className="flex list-disc flex-col gap-1 pl-5">
            {questions.map((question) => (
              <li key={question}>{question}</li>
            ))}
          </ul>
        </div>
      ) : null}
      <p className="text-sm text-zinc-500">{t("speakingNote")}</p>
    </div>
  );
}
