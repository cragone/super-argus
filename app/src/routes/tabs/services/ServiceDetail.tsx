import { useState } from 'react';
import { useParams } from 'react-router-dom';

import { ServiceHistory } from '@/components/ServiceHistory';
import { ServiceLive } from '@/components/ServiceLive';
import { RestartControl } from '@/components/RestartControl';
import { Chip } from '@/components/ui';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { serviceQuery } from '@/lib/serviceQuery';
import { useTheme } from '@/theme';

type Mode = 'live' | 'history';

export function ServiceDetail() {
  const t = useTheme();
  const { name: rawName = '' } = useParams<{ name: string }>();
  const name = decodeURIComponent(rawName);
  useDocumentTitle(name);
  const [mode, setMode] = useState<Mode>('live');
  const [draft, setDraft] = useState('');
  const [filter, setFilter] = useState('');
  const query = serviceQuery(name, filter);

  return (
    <div style={{ height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column', backgroundColor: t.bg }}>
      <div style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 8, backgroundColor: t.surface, borderBottom: `1px solid ${t.border}` }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
          <div style={{ display: 'flex' }}>
            <Chip label="Live" selected={mode === 'live'} onPress={() => setMode('live')} />
            <Chip label="History" selected={mode === 'history'} onPress={() => setMode('history')} />
          </div>
          <RestartControl service={name} />
        </div>
        <input
          aria-label="Filter lines"
          placeholder={'Filter lines, e.g. timeout  (or | json | level="error")'}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && setFilter(draft)}
          autoCapitalize="none"
          autoCorrect="off"
          style={{ color: t.text, backgroundColor: t.bg, borderColor: t.border, borderWidth: 1, borderStyle: 'solid', borderRadius: 8, padding: '8px 10px', fontSize: 15 }}
        />
      </div>
      {mode === 'live' ? <ServiceLive key={query} query={query} /> : <ServiceHistory key={query} query={query} />}
    </div>
  );
}
