import { useState, type CSSProperties } from 'react';

import { errorMessage, type AlertRule, type AlertRuleInput, type Channel } from '@/api/client';
import { COMPARATORS, SEVERITIES, validDuration } from '@/lib/alerts';
import { mono, useTheme } from '@/theme';

import { QueryInput } from './QueryInput';
import { Button, Chip, ErrorText } from './ui';

const EXAMPLE = 'sum(count_over_time({service="api"} |= "error" [5m]))';

/** Create or edit an alert rule. The server re-validates everything. */
export function AlertRuleForm({
  initial,
  channels,
  submitLabel,
  onSubmit,
  onCancel,
}: {
  initial?: AlertRule;
  channels: Channel[];
  submitLabel: string;
  onSubmit: (body: AlertRuleInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const t = useTheme();
  const [name, setName] = useState(initial?.name ?? '');
  const [query, setQuery] = useState(initial?.query ?? '');
  const [comparator, setComparator] = useState<AlertRule['comparator']>(initial?.comparator ?? 'gt');
  const [threshold, setThreshold] = useState(String(initial?.threshold ?? 0));
  const [windowText, setWindowText] = useState(initial?.window ?? '5m');
  const [forText, setForText] = useState(initial?.for_duration ?? '0s');
  const [cooldown, setCooldown] = useState(initial?.cooldown ?? '0s');
  const [severity, setSeverity] = useState<AlertRule['severity']>(initial?.severity ?? 'warning');
  const [enabled, setEnabled] = useState(initial?.enabled ?? true);
  const [picked, setPicked] = useState<string[]>(initial?.notify_channel_ids ?? []);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const save = async () => {
    const th = Number(threshold);
    if (!name.trim()) return setError('Give the rule a name');
    if (!query.trim()) return setError('Enter a metric query');
    if (threshold.trim() === '' || !Number.isFinite(th)) return setError('Threshold must be a number');
    if (!validDuration(windowText, false)) return setError('Window must be a duration such as 5m');
    if (!validDuration(forText, true)) return setError('"For" must be a duration such as 2m, or 0s');
    if (!validDuration(cooldown, true)) return setError('Cooldown must be a duration such as 30m, or 0s');
    setBusy(true);
    setError(null);
    try {
      await onSubmit({
        name: name.trim(),
        query: query.trim(),
        comparator,
        threshold: th,
        window: windowText.trim(),
        for_duration: forText.trim() || '0s',
        cooldown: cooldown.trim() || '0s',
        severity,
        enabled,
        notify_channel_ids: picked,
      });
    } catch (e) {
      setError(errorMessage(e, 'Could not save the rule'));
    } finally {
      setBusy(false);
    }
  };

  const field: CSSProperties = {
    color: t.text, backgroundColor: t.surface, borderColor: t.border, borderWidth: 1, borderStyle: 'solid',
    borderRadius: 8, padding: '8px 10px', fontSize: 14,
  };
  const label: CSSProperties = { color: t.textDim, fontSize: 12 };
  const toggle = (id: string) => setPicked((p) => (p.includes(id) ? p.filter((x) => x !== id) : [...p, id]));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10, padding: 12, backgroundColor: t.surfaceAlt, borderRadius: 10, borderWidth: 1, borderStyle: 'solid', borderColor: t.border }}>
      <input aria-label="Rule name" placeholder="Rule name" value={name} onChange={(e) => setName(e.target.value)} style={{ ...field, width: '100%' }} />
      <QueryInput value={query} onChange={setQuery} onSubmit={() => void save()} />
      {!query && (
        <span style={{ ...mono, color: t.textDim, fontSize: 11 }} aria-label="Example query">
          e.g. {EXAMPLE}
        </span>
      )}
      <span style={label}>Fire when the query value is</span>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
        {COMPARATORS.map((c) => (
          <Chip key={c.id} label={c.symbol} accessibilityLabel={`Comparator ${c.id}`} selected={comparator === c.id} onPress={() => setComparator(c.id)} />
        ))}
        <input
          aria-label="Threshold"
          placeholder="Threshold"
          value={threshold}
          onChange={(e) => setThreshold(e.target.value)}
          inputMode="decimal"
          style={{ ...field, minWidth: 100 }}
        />
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        {[
          { l: 'Window', v: windowText, set: setWindowText },
          { l: 'For', v: forText, set: setForText },
          { l: 'Cooldown', v: cooldown, set: setCooldown },
        ].map((f) => (
          <div key={f.l} style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={label}>{f.l}</span>
            <input aria-label={f.l} value={f.v} onChange={(e) => f.set(e.target.value)} autoCapitalize="none" style={{ ...field, width: '100%' }} />
          </div>
        ))}
      </div>
      <span style={label}>Severity</span>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
        {SEVERITIES.map((s) => (
          <Chip key={s} label={s} accessibilityLabel={`Severity ${s}`} selected={severity === s} onPress={() => setSeverity(s)} />
        ))}
      </div>
      <span style={label}>Notify</span>
      {channels.length === 0 ? (
        <span style={{ color: t.textDim, fontSize: 12 }}>No channels yet. An admin can add one in Settings.</span>
      ) : (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {channels.map((c) => (
            <Chip key={c.id} label={c.name} accessibilityLabel={`Channel ${c.name}`} selected={picked.includes(c.id)} onPress={() => toggle(c.id)} />
          ))}
        </div>
      )}
      <Chip label={enabled ? 'Enabled' : 'Disabled'} accessibilityLabel="Enabled" selected={enabled} onPress={() => setEnabled((e) => !e)} />
      {error && <ErrorText>{error}</ErrorText>}
      <div style={{ display: 'flex', gap: 8 }}>
        <Button title={submitLabel} onPress={() => void save()} loading={busy} style={{ flex: 1 }} />
        <Button title="Cancel" variant="ghost" onPress={onCancel} />
      </div>
    </div>
  );
}
