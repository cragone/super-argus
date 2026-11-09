import { useNavigate } from 'react-router-dom';

import { errorMessage } from '@/api/client';
import { Card, ErrorText } from '@/components/ui';
import { useServices } from '@/hooks/useLogs';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { formatDateTime } from '@/lib/time';
import { useTheme } from '@/theme';

function ago(iso: string): string {
  const s = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000));
  if (s < 5) return 'just now';
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return formatDateTime(String(new Date(iso).getTime() * 1_000_000));
}

export function ServicesIndex() {
  useDocumentTitle('Services');
  const t = useTheme();
  const navigate = useNavigate();
  const list = useServices();

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12, maxWidth: 720, width: '100%', margin: '0 auto' }}>
      {list.isLoading && <span style={{ color: t.textDim }}>Loading…</span>}
      {list.isError && <ErrorText>{errorMessage(list.error)}</ErrorText>}
      {list.data?.length === 0 && (
        <Card style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <span style={{ color: t.text, fontWeight: 600 }}>No services yet</span>
          <span style={{ color: t.textDim }}>
            Services appear automatically once something ships logs with a service label. Run the Argus agent on a host or node, or point an SDK or Loki client at Argus with an API key.
          </span>
        </Card>
      )}
      {list.data?.map((s) => (
        <button
          key={s.name}
          type="button"
          aria-label={`Open service ${s.name}`}
          onClick={() => navigate(`/services/${encodeURIComponent(s.name)}`)}
          style={{ background: 'none', border: 0, padding: 0, textAlign: 'left', cursor: 'pointer' }}>
          <Card style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ width: 10, height: 10, borderRadius: 5, backgroundColor: s.active ? t.levels.info : t.textDim }} />
              <span style={{ color: t.text, fontSize: 16, fontWeight: 600, flex: 1 }}>{s.name}</span>
              <span style={{ color: t.textDim, fontSize: 12 }}>{s.active ? 'active' : `seen ${ago(s.last_seen)}`}</span>
            </div>
            <span style={{ color: t.textDim, fontSize: 13 }}>
              {s.streams} {s.streams === 1 ? 'stream' : 'streams'} · {s.lines.toLocaleString()} lines stored
            </span>
          </Card>
        </button>
      ))}
    </div>
  );
}
