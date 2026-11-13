import { useTheme } from '@/theme';

export type LivenessState = 'alive' | 'down' | 'unknown';

const LABEL: Record<LivenessState, string> = { alive: 'alive', down: 'down', unknown: 'unknown' };

/**
 * Color dot + word for a service's real liveness (systemd/Kubernetes collector
 * state), distinct from log-recency "active". Renders nothing when no
 * collector has reported for this service.
 */
export function LivenessBadge({ state, size = 10 }: { state: LivenessState | null | undefined; size?: number }) {
  const t = useTheme();
  if (!state) return null;
  const color = state === 'alive' ? t.levels.info : state === 'down' ? t.danger : t.levels.warning;
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span
        aria-hidden
        style={{ width: size, height: size, borderRadius: size / 2, backgroundColor: color, flexShrink: 0 }}
      />
      <span style={{ color, fontSize: 13, fontWeight: 600 }}>{LABEL[state]}</span>
    </span>
  );
}
