// Korean rendering catalog for the log viewer. The backend logs stable English
// event keys (msg) with structured fields; this module is the only place that
// translates them. Unknown keys and legacy (pre-JSON) lines fall back to the
// raw msg, so catalog drift degrades to English instead of breaking.

export interface LogEntry {
  time: string;
  level: string;
  msg: string;
  legacy?: boolean;
  fields: Record<string, unknown>;
}

type Renderer = (f: Record<string, unknown>) => string;

const modeLabels: Record<string, string> = {
  paste: '붙여넣기',
  auto: '자동',
};

const catalog: Record<string, Renderer> = {
  'server started': (f) => `서버 시작 (${f.addr})`,
  'request': (f) => `${f.method} ${f.path} → ${f.status} (${f.duration_ms}ms)`,
  'label generated': (f) => `라벨 생성: ${f.file} (${modeLabels[String(f.mode)] ?? f.mode}, ${f.ip})`,
  'logs cleared': () => '로그 초기화됨',
  'log clear failed': (f) => `로그 초기화 실패: ${f.err}`,
  'server exited': (f) => `서버 종료: ${f.err}`,
};

export function renderMessage(e: LogEntry): string {
  if (e.legacy) return e.msg;
  const render = catalog[e.msg];
  return render ? render(e.fields) : e.msg;
}

const levelLabels: Record<string, string> = {
  DEBUG: '디버그',
  INFO: '정보',
  WARN: '경고',
  ERROR: '오류',
};

export function levelLabel(level: string): string {
  return levelLabels[level] ?? level;
}
