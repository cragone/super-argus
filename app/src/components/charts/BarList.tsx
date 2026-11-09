import { formatValue, lastValue, type ChartSeries } from '@/lib/chart';
import { useChartColors } from '@/theme/chartColors';

export interface BarItem {
  key: string;
  label: string;
  value: number;
}

/** Top-N items from series: the newest value of each, largest first. */
export function barItems(series: ChartSeries[], max = 10): BarItem[] {
  return series
    .map((s) => ({ key: s.key, label: s.label, value: lastValue(s) }))
    .filter((i) => i.value > 0)
    .sort((a, b) => b.value - a.value || (a.label < b.label ? -1 : 1))
    .slice(0, max);
}

/**
 * Ranked horizontal bars, largest first, one hue: bars share one baseline, so close
 * values stay comparable. 16px thick, square at the baseline, 4px rounded at the tip, value at the tip.
 */
export function BarList({ items, format = formatValue }: { items: BarItem[]; format?: (v: number) => string }) {
  const c = useChartColors();
  const max = Math.max(...items.map((i) => i.value), 0);
  return (
    <div role="list" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {items.map((it) => (
        <div key={it.key} role="text" aria-label={`${it.label}: ${format(it.value)}`}>
          <div style={{ color: c.inkSecondary, fontSize: 12, marginBottom: 3, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {it.label}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div style={{ flex: 1, height: 16, display: 'flex', alignItems: 'center' }}>
              <div
                style={{
                  width: `${max > 0 ? Math.max(2, (it.value / max) * 100) : 0}%`,
                  height: 16,
                  backgroundColor: c.sequential,
                  borderTopRightRadius: 4,
                  borderBottomRightRadius: 4,
                }}
              />
            </div>
            <span style={{ color: c.ink, fontSize: 12, fontWeight: 700, minWidth: 44, textAlign: 'right' }}>{format(it.value)}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
