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
    const language = normalizeLanguage(code.lang);
    const source = code.lines.join('\n');
    const collapsed = code.lines.length > 18 ? '' : ' open';
    html.push(
      `<details class="code-block"${collapsed}>` +
        `<summary><span class="code-block-language">${escapeHTML(language.label)}</span>` +
        `<span class="code-block-actions"><span class="code-block-toggle" data-expand="Expand" data-collapse="Collapse">${collapsed ? 'Collapse' : 'Expand'}</span>` +
        `<button type="button" class="code-copy" aria-label="Copy code">Copy</button></span></summary>` +
        `<pre><code class="language-${escapeAttr(language.id)}">${highlightCode(source, language.id)}</code></pre>` +
      `</details>`
    );
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

export function highlightedCodeToHTML(value, path = '') {
  return highlightCode(String(value || ''), languageFromPath(path));
}

function languageFromPath(path) {
  const match = String(path).toLowerCase().match(/\.([a-z0-9]+)$/);
  const extensions = {
    go: 'go', js: 'javascript', jsx: 'javascript', mjs: 'javascript', cjs: 'javascript',
    ts: 'typescript', tsx: 'typescript', py: 'python', rb: 'ruby', sh: 'bash', bash: 'bash',
    zsh: 'bash', json: 'json', yaml: 'yaml', yml: 'yaml', sql: 'sql',
    html: 'markup', htm: 'markup', xml: 'markup', svg: 'markup', md: 'markdown'
  };
  return extensions[match?.[1]] || 'plaintext';
}

function normalizeLanguage(value) {
  const id = String(value || '').trim().toLowerCase();
  const aliases = {
    js: 'javascript', jsx: 'javascript', ts: 'typescript', tsx: 'typescript',
    py: 'python', rb: 'ruby', sh: 'bash', shell: 'bash', yml: 'yaml',
    html: 'markup', xml: 'markup', svg: 'markup', md: 'markdown', text: 'plaintext'
  };
  const normalized = aliases[id] || id || 'plaintext';
  return { id: normalized.replace(/[^a-z0-9_-]/g, '') || 'plaintext', label: normalized || 'plaintext' };
}

function highlightCode(value, language) {
  const source = escapeHTML(value);
  if (language === 'plaintext') return source;

  const keywords = {
    javascript: /\b(?:await|async|break|case|catch|class|const|continue|default|delete|do|else|export|extends|false|finally|for|from|function|if|import|in|instanceof|let|new|null|of|return|static|super|switch|this|throw|true|try|typeof|undefined|var|void|while|with|yield)\b/g,
    typescript: /\b(?:abstract|any|as|async|await|boolean|break|case|catch|class|const|continue|declare|default|delete|do|else|enum|export|extends|false|finally|for|from|function|if|implements|import|in|interface|keyof|let|namespace|new|null|number|of|private|protected|public|readonly|return|static|string|super|switch|this|throw|true|try|type|typeof|undefined|var|void|while|yield)\b/g,
    go: /\b(?:break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b/g,
    python: /\b(?:and|as|assert|async|await|break|class|continue|def|del|elif|else|except|False|finally|for|from|global|if|import|in|is|lambda|None|nonlocal|not|or|pass|raise|return|True|try|while|with|yield)\b/g,
    bash: /\b(?:case|do|done|echo|elif|else|esac|export|fi|for|function|if|in|local|then|while)\b/g,
    json: /\b(?:true|false|null)\b/g,
    yaml: /\b(?:true|false|null|yes|no)\b/g,
    sql: /\b(?:SELECT|FROM|WHERE|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|JOIN|LEFT|RIGHT|INNER|ORDER|GROUP|BY|LIMIT|AS|AND|OR|NULL|VALUES|SET)\b/gi
  };
  const keyword = keywords[language];
  if (!keyword) return source;

  const token = /(?:\/\/[^\n]*|\/\*[\s\S]*?\*\/|#[^\n]*|&quot;(?:\\.|[^&]|&(?!quot;))*?&quot;|&#39;(?:\\.|[^&]|&(?!#39;))*?&#39;|`(?:\\.|[^`])*?`|\b\d+(?:\.\d+)?\b)/g;
  let result = '';
  let last = 0;
  for (const match of source.matchAll(token)) {
    result += highlightKeywords(source.slice(last, match.index), keyword);
    const text = match[0];
    const type = /^\d/.test(text) ? 'number' : /^#|^\/\//.test(text) || text.startsWith('/*') ? 'comment' : 'string';
    result += `<span class="tok-${type}">${text}</span>`;
    last = match.index + text.length;
  }
  return result + highlightKeywords(source.slice(last), keyword);
}

function highlightKeywords(value, pattern) {
  return value.replace(pattern, (match) => `<span class="tok-keyword">${match}</span>`);
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
  const codeTagPattern = /<code\b[^>]*>([\s\S]*?)<\/code>/gi;
  let out = '';
  let last = 0;
  let match;
  while ((match = codeTagPattern.exec(value))) {
    out += renderInlineTextWithoutCodeTags(value.slice(last, match.index));
    out += `<code>${escapeHTML(match[1])}</code>`;
    last = match.index + match[0].length;
  }
  return out + renderInlineTextWithoutCodeTags(value.slice(last));
}

function renderInlineTextWithoutCodeTags(value) {
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
