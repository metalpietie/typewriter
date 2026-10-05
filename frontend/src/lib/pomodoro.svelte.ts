import { Window } from '@wailsio/runtime';
import { DEFAULTS, LABEL, cleanSettings, endMessage, mmss, nextAfter, phaseMs, summarize, type Phase, type Round, type Settings } from './pomodoro-core';

// The Pomodoro timer. It lives here, not in a screen, so it keeps running while you change tabs.
// The settings and the rounds you finished today are choices of this computer, like the colour theme.
// They are not in the project data.

const SETTINGS_KEY = 'typewriter-pomodoro';
const LOG_KEY = 'typewriter-pomodoro-log';

const today = () => { const d = new Date(); return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`; };

function loadSettings(): Settings {
  try { return cleanSettings(JSON.parse(localStorage.getItem(SETTINGS_KEY) ?? 'null')); } catch { return { ...DEFAULTS }; }
}
function loadLog(): Round[] {
  try {
    const saved = JSON.parse(localStorage.getItem(LOG_KEY) ?? 'null');
    if (saved?.day !== today() || !Array.isArray(saved.rounds)) return [];
    return saved.rounds.map((r: any) => ({ words: Number(r.words) || 0, minutes: Number(r.minutes) || 0 }));
  } catch { return []; }
}

export const pomo = $state({
  settings: loadSettings(),
  phase: 'work' as Phase,
  round: 1,                       // the number of the work round
  running: false,
  endsAt: 0,                      // when the running part ends (a time in milliseconds)
  leftMs: null as number | null,  // time left while paused. null: not started, so the full length
  now: Date.now(),                // moves every half second while the timer runs
  words: 0,                       // words written in the work round now
  message: '',
  alert: false,                   // a part just ended: the screen shows a message until you answer it
  done: loadLog(),                // the rounds you finished today
});

let ticker: ReturnType<typeof setInterval> | undefined;
let audio: AudioContext | undefined;

/** Time left in this part, in milliseconds. */
export const left = (): number => (pomo.running ? Math.max(0, pomo.endsAt - pomo.now) : (pomo.leftMs ?? phaseMs(pomo.settings, pomo.phase)));
/** How far this part has come, from 0 to 1. */
export const progress = (): number => 1 - left() / phaseMs(pomo.settings, pomo.phase);
export const started = (): boolean => pomo.running || pomo.leftMs !== null;
export const clock = (): string => mmss(left());
/** A short text for the corner of the screen. Empty when the timer is not running. */
export const miniText = (): string => (pomo.running ? `${LABEL[pomo.phase]} ${clock()}` : '');
export const todaySummary = () => summarize(pomo.done);

function saveLog() {
  try { localStorage.setItem(LOG_KEY, JSON.stringify({ day: today(), rounds: pomo.done })); } catch { /* the list lasts until the app closes */ }
}

/** Called while you type, with the words added (below 0 when words were cut). */
export function noteWords(delta: number) {
  if (pomo.running && pomo.phase === 'work') pomo.words += delta;
}

function ring() {
  if (!pomo.settings.sound) return;
  try {
    audio ??= new AudioContext();
    void audio.resume();
    // three soft beeps
    for (let i = 0; i < 3; i++) {
      const t = audio.currentTime + i * 0.35;
      const osc = audio.createOscillator(), gain = audio.createGain();
      osc.type = 'sine'; osc.frequency.value = 880;
      gain.gain.setValueAtTime(0.0001, t);
      gain.gain.exponentialRampToValueAtTime(0.25, t + 0.03);
      gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.3);
      osc.connect(gain).connect(audio.destination);
      osc.start(t); osc.stop(t + 0.32);
    }
  } catch { /* no sound on this computer: the message still shows */ }
}

const APP_TITLE = 'Typewriter';
/** The window title shows in the task bar, so you see the alert when the app is behind other windows. */
function setTitle(text: string) {
  try { Window.SetTitle(text)?.catch?.(() => {}); } catch { /* no window title in a browser test */ }
}
let alertTimer: ReturnType<typeof setTimeout> | undefined;

/** You saw it: take the message away. */
export function dismiss() {
  clearTimeout(alertTimer);
  if (!pomo.alert) return;
  pomo.alert = false;
  setTitle(APP_TITLE);
}

function raiseAlert() {
  pomo.alert = true;
  setTitle(`${pomo.message.split('.')[0]} · ${APP_TITLE}`);
  clearTimeout(alertTimer);
  // when the next part started by itself, nothing waits for you, so the message goes away alone
  if (pomo.running) alertTimer = setTimeout(dismiss, 10_000);
}

function stopTicker() { clearInterval(ticker); ticker = undefined; }
function runTicker() {
  stopTicker();
  ticker = setInterval(() => {
    pomo.now = Date.now();
    if (pomo.running && pomo.now >= pomo.endsAt) finish(true);
  }, 500);
}

function begin(fromMs: number) {
  pomo.now = Date.now();
  pomo.endsAt = pomo.now + fromMs;
  pomo.leftMs = null;
  pomo.running = true;
  runTicker();
}

export function start() {
  if (pomo.running) return;
  if (pomo.leftMs === null) { // a new part, not a continue after a pause
    pomo.message = '';
    if (pomo.phase === 'work') pomo.words = 0;
  }
  dismiss();
  try { audio ??= new AudioContext(); void audio.resume(); } catch { /* the first click lets the sound play later */ }
  begin(left());
  // the card shows above Search, so bring the top of the menu into view
  document.querySelector('.nav')?.scrollTo({ top: 0 });
}

export function pause() {
  if (!pomo.running) return;
  pomo.leftMs = Math.max(0, pomo.endsAt - Date.now());
  pomo.running = false;
  stopTicker();
}

/** The part is over: it ran out (completed), or you pressed Skip. */
function finish(completed: boolean) {
  const ended = pomo.phase;
  const full = phaseMs(pomo.settings, ended);
  const elapsed = full - Math.min(full, left());
  const words = Math.max(0, pomo.words);
  // a round you stop early still counts when you wrote something or worked for a minute
  if (ended === 'work' && (completed || words > 0 || elapsed >= 60_000)) {
    pomo.done.push({ words, minutes: Math.round(elapsed / 60_000) });
    saveLog();
  }
  const next = nextAfter(ended, pomo.round, pomo.settings);
  pomo.message = endMessage(ended, next.phase, next.round, words);
  pomo.phase = next.phase;
  pomo.round = next.round;
  pomo.leftMs = null;
  if (completed) ring();
  if (pomo.settings.auto && completed) {
    if (next.phase === 'work') pomo.words = 0;
    begin(phaseMs(pomo.settings, next.phase));
  } else {
    pomo.running = false;
    stopTicker();
  }
  if (completed && pomo.settings.flash) raiseAlert();
}

/** End this part now. A part that did not start counts for nothing. */
export const skip = () => { dismiss(); if (!started()) pomo.words = 0; finish(false); };

/** Stop and go back to round 1. */
export function reset() {
  dismiss();
  pomo.running = false;
  stopTicker();
  pomo.phase = 'work'; pomo.round = 1; pomo.leftMs = null; pomo.words = 0; pomo.message = '';
}

export function clearToday() { pomo.done = []; saveLog(); }

export function setSettings(patch: Partial<Settings>) {
  const next = cleanSettings({ ...pomo.settings, ...patch });
  Object.assign(pomo.settings, next);
  // a part that is paused cannot be longer than the new length
  if (pomo.leftMs !== null) pomo.leftMs = Math.min(pomo.leftMs, phaseMs(next, pomo.phase));
  // the part that runs now keeps its end time; the new length counts from the next part
  if (pomo.round > next.rounds) pomo.round = next.rounds;
  try { localStorage.setItem(SETTINGS_KEY, JSON.stringify({ ...pomo.settings })); } catch { /* the choice lasts until the app closes */ }
}

export const resetSettings = () => setSettings({ ...DEFAULTS });
