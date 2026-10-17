// Small effects for the whole page: fade in on scroll, the glow that follows the mouse,
// screenshots that tilt, and cards that light up under the mouse. All of them respect "reduce motion".
export const reduceMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;

const io = new IntersectionObserver(
  (entries) => entries.forEach((e) => { if (e.isIntersecting) { e.target.classList.add('in'); io.unobserve(e.target); } }),
  { threshold: 0.12 },
);
document.querySelectorAll<HTMLElement>('.reveal').forEach((el, i) => { el.style.transitionDelay = (i % 4) * 80 + 'ms'; io.observe(el); });

const aurora = document.querySelector<HTMLElement>('.aurora');
if (aurora && !reduceMotion) addEventListener('pointermove', (e) => aurora.style.setProperty('--mx', (e.clientX / innerWidth) * 100 + '%'), { passive: true });

document.querySelectorAll<HTMLElement>('[data-tilt]').forEach((el) => {
  const amount = Number(el.dataset.tilt) || 10;
  el.addEventListener('pointermove', (e) => {
    if (reduceMotion || e.pointerType === 'touch') return;
    const r = el.getBoundingClientRect(), x = (e.clientX - r.left) / r.width, y = (e.clientY - r.top) / r.height;
    el.style.transform = `perspective(1600px) rotateY(${(x - 0.5) * amount}deg) rotateX(${(0.5 - y) * amount * 0.8}deg)`;
    el.style.setProperty('--gx', x * 100 + '%');
    el.style.setProperty('--gy', y * 100 + '%');
  });
  el.addEventListener('pointerleave', () => (el.style.transform = ''));
});

document.querySelectorAll<HTMLElement>('[data-spotlight]').forEach((el) => el.addEventListener('pointermove', (e) => {
  const r = el.getBoundingClientRect();
  el.style.setProperty('--x', e.clientX - r.left + 'px');
  el.style.setProperty('--y', e.clientY - r.top + 'px');
}));
