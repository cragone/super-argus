import { useMemo, useState, type KeyboardEvent } from 'react';

import { applySuggestion, filterSuggestions, getContext } from '@/lib/autocomplete';
import { useLabelNames, useLabelValues } from '@/hooks/useLogs';
import { mono, useTheme } from '@/theme';

import { Chip } from './ui';

/** ArgusQL input with label and label-value autocomplete inside the stream selector. */
export function QueryInput({
  value,
  onChange,
  onSubmit,
}: {
  value: string;
  onChange: (v: string) => void;
  onSubmit: () => void;
}) {
  const t = useTheme();
  const [cursor, setCursor] = useState(value.length);

  const ctx = useMemo(() => getContext(value, cursor), [value, cursor]);
  const names = useLabelNames();
  const values = useLabelValues(ctx?.kind === 'value' ? ctx.label : null);

  const suggestions = useMemo(() => {
    if (!ctx) return [];
    const pool = ctx.kind === 'label' ? names.data : values.data;
    return filterSuggestions(pool ?? [], ctx.prefix);
  }, [ctx, names.data, values.data]);

  const pick = (s: string) => {
    if (!ctx) return;
    const r = applySuggestion(value, cursor, ctx, s);
    onChange(r.text);
    setCursor(r.cursor);
  };

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onSubmit();
    }
  };

  return (
    <div>
      <textarea
        aria-label="Query"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onSelect={(e) => setCursor(e.currentTarget.selectionEnd)}
        onKeyDown={onKeyDown}
        placeholder={'{service="api"} |= "error"'}
        autoCapitalize="none"
        autoCorrect="off"
        spellCheck={false}
        style={{
          ...mono,
          display: 'block',
          width: '100%',
          minHeight: 44,
          maxHeight: 120,
          resize: 'vertical',
          color: t.text,
          backgroundColor: t.surface,
          borderColor: t.border,
          borderWidth: 1,
          borderStyle: 'solid',
          borderRadius: 8,
          padding: '8px 10px',
          fontSize: 14,
        }}
      />
      {suggestions.length > 0 && (
        <div style={{ display: 'flex', overflowX: 'auto', marginTop: 6 }}>
          {suggestions.map((s) => (
            <Chip key={s} label={s} accessibilityLabel={`Insert ${s}`} onPress={() => pick(s)} />
          ))}
        </div>
      )}
    </div>
  );
}
