"use client";

import { useSyncExternalStore } from "react";
import { getSelectedProfileId } from "./profile";

function subscribe(callback: () => void): () => void {
  window.addEventListener("storage", callback);
  return () => window.removeEventListener("storage", callback);
}

/**
 * The persisted profile id. Reading it through `useSyncExternalStore` keeps the
 * server render (no profile) and the first client render in sync.
 */
export function useSelectedProfileId(): number | undefined {
  return useSyncExternalStore(subscribe, getSelectedProfileId, () => undefined);
}
