import { useCallback, useEffect, useRef, useState } from 'react';

import { LogRow } from '@/components/LogRow';
import { QueryInput } from '@/components/QueryInput';
import { Button, Chip, ErrorText } from '@/components/ui';
import { VirtualList, type VirtualListRef } from '@/components/VirtualList';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { MAX_TAIL_LINES, useTail } from '@/hooks/useTail';
import { useTheme } from '@/theme';

const STATUS_TEXT = {
  stopped: 'Not connected',
  connecting: 'Connecting…',
  live: 'Live',
  reconnecting: 'Reconnecting…',
  error: 'Stopped',
} as const;

export function Tail() {
  useDocumentTitle('Live Tail');
  const t = useTheme();
  const [query, setQuery] = useState('');
  const [autoScroll, setAutoScroll] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [formError, setFormError] = useState<string | null>(null);
  const tail = useTail();
  const list = useRef<VirtualListRef>(null);

  const running = tail.status === 'connecting' || tail.status === 'live' || tail.status === 'reconnecting';

  const start = useCallback(() => {
    if (!query.trim()) {
      setFormError('Enter a query with a stream selector, e.g. {service="api"}');
      return;
    }
    setFormError(null);
    setExpanded(new Set());
    tail.start(query.trim());
  }, [query, tail]);

  const toggle = useCallback((key: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (!next.delete(key)) next.add(key);
      return next;
    });
  }, []);

  useEffect(() => {
    if (autoScroll && !tail.paused && tail.lines.length > 0) list.current?.scrollToEnd();
  }, [tail.lines, autoScroll, tail.paused]);

  const dot = tail.status === 'live' ? t.levels.info : tail.status === 'error' ? t.danger : t.levels.warn;

  return (
    <div style={{ height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column', backgroundColor: t.bg }}>
      <div style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 10, borderBottom: `1px solid ${t.border}`, backgroundColor: t.surface }}>
        <QueryInput value={query} onChange={setQuery} onSubmit={start} />
        <div style={{ display: 'flex', gap: 8 }}>
          {running ? (
            <Button title="Stop" variant="ghost" onPress={tail.stop} style={{ flex: 1 }} />
          ) : (
            <Button title="Start tail" onPress={start} style={{ flex: 1 }} />
          )}
          <Button
            title={tail.paused ? 'Resume' : 'Pause'}
            variant="ghost"
            disabled={!running}
            onPress={() => tail.setPaused(!tail.paused)}
          />
          <Button title="Clear" variant="ghost" onPress={tail.clear} />
        </div>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Chip label={autoScroll ? 'Auto-scroll on' : 'Auto-scroll off'} selected={autoScroll} onPress={() => setAutoScroll((v) => !v)} />
          <span style={{ width: 9, height: 9, borderRadius: 5, backgroundColor: dot, marginRight: 6 }} />
          <span aria-label="Tail status" style={{ color: t.textDim, fontSize: 13 }}>
            {STATUS_TEXT[tail.status]}
            {tail.paused && tail.buffered > 0 ? ` · paused, ${tail.buffered} new` : tail.paused ? ' · paused' : ''}
          </span>
        </div>
        {formError && <ErrorText>{formError}</ErrorText>}
        {tail.error && <ErrorText>{tail.error}</ErrorText>}
        {tail.droppedLines > 0 && (
          <span aria-label="Dropped lines" style={{ color: t.levels.warn, fontSize: 13 }}>
            Too fast to keep up: {tail.droppedLines.toLocaleString()} lines were skipped. Narrow the query to see everything.
          </span>
        )}
      </div>

      {tail.lines.length === 0 ? (
        <span style={{ color: t.textDim, padding: 16 }}>
          {running ? 'Waiting for new lines…' : 'Write a query and start the tail to watch new lines arrive.'}
        </span>
      ) : (
        <VirtualList
          ref={list}
          data={tail.lines}
          keyExtractor={(l) => String(l.id)}
          renderItem={(item) => (
            <LogRow rowKey={String(item.id)} entry={item} labels={item.labels} expanded={expanded.has(String(item.id))} onToggle={toggle} />
          )}
        />
      )}
      <span style={{ color: t.textDim, fontSize: 11, padding: 6, textAlign: 'center' }}>
        {tail.lines.length.toLocaleString()} / {MAX_TAIL_LINES.toLocaleString()} lines kept in memory
      </span>
    </div>
  );
}
