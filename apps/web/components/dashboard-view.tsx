"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { useDashboard } from "@/lib/api";
import { useSelectedProfileId } from "@/lib/use-selected-profile";
import { ProfilePrompt } from "./profile-prompt";

export function DashboardView() {
  const t = useTranslations("dashboard");
  const profileId = useSelectedProfileId();
  const dashboard = useDashboard(profileId);

  if (profileId === undefined) return <ProfilePrompt />;
  if (dashboard.isPending) {
    return <p className="text-zinc-500">{t("loading")}</p>;
  }
  if (dashboard.isError || !dashboard.data) {
    return <p className="text-red-600">{t("loadFailed")}</p>;
  }

  const data = dashboard.data;
  const weekly = data.weekly_minutes;
  const maxMinutes = Math.max(1, ...weekly.map((week) => week.minutes));

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-1">
        <h2 className="text-xl font-semibold tracking-tight">{data.profile.name}</h2>
        <p className="text-sm text-zinc-500">
          {data.profile.track} · {data.profile.target_exam}
        </p>
      </section>

      <section className="grid grid-cols-3 gap-3">
        <StatCard label={t("streak")} value={`${data.streak_days}`} unit={t("days")} />
        <StatCard
          label={t("weekMinutes")}
          value={`${data.week_minutes}`}
          unit={t("minutes")}
        />
        <StatCard label={t("level")} value={data.level} />
      </section>

      <section className="flex flex-col gap-3">
        <h3 className="text-lg font-medium">{t("skills")}</h3>
        <ul className="flex flex-col gap-2">
          {data.skills.map((skill) => (
            <li
              key={skill.skill}
              className="flex items-center justify-between rounded-lg border border-zinc-200 px-3 py-2 dark:border-zinc-800"
            >
              <span className="capitalize">{skill.skill}</span>
              <span className="text-sm text-zinc-500">
                {Math.round(skill.rating)} · {skill.level} · {skill.attempts_count}{" "}
                {t("attempts")}
              </span>
            </li>
          ))}
        </ul>
      </section>

      <section className="flex flex-col gap-3">
        <h3 className="text-lg font-medium">{t("weeklyMinutes")}</h3>
        {weekly.length === 0 ? (
          <p className="text-sm text-zinc-500">{t("noData")}</p>
        ) : (
          <div className="flex h-24 items-end gap-1">
            {weekly.map((week) => (
              <div
                key={week.week_start}
                className="flex-1 rounded-t bg-zinc-300 dark:bg-zinc-700"
                style={{ height: `${(week.minutes / maxMinutes) * 100}%` }}
                title={`${week.week_start}: ${week.minutes} ${t("minutes")}`}
              />
            ))}
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h3 className="text-lg font-medium">{t("nextUp")}</h3>
        {data.next_activities.length === 0 ? (
          <p className="text-sm text-zinc-500">{t("noNext")}</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {data.next_activities.map((activity) => (
              <li key={activity.id}>
                <Link
                  href={`/practice/activities?id=${activity.id}`}
                  className="flex items-center justify-between rounded-lg border border-zinc-200 px-3 py-2 transition-colors hover:border-zinc-400 dark:border-zinc-800 dark:hover:border-zinc-600"
                >
                  <span className="flex flex-col">
                    <span className="font-medium">{activity.unit_title}</span>
                    <span className="text-sm text-zinc-500 capitalize">
                      {activity.skill} · {activity.type}
                    </span>
                  </span>
                  <span className="text-sm">{t("open")}</span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function StatCard({
  label,
  value,
  unit,
}: {
  label: string;
  value: string;
  unit?: string;
}) {
  return (
    <div className="flex flex-col gap-1 rounded-xl border border-zinc-200 p-3 dark:border-zinc-800">
      <span className="text-xs text-zinc-500">{label}</span>
      <span className="text-xl font-semibold">
        {value}
        {unit ? <span className="ml-1 text-xs font-normal">{unit}</span> : null}
      </span>
    </div>
  );
}
