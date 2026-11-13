/** The five canonical levels. Mirrors internal/level on the server, which stores one in the `level` stream label. */
export type LogLevel = 'fatal' | 'error' | 'warning' | 'info' | 'unknown';

export const LEVELS: readonly LogLevel[] = ['fatal', 'error', 'warning', 'info', 'unknown'];

export const LEVEL_LABEL: Record<LogLevel, string> = {
  fatal: 'Fatal',
  error: 'Error',
  warning: 'Warning',
  info: 'Info',
  unknown: 'Unknown',
};

const ALIASES: Record<string, LogLevel> = {
  fatal: 'fatal', panic: 'fatal', emerg: 'fatal', emergency: 'fatal', alert: 'fatal', crit: 'fatal', critical: 'fatal', severe: 'fatal',
  error: 'error', err: 'error', errors: 'error',
  warn: 'warning', warning: 'warning',
  info: 'info', information: 'info', notice: 'info',
};

const TOKEN = /\b(fatal|panic|emerg|emergency|alert|crit|critical|severe|errors?|err|warn(?:ing)?|info(?:rmation)?|notice)\b/i;

/** Level from a parsed level/severity field, else from the first level-looking token in the line; null if none. */
export function detectLevel(line: string, fields?: Record<string, string>): LogLevel | null {
  const f = fields?.level ?? fields?.severity ?? fields?.lvl;
  if (f && ALIASES[f.toLowerCase()]) return ALIASES[f.toLowerCase()];
  const m = TOKEN.exec(line.slice(0, 200));
  return m ? ALIASES[m[1].toLowerCase()] : null;
}

/** Canonical level of a log row: the server-assigned `level` stream label, else a client-side guess, else unknown. */
export function levelOf(line: string, fields?: Record<string, string>, labels?: Record<string, string>): LogLevel {
  const l = labels?.level?.toLowerCase();
  if (l && (LEVELS as readonly string[]).includes(l)) return l as LogLevel;
  return detectLevel(line, fields) ?? 'unknown';
}

/** Pretty-printed JSON for lines that are JSON objects or arrays, else null. */
export function prettyJson(line: string): string | null {
  const t = line.trim();
  if (!(t.startsWith('{') || t.startsWith('['))) return null;
  try {
    return JSON.stringify(JSON.parse(t), null, 2);
  } catch {
    return null;
  }
}
