"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";

export function AppHeader() {
  const t = useTranslations("nav");

  return (
    <header className="border-b border-zinc-200 dark:border-zinc-800">
      <div className="mx-auto flex w-full max-w-3xl items-center justify-between px-4 py-3">
        <Link href="/" className="text-lg font-semibold tracking-tight">
          eng-geek
        </Link>
        <nav className="flex gap-4 text-sm">
          <Link
            href="/"
            className="text-zinc-600 transition-colors hover:text-foreground dark:text-zinc-400"
          >
            {t("home")}
          </Link>
          <Link
            href="/practice"
            className="text-zinc-600 transition-colors hover:text-foreground dark:text-zinc-400"
          >
            {t("practice")}
          </Link>
          <Link
            href="/dashboard"
            className="text-zinc-600 transition-colors hover:text-foreground dark:text-zinc-400"
          >
            {t("dashboard")}
          </Link>
          <Link
            href="/levels"
            className="text-zinc-600 transition-colors hover:text-foreground dark:text-zinc-400"
          >
            {t("level")}
          </Link>
        </nav>
      </div>
    </header>
  );
}
