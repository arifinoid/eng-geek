"use client";

import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/routing";
import { useProfiles } from "@/lib/api";
import { setSelectedProfileId } from "@/lib/profile";

export function ProfilePicker() {
  const t = useTranslations("home");
  const router = useRouter();
  const profiles = useProfiles();

  if (profiles.isPending) {
    return <p className="text-zinc-500">{t("loadingProfiles")}</p>;
  }
  if (profiles.isError) {
    return <p className="text-red-600">{t("loadFailed")}</p>;
  }

  const list = profiles.data ?? [];
  if (list.length === 0) {
    return <p className="text-zinc-500">{t("noProfiles")}</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-zinc-600 dark:text-zinc-400">{t("pickProfile")}</p>
      <div className="flex flex-col gap-3">
        {list.map((profile) => (
          <button
            key={profile.id}
            type="button"
            onClick={() => {
              setSelectedProfileId(profile.id);
              router.push("/dashboard");
            }}
            className="rounded-xl border border-zinc-200 p-4 text-left transition-colors hover:border-zinc-400 dark:border-zinc-800 dark:hover:border-zinc-600"
          >
            <span className="block font-medium">{profile.name}</span>
            <span className="block text-sm text-zinc-500">
              {profile.track} · {profile.target_exam}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}
