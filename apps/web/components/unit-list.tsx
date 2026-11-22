"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { useUnits } from "@/lib/api";
import { useSelectedProfileId } from "@/lib/use-selected-profile";
import { ProfilePrompt } from "./profile-prompt";

export function UnitList() {
  const t = useTranslations("practice");
  const profileId = useSelectedProfileId();
  const units = useUnits(profileId);

  if (profileId === undefined) return <ProfilePrompt />;
  if (units.isPending) return <p className="text-zinc-500">{t("loading")}</p>;
  if (units.isError || !units.data) {
    return <p className="text-red-600">{t("loadFailed")}</p>;
  }

  if (units.data.length === 0) {
    return <p className="text-zinc-500">{t("noUnits")}</p>;
  }

  return (
    <ul className="flex flex-col gap-2">
      {units.data.map((unit) => (
        <li key={unit.id}>
          <Link
            href={`/practice/units?id=${unit.id}`}
            className="flex items-center gap-3 rounded-xl border border-zinc-200 px-4 py-3 transition-colors hover:border-zinc-400 dark:border-zinc-800 dark:hover:border-zinc-600"
          >
            <span className="w-8 text-sm text-zinc-500">{unit.position}</span>
            <span className="flex flex-col">
              <span className="font-medium">{unit.title}</span>
              <span className="text-sm text-zinc-500">{unit.level_code}</span>
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
