import { useSyncExternalStore } from "react";

/** A model the user starred in the agent picker, with the agent it belongs to. */
export interface ModelFavorite {
	harness: string;
	model: string;
	label: string;
}

const STORAGE_KEY = "ao.chat.modelFavorites";
const listeners = new Set<() => void>();
let cached: ModelFavorite[] | undefined;

function read(): ModelFavorite[] {
	if (cached) return cached;
	try {
		const parsed: unknown = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "[]");
		cached = Array.isArray(parsed)
			? parsed.filter((entry): entry is ModelFavorite =>
				typeof entry?.harness === "string" && typeof entry?.model === "string" && typeof entry?.label === "string")
			: [];
	} catch {
		cached = [];
	}
	return cached;
}

function write(next: ModelFavorite[]) {
	cached = next;
	try {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
	} catch {
		// A full or blocked store only loses the favourite across restarts.
	}
	for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
	listeners.add(listener);
	return () => listeners.delete(listener);
}

/** Adds the model to the favourites, or removes it if it is already one. */
export function toggleModelFavorite(favorite: ModelFavorite) {
	const current = read();
	const kept = current.filter((entry) => entry.harness !== favorite.harness || entry.model !== favorite.model);
	write(kept.length === current.length ? [...current, favorite] : kept);
}

/** The starred models, in the order they were starred. */
export function useModelFavorites(): ModelFavorite[] {
	return useSyncExternalStore(subscribe, read, read);
}
