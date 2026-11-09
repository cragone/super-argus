import type { ChartSeries } from '@/lib/chart';
import { useChartColors } from '@/theme/chartColors';

/** Identity channel for 2+ series: line key + name in text ink (never the series color). */
export function Legend({ series, slots }: { series: ChartSeries[]; slots: number[] }) {
  const c = useChartColors();
  if (series.length < 2) return null; // a single series is named by the panel title
  return (
    <div role="list" style={{ display: 'flex', flexWrap: 'wrap', gap: 12, paddingTop: 6 }}>
      {series.map((s, i) => (
        <div key={s.key} role="text" style={{ display: 'flex', alignItems: 'center', gap: 6, maxWidth: 220 }}>
          <span style={{ width: 14, height: 2, borderRadius: 1, backgroundColor: c.categorical[slots[i] % 8] }} />
          <span style={{ color: c.inkSecondary, fontSize: 12, flexShrink: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {s.label}
          </span>
        </div>
      ))}
    </div>
  );
}
