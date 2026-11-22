"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";

/** Shown when a page needs a profile but none is selected yet. */
export function ProfilePrompt() {
  const t = useTranslations("common");

  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-zinc-300 p-6 text-center dark:border-zinc-700">
      <p className="text-zinc-600 dark:text-zinc-400">{t("needProfile")}</p>
      <Link
        href="/"
        className="rounded-full bg-foreground px-4 py-2 text-sm text-background transition-colors hover:bg-[#383838] dark:hover:bg-[#ccc]"
      >
        {t("goHome")}
      </Link>
    </div>
  );
}
