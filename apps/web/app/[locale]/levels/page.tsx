import { getTranslations } from "next-intl/server";
import { LevelsView } from "@/components/levels-view";

export default async function LevelsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "levels" });

  return (
    <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 px-4 py-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="text-zinc-600 dark:text-zinc-400">{t("subtitle")}</p>
      </div>
      <LevelsView />
    </main>
  );
}
