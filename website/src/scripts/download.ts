// Picks the right download for this computer, and asks GitHub for the newest release
// (the page was built with the release of that day; a newer one shows up without a rebuild).
// Elements it fills:
//   data-dl="primary|installer|portable|linux|release"   the link
//   data-version                                        "v1.12.1"
//   data-os-label                                       "for Windows" / "for Linux" / "" (unknown system)
//   data-os-only="windows|linux|other"                  shown only on that system
const REPO = 'metalpietie/typewriter';

type OS = 'windows' | 'linux' | 'other';
type Links = { version: string; page: string; installer: string; portable: string; linux: string };

export function detectOS(): OS {
  const nav = navigator as Navigator & { userAgentData?: { platform?: string } };
  const p = (nav.userAgentData?.platform || navigator.platform || '').toLowerCase();
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes('android') || /iphone|ipad/.test(ua)) return 'other';
  if (p.includes('win') || ua.includes('windows')) return 'windows';
  if (p.includes('linux') || ua.includes('linux') || ua.includes('x11')) return 'linux';
  return 'other';
}

async function latest(): Promise<Links | null> {
  try {
    const hit = sessionStorage.getItem('tw-release');
    if (hit) return JSON.parse(hit);
  } catch { /* storage blocked: ask GitHub */ }
  try {
    const r = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`, { headers: { Accept: 'application/vnd.github+json' } });
    if (!r.ok) return null;
    const j = await r.json();
    const find = (end: string): string => j.assets.find((a: { name: string }) => a.name.endsWith(end))?.browser_download_url ?? j.html_url;
    const links: Links = { version: String(j.tag_name).replace(/^v/, ''), page: j.html_url, installer: find('-windows-x64-installer.exe'), portable: find('-windows-x64.exe'), linux: find('-linux-x64') };
    try { sessionStorage.setItem('tw-release', JSON.stringify(links)); } catch { /* fine */ }
    return links;
  } catch {
    return null; // offline or rate-limited: keep the links from the build
  }
}

function primary(os: OS, l: Pick<Links, 'installer' | 'linux' | 'page'>): string {
  return os === 'windows' ? l.installer : os === 'linux' ? l.linux : l.page;
}

const os = detectOS();
document.documentElement.dataset.os = os;
document.querySelectorAll<HTMLElement>('[data-os-only]').forEach((el) => (el.hidden = el.dataset.osOnly !== os));
document.querySelectorAll<HTMLElement>('[data-os-label]').forEach((el) => (el.textContent = os === 'windows' ? 'for Windows' : os === 'linux' ? 'for Linux' : ''));
// Umami counts the click on the main button by the file it really gave (see data-umami-event in the components)
document.querySelectorAll<HTMLElement>('[data-dl="primary"]').forEach((el) => (el.dataset.umamiEventFile = os === 'windows' ? 'installer' : os === 'linux' ? 'linux' : 'release page'));

// first with the links from the build, then with the newest ones
const built = (kind: string) => document.querySelector<HTMLAnchorElement>(`[data-dl="${kind}"]`)?.href ?? '';
const fill = (l: Links) => {
  const map: Record<string, string> = { installer: l.installer, portable: l.portable, linux: l.linux, release: l.page, primary: primary(os, l) };
  document.querySelectorAll<HTMLAnchorElement>('[data-dl]').forEach((a) => { const href = map[a.dataset.dl!]; if (href) a.href = href; });
  if (l.version) document.querySelectorAll('[data-version]').forEach((el) => (el.textContent = 'v' + l.version));
};
fill({ version: '', page: built('release'), installer: built('installer'), portable: built('portable'), linux: built('linux') });
latest().then((l) => l && fill(l));
