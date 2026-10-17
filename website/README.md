# Typewriter website

The web page for Typewriter: what the app is, who it is for, and the download. Built with [Astro](https://astro.build).

## Run it

```bash
npm install
npm run dev
```

Open http://localhost:4321.

## Build it

```bash
npm run build
```

The finished site is in `dist/`. Put that folder on any static web host.

- The build asks GitHub for the newest release and puts its download links in the page. Without internet, the links go to the GitHub release page.
- In the browser, the page asks GitHub again. A new release shows up without a new build.
- On a host that is not the root of its domain (for example GitHub Pages), set the address when you build:

  ```bash
  SITE_URL=https://metalpietie.github.io BASE_PATH=/typewriter npm run build
  ```

## Put it online with Dokploy

The `Dockerfile` builds the site and serves it with nginx (`nginx.conf`: caching and security headers).

1. In Dokploy, make an **Application**.
2. **Provider:** GitHub, repository `metalpietie/typewriter`, branch `main`.
3. **Build type:** Dockerfile. **Build path:** `website`. **Dockerfile path:** `Dockerfile`.
4. **Watch paths:** `website/**`, so only changes to the site start a new build.
5. **Domains:** add your domain, set the **container port** to `80`, and switch on HTTPS (Let's Encrypt).
6. Optional, under **Build-time arguments**: `SITE_URL=https://your-domain`. It gives link previews a full address for the picture.
7. Press **Deploy**. With **Auto deploy** on, every push to `main` that changes `website/` builds the site again.

A new app release needs no new build: the page reads the newest release from GitHub in the browser.

To test the image on your computer:

```bash
docker build -t typewriter-website website
docker run --rm -p 8088:80 typewriter-website
```

Then open http://localhost:8088.

## Where things are

| Path | What it holds |
| --- | --- |
| `src/data/content.ts` | All texts: the books on the shelf, the cards, who it is for |
| `src/data/release.ts` | Reads the newest release from GitHub when the site is built |
| `src/scripts/download.ts` | Picks the download for the visitor's system (Windows: installer, Linux: the app) |
| `src/scripts/effects.ts` | Fade-in on scroll, tilting screenshots, glowing cards |
| `src/components/` | One file per part of the page |
| `src/assets/screens/` | The screenshots (copies of `docs/images`). Astro makes small `.webp` files from them. |
| `public/` | The icon and the picture for link previews (`og.png`) |
