const STORAGE_KEY = "eng-geek.profileId";

/** Reads the persisted profile id, or undefined when nothing valid is stored. */
export function getSelectedProfileId(): number | undefined {
  if (typeof window === "undefined") return undefined;
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (raw === null) return undefined;
  const id = Number(raw);
  return Number.isInteger(id) && id > 0 ? id : undefined;
}

export function setSelectedProfileId(profileId: number): void {
  window.localStorage.setItem(STORAGE_KEY, String(profileId));
}

export function clearSelectedProfileId(): void {
  window.localStorage.removeItem(STORAGE_KEY);
}
