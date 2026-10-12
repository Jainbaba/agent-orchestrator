import { useSyncExternalStore } from "react";

/**
 * A model the user starred in the agent picker, with the settings it had: one
 * favourite per model row. For an agent whose catalog folds settings into the
 * model id (Cursor), `model` is that exact variant.
 */
export interface ModelFavorite {
	harness: string;
	/** The row in the agent's model list. */
	row: string;
	/** The model id to select. */
	model: string;
	label: string;
	effort?: string;
	effortLabel?: string;
	fast?: boolean;
	thinking?: boolean;
}

const STORAGE_KEY = "ao.chat.modelFavorites";
const listeners = new Set<() => void>();
let cached: ModelFavorite[] | undefined;

function read(): ModelFavorite[] {
	if (cached) return cached;
	try {
		const parsed: unknown = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "[]");
		cached = Array.isArray(parsed)
			? parsed
				.filter((entry) => typeof entry?.harness === "string" && typeof entry?.model === "string" && typeof entry?.label === "string")
				// A favourite saved before rows were recorded was a plain model.
				.map((entry) => ({ ...entry, row: typeof entry.row === "string" ? entry.row : entry.model }) as ModelFavorite)
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
	const kept = current.filter((entry) => entry.harness !== favorite.harness || entry.row !== favorite.row);
	write(kept.length === current.length ? [...current, favorite] : kept);
}

/** The starred models, in the order they were starred. */
export function useModelFavorites(): ModelFavorite[] {
	return useSyncExternalStore(subscribe, read, read);
}
