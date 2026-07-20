# Changelog

All notable changes to this project will be documented in this file.

## [3.1.0] - 2026-07-20

### Changed
- 라벨 출력 xlsx → PDF 전환: `internal/pdf`(Times·바탕 폰트 임베드, 한/영 run 분리 폰트 라우팅, 텍스트 자동 축소, A4 셸프 패킹), 실물 인쇄 실측 보정(7cm 라벨 47×150mm), 바탕 합성 볼드
- 로그 개선: slog JSON lines + lumberjack 로테이션(10MB×5), request ID 추적, 액세스 로그(비즈니스 라우트만), 뷰어 한글 렌더/레벨 뱃지/요청 필터. `/api/logs`의 `search` 파라미터 제거(클라이언트 검색), clear의 `backup_file` 응답 제거
- 로거 Init에 로그 경로 쓰기 프로브 추가 — 경로 쓰기 불가 시 기동 실패(fail-fast)

### Removed
- 레거시 Python(Flask) 앱, xlsx 생성(`internal/excel`)·골든 패리티 자산, Docker 구성 — PDF 경로가 대체

## [3.0.0] - 2026-06-18

### Changed
- Python(Flask) → Go(Fiber) + Vite/Svelte SPA 전면 재작성 — 단일 정적 바이너리(`embed.FS`), 스트리밍 응답, CP949 QR 인코딩 유지, 레거시 출력과 골든 패리티 검증

## [2.1.1.0] - 2026-05-19

### Changed
- Extracted document schema and validation from `app.py` into `document_schema.py` — `parse_label_request`, `ValidationError`, field constants, and `EquipmentLabel`/`ProjectLabel` dataclasses now live in a single schema module
- Replaced inline binder layout config dict in `excel_generator.py` with `label_layout.get_qr_config()` — binder size → QR cell placement is now a named, testable seam
- Replaced daemon-thread file deletion helpers (`delete_file_later`, `delete_dir_later`) with `FileLifecycleManager` — cleanup state is now observable via `pending()` and testable without race conditions
- `ExcelLabelGenerator` now delegates cell values, QR payloads, and document counts to label objects rather than computing them inline

### Fixed
- Path traversal vulnerability in `/download/<filename>` — now uses `werkzeug.utils.safe_join` to prevent directory escape
- Unbounded input in `/api/qr_image` and `/api/qr_image_base64` — enforces 500-character limit on QR text

### Added
- `document_schema.py` — schema, validation, and label dataclasses (`EquipmentLabel`, `ProjectLabel`, `make_label`, `get_doc_count`)
- `label_layout.py` — binder size → QR placement config (`get_qr_config`)
- `file_lifecycle.py` — thread-safe deferred cleanup registry (`FileLifecycleManager`)
- 67 tests across 7 test modules (up from 15)
