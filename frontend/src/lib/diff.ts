import { toHtml } from './db.svelte';

/** The text of a scene with a line break between paragraphs, ready for the word diff. */
export function blockText(body: string): string {
  const html = toHtml(body ?? '').replace(/<\/(p|h[1-6]|li|blockquote)>|<br\s*\/?>|<hr\s*\/?>/gi, '\n');
  return new DOMParser().parseFromString(html, 'text/html').body.textContent ?? '';
}
