# CLAUDE.md

## 하네스: QR_Web Go+Vite 개발

**목표:** QR/Excel 라벨 앱을 Go(Fiber)+Svelte SPA 단일 정적 바이너리로 개발·마이그레이션·유지한다.

**트리거:** Go 백엔드 / Svelte 프론트 / Excel 라벨 생성 / QR / 마이그레이션 / 패리티 검증 관련 작업 요청 시 `qr-web-dev` 스킬을 사용하라. 단순 질문(개념 설명, 단일 파일 조회)은 직접 응답 가능.

**팀:** excel-parity-engineer · go-backend-engineer · svelte-frontend-engineer · parity-qa (정의는 `.claude/agents/`, 스킬은 `.claude/skills/`).

**변경 이력:**
| 날짜 | 변경 내용 | 대상 | 사유 |
|------|----------|------|------|
| 2026-06-18 | 초기 구성 | 전체 | Python Flask → Go(Fiber)+Svelte SPA 전면 재작성 하네스 구축 |
