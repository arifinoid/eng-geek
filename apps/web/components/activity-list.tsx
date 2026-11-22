"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { useSearchParams } from "next/navigation";
import { useSelectedProfileId } from "@/lib/use-selected-profile";
import { useUnitActivities, useUnits } from "@/lib/api";
import { ProfilePrompt } from "./profile-prompt";

function parseId(value: string | null): number | undefined {
  if (value === null) return undefined;
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : undefined;
}

export function ActivityList() {
  const t = useTranslations("unit");
  const params = useSearchParams();
  const unitId = parseId(params.get("id"));
  const profileId = useSelectedProfileId();
  const units = useUnits(profileId);
  const activities = useUnitActivities(unitId);

  if (profileId === undefined) return <ProfilePrompt />;
  if (unitId === undefined) {
    return <p className="text-zinc-500">{t("invalidUnit")}</p>;
  }
  if (activities.isPending) return <p className="text-zinc-500">{t("loading")}</p>;
  if (activities.isError || !activities.data) {
    return <p className="text-red-600">{t("loadFailed")}</p>;
  }

  const unit = units.data?.find((candidate) => candidate.id === unitId);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <Link
          href="/practice"
          className="text-sm text-zinc-500 transition-colors hover:text-foreground"
        >
          {t("backToUnits")}
        </Link>
        <h2 className="text-xl font-semibold tracking-tight">{unit?.title ?? t("title")}</h2>
      </div>

      {activities.data.length === 0 ? (
        <p className="text-zinc-500">{t("noActivities")}</p>
      ) : (
        <ol className="flex flex-col gap-2">
          {activities.data.map((activity) => (
            <li key={activity.id}>
              <Link
                href={`/practice/activities?id=${activity.id}`}
                className="flex items-center gap-3 rounded-xl border border-zinc-200 px-4 py-3 transition-colors hover:border-zinc-400 dark:border-zinc-800 dark:hover:border-zinc-600"
              >
                <span className="w-6 text-sm text-zinc-500">{activity.position}</span>
                <span className="flex flex-col">
                  <span className="font-medium capitalize">{activity.type}</span>
                  <span className="text-sm text-zinc-500 capitalize">
                    {activity.skill}
                  </span>
                </span>
              </Link>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
