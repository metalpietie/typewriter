// Runs the app in a plain browser, with the demo backend instead of the Go one. For the tutorial screenshots only.
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const frontend = path.resolve(here, '../../frontend');
const need = createRequire(path.join(frontend, 'package.json'));
const { defineConfig } = need('vite');
const { svelte } = need('@sveltejs/vite-plugin-svelte');

export default defineConfig({
  root: frontend,
  server: { host: '127.0.0.1', port: 5199, strictPort: true, fs: { strict: false } },
  resolve: { alias: [{ find: /^.*bindings\/typewriter\/internal\/app$/, replacement: path.join(here, 'stub.ts') }] },
  plugins: [svelte()],
});
