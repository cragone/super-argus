import { useState } from 'react';

import { errorMessage, type SavedQuery } from '@/api/client';
import { PanelForm } from '@/components/PanelForm';
import { Button, Chip, ErrorText } from '@/components/ui';
import { useCanEdit, useDashboardMutations, useDashboards, useSavedQueries, type PanelType } from '@/hooks/useDashboards';
import { RANGE_PRESETS, type TimeRange } from '@/lib/time';
import { useTheme } from '@/theme';

/** A query is a metric query when it starts with a function name rather than a selector. */
const isMetric = (q: string) => /^\s*[a-z_]/i.test(q);

/**
 * Explore actions: load a saved query (everyone), save the current query or add it
 * to a dashboard as a panel (editors and admins).
 */
export function SaveBar({
  query,
  range,
  onLoad,
}: {
  query: string;
  range: TimeRange;
  onLoad: (q: SavedQuery) => void;
}) {
  const t = useTheme();
  const canEdit = useCanEdit();
  const saved = useSavedQueries();
  const dashboards = useDashboards();
  const m = useDashboardMutations();
  const [mode, setMode] = useState<'none' | 'save' | 'dashboard'>('none');
  const [name, setName] = useState('');
  const [target, setTarget] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const presetId = range.kind === 'preset' && RANGE_PRESETS.some((p) => p.id === range.id) ? range.id : '1h';

  const save = async () => {
    setError(null);
    if (!query.trim()) return setError('Write a query first');
    if (!name.trim()) return setError('Give the query a name');
    try {
      await m.saveQuery.mutateAsync({ name: name.trim(), query: query.trim(), default_range: presetId });
      setMessage(`Saved "${name.trim()}"`);
      setName('');
      setMode('none');
    } catch (e) {
      setError(errorMessage(e, 'Could not save the query'));
    }
  };

  const dashId = target ?? dashboards.data?.[0]?.id ?? null;
  const initialType: PanelType = isMetric(query) ? 'timeseries' : 'logs';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {!!saved.data?.length && (
        <div style={{ display: 'flex', overflowX: 'auto', alignItems: 'center' }} aria-label="Saved queries">
          <span style={{ color: t.textDim, fontSize: 13, marginRight: 6 }}>Saved</span>
          {saved.data.map((q) => (
            <Chip key={q.id} label={q.name} accessibilityLabel={`Load saved query ${q.name}`} onPress={() => onLoad(q)} />
          ))}
        </div>
      )}
      {canEdit && (
        <div style={{ display: 'flex', gap: 6 }}>
          <Chip label="Save query" selected={mode === 'save'} onPress={() => { setMode(mode === 'save' ? 'none' : 'save'); setMessage(null); }} />
          <Chip label="Add to dashboard" selected={mode === 'dashboard'} onPress={() => { setMode(mode === 'dashboard' ? 'none' : 'dashboard'); setMessage(null); }} />
        </div>
      )}
      {message && <span aria-label="Save status" style={{ color: t.levels.info, fontSize: 13 }}>{message}</span>}

      {mode === 'save' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <input
            aria-label="Saved query name"
            placeholder="Name, e.g. API errors"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && void save()}
            style={{ color: t.text, backgroundColor: t.surface, borderColor: t.border, borderWidth: 1, borderStyle: 'solid', borderRadius: 8, padding: '8px 10px', fontSize: 14 }}
          />
          {error && <ErrorText>{error}</ErrorText>}
          <Button title="Save" onPress={() => void save()} loading={m.saveQuery.isPending} />
        </div>
      )}

      {mode === 'dashboard' &&
        (dashboards.data?.length ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div style={{ display: 'flex', overflowX: 'auto' }}>
              {dashboards.data.map((d) => (
                <Chip key={d.id} label={d.title} selected={dashId === d.id} accessibilityLabel={`Dashboard ${d.title}`} onPress={() => setTarget(d.id)} />
              ))}
            </div>
            <PanelForm
              key={`${initialType}-${query}`}
              initial={{ query, type: initialType, title: '' }}
              submitLabel="Add panel"
              onCancel={() => setMode('none')}
              onSubmit={async (draft) => {
                if (!dashId) throw new Error('Pick a dashboard');
                await m.addPanel.mutateAsync({ dashboardId: dashId, draft });
                setMessage(`Added "${draft.title}" to ${dashboards.data?.find((d) => d.id === dashId)?.title}`);
                setMode('none');
              }}
            />
          </div>
        ) : (
          <span style={{ color: t.textDim, fontSize: 13 }}>Create a dashboard first (Dashboards tab).</span>
        ))}
    </div>
  );
}
