const MATCHER = /([A-Za-z_][A-Za-z0-9_]*)\s*(=~|!~|!=|=)\s*"((?:[^"\\]|\\.)*)"/g;

const quote = (v: string) => `"${v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`;

/**
 * Adds name="value" to the stream selector of the query, replacing an existing
 * equality matcher for the same label. Creates a selector if there is none.
 */
export function addMatcher(query: string, name: string, value: string): string {
  const matcher = `${name}=${quote(value)}`;
  const open = query.indexOf('{');
  const close = open >= 0 ? query.indexOf('}', open) : -1;
  if (open < 0 || close < 0) {
    const rest = query.trim();
    return `{${matcher}}${rest ? ' ' + rest : ''}`;
  }
  const inner = query.slice(open + 1, close);
  let replaced = false;
  const next = inner.replace(MATCHER, (m, n: string, op: string) => {
    if (n === name && op === '=' && !replaced) {
      replaced = true;
      return matcher;
    }
    return m;
  });
  const body = replaced ? next : inner.trim() === '' ? matcher : `${inner.replace(/[,\s]+$/, '')}, ${matcher}`;
  return query.slice(0, open + 1) + body + query.slice(close);
}

const LEVEL_MATCHER = /^level\s*(=~|=)\s*"((?:[^"\\]|\\.)*)"$/;

/** Splits a selector body on commas that are outside quotes. */
function splitMatchers(inner: string): string[] {
  return inner
    .split(/,(?=(?:[^"]*"[^"]*")*[^"]*$)/)
    .map((p) => p.trim())
    .filter(Boolean);
}

/** Levels selected by a `level="x"` / `level=~"a|b"` matcher in the stream selector, or [] if none. */
export function getLevelMatcher(query: string): string[] {
  const open = query.indexOf('{');
  const close = open >= 0 ? query.indexOf('}', open) : -1;
  if (open < 0 || close < 0) return [];
  for (const part of splitMatchers(query.slice(open + 1, close))) {
    const m = LEVEL_MATCHER.exec(part);
    if (m) return m[2].split('|').filter(Boolean);
  }
  return [];
}

/**
 * Sets the level filter in the stream selector: nothing removes it, one level
 * writes level="x", several write level=~"a|b". Selecting every level in `all`
 * is the same as no filter.
 */
export function setLevelMatcher(query: string, levels: string[], all: readonly string[] = []): string {
  const wanted = all.length > 0 && levels.length === all.length ? [] : levels;
  const matcher = wanted.length === 0 ? '' : wanted.length === 1 ? `level=${quote(wanted[0])}` : `level=~${quote(wanted.join('|'))}`;
  const open = query.indexOf('{');
  const close = open >= 0 ? query.indexOf('}', open) : -1;
  if (open < 0 || close < 0) {
    if (!matcher) return query;
    const rest = query.trim();
    return `{${matcher}}${rest ? ' ' + rest : ''}`;
  }
  const parts = splitMatchers(query.slice(open + 1, close)).filter((p) => !LEVEL_MATCHER.test(p));
  if (matcher) parts.push(matcher);
  return query.slice(0, open + 1) + parts.join(', ') + query.slice(close);
}
