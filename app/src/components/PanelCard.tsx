import { useMemo, useRef, useState } from 'react';

import { errorMessage, type Panel } from '@/api/client';
import { usePanelData } from '@/hooks/useDashboards';
import { foldSeries, toChartSeries } from '@/lib/chart';
import type { TimeRange } from '@/lib/time';
import { ColorAssigner, useChartColors } from '@/theme/chartColors';

import { BarList, barItems } from './charts/BarList';
import { ChartTable } from './charts/ChartTable';
import { Legend } from './charts/Legend';
import { StatTile } from './charts/StatTile';
import { TimeseriesChart } from './charts/TimeseriesChart';
import { LogRow } from './LogRow';

export type PanelHeight = 's' | 'm' | 'l';
export const HEIGHTS: Record<PanelHeight, number> = { s: 140, m: 210, l: 320 };

export function panelHeight(p: Panel): PanelHeight {
  const h = (p.options as { height?: string } | undefined)?.height;
  return h === 's' || h === 'l' ? h : 'm';
}

export function panelWidth(p: Panel): 'half' | 'full' {
  return (p.options as { width?: string } | undefined)?.width === 'full' ? 'full' : 'half';
}

/** One dashboard panel: title, body for its type, and a table view for charts. */
export function PanelCard({ panel, range, refreshMs }: { panel: Panel; range: TimeRange; refreshMs: number }) {
  const c = useChartColors();
  const q = usePanelData(panel, range, refreshMs);
  const assigner = useRef(new ColorAssigner());
  const [table, setTable] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const height = HEIGHTS[panelHeight(panel)];

  const metric = q.data?.kind === 'metric' ? q.data.metric : null;
  const series = useMemo(() => (metric ? foldSeries(toChartSeries(metric.series)) : []), [metric]);
  // Mutating during render is intentional: slot assignment must stay stable across
  // renders as series come and go, and `assign` is idempotent for an unchanged key set.
  // eslint-disable-next-line react-hooks/refs
  const slots = assigner.current.assign(series.map((s) => s.key));
  const chartPanel = panel.type === 'timeseries';

  const refetching = q.isFetching && !!q.data; // keep the frame, dim it
  let body: React.ReactNode;
  if (q.isError) {
    body = <span role="alert" style={{ color: '#d03b3b', fontSize: 13 }}>{errorMessage(q.error)}</span>;
  } else if (!q.data) {
    body = <span style={{ color: c.inkSecondary, fontSize: 13 }}>Loading…</span>;
  } else if (q.data.kind === 'logs') {
    const { entries, streams } = q.data.logs;
    body =
      entries.length === 0 ? (
        <span style={{ color: c.inkSecondary, fontSize: 13 }}>No lines in this range.</span>
      ) : (
        <div style={{ maxHeight: height + 60, overflow: 'hidden' }}>
          {entries.slice(0, 50).map((e, i) => {
            const key = `${e.ts}-${i}`;
            return (
              <LogRow
                key={key}
                rowKey={key}
                entry={e}
                labels={streams[e.stream]}
                expanded={expanded.has(key)}
                onToggle={(k) => setExpanded((p) => (p.has(k) ? new Set([...p].filter((x) => x !== k)) : new Set(p).add(k)))}
              />
            );
          })}
        </div>
      );
  } else if (series.length === 0) {
    body = <span style={{ color: c.inkSecondary, fontSize: 13 }}>No data in this range.</span>;
  } else if (panel.type === 'stat') {
    body = <StatTile series={series} />;
  } else if (panel.type === 'top_n') {
    const items = barItems(series);
    body = items.length === 0 ? <span style={{ color: c.inkSecondary, fontSize: 13 }}>No data in this range.</span> : <BarList items={items} />;
  } else if (table) {
    body = <ChartTable series={series} />;
  } else if (metric) {
    body = (
      <>
        <TimeseriesChart series={series} slots={slots} startMs={metric.start_ms} endMs={metric.end_ms} stepMs={metric.step_ms} height={height} />
        <Legend series={series} slots={slots} />
      </>
    );
  }

  return (
    <div
      aria-label={`Panel ${panel.title}`}
      style={{ backgroundColor: c.surface, borderColor: c.border, borderWidth: 1, borderStyle: 'solid', borderRadius: 10, padding: 12, display: 'flex', flexDirection: 'column', gap: 8, opacity: refetching ? 0.6 : 1 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
        <span
          role="heading"
          aria-level={3}
          style={{ color: c.ink, fontSize: 14, fontWeight: 600, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {panel.title}
        </span>
        {chartPanel && q.data?.kind === 'metric' && series.length > 0 && (
          <button
            type="button"
            aria-label={table ? 'Show chart' : 'Show table'}
            onClick={() => setTable((t) => !t)}
            style={{ background: 'none', border: 0, padding: 0, cursor: 'pointer', color: c.inkSecondary, fontSize: 12 }}>
            {table ? 'Chart' : 'Table'}
          </button>
        )}
      </div>
      {body}
    </div>
  );
}
