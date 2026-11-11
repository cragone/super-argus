import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, vi } from 'vitest';

import type { Action, RestartTarget } from '@/api/client';
import { RestartControl, describeTarget } from '@/components/RestartControl';

const h = vi.hoisted(() => ({
  admin: true,
  targets: [] as RestartTarget[],
  mutate: vi.fn(),
  mutation: {} as Record<string, unknown>,
  action: undefined as Action | undefined,
}));

vi.mock('@/hooks/useAlerts', () => ({ useIsAdmin: () => h.admin }));
vi.mock('@/hooks/useServiceActions', async (orig) => ({
  ...(await orig<typeof import('@/hooks/useServiceActions')>()),
  useRestartTargets: () => ({ data: h.targets }),
  useRestartService: () => ({ mutate: h.mutate, reset: vi.fn(), isPending: false, isSuccess: false, isError: false, ...h.mutation }),
  useAction: () => ({ data: h.action, isError: false }),
}));

const unit: RestartTarget = {
  kind: 'systemd_restart',
  target: { host: 'h1', unit: 'nginx.service' },
  labels: { source: 'journald', host: 'h1', unit: 'nginx.service' },
  last_seen: '2026-01-01T00:00:00Z',
};
const pod: RestartTarget = {
  kind: 'k8s_restart',
  target: { namespace: 'prod', pod: 'web-abc-1' },
  labels: { source: 'kubernetes', namespace: 'prod', pod: 'web-abc-1' },
  last_seen: '2026-01-01T00:00:00Z',
};
const act = (status: Action['status'], result?: string): Action => ({
  id: 'a1', kind: 'systemd_restart', service: 'nginx', target: {}, status, requested_by: 'u', requested_at: '2026-01-01T00:00:00Z', result,
});

beforeEach(() => {
  h.admin = true;
  h.targets = [unit];
  h.mutation = {};
  h.action = undefined;
  h.mutate.mockClear();
});

describe('describeTarget', () => {
  it('names the unit and host, or the pod and namespace', () => {
    expect(describeTarget(unit)).toBe('nginx.service on h1');
    expect(describeTarget(pod)).toBe('pod prod/web-abc-1');
  });
});

describe('RestartControl', () => {
  it('renders nothing for non-admins or when nothing is restartable', () => {
    h.admin = false;
    const { container, rerender } = render(<RestartControl service="nginx" />);
    expect(container).toBeEmptyDOMElement();
    h.admin = true;
    h.targets = [];
    rerender(<RestartControl service="nginx" />);
    expect(container).toBeEmptyDOMElement();
  });

  it('asks for confirmation before restarting, then sends the target', () => {
    render(<RestartControl service="nginx" />);
    fireEvent.click(screen.getByRole('button', { name: 'Restart…' }));
    expect(h.mutate).not.toHaveBeenCalled();
    expect(screen.getByText('nginx.service on h1')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Confirm restart' }));
    expect(h.mutate).toHaveBeenCalledWith(unit);
  });

  it('lets the admin cancel without restarting', () => {
    render(<RestartControl service="nginx" />);
    fireEvent.click(screen.getByRole('button', { name: 'Restart…' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(h.mutate).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Restart…' })).toBeInTheDocument();
  });

  it('picks among several targets', () => {
    h.targets = [unit, pod];
    render(<RestartControl service="web" />);
    fireEvent.click(screen.getByRole('button', { name: 'Restart…' }));
    fireEvent.change(screen.getByLabelText('Restart target'), { target: { value: '1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirm restart' }));
    expect(h.mutate).toHaveBeenCalledWith(pod);
  });

  it('shows progress, success and failure from the polled action', () => {
    h.mutation = { isSuccess: true, data: act('pending') };
    h.action = act('running');
    const { rerender } = render(<RestartControl service="nginx" />);
    fireEvent.click(screen.getByRole('button', { name: 'Restart…' }));
    expect(screen.getByRole('status')).toHaveTextContent('Waiting for the agent');

    h.action = act('succeeded', 'restarted nginx.service');
    rerender(<RestartControl service="nginx" />);
    expect(screen.getByRole('status')).toHaveTextContent('Done: restarted nginx.service');

    h.action = act('failed', 'authorization required');
    rerender(<RestartControl service="nginx" />);
    expect(screen.getByRole('alert')).toHaveTextContent('Failed: authorization required');

    h.action = act('expired');
    rerender(<RestartControl service="nginx" />);
    expect(screen.getByRole('alert')).toHaveTextContent('No agent picked this up');
  });
});
