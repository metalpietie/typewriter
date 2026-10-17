// Astro settings for the Typewriter website.
// SITE_URL and BASE_PATH are only needed when the site is not at the root of its domain
// (for example GitHub Pages: SITE_URL=https://metalpietie.github.io BASE_PATH=/typewriter).
import { defineConfig } from 'astro/config';

export default defineConfig({
  site: process.env.SITE_URL || undefined,
  base: process.env.BASE_PATH || '/',
});
