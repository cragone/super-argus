import { addMatcher, getLevelMatcher, setLevelMatcher } from '@/lib/selector';

describe('addMatcher', () => {
  it('creates a selector from nothing', () => {
    expect(addMatcher('', 'service', 'api')).toBe('{service="api"}');
    expect(addMatcher('  ', 'service', 'api')).toBe('{service="api"}');
  });

  it('keeps existing pipeline text when there is no selector yet', () => {
    expect(addMatcher('|= "err"', 'service', 'api')).toBe('{service="api"} |= "err"');
  });

  it('fills an empty selector', () => {
    expect(addMatcher('{}', 'env', 'prod')).toBe('{env="prod"}');
  });

  it('appends to an existing selector and keeps the rest', () => {
    expect(addMatcher('{service="api"} |= "x"', 'env', 'prod')).toBe('{service="api", env="prod"} |= "x"');
    expect(addMatcher('{service="api",}', 'env', 'prod')).toBe('{service="api", env="prod"}');
  });

  it('replaces the value of an existing equality matcher for that label', () => {
    expect(addMatcher('{service="api", env="dev"}', 'env', 'prod')).toBe('{service="api", env="prod"}');
  });

  it('does not replace regex matchers, it adds alongside', () => {
    expect(addMatcher('{env=~"a|b"}', 'env', 'prod')).toBe('{env=~"a|b", env="prod"}');
  });

  it('escapes the value', () => {
    expect(addMatcher('', 'msg', 'say "hi"')).toBe('{msg="say \\"hi\\""}');
  });
});

describe('level matcher', () => {
  const ALL = ['fatal', 'error', 'warning', 'info', 'unknown'];

  it('sets, merges and clears the level matcher', () => {
    expect(setLevelMatcher('{service="api"}', ['error'], ALL)).toBe('{service="api", level="error"}');
    expect(setLevelMatcher('{service="api"}', ['error', 'fatal'], ALL)).toBe('{service="api", level=~"error|fatal"}');
    expect(setLevelMatcher('{service="api", level="error"}', ['warning'], ALL)).toBe('{service="api", level="warning"}');
    expect(setLevelMatcher('{service="api", level="error"}', [], ALL)).toBe('{service="api"}');
    expect(setLevelMatcher('{service="api"}', ALL, ALL)).toBe('{service="api"}');
  });

  it('creates a selector when there is none and keeps the pipeline', () => {
    expect(setLevelMatcher('', ['info'], ALL)).toBe('{level="info"}');
    expect(setLevelMatcher('', [], ALL)).toBe('');
    expect(setLevelMatcher('{a="b,c"} |= "x"', ['error'], ALL)).toBe('{a="b,c", level="error"} |= "x"');
  });

  it('reads the selected levels back', () => {
    expect(getLevelMatcher('{service="api", level=~"error|fatal"}')).toEqual(['error', 'fatal']);
    expect(getLevelMatcher('{level="info"} | json')).toEqual(['info']);
    expect(getLevelMatcher('{service="api"}')).toEqual([]);
    expect(getLevelMatcher('')).toEqual([]);
  });
});
