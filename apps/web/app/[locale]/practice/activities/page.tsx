import { Suspense } from "react";
import { getTranslations } from "next-intl/server";
import { ActivityRunner } from "@/components/activity-runner";

export default async function ActivityPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "activity" });
  const common = await getTranslations({ locale, namespace: "common" });

  return (
    <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 px-4 py-6">
      <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
      <Suspense fallback={<p className="text-zinc-500">{common("loading")}</p>}>
        <ActivityRunner />
      </Suspense>
    </main>
  );
}
