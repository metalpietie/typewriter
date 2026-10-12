import type { TransitionConfig } from 'svelte/transition';
import { PARAMS, bezier, cleanLevel, cssVars, nextLevel, startLevel, type Level, type Params } from './motion-core';

// Motion: ripples on buttons, menus that open and close, pages that arrive, toasts that slide.
// How much of it you see is one choice, the level (Off, Calm, Smooth, Lively). It belongs to this computer, like the colour theme.
// The text of the editor never moves.

const KEY = 'typewriter-motion';

const wantsLess = (): boolean => { try { return window.matchMedia('(prefers-reduced-motion: reduce)').matches; } catch { return false; } };
const savedChoice = (): string | null => { try { return localStorage.getItem(KEY); } catch { return null; } };

export const motion = $state<{ level: Level }>({ level: startLevel(savedChoice(), wantsLess()) });
export const params = (): Params => PARAMS[motion.level];

/** Put the numbers of the level on the page, as CSS variables, so the style sheets can use them. */
export function applyMotion() {
  const root = document.documentElement;
  for (const [name, value] of Object.entries(cssVars(params()))) root.style.setProperty(name, value);
  root.dataset.motion = motion.level;
}

export function setLevel(level: Level) {
  motion.level = cleanLevel(level, 'smooth');
  try { localStorage.setItem(KEY, motion.level); } catch { /* the choice lasts until the app closes */ }
  applyMotion();
}
export const cycleLevel = () => setLevel(nextLevel(motion.level));

// ---- transitions for Svelte blocks that come and go ----

const none = (): TransitionConfig => ({ duration: 0 });

/** A menu or pop-up: it fades in and slides a little, from the side it hangs from. */
export function pop(_node: Element, opts: { origin?: string } = {}): TransitionConfig {
  const p = params();
  if (!p.dur) return none();
  return { duration: p.dur, easing: bezier(p.ease), css: (t, u) => `opacity:${Math.min(1, t * 1.4)};transform-origin:${opts.origin ?? 'top left'};transform:translateY(${-p.travel * 0.6 * u}px) scale(${1 - 0.03 * u});` };
}

/** The dark veil behind a window. */
export function veil(_node: Element): TransitionConfig {
  const p = params();
  return p.dur ? { duration: p.dur, css: (t) => `opacity:${t}` } : none();
}

/** A window: it fades in, grows a little and slides down into place. */
export function dialog(_node: Element): TransitionConfig {
  const p = params();
  if (!p.dur) return none();
  return { duration: p.dur * 1.1, easing: bezier(p.ease), css: (t, u) => `opacity:${Math.min(1, t * 1.5)};transform:translateY(${-p.travel * 0.8 * u}px) scale(${1 - 0.02 * u});` };
}

/** A message at the edge of the screen: it slides up and fades in. It uses "translate", so a centring "transform" still works. */
export function slideUp(_node: Element): TransitionConfig {
  const p = params();
  if (!p.dur) return none();
  return { duration: p.dur * 1.2, easing: bezier(p.ease), css: (t, u) => `opacity:${Math.min(1, t * 1.5)};translate:0 ${p.travel * 2 * u}px;` };
}

// ---- things that run on the whole page ----

/** A ripple where you press a button. One listener serves every button of the app. */
export function installRipple(): () => void {
  const down = (e: PointerEvent) => {
    const p = params();
    if (!p.rippleMs || e.button !== 0 || !(e.target instanceof Element)) return;
    const b = e.target.closest('button') as HTMLButtonElement | null;
    if (!b || b.disabled || b.classList.contains('noripple') || b.closest('.ProseMirror')) return;
    const r = b.getBoundingClientRect();
    if (r.width < 4 || r.height < 4) return;
    // the ripple is cut off at the edge of the button. This is set only while it runs.
    const wasStatic = getComputedStyle(b).position === 'static';
    const overflow = b.style.overflow;
    if (wasStatic) b.style.position = 'relative';
    b.style.overflow = 'hidden';
    const size = Math.max(r.width, r.height) * p.rippleScale;
    const dot = document.createElement('span');
    dot.setAttribute('aria-hidden', 'true');
    dot.dataset.rip = '1';
    dot.style.cssText = `position:absolute;pointer-events:none;border-radius:50%;background:currentColor;width:${size}px;height:${size}px;left:${e.clientX - r.left - size / 2}px;top:${e.clientY - r.top - size / 2}px`;
    b.appendChild(dot);
    let finished = false;
    const done = () => {
      if (finished) return;
      finished = true;
      dot.remove();
      if (!b.querySelector(':scope > [data-rip]')) {
        b.style.overflow = overflow;
        if (wasStatic) b.style.position = '';
      }
    };
    dot.animate([{ transform: 'scale(0)', opacity: p.rippleAlpha }, { transform: 'scale(1)', opacity: 0 }], { duration: p.rippleMs, easing: 'ease-out' }).onfinish = done;
    setTimeout(done, p.rippleMs + 150); // the ripple must go even if the animation is paused, for example in a hidden window
  };
  document.addEventListener('pointerdown', down);
  return () => document.removeEventListener('pointerdown', down);
}

/** A page that has just opened: its parts fade in and slide up, one after another. */
export function animatePage(main: HTMLElement | null) {
  const p = params();
  if (!main || !p.dur) return;
  const root = [...main.children].find((c) => !c.classList.contains('resizer')) as HTMLElement | undefined;
  if (!root) return;
  const parts = ([...root.children] as HTMLElement[]).slice(0, 10);
  (parts.length ? parts : [root]).forEach((el, i) => {
    el.animate([{ opacity: 0, transform: `translateY(${p.travel}px)` }, { opacity: 1, transform: 'none' }],
      { duration: p.dur * 1.3, delay: i * p.stagger, easing: p.ease, fill: 'backwards' });
  });
}

/** The whole window fades a little when you go into focus mode or come out of it. */
export function fadeWindow(el: HTMLElement | null) {
  const p = params();
  if (!el || !p.dur) return;
  el.animate([{ opacity: 0.25 }, { opacity: 1 }], { duration: p.dur * 2, easing: p.ease });
}
