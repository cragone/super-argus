import { formatValue, lastValue, type ChartSeries } from '@/lib/chart';
import { useElementWidth } from '@/hooks/useElementWidth';
import { useChartColors } from '@/theme/chartColors';

/** Total of the newest value across series (a stat panel normally has one series). */
export function statValue(series: ChartSeries[]): number | null {
  if (series.length === 0) return null;
  return series.reduce((sum, s) => sum + lastValue(s), 0);
}

/**
 * One headline number with a quiet sparkline: the trend in the de-emphasis gray,
 * the current point in the accent. No axes: it is a stat, not a plot.
 */
export function StatTile({ series, format = formatValue, height = 36 }: { series: ChartSeries[]; format?: (v: number) => string; height?: number }) {
  const c = useChartColors();
  const [ref, w] = useElementWidth<HTMLDivElement>();
  const value = statValue(series);

  // Sum the series pointwise for the sparkline.
  const byTime = new Map<number, number>();
  for (const s of series) for (const [t, v] of s.points) byTime.set(t, (byTime.get(t) ?? 0) + v);
  const pts = [...byTime.entries()].sort((a, b) => a[0] - b[0]);

  const min = Math.min(...pts.map((p) => p[1]), 0);
  const max = Math.max(...pts.map((p) => p[1]), 1);
  const x = (i: number) => 3 + (pts.length > 1 ? (i / (pts.length - 1)) * (w - 6) : 0);
  const y = (v: number) => 4 + (1 - (v - min) / (max - min || 1)) * (height - 8);
  const d = pts.map((p, i) => `${i ? 'L' : 'M'} ${x(i).toFixed(1)} ${y(p[1]).toFixed(1)}`).join(' ');

  return (
    <div ref={ref}>
      <span aria-label={`Value ${value === null ? 'no data' : format(value)}`} style={{ color: c.ink, fontSize: 34, fontWeight: 600, display: 'block' }}>
        {value === null ? '-' : format(value)}
      </span>
      {w > 0 && pts.length > 1 && (
        <svg width={w} height={height} style={{ marginTop: 6, display: 'block' }}>
          <path d={d} stroke={c.deEmphasis} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" fill="none" />
          <circle cx={x(pts.length - 1)} cy={y(pts[pts.length - 1][1])} r={4} fill={c.categorical[0]} stroke={c.surface} strokeWidth={2} />
        </svg>
      )}
    </div>
  );
}
