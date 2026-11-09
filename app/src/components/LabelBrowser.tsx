import { useState } from 'react';

import { useLabelNames, useLabelValues } from '@/hooks/useLogs';
import { useTheme } from '@/theme';

import { Chip } from './ui';

/** Browse label names, then their values; picking a value adds it to the selector. */
export function LabelBrowser({ onPick }: { onPick: (name: string, value: string) => void }) {
  const t = useTheme();
  const [open, setOpen] = useState<string | null>(null);
  const names = useLabelNames();
  const values = useLabelValues(open);

  if (names.isLoading) return <span style={{ color: t.textDim }}>Loading labels…</span>;
  if (names.isError) return <span style={{ color: t.danger }}>Could not load labels.</span>;
  if (!names.data?.length) return <span style={{ color: t.textDim }}>No labels yet. Push some logs first.</span>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div style={{ display: 'flex', overflowX: 'auto' }}>
        {names.data.map((n) => (
          <Chip key={n} label={n} selected={open === n} onPress={() => setOpen(open === n ? null : n)} />
        ))}
      </div>
      {open && (
        <div style={{ display: 'flex', overflowX: 'auto' }} aria-label={`Values of ${open}`}>
          {values.isLoading && <span style={{ color: t.textDim }}>Loading…</span>}
          {values.data?.length === 0 && <span style={{ color: t.textDim }}>No values</span>}
          {values.data?.map((v) => (
            <Chip key={v} label={v} accessibilityLabel={`Add ${open}=${v}`} onPress={() => onPick(open, v)} />
          ))}
        </div>
      )}
    </div>
  );
}
