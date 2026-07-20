import { describe, expect, it } from 'vitest';
import { levelLabel, renderMessage, type LogEntry } from './logMessages';

function entry(msg: string, fields: Record<string, unknown> = {}, legacy = false): LogEntry {
  return { time: '2026-07-20T14:00:00+09:00', level: 'INFO', msg, fields, legacy };
}

describe('renderMessage', () => {
  it('renders label generated in Korean', () => {
    const s = renderMessage(entry('label generated', { mode: 'paste', file: 'a.pdf', ip: '10.0.0.5' }));
    expect(s).toBe('라벨 생성: a.pdf (붙여넣기, 10.0.0.5)');
  });

  it('renders auto mode label', () => {
    const s = renderMessage(entry('label generated', { mode: 'auto', file: 'b.pdf', ip: '10.0.0.6' }));
    expect(s).toBe('라벨 생성: b.pdf (자동, 10.0.0.6)');
  });

  it('renders request with duration', () => {
    const s = renderMessage(entry('request', { method: 'POST', path: '/api/create_label', status: 200, duration_ms: 142 }));
    expect(s).toBe('POST /api/create_label → 200 (142ms)');
  });

  it('renders server started and logs cleared', () => {
    expect(renderMessage(entry('server started', { addr: '0.0.0.0:5000' }))).toBe('서버 시작 (0.0.0.0:5000)');
    expect(renderMessage(entry('logs cleared'))).toBe('로그 초기화됨');
  });

  it('falls back to raw msg for unknown keys and legacy lines', () => {
    expect(renderMessage(entry('some new event', { a: 1 }))).toBe('some new event');
    expect(renderMessage(entry('2026-06-01 old text line', {}, true))).toBe('2026-06-01 old text line');
  });
});

describe('levelLabel', () => {
  it('maps slog levels to Korean', () => {
    expect(levelLabel('DEBUG')).toBe('디버그');
    expect(levelLabel('INFO')).toBe('정보');
    expect(levelLabel('WARN')).toBe('경고');
    expect(levelLabel('ERROR')).toBe('오류');
    expect(levelLabel('WEIRD')).toBe('WEIRD');
  });
});
