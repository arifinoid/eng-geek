"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent } from "react";
import type { Assessment, Gate } from "@eng-geek/api-client";
import { useAssessments, useCreateAssessment, useGate } from "@/lib/api";
import { useSelectedProfileId } from "@/lib/use-selected-profile";
import { ProfilePrompt } from "./profile-prompt";

const LEVELS = ["A1", "A2", "B1", "B2", "C1", "C2"];
const SKILLS = ["listening", "reading", "writing", "speaking"] as const;

const EMPTY_SCORES: Record<string, string> = {
  listening: "",
  reading: "",
  writing: "",
  speaking: "",
};

export function LevelsView() {
  const t = useTranslations("levels");
  const profileId = useSelectedProfileId();
  const [level, setLevel] = useState("");
  const gate = useGate(profileId, level === "" ? undefined : level);

  if (profileId === undefined) return <ProfilePrompt />;

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-2">
        <h2 className="text-lg font-medium">{t("currentGate")}</h2>
        <label className="flex items-center gap-2 text-sm">
          <span className="text-zinc-500">{t("level")}</span>
          <select
            value={level}
            onChange={(event) => setLevel(event.target.value)}
            className="rounded-lg border border-zinc-200 px-2 py-1 dark:border-zinc-800"
          >
            <option value="">{t("current")}</option>
            {LEVELS.map((code) => (
              <option key={code} value={code}>
                {code}
              </option>
            ))}
          </select>
        </label>
        {gate.isPending ? (
          <p className="text-sm text-zinc-500">{t("loading")}</p>
        ) : gate.isError || !gate.data ? (
          <p className="text-sm text-red-600">{t("loadFailed")}</p>
        ) : (
          <GateCard gate={gate.data} />
        )}
      </section>

      <AssessmentForm
        key={gate.data?.level ?? "current"}
        profileId={profileId}
        level={gate.data?.level ?? ""}
      />

      <History profileId={profileId} />
    </div>
  );
}

function GateCard({ gate }: { gate: Gate }) {
  const t = useTranslations("levels");
  const percent = Math.round(gate.completion_ratio * 100);

  return (
    <div className="flex flex-col gap-4 rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
      <div className="flex items-center justify-between">
        <span className="text-sm text-zinc-500">{t("level")}</span>
        <span className="font-medium">
          {gate.level}
          {gate.next_level ? ` → ${gate.next_level}` : null}
        </span>
      </div>

      {gate.passed ? (
        <p className="text-sm text-green-600 dark:text-green-500">{t("passed")}</p>
      ) : (
        <ul className="flex list-disc flex-col gap-1 pl-5 text-sm text-zinc-600 dark:text-zinc-400">
          {gate.reasons.map((reason) => (
            <li key={reason}>{reason}</li>
          ))}
        </ul>
      )}

      <dl className="grid grid-cols-2 gap-3 text-sm">
        <Metric label={t("completion")} value={`${gate.completion_done}/${gate.completion_total} (${percent}%)`} />
        <Metric
          label={t("checkpointScore")}
          value={gate.checkpoint_overall === null ? "—" : `${Math.round(gate.checkpoint_overall * 100)}%`}
        />
        <Metric label={t("weakestSkill")} value={`${gate.min_skill} · ${Math.round(gate.min_rating)}`} />
      </dl>

      <ul className="flex flex-col gap-1 text-sm">
        {SKILLS.map((skill) => (
          <li key={skill} className="flex items-center justify-between">
            <span className="capitalize">{skill}</span>
            <span className="text-zinc-500">{Math.round(gate.ratings[skill] ?? 0)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-zinc-500">{label}</dt>
      <dd className="font-medium">{value}</dd>
    </div>
  );
}

function AssessmentForm({ profileId, level }: { profileId: number; level: string }) {
  const t = useTranslations("levels");
  const queryClient = useQueryClient();
  const create = useCreateAssessment();
  const [kind, setKind] = useState<"checkpoint" | "placement">("checkpoint");
  const [levelCode, setLevelCode] = useState(level);
  const [scores, setScores] = useState(EMPTY_SCORES);
  const [error, setError] = useState<string | null>(null);

  function submit(event: FormEvent) {
    event.preventDefault();
    const code = levelCode.trim();
    if (code === "") {
      setError(t("invalidLevel"));
      return;
    }
    const parsed = {
      listening: Number(scores.listening),
      reading: Number(scores.reading),
      writing: Number(scores.writing),
      speaking: Number(scores.speaking),
    };
    for (const skill of SKILLS) {
      const value = parsed[skill];
      if (!Number.isFinite(value) || value < 0 || value > 1) {
        setError(t("invalidScores"));
        return;
      }
    }

    create.mutate(
      {
        profileId,
        levelCode: code,
        kind,
        scores: parsed,
      },
      {
        onSuccess: () => {
          setError(null);
          setScores(EMPTY_SCORES);
          void queryClient.invalidateQueries({ queryKey: ["gate"] });
          void queryClient.invalidateQueries({ queryKey: ["assessments"] });
          void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
        },
      },
    );
  }

  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-medium">{t("record")}</h2>
      <form
        onSubmit={submit}
        className="flex flex-col gap-4 rounded-xl border border-zinc-200 p-4 dark:border-zinc-800"
      >
        <div className="flex flex-wrap items-center gap-3 text-sm">
          <label className="flex items-center gap-2">
            <span className="text-zinc-500">{t("level")}</span>
            <input
              value={levelCode}
              onChange={(event) => setLevelCode(event.target.value)}
              className="w-20 rounded-lg border border-zinc-200 px-2 py-1 dark:border-zinc-800"
            />
          </label>
          <label className="flex items-center gap-2">
            <span className="text-zinc-500">{t("kind")}</span>
            <select
              value={kind}
              onChange={(event) => setKind(event.target.value as "checkpoint" | "placement")}
              className="rounded-lg border border-zinc-200 px-2 py-1 dark:border-zinc-800"
            >
              <option value="checkpoint">{t("checkpoint")}</option>
              <option value="placement">{t("placement")}</option>
            </select>
          </label>
        </div>

        <div className="grid grid-cols-2 gap-3">
          {SKILLS.map((skill) => (
            <label key={skill} className="flex flex-col gap-1 text-sm">
              <span className="capitalize text-zinc-500">{skill}</span>
              <input
                type="number"
                min="0"
                max="1"
                step="0.01"
                value={scores[skill]}
                onChange={(event) =>
                  setScores((previous) => ({ ...previous, [skill]: event.target.value }))
                }
                className="rounded-lg border border-zinc-200 px-2 py-1 dark:border-zinc-800"
              />
            </label>
          ))}
        </div>

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={create.isPending}
            className="rounded-full bg-foreground px-5 py-2 text-sm text-background transition-colors hover:bg-[#383838] disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-[#ccc]"
          >
            {create.isPending ? t("loading") : t("save")}
          </button>
          {error ? <span className="text-sm text-red-600">{error}</span> : null}
          {create.isError ? (
            <span className="text-sm text-red-600">{t("loadFailed")}</span>
          ) : null}
          {create.data?.level_completed ? (
            <span className="text-sm text-green-600 dark:text-green-500">{t("completed")}</span>
          ) : null}
        </div>
      </form>
    </section>
  );
}

function History({ profileId }: { profileId: number }) {
  const t = useTranslations("levels");
  const assessments = useAssessments(profileId);

  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-medium">{t("history")}</h2>
      {assessments.isPending ? (
        <p className="text-sm text-zinc-500">{t("loading")}</p>
      ) : assessments.isError || !assessments.data ? (
        <p className="text-sm text-red-600">{t("loadFailed")}</p>
      ) : assessments.data.length === 0 ? (
        <p className="text-sm text-zinc-500">{t("noData")}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {assessments.data.map((assessment: Assessment) => (
            <li
              key={assessment.id}
              className="flex items-center justify-between rounded-lg border border-zinc-200 px-3 py-2 text-sm dark:border-zinc-800"
            >
              <span className="flex flex-col">
                <span className="font-medium capitalize">{assessment.kind}</span>
                <span className="text-zinc-500">
                  {assessment.level_code} · {assessment.created_at.slice(0, 10)}
                </span>
              </span>
              <span>{Math.round(assessment.overall * 100)}%</span>
            </li>
          ))}
        </ul>
      )}
      <Link href="/dashboard" className="text-sm text-zinc-500 transition-colors hover:text-foreground">
        {t("backToDashboard")}
      </Link>
    </section>
  );
}
