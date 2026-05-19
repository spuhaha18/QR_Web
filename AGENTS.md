# Repository Guidelines

## Project Structure & Module Organization

This is a Python 3.13 Flask application for generating QR label Excel files. Core app logic lives at the repository root: `app.py` defines routes and form handling, `qr_generator.py` builds QR images, `excel_generator.py` creates workbook output, and `config.py` manages settings.

Schema, validation, and label dataclasses (`EquipmentLabel`, `ProjectLabel`, `parse_label_request`, `get_doc_count`, `make_label`) live in `document_schema.py`. Binder size → QR cell placement configuration is provided by `label_layout.get_qr_config()` in `label_layout.py`. Deferred cleanup of temporary files and directories is handled by `FileLifecycleManager` (singleton: `file_lifecycle`) in `file_lifecycle.py`; do not use bare `threading.Timer` calls for file deletion.

Shared helpers are in `utils.py`, `cache_manager.py`, and `performance_monitor.py`. HTML templates live in `templates/`; CSS and static assets live in `static/`. Tests live under `tests/` and use pytest. Docker runtime files are `Dockerfile` and `docker-compose.yml`. `app_backup.py` is historical backup code; avoid extending it unless restoring behavior.

## Build, Test, and Development Commands

- `uv add -r requirements.txt`: install Python dependencies into the uv-managed environment.
- `uv run app.py`: run the Flask app locally at `http://localhost:5000`.
- `python -m venv venv && pip install -r requirements.txt`: create a traditional virtualenv when uv is unavailable.
- `python app.py`: run the app from an activated virtualenv.
- `docker build -t qr-web:v1.1 .`: build the container image.
- `docker-compose up -d`: start the app in the background using Compose.

## Coding Style & Naming Conventions

Use 4-space indentation and standard Python naming: `snake_case` for functions and variables, `PascalCase` for classes, and uppercase names for constants. Keep route handlers thin when possible; move QR, Excel, validation, caching, and monitoring behavior into focused helper modules. Prefer explicit validation and clear error responses for user-submitted form data. Template files should use descriptive names such as `api_docs.html` or `logs.html`; static files should remain grouped by asset type under `static/`.

## Testing Guidelines

Tests live under `tests/` and run with `uv run pytest` (or `python -m pytest`). Current coverage includes `document_schema`, `label_layout`, `file_lifecycle`, `excel_generator`, and related helpers — 67 tests across 7 modules. For new behavior, add focused tests before or with the change using names like `test_<module>.py`. Use Flask's test client for routes and direct unit tests for QR generation, workbook creation, validation helpers, and config selection. Always verify `uv run app.py`, the main form flow, Excel downloads, and relevant API responses before shipping.

## Commit & Pull Request Guidelines

Recent history uses concise conventional prefixes such as `fix:`, `feat:`, `build:`, and `docs:`. Follow that pattern, for example `fix: validate binder size input`. Pull requests should include a short summary, test or manual verification notes, linked issues when available, and screenshots for UI changes. Call out configuration, Docker, or environment-variable changes explicitly.

## Security & Configuration Tips

Use environment variables for sensitive or deployment-specific settings: `SECRET_KEY`, `FLASK_ENV`, and `FLASK_PORT`. Do not commit generated files, local virtualenvs, secrets, or production data.
