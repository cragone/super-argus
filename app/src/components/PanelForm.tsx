import { useState, type CSSProperties } from 'react';

import { errorMessage } from '@/api/client';
import type { PanelDraft, PanelType } from '@/hooks/useDashboards';
import { mono, useTheme } from '@/theme';

import { QueryInput } from './QueryInput';
import { Button, Chip, ErrorText } from './ui';

const TYPES: { id: PanelType; label: string; hint: string }[] = [
  { id: 'timeseries', label: 'Time series', hint: 'A line per series over time (metric query)' },
  { id: 'top_n', label: 'Top N', hint: 'Ranked bars of the newest values (metric query, e.g. topk)' },
  { id: 'stat', label: 'Stat', hint: 'One headline number (metric query)' },
  { id: 'logs', label: 'Logs', hint: 'The newest matching lines (log query)' },
];

const EXAMPLES: Record<PanelType, string> = {
  timeseries: 'sum by (level) (count_over_time({service="api"} | json [1m]))',
  top_n: 'topk(10, sum by (route) (count_over_time({service="api"} | json [5m])))',
  stat: 'sum(count_over_time({service="api"} |= "error" [1h]))',
  logs: '{service="api"} |= "error"',
};

/** Create or edit a panel. The server validates the query for the chosen type. */
export function PanelForm({
  initial,
  submitLabel,
  onSubmit,
  onCancel,
}: {
  initial?: Partial<PanelDraft>;
  submitLabel: string;
  onSubmit: (d: PanelDraft) => Promise<unknown>;
  onCancel: () => void;
}) {
  const t = useTheme();
  const [title, setTitle] = useState(initial?.title ?? '');
  const [type, setType] = useState<PanelType>(initial?.type ?? 'timeseries');
  const [query, setQuery] = useState(initial?.query ?? '');
  const [step, setStep] = useState(initial?.step ?? '');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!title.trim()) return setError('Give the panel a title');
    if (!query.trim()) return setError('Enter a query');
    setBusy(true);
    setError(null);
    try {
      await onSubmit({ title: title.trim(), type, query: query.trim(), step: type === 'logs' ? '' : step.trim(), options: initial?.options ?? {}, position: initial?.position });
    } catch (e) {
      setError(errorMessage(e, 'Could not save the panel'));
    } finally {
      setBusy(false);
    }
  };

  const field: CSSProperties = {
    color: t.text,
    backgroundColor: t.surface,
    borderColor: t.border,
    borderWidth: 1,
    borderStyle: 'solid',
    borderRadius: 8,
    padding: '8px 10px',
    fontSize: 14,
    width: '100%',
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: 12, backgroundColor: t.surfaceAlt, borderRadius: 10, borderWidth: 1, borderStyle: 'solid', borderColor: t.border }}>
      <input aria-label="Panel title" placeholder="Panel title" value={title} onChange={(e) => setTitle(e.target.value)} style={field} />
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
        {TYPES.map((x) => (
          <Chip key={x.id} label={x.label} selected={type === x.id} onPress={() => setType(x.id)} />
        ))}
      </div>
      <span style={{ color: t.textDim, fontSize: 12 }}>{TYPES.find((x) => x.id === type)?.hint}</span>
      <QueryInput value={query} onChange={setQuery} onSubmit={() => void save()} />
      {!query && (
        <span style={{ ...mono, color: t.textDim, fontSize: 11 }} aria-label="Example query">
          e.g. {EXAMPLES[type]}
        </span>
      )}
      {type !== 'logs' && (
        <input
          aria-label="Step"
          placeholder="Step, e.g. 30s (empty = automatic)"
          value={step}
          onChange={(e) => setStep(e.target.value)}
          autoCapitalize="none"
          style={field}
        />
      )}
      {error && <ErrorText>{error}</ErrorText>}
      <div style={{ display: 'flex', gap: 8 }}>
        <Button title={submitLabel} onPress={() => void save()} loading={busy} style={{ flex: 1 }} />
        <Button title="Cancel" variant="ghost" onPress={onCancel} />
      </div>
    </div>
  );
}
