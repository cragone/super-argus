import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { errorMessage } from '@/api/client';
import { Button, Card, ErrorText } from '@/components/ui';
import { useCanEdit, useDashboardMutations, useDashboards } from '@/hooks/useDashboards';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { useTheme } from '@/theme';

export function DashboardsIndex() {
  useDocumentTitle('Dashboards');
  const t = useTheme();
  const navigate = useNavigate();
  const list = useDashboards();
  const canEdit = useCanEdit();
  const { createDashboard } = useDashboardMutations();
  const [title, setTitle] = useState('');
  const [error, setError] = useState<string | null>(null);

  const create = async () => {
    if (!title.trim()) return setError('Give the dashboard a title');
    setError(null);
    try {
      const d = await createDashboard.mutateAsync(title.trim());
      setTitle('');
      navigate(`/dashboards/${d.id}`);
    } catch (e) {
      setError(errorMessage(e, 'Could not create the dashboard'));
    }
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12, maxWidth: 720, width: '100%', margin: '0 auto' }}>
      {canEdit && (
        <Card style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <span style={{ color: t.text, fontWeight: 600 }}>New dashboard</span>
          <input
            aria-label="Dashboard title"
            placeholder="Title, e.g. API health"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && void create()}
            style={{ color: t.text, backgroundColor: t.surface, borderColor: t.border, borderWidth: 1, borderStyle: 'solid', borderRadius: 8, padding: '8px 10px', fontSize: 15 }}
          />
          {error && <ErrorText>{error}</ErrorText>}
          <Button title="Create dashboard" onPress={() => void create()} loading={createDashboard.isPending} />
        </Card>
      )}

      {list.isLoading && <span style={{ color: t.textDim }}>Loading…</span>}
      {list.isError && <ErrorText>{errorMessage(list.error)}</ErrorText>}
      {list.data?.length === 0 && (
        <span style={{ color: t.textDim }}>{canEdit ? 'No dashboards yet. Create one above.' : 'No dashboards yet. Ask an editor to create one.'}</span>
      )}
      {list.data?.map((d) => (
        <button
          key={d.id}
          type="button"
          aria-label={`Open dashboard ${d.title}`}
          onClick={() => navigate(`/dashboards/${d.id}`)}
          style={{ background: 'none', border: 0, padding: 0, textAlign: 'left', cursor: 'pointer' }}>
          <Card style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ color: t.text, fontSize: 16, fontWeight: 600 }}>{d.title}</span>
            {!!d.description && <span style={{ color: t.textDim }}>{d.description}</span>}
          </Card>
        </button>
      ))}
    </div>
  );
}
