// The newest release on GitHub, read once when the site is built, so the links work without JavaScript.
// The page asks GitHub again in the browser (scripts/download.ts), so a new release shows up without a rebuild.
export const REPO = 'metalpietie/typewriter';
export const REPO_URL = `https://github.com/${REPO}`;
export const RELEASES_URL = `${REPO_URL}/releases/latest`;

export type Release = { version: string; title: string; page: string; installer: string; portable: string; linux: string };

let cached: Promise<Release> | undefined;

export function latestRelease(): Promise<Release> {
  cached ??= load();
  return cached;
}

async function load(): Promise<Release> {
  const fallback: Release = { version: '', title: '', page: RELEASES_URL, installer: RELEASES_URL, portable: RELEASES_URL, linux: RELEASES_URL };
  try {
    const token = process.env.GITHUB_TOKEN;
    const r = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`, {
      headers: { Accept: 'application/vnd.github+json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    });
    if (!r.ok) throw new Error('GitHub ' + r.status);
    const j = await r.json();
    const find = (end: string): string => j.assets.find((a: { name: string }) => a.name.endsWith(end))?.browser_download_url ?? j.html_url;
    return {
      version: String(j.tag_name).replace(/^v/, ''),
      title: String(j.name ?? '').replace(/^Version [^:]+:\s*/, ''),
      page: j.html_url,
      installer: find('-windows-x64-installer.exe'),
      portable: find('-windows-x64.exe'),
      linux: find('-linux-x64'),
    };
  } catch (e) {
    console.warn('[release] could not read the newest release, the links go to the release page:', (e as Error).message);
    return fallback;
  }
}
