import { detectLevel, levelOf, prettyJson } from '@/lib/logLevel';

describe('detectLevel', () => {
  it('prefers a parsed level field', () => {
    expect(detectLevel('anything', { level: 'ERROR' })).toBe('error');
    expect(detectLevel('anything', { severity: 'warning' })).toBe('warning');
    expect(detectLevel('anything', { level: 'critical' })).toBe('fatal');
  });

  it('falls back to the line text', () => {
    expect(detectLevel('2026 level=error msg=boom')).toBe('error');
    expect(detectLevel('[WARN] disk almost full')).toBe('warning');
    expect(detectLevel('INFO started')).toBe('info');
    expect(detectLevel('FATAL out of memory')).toBe('fatal');
    expect(detectLevel('trace id=1')).toBeNull();
    expect(detectLevel('nothing to see')).toBeNull();
  });

  it('does not match levels inside other words', () => {
    expect(detectLevel('information_schema query')).toBeNull();
    expect(detectLevel('terrorist')).toBeNull();
  });

  it('ignores an unknown level field value', () => {
    expect(detectLevel('level=error', { level: 'weird' })).toBe('error');
  });
});

describe('levelOf', () => {
  it('uses the server-assigned level label first', () => {
    expect(levelOf('ERROR boom', undefined, { level: 'warning' })).toBe('warning');
  });
  it('falls back to detection, then unknown', () => {
    expect(levelOf('ERROR boom', undefined, { service: 'api' })).toBe('error');
    expect(levelOf('plain', undefined, { level: 'bogus' })).toBe('unknown');
  });
});

describe('prettyJson', () => {
  it('pretty-prints objects and arrays', () => {
    expect(prettyJson('{"a":1,"b":[2]}')).toBe('{\n  "a": 1,\n  "b": [\n    2\n  ]\n}');
    expect(prettyJson('  [1]  ')).toBe('[\n  1\n]');
  });

  it('returns null for non-JSON and invalid JSON', () => {
    expect(prettyJson('plain text')).toBeNull();
    expect(prettyJson('{broken')).toBeNull();
  });
});
