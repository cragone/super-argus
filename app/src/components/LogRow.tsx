import { memo } from 'react';

import type { QueryEntry } from '@/api/client';
import { detectLevel, prettyJson } from '@/lib/logLevel';
import { formatDateTime, formatTime } from '@/lib/time';
import { mono, useTheme } from '@/theme';

export interface LogRowProps {
  entry: QueryEntry;
  labels?: Record<string, string>;
  expanded: boolean;
  onToggle: (key: string) => void;
  rowKey: string;
}

/** One log line: time, level color bar, truncated text. Tap to expand (pretty JSON, fields, stream labels). */
export const LogRow = memo(function LogRow({ entry, labels, expanded, onToggle, rowKey }: LogRowProps) {
  const t = useTheme();
  const level = detectLevel(entry.line, entry.fields);
  const color = level ? t.levels[level] : t.border;
  const json = expanded ? prettyJson(entry.line) : null;

  return (
    <button
      type="button"
      aria-label={`Log line at ${formatTime(entry.ts)}`}
      aria-expanded={expanded}
      onClick={() => onToggle(rowKey)}
      style={{
        display: 'flex',
        width: '100%',
        textAlign: 'left',
        borderWidth: 0,
        borderBottom: `1px solid ${t.border}`,
        backgroundColor: t.surface,
        cursor: 'pointer',
        padding: 0,
      }}>
      <span style={{ width: 4, alignSelf: 'stretch', backgroundColor: color, flexShrink: 0 }} />
      <span style={{ flex: 1, padding: '6px 10px', display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <span style={{ ...mono, fontSize: 11, color: t.textDim }}>{expanded ? formatDateTime(entry.ts) : formatTime(entry.ts)}</span>
        <span
          style={{
            ...mono,
            fontSize: 12.5,
            color: t.text,
            userSelect: expanded ? 'text' : 'none',
            whiteSpace: expanded ? 'pre-wrap' : 'nowrap',
            overflow: expanded ? 'visible' : 'hidden',
            textOverflow: expanded ? 'clip' : 'ellipsis',
            display: expanded ? 'block' : '-webkit-box',
            WebkitLineClamp: expanded ? undefined : 2,
            WebkitBoxOrient: expanded ? undefined : 'vertical',
          }}>
          {json ?? entry.line}
        </span>
        {expanded && labels && (
          <span style={{ ...mono, fontSize: 11, color: t.textDim, marginTop: 6 }}>
            {Object.entries(labels)
              .map(([k, v]) => `${k}="${v}"`)
              .join('  ')}
          </span>
        )}
        {expanded && entry.fields && Object.keys(entry.fields).length > 0 && (
          <span style={{ marginTop: 6, display: 'flex', flexDirection: 'column' }}>
            {Object.entries(entry.fields).map(([k, v]) => (
              <span key={k} style={{ ...mono, fontSize: 11, color: t.textDim }}>
                <span style={{ color: t.accent }}>{k}</span>={v}
              </span>
            ))}
          </span>
        )}
      </span>
    </button>
  );
});
