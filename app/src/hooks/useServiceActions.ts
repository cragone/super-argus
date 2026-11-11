import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { api, unwrap, type Action, type RestartTarget } from '@/api/client';
import { useSession } from '@/auth/session';

const POLL_MS = 2_000;

export const isDone = (a: Pick<Action, 'status'> | undefined) =>
  a !== undefined && a.status !== 'pending' && a.status !== 'running';

/** What can be restarted for a service. Admin-only on the server, so viewers/editors never call it. */
export function useRestartTargets(service: string, enabled: boolean) {
  const on = useSession((s) => s.status === 'signedIn');
  return useQuery({
    queryKey: ['restart-targets', service],
    enabled: on && enabled,
    queryFn: async () => unwrap(await api.GET('/services/{name}/restart-targets', { params: { path: { name: service } } })).items,
  });
}

export function useRestartService(service: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (target: RestartTarget) =>
      unwrap(await api.POST('/services/{name}/restart', { params: { path: { name: service } }, body: { labels: target.labels } })),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['actions'] }),
  });
}

/** One action, polled until it finishes. */
export function useAction(id: string | undefined) {
  return useQuery({
    queryKey: ['actions', id],
    enabled: !!id,
    refetchInterval: (q) => (isDone(q.state.data) ? false : POLL_MS),
    queryFn: async () => unwrap(await api.GET('/actions/{id}', { params: { path: { id: id ?? '' } } })),
  });
}
