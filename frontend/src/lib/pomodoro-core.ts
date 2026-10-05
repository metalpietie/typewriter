// The rules of the Pomodoro timer. No imports, so it can be tested alone.
// A round of work, then a short break. After a set number of rounds, a long break.

export type Phase = 'work' | 'short' | 'long';

export type Settings = {
  work: number;   // minutes of work in a round
  short: number;  // minutes of a short break
  long: number;   // minutes of a long break
  rounds: number; // rounds of work before the long break
  auto: boolean;  // start the next part by itself
  sound: boolean; // ring when a part ends
  flash: boolean; // show a message and flash the screen when a part ends
};

export const DEFAULTS: Settings = { work: 25, short: 5, long: 15, rounds: 4, auto: false, sound: true, flash: true };

// the smallest and largest value you can choose
export const LIMITS: Record<'work' | 'short' | 'long' | 'rounds', [number, number]> = {
  work: [1, 180], short: [1, 60], long: [1, 120], rounds: [2, 12],
};

export const LABEL: Record<Phase, string> = { work: 'Work', short: 'Short break', long: 'Long break' };

const whole = (v: unknown, [lo, hi]: [number, number], fallback: number): number => {
  const n = Math.round(Number(v));
  return Number.isFinite(n) && v !== '' && v !== null ? Math.min(hi, Math.max(lo, n)) : fallback;
};

/** Saved or typed values made safe. A wrong or missing value becomes the normal one. */
export function cleanSettings(raw: unknown): Settings {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>;
  return {
    work: whole(r.work, LIMITS.work, DEFAULTS.work),
    short: whole(r.short, LIMITS.short, DEFAULTS.short),
    long: whole(r.long, LIMITS.long, DEFAULTS.long),
    rounds: whole(r.rounds, LIMITS.rounds, DEFAULTS.rounds),
    auto: typeof r.auto === 'boolean' ? r.auto : DEFAULTS.auto,
    sound: typeof r.sound === 'boolean' ? r.sound : DEFAULTS.sound,
    flash: typeof r.flash === 'boolean' ? r.flash : DEFAULTS.flash,
  };
}

export const phaseMs = (s: Settings, p: Phase): number => s[p === 'work' ? 'work' : p] * 60_000;

/** What comes after a part. round is the number of the work round (1 and up). */
export function nextAfter(phase: Phase, round: number, s: Settings): { phase: Phase; round: number } {
  if (phase === 'work') return { phase: round >= s.rounds ? 'long' : 'short', round };
  return { phase: 'work', round: phase === 'long' ? 1 : round + 1 };
}

/** "25:00". The seconds are rounded up, so the clock shows 0:01 until the very end. */
export function mmss(ms: number): string {
  const total = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}

/** A finished round of work. */
export type Round = { words: number; minutes: number };

export function summarize(rounds: Round[]): { count: number; words: number; minutes: number } {
  return rounds.reduce<{ count: number; words: number; minutes: number }>((a, r) => ({ count: a.count + 1, words: a.words + r.words, minutes: a.minutes + r.minutes }), { count: 0, words: 0, minutes: 0 });
}

const plural = (n: number, one: string) => `${n.toLocaleString()} ${one}${n === 1 ? '' : 's'}`;

/** The message shown when a part ends. */
export function endMessage(ended: Phase, next: Phase, round: number, words: number): string {
  if (ended === 'work') {
    return `Round done. You wrote ${plural(Math.max(0, words), 'word')}. Time for a ${next === 'long' ? 'long' : 'short'} break.`;
  }
  return `Break over. Ready for round ${round}?`;
}
