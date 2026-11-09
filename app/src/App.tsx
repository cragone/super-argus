import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { useSession } from '@/auth/session';
import { AppRoutes } from '@/router';
import { useTheme } from '@/theme';

export function App() {
  const t = useTheme();
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } } }),
  );
  const status = useSession((s) => s.status);
  const bootstrap = useSession((s) => s.bootstrap);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  // A signed-out session must not leak cached data to the next user.
  useEffect(() => {
    if (status === 'signedOut') client.clear();
  }, [status, client]);

  if (status === 'loading') {
    return (
      <div
        style={{
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: t.bg,
        }}>
        <span className="spinner" style={{ color: t.accent, width: 24, height: 24 }} />
      </div>
    );
  }

  return (
    <QueryClientProvider client={client}>
      <div style={{ height: '100%', backgroundColor: t.bg }}>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </div>
    </QueryClientProvider>
  );
}
