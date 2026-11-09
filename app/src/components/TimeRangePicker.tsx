import type { CSSProperties } from 'react';

import { RANGE_PRESETS, type TimeRange } from '@/lib/time';
import { mono, useTheme } from '@/theme';

import { Chip } from './ui';

export function TimeRangePicker({ value, onChange }: { value: TimeRange; onChange: (r: TimeRange) => void }) {
  const t = useTheme();
  const input: CSSProperties = {
    flex: 1,
    color: t.text,
    backgroundColor: t.surface,
    borderColor: t.border,
    borderWidth: 1,
    borderStyle: 'solid',
    borderRadius: 8,
    padding: '6px 8px',
    fontSize: 13,
    ...mono,
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div style={{ display: 'flex', overflowX: 'auto' }}>
        {RANGE_PRESETS.map((p) => (
          <Chip
            key={p.id}
            label={`Last ${p.label}`}
            selected={value.kind === 'preset' && value.id === p.id}
            onPress={() => onChange({ kind: 'preset', id: p.id })}
          />
        ))}
        <Chip
          label="Custom"
          selected={value.kind === 'custom'}
          onPress={() => {
            if (value.kind === 'custom') return;
            const end = new Date();
            onChange({ kind: 'custom', start: new Date(end.getTime() - 3_600_000).toISOString(), end: end.toISOString() });
          }}
        />
      </div>
      {value.kind === 'custom' && (
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <span style={{ color: t.textDim }}>From</span>
          <input
            aria-label="Start time"
            value={value.start}
            onChange={(e) => onChange({ ...value, start: e.target.value })}
            autoCapitalize="none"
            autoCorrect="off"
            style={input}
          />
          <span style={{ color: t.textDim }}>to</span>
          <input
            aria-label="End time"
            value={value.end}
            onChange={(e) => onChange({ ...value, end: e.target.value })}
            autoCapitalize="none"
            autoCorrect="off"
            style={input}
          />
        </div>
      )}
    </div>
  );
}
