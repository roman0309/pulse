import { create } from "zustand";

export type RangeKey = "15m" | "1h" | "6h" | "24h";

const MINUTES: Record<RangeKey, number> = { "15m": 15, "1h": 60, "6h": 360, "24h": 1440 };
const STEP: Record<RangeKey, number> = { "15m": 30, "1h": 60, "6h": 300, "24h": 600 };

export const RANGE_KEYS: RangeKey[] = ["15m", "1h", "6h", "24h"];

interface RangeState {
  range: RangeKey;
  live: boolean;
  setRange: (r: RangeKey) => void;
  toggleLive: () => void;
}

// Shared across Overview / Metrics / Logs / Traces so the time window and
// auto-refresh state stay consistent as you move between views.
export const useRangeStore = create<RangeState>((set) => ({
  range: "1h",
  live: true,
  setRange: (range) => set({ range }),
  toggleLive: () => set((s) => ({ live: !s.live })),
}));

// rangeWindow computes the absolute from/to (and a sensible chart step) for the
// selected key. `from` is evaluated at call time so each query gets a fresh window.
export function rangeWindow(range: RangeKey) {
  const minutes = MINUTES[range];
  const now = Date.now();
  return {
    minutes,
    step: STEP[range],
    from: new Date(now - minutes * 60_000).toISOString(),
    to: new Date(now).toISOString(),
  };
}
