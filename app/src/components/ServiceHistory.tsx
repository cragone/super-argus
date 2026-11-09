import { useCallback, useMemo, useState } from 'react';

import { errorMessage } from '@/api/client';
import { LogRow } from '@/components/LogRow';
import { TimeRangePicker } from '@/components/TimeRangePicker';
import { Button, ErrorText } from '@/components/ui';
import { VirtualList } from '@/components/VirtualList';
import { useRangeQuery, type SubmittedQuery } from '@/hooks/useLogs';
import { resolveRange, type TimeRange } from '@/lib/time';
import { useTheme } from '@/theme';

const LIMIT = 1000;

/** Older lines for one service query, over a chosen time range. */
export function ServiceHistory({ query }: { query: string }) {
  const t = useTheme();
  const [range, setRange] = useState<TimeRange>({ kind: 'preset', id: '1h' });
  const [direction, setDirection] = useState<'backward' | 'forward'>('backward');
  // the parent remounts this view when the query changes, so the first load is just initial state
  const [submitted, setSubmitted] = useState<SubmittedQuery | null>(() => {
    const r = resolveRange({ kind: 'preset', id: '1h' });
    return 'error' in r ? null : { query, start: r.start, end: r.end, direction: 'backward', limit: LIMIT, nonce: 1 };
  });
  const [formError, setFormError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const result = useRangeQuery(submitted);

  const run = useCallback(() => {
    const r = resolveRange(range);
    if ('error' in r) {
      setFormError(r.error);
      return;
    }
    setFormError(null);
    setExpanded(new Set());
    setSubmitted((prev) => ({ query, start: r.start, end: r.end, direction, limit: LIMIT, nonce: (prev?.nonce ?? 0) + 1 }));
  }, [query, range, direction]);

  const toggle = useCallback((key: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (!next.delete(key)) next.add(key);
      return next;
    });
  }, []);

  const rows = useMemo(() => result.data?.entries.map((e, i) => ({ key: `${e.ts}-${e.stream}-${i}`, entry: e })) ?? [], [result.data]);
  const stats = result.data?.stats;

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
      <div style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 8, borderBottom: `1px solid ${t.border}` }}>
        <TimeRangePicker value={range} onChange={setRange} />
        <div style={{ display: 'flex', gap: 8 }}>
          <Button title="Run" onPress={run} loading={result.isFetching} style={{ flex: 1 }} />
          <Button
            title={direction === 'backward' ? 'Newest first' : 'Oldest first'}
            variant="ghost"
            onPress={() => setDirection((d) => (d === 'backward' ? 'forward' : 'backward'))}
          />
        </div>
        {formError && <ErrorText>{formError}</ErrorText>}
      </div>
      {result.isError && (
        <div style={{ padding: 12 }}>
          <ErrorText>{errorMessage(result.error)}</ErrorText>
        </div>
      )}
      {stats && (
        <span aria-label="Query stats" style={{ color: t.textDim, fontSize: 12, padding: '6px 12px' }}>
          {rows.length} {rows.length === 1 ? 'line' : 'lines'}
          {rows.length >= LIMIT ? ` (limit ${LIMIT})` : ''} · {stats.elapsed_ms} ms
        </span>
      )}
      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        {result.isFetching && !result.data ? (
          <span style={{ color: t.textDim, padding: 16 }}>Running…</span>
        ) : rows.length === 0 && !result.isError ? (
          <span style={{ color: t.textDim, padding: 16 }}>No lines from this service in this time range.</span>
        ) : (
          <VirtualList
            data={rows}
            keyExtractor={(r) => r.key}
            renderItem={(item) => (
              <LogRow rowKey={item.key} entry={item.entry} labels={result.data?.streams[item.entry.stream]} expanded={expanded.has(item.key)} onToggle={toggle} />
            )}
          />
        )}
      </div>
    </div>
  );
}
