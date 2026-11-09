import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { errorMessage, type Panel } from '@/api/client';
import { PanelCard, panelHeight, panelWidth, type PanelHeight } from '@/components/PanelCard';
import { PanelForm } from '@/components/PanelForm';
import { TimeRangePicker } from '@/components/TimeRangePicker';
import { Button, Chip, ErrorText } from '@/components/ui';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { REFRESH_OPTIONS, panelDraft, useCanEdit, useDashboard, useDashboardMutations, type PanelDraft } from '@/hooks/useDashboards';
import type { TimeRange } from '@/lib/time';
import { useTheme } from '@/theme';

const NEXT_HEIGHT: Record<PanelHeight, PanelHeight> = { s: 'm', m: 'l', l: 's' };

export function DashboardDetail() {
  const t = useTheme();
  const { id = '' } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const dash = useDashboard(id);
  const canEdit = useCanEdit();
  const m = useDashboardMutations(id);
  const qc = useQueryClient();
  const wide = useMediaQuery('(min-width: 900px)');

  const [range, setRange] = useState<TimeRange>({ kind: 'preset', id: '1h' });
  const [refreshId, setRefreshId] = useState<(typeof REFRESH_OPTIONS)[number]['id']>('off');
  const [edit, setEdit] = useState(false);
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [title, setTitle] = useState<string | null>(null);

  useDocumentTitle(dash.data?.title ?? 'Dashboard');

  const refreshMs = REFRESH_OPTIONS.find((r) => r.id === refreshId)?.ms ?? 0;
  const panels = dash.data?.panels ?? [];

  const run = async (fn: () => Promise<unknown>) => {
    setError(null);
    try {
      await fn();
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const put = (p: Panel, patch: Partial<PanelDraft>) =>
    m.updatePanel.mutateAsync({ dashboardId: id, panelId: p.id, draft: { ...panelDraft(p), ...patch } });

  const move = (i: number, delta: number) =>
    run(async () => {
      const order = [...panels];
      const j = i + delta;
      if (j < 0 || j >= order.length) return;
      [order[i], order[j]] = [order[j], order[i]];
      await Promise.all(order.map((p, pos) => (p.position === pos ? null : put(p, { position: pos }))));
    });

  const setOpt = (p: Panel, opt: Record<string, unknown>) => run(() => put(p, { options: { ...(p.options as object), ...opt } }));

  if (dash.isLoading) return <span style={{ color: t.textDim, padding: 16, display: 'block' }}>Loading…</span>;
  if (dash.isError || !dash.data) return <div style={{ padding: 16 }}><ErrorText>{errorMessage(dash.error, 'Dashboard not found')}</ErrorText></div>;

  return (
    <div style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* One row of filters above the content they scope. */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, padding: 12, backgroundColor: t.surface, borderRadius: 10, borderWidth: 1, borderStyle: 'solid', borderColor: t.border }}>
        <TimeRangePicker value={range} onChange={setRange} />
        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 6 }}>
          <span style={{ color: t.textDim, fontSize: 13, marginRight: 4 }}>Auto-refresh</span>
          {REFRESH_OPTIONS.map((r) => (
            <Chip key={r.id} label={r.label} selected={refreshId === r.id} accessibilityLabel={`Refresh every ${r.label}`} onPress={() => setRefreshId(r.id)} />
          ))}
          <Chip label="Refresh now" onPress={() => void qc.invalidateQueries({ queryKey: ['panel'] })} />
          {canEdit && <Chip label={edit ? 'Done editing' : 'Edit'} selected={edit} onPress={() => { setEdit((v) => !v); setAdding(false); setEditing(null); }} />}
        </div>
      </div>

      {error && <ErrorText>{error}</ErrorText>}

      {edit && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, padding: 12, backgroundColor: t.surface, borderRadius: 10, borderWidth: 1, borderStyle: 'solid', borderColor: t.border }}>
          <input
            aria-label="Dashboard title"
            value={title ?? dash.data.title}
            onChange={(e) => setTitle(e.target.value)}
            onBlur={() => {
              if (title !== null && title.trim() && title.trim() !== dash.data?.title) void run(() => m.renameDashboard.mutateAsync({ id, title: title.trim(), description: dash.data?.description }));
            }}
            style={{ color: t.text, borderColor: t.border, borderWidth: 1, borderStyle: 'solid', borderRadius: 8, padding: '8px 10px', fontSize: 16, fontWeight: 600 }}
          />
          <div style={{ display: 'flex', gap: 8 }}>
            <Button title="Add panel" onPress={() => { setAdding(true); setEditing(null); }} style={{ flex: 1 }} />
            {confirmDelete === 'dashboard' ? (
              <Button title="Confirm delete" variant="ghost" onPress={() => void run(async () => { await m.deleteDashboard.mutateAsync(id); navigate('/dashboards'); })} />
            ) : (
              <Button title="Delete dashboard" variant="ghost" onPress={() => setConfirmDelete('dashboard')} />
            )}
          </div>
        </div>
      )}

      {adding && (
        <PanelForm
          submitLabel="Add panel"
          onCancel={() => setAdding(false)}
          onSubmit={async (draft) => {
            await m.addPanel.mutateAsync({ dashboardId: id, draft });
            setAdding(false);
          }}
        />
      )}

      {panels.length === 0 && !adding && (
        <span style={{ color: t.textDim }}>{canEdit ? 'This dashboard has no panels. Press Edit, then Add panel.' : 'This dashboard has no panels yet.'}</span>
      )}

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
        {panels.map((p, i) => {
          const full = panelWidth(p) === 'full' || !wide;
          return (
            <div key={p.id} style={{ width: full ? '100%' : '49%', display: 'flex', flexDirection: 'column', gap: 6 }}>
              {editing === p.id ? (
                <PanelForm
                  initial={panelDraft(p)}
                  submitLabel="Save panel"
                  onCancel={() => setEditing(null)}
                  onSubmit={async (draft) => {
                    await put(p, draft);
                    setEditing(null);
                  }}
                />
              ) : (
                <PanelCard panel={p} range={range} refreshMs={refreshMs} />
              )}
              {edit && editing !== p.id && (
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                  <Chip label="Up" accessibilityLabel={`Move ${p.title} up`} onPress={() => void move(i, -1)} />
                  <Chip label="Down" accessibilityLabel={`Move ${p.title} down`} onPress={() => void move(i, 1)} />
                  <Chip label={panelWidth(p) === 'full' ? 'Width: full' : 'Width: half'} accessibilityLabel={`Toggle width of ${p.title}`} onPress={() => void setOpt(p, { width: panelWidth(p) === 'full' ? 'half' : 'full' })} />
                  <Chip label={`Height: ${panelHeight(p).toUpperCase()}`} accessibilityLabel={`Change height of ${p.title}`} onPress={() => void setOpt(p, { height: NEXT_HEIGHT[panelHeight(p)] })} />
                  <Chip label="Edit" accessibilityLabel={`Edit ${p.title}`} onPress={() => { setEditing(p.id); setAdding(false); }} />
                  {confirmDelete === p.id ? (
                    <Chip label="Confirm delete" accessibilityLabel={`Confirm delete ${p.title}`} onPress={() => void run(async () => { await m.deletePanel.mutateAsync({ dashboardId: id, panelId: p.id }); setConfirmDelete(null); })} />
                  ) : (
                    <Chip label="Delete" accessibilityLabel={`Delete ${p.title}`} onPress={() => setConfirmDelete(p.id)} />
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
