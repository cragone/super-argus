import { useState } from 'react';

import { errorMessage, type RestartTarget } from '@/api/client';
import { Button, ErrorText } from '@/components/ui';
import { useIsAdmin } from '@/hooks/useAlerts';
import { isDone, useAction, useRestartService, useRestartTargets } from '@/hooks/useServiceActions';
import { useTheme } from '@/theme';

/** Human label for a target: the unit on its host, or the pod in its namespace. */
export function describeTarget(t: Pick<RestartTarget, 'kind' | 'target'>): string {
  return t.kind === 'systemd_restart' ? `${t.target.unit} on ${t.target.host}` : `pod ${t.target.namespace}/${t.target.pod}`;
}

/**
 * Admin-only "Restart" control for a service. Nothing renders unless the signed-in
 * user is an admin and the service has a restartable target; the click opens an
 * inline confirmation, and the outcome is polled until the agent reports back.
 */
export function RestartControl({ service }: { service: string }) {
  const t = useTheme();
  const admin = useIsAdmin();
  const targets = useRestartTargets(service, admin);
  const restart = useRestartService(service);
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState(0);
  const action = useAction(restart.data?.id);

  const items = targets.data ?? [];
  if (!admin || items.length === 0) return null;
  const target = items[Math.min(picked, items.length - 1)];
  const a = action.data ?? restart.data;
  const running = restart.isPending || (restart.isSuccess && !isDone(a));

  const close = () => {
    setOpen(false);
    restart.reset();
  };

  if (!open) return <Button title="Restart…" variant="ghost" onPress={() => setOpen(true)} />;

  return (
    <div role="group" aria-label="Restart service" style={{ display: 'flex', flexDirection: 'column', gap: 8, color: t.text }}>
      {items.length > 1 && !restart.isSuccess ? (
        <select aria-label="Restart target" value={picked} onChange={(e) => setPicked(Number(e.target.value))}>
          {items.map((it, i) => (
            <option key={describeTarget(it)} value={i}>
              {describeTarget(it)}
            </option>
          ))}
        </select>
      ) : null}
      {!restart.isSuccess ? (
        <>
          <span>
            Restart <strong>{describeTarget(target)}</strong>? {target.kind === 'k8s_restart' ? 'The owning workload is rolled.' : 'The service is briefly unavailable.'}
          </span>
          {restart.isError ? <ErrorText>{errorMessage(restart.error)}</ErrorText> : null}
          <div style={{ display: 'flex', gap: 8 }}>
            <Button title="Confirm restart" loading={restart.isPending} onPress={() => restart.mutate(target)} />
            <Button title="Cancel" variant="ghost" onPress={close} />
          </div>
        </>
      ) : (
        <>
          <span role="status">
            {running ? 'Waiting for the agent…' : a?.status === 'succeeded' ? `Done: ${a.result ?? 'restarted'}` : null}
          </span>
          {a && isDone(a) && a.status !== 'succeeded' ? (
            <ErrorText>{a.status === 'expired' ? 'No agent picked this up in time.' : `Failed: ${a.result ?? a.status}`}</ErrorText>
          ) : null}
          {action.isError ? <ErrorText>{errorMessage(action.error)}</ErrorText> : null}
          {!running ? <Button title="Close" variant="ghost" onPress={close} /> : null}
        </>
      )}
    </div>
  );
}
