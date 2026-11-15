import { getTranslations } from "next-intl/server";

export default async function Practice({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "practice" });

  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
      <p className="max-w-md text-zinc-600 dark:text-zinc-400">{t("subtitle")}</p>
    </main>
  );
}
