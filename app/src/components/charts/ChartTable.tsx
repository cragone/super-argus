import { formatFullTime, formatValue, unionTimes, valueAt, type ChartSeries } from '@/lib/chart';
import { useChartColors } from '@/theme/chartColors';

/** The table view of a chart: every value is reachable without hover. Newest rows first. */
export function ChartTable({ series, rows = 30, format = formatValue }: { series: ChartSeries[]; rows?: number; format?: (v: number) => string }) {
  const c = useChartColors();
  const times = unionTimes(series).slice(-rows).reverse();
  const cols = series.slice(0, 8);
  return (
    <div style={{ overflowX: 'auto' }}>
      <table role="table" aria-label="Data table" style={{ borderCollapse: 'collapse' }}>
        <thead>
          <tr style={{ borderBottom: `1px solid ${c.grid}` }}>
            <th style={{ width: 150, color: c.inkSecondary, fontSize: 11, fontWeight: 400, padding: '4px 0', textAlign: 'left' }}>Time</th>
            {cols.map((s) => (
              <th
                key={s.key}
                style={{ width: 90, color: c.inkSecondary, fontSize: 11, fontWeight: 400, padding: '4px 0', textAlign: 'right', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {s.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {times.map((t) => (
            <tr key={t} style={{ borderBottom: `1px solid ${c.grid}` }}>
              <td style={{ width: 150, color: c.inkSecondary, fontSize: 11, padding: '3px 0' }}>{formatFullTime(t)}</td>
              {cols.map((s) => {
                const v = valueAt(s, t);
                return (
                  <td key={s.key} style={{ width: 90, color: c.ink, fontSize: 11, padding: '3px 0', textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
                    {v === null ? '' : format(v)}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
