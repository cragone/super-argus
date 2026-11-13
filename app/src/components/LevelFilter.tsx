import { LEVEL_LABEL, LEVELS } from '@/lib/logLevel';
import { getLevelMatcher, setLevelMatcher } from '@/lib/selector';
import { useTheme } from '@/theme';

const ERROR_AND_ABOVE = 'fatal,error';

/** Dropdown choices: everything, each level, and "Error and above" (fatal + error). */
const OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'All levels' },
  ...LEVELS.map((l) => ({ value: l, label: LEVEL_LABEL[l] })),
  { value: ERROR_AND_ABOVE, label: 'Error and above' },
];

/**
 * Level dropdown driven by the query's `level` matcher, so the query text stays
 * the single source of truth. Writes `level="x"` or `level=~"a|b"` into the selector.
 */
export function LevelFilter({ query, onChange }: { query: string; onChange: (query: string) => void }) {
  const t = useTheme();
  const selected = getLevelMatcher(query);
  const current = [...selected].sort().join(',') === [...ERROR_AND_ABOVE.split(',')].sort().join(',') ? ERROR_AND_ABOVE : selected.join(',');
  const known = OPTIONS.some((o) => o.value === current);

  return (
    <select
      aria-label="Filter by level"
      value={known ? current : ''}
      onChange={(e) => onChange(setLevelMatcher(query, e.target.value ? e.target.value.split(',') : [], LEVELS))}
      style={{
        color: t.text,
        backgroundColor: t.bg,
        border: `1px solid ${t.border}`,
        borderRadius: 8,
        padding: '8px 10px',
        fontSize: 14,
        cursor: 'pointer',
      }}>
      {OPTIONS.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );
}
