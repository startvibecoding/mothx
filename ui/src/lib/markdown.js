export function markdownToHTML(value) {
  if (!value) return '';
  const lines = value.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
  const html = [];
  let paragraph = [];
  let list = null;
  let quote = [];
  let code = null;

  function closeParagraph() {
    if (paragraph.length === 0) return;
    html.push(`<p>${renderInline(paragraph.join('\n')).replace(/\n/g, '<br>')}</p>`);
    paragraph = [];
  }

  function closeList() {
    if (!list) return;
    const items = list.items.map((item) => `<li>${renderInline(item)}</li>`).join('');
    html.push(`<${list.type}>${items}</${list.type}>`);
    list = null;
  }

  function closeQuote() {
    if (quote.length === 0) return;
    html.push(`<blockquote>${quote.map((line) => `<p>${renderInline(line)}</p>`).join('')}</blockquote>`);
    quote = [];
  }

  function closeCode() {
    if (!code) return;
    const lang = code.lang ? ` class="language-${escapeAttr(code.lang)}"` : '';
    html.push(`<pre><code${lang}>${escapeHTML(code.lines.join('\n'))}</code></pre>`);
    code = null;
  }

  function closeBlocks() {
    closeParagraph();
    closeList();
    closeQuote();
  }

  for (const line of lines) {
    const fence = line.match(/^```([A-Za-z0-9_-]+)?\s*$/);
    if (fence) {
      if (code) {
        closeCode();
      } else {
        closeBlocks();
        code = { lang: fence[1] || '', lines: [] };
      }
      continue;
    }

    if (code) {
      code.lines.push(line);
      continue;
    }

    if (!line.trim()) {
      closeBlocks();
      continue;
    }

    const heading = line.match(/^(#{1,4})\s+(.+)$/);
    if (heading) {
      closeBlocks();
      const level = heading[1].length + 1;
      html.push(`<h${level}>${renderInline(heading[2].trim())}</h${level}>`);
      continue;
    }

    const unordered = line.match(/^\s*[-*]\s+(.+)$/);
    const ordered = line.match(/^\s*\d+\.\s+(.+)$/);
    if (unordered || ordered) {
      closeParagraph();
      closeQuote();
      const type = ordered ? 'ol' : 'ul';
      if (!list || list.type !== type) {
        closeList();
        list = { type, items: [] };
      }
      list.items.push((unordered || ordered)[1]);
      continue;
    }

    const quoted = line.match(/^\s*>\s?(.*)$/);
    if (quoted) {
      closeParagraph();
      closeList();
      quote.push(quoted[1]);
      continue;
    }

    closeList();
    closeQuote();
    paragraph.push(line);
  }

  closeCode();
  closeBlocks();
  return html.join('');
}

function renderInline(value) {
  const parts = value.split(/(`[^`]*`)/g);
  return parts
    .map((part) => {
      if (part.startsWith('`') && part.endsWith('`')) {
        return `<code>${escapeHTML(part.slice(1, -1))}</code>`;
      }
      return renderInlineText(part);
    })
    .join('');
}

function renderInlineText(value) {
  const linkPattern = /\[([^\]]+)\]\(([^)\s]+)\)/g;
  let out = '';
  let last = 0;
  let match;
  while ((match = linkPattern.exec(value))) {
    out += renderEmphasis(escapeHTML(value.slice(last, match.index)));
    const text = renderEmphasis(escapeHTML(match[1]));
    const href = safeURL(match[2]);
    if (href) {
      out += `<a href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${text}</a>`;
    } else {
      out += escapeHTML(match[0]);
    }
    last = match.index + match[0].length;
  }
  out += renderEmphasis(escapeHTML(value.slice(last)));
  return out;
}

function renderEmphasis(value) {
  return value.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>').replace(/\*([^*]+)\*/g, '<em>$1</em>');
}

function safeURL(value) {
  if (!value) return '';
  try {
    const base = typeof window === 'undefined' ? 'http://localhost' : window.location.origin;
    const url = new URL(value, base);
    if (['http:', 'https:', 'mailto:'].includes(url.protocol)) {
      return value;
    }
  } catch {
    return '';
  }
  return '';
}

function escapeHTML(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function escapeAttr(value) {
  return escapeHTML(value).replace(/`/g, '&#96;');
}
