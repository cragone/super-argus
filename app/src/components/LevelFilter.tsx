import { Pressable, Text, View } from 'react-native';

import { LEVEL_LABEL, LEVELS, type LogLevel } from '@/lib/logLevel';
import { getLevelMatcher, setLevelMatcher } from '@/lib/selector';
import { useTheme } from '@/theme';

/**
 * Level chips driven by the query's `level` matcher, so the query text stays the
 * single source of truth. No chip selected = all levels.
 */
export function LevelFilter({ query, onChange }: { query: string; onChange: (query: string) => void }) {
  const t = useTheme();
  const selected = getLevelMatcher(query);

  const toggle = (l: LogLevel) => {
    const next = selected.includes(l) ? selected.filter((x) => x !== l) : [...selected, l];
    onChange(setLevelMatcher(query, next, LEVELS));
  };

  return (
    <View accessibilityLabel="Level filter" style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
      {LEVELS.map((l) => {
        const on = selected.includes(l);
        return (
          <Pressable
            key={l}
            accessibilityRole="button"
            accessibilityLabel={`Level ${LEVEL_LABEL[l]}`}
            accessibilityState={{ selected: on }}
            onPress={() => toggle(l)}
            style={{
              flexDirection: 'row',
              alignItems: 'center',
              gap: 6,
              paddingHorizontal: 10,
              paddingVertical: 5,
              borderRadius: 14,
              borderWidth: 1,
              borderColor: on ? t.levels[l] : t.border,
              backgroundColor: on ? t.surfaceAlt : t.surface,
            }}>
            <View style={{ width: 8, height: 8, borderRadius: 4, backgroundColor: t.levels[l] }} />
            <Text style={{ color: t.text, fontSize: 13, fontWeight: on ? '700' : '400' }}>{LEVEL_LABEL[l]}</Text>
          </Pressable>
        );
      })}
    </View>
  );
}
