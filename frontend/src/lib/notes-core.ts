// The release text of a new version (Markdown, from GitHub) as simple HTML for the update window. No imports, so it can be tested alone.
// Only a small part of Markdown is needed: headings, lists, paragraphs, **bold**, *italic* and `code`.
// Every character of the text is escaped first, so the text can never add HTML of its own.

const esc = (t: string) => t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

function inline(t: string): string {
  return esc(t)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/(^|[^*\w])\*([^*\s][^*]*?)\*(?=[^*\w]|$)/g, '$1<em>$2</em>');
}

export function notesHtml(md: string): string {
  const out: string[] = [];
  let list: 'ul' | 'ol' | null = null;
  let para: string[] = [];
  const closePara = () => { if (para.length) { out.push('<p>' + para.map(inline).join(' ') + '</p>'); para = []; } };
  const closeList = () => { if (list) { out.push(`</${list}>`); list = null; } };
  for (const raw of md.replace(/\r\n/g, '\n').split('\n')) {
    const line = raw.trim();
    if (line.startsWith('```')) continue; // a text pasted inside a code block shows as ordinary text
    if (!line) { closePara(); closeList(); continue; }
    const h = /^(#{1,6})\s+(.*)$/.exec(line);
    if (h) {
      closePara(); closeList();
      const level = Math.min(6, h[1].length + 2); // ## becomes a small heading in the window
      out.push(`<h${level}>${inline(h[2])}</h${level}>`);
      continue;
    }
    const li = /^(?:[-*]|(\d+)\.)\s+(.*)$/.exec(line);
    if (li) {
      closePara();
      const kind = li[1] ? 'ol' : 'ul';
      if (list !== kind) { closeList(); out.push(`<${kind}>`); list = kind; }
      out.push(`<li>${inline(li[2])}</li>`);
      continue;
    }
    if (list && /^\s{2,}\S/.test(raw)) { // a line that goes on with the list item above
      out[out.length - 1] = out[out.length - 1].replace(/<\/li>$/, ' ' + inline(line) + '</li>');
      continue;
    }
    closeList();
    para.push(line);
  }
  closePara(); closeList();
  return out.join('');
}
