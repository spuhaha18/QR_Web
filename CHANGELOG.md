# Changelog

All notable changes to this project will be documented in this file.

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
