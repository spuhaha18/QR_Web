# 설계문서: QR 중앙 정렬 + 로고 개선

> 작성일 2026-06-18 · 상태: 설계 승인(구현 전) · 브랜치 feat/go-vite-migration

## Context

Go(excelize)+Svelte 재작성이 레거시 Python 앱과 패리티를 달성한 뒤(`doc/spec/go-vite-migration.md`), 사용자가 두 가지 **의도적 개선**(패리티 아닌 디자인 변경)을 요청했다:

1. **QR 위치** — 현재 하단 QR 박스 좌상단에 붙어 "오른쪽으로 치우쳐" 보임. 박스 정중앙으로 옮긴다.
2. **회사 로고** — `logo.png` 이미지가 조악함. 흰 배경 + 파란 글자 텍스트 배지로 교체한다.

레거시 Python 앱은 은퇴 예정이라 건드리지 않는다. **Go/Svelte 산출물만 개선**한다. 따라서 QR 위치는 레거시 골든과 의도적으로 달라진다.

---

## 기능 1 — QR 중앙 정렬 (Go / excelize)

### 현재 동작
`internal/label/layout.go`의 `GetQRConfig`가 바인더·문서타입별 `CellPos`(기기 B9/D9/D9/E9, 과제 B9/D8/D8/E8)를 반환하고, `internal/excel/generator.go`가 그 셀 좌상단에 75×75px QR을 oneCellAnchor(offset 0)로 임베드한다.

### 목표 동작
QR을 **하단 QR 박스(테두리로 둘러싸인 내부 영역)의 정중앙**(가로+세로)에 배치한다.
- **박스 범위 = 테두리 영역**: 기기 = `B8:M17`, 과제 = `B7:M17`. 가로는 컬럼 **B–M**(둘 다 공통).
  - 행 높이(point): 기기 박스 = 8–17행 각 6.75 → 합 67.5pt(=90px). 과제 박스 = 7행 27 + 8–17행 각 6.75 → 합 94.5pt(=126px). **과제 7행이 27pt로 큰 점**을 환산에 그대로 반영(테두리 영역 기준 정중앙).
- **QR 크기**: 75×75px(714375 EMU) **고정**(현행 유지).
- **전 범위 균일**: 기기·과제 × 바인더 1/3/5/7cm 전부.

### 계산 방식
박스 크기를 EMU로 구해 중앙 오프셋을 계산한다.
- **가로폭(EMU)**: 컬럼 B–M 각 폭(char 단위, 바인더별 0.75/1.0/1.25/1.875)을 px로 환산 후 EMU(`px × 9525`)로 합산.
  - Excel char→px 환산: `px = round(width × MDW + 5)` (MDW=최대자릿너비, Calibri 11 ≈ 7px). excelize 내부 환산 헬퍼가 있으면 그것을 사용하고, 없으면 이 공식을 단일 함수로 둔다.
- **세로높이(EMU)**: 박스 행(기기 8–17, 과제 7–17) 각 높이(point)를 px(`pt × 4/3`)로 환산 후 EMU 합산.
- **오프셋**: `offX = (boxW − 714375) / 2`, `offY = (boxH − 714375) / 2`. 음수면(1cm처럼 QR>박스) 그대로 사용 → 대칭으로 양옆/위아래 약간 넘침(허용).
- **앵커**: 박스 좌상단 셀(기기 B8 = col 1·row 7, 과제 B7 = col 1·row 6) 기준 oneCellAnchor. `colOff`/`rowOff`가 셀 폭/높이를 넘으면 실제 셀 + 잔여 오프셋으로 정규화(누적 폭/높이로 from-cell 결정 후 잔여를 colOff/rowOff에 둔다).

### 코드 변경
- `internal/label/layout.go`: `QRConfig.CellPos` 폐기. `ColumnWidth`는 박스폭 계산에 계속 사용. 새 함수 또는 박스 정의(`BoxTopLeft`, 박스 행 범위)를 layout이 제공하거나, 박스 기하를 generator가 보유.
- `internal/excel/generator.go`: QR 임베드부를 "중앙 오프셋 계산 → oneCellAnchor(from-cell + colOff/rowOff)"로 교체. 멀티시트 복제본에도 동일 적용(QR은 시트별 임베드라 자연 반영).
- EMU 환산 헬퍼(char→px, pt→px, px→EMU)를 `internal/excel`에 단일 위치로 둔다.

### 패리티/테스트 영향 (중요)
QR 위치가 레거시 골든과 **의도적으로 달라진다.**
- `parity-qa` 비교기 `.claude/skills/parity-qa/scripts/compare_xlsx.py`: **이미지 앵커(`images`) 비교 항목 제외**(주석으로 사유 명시). 셀값/병합/열너비/행높이/시트명/테두리/폰트 패리티는 **유지**.
- `internal/excel/parity_test.go`의 골든 비교: 위 비교기 변경으로 통과 유지. QR 앵커는 더 이상 골든과 같지 않음을 전제.
- **신규 중앙 정렬 테스트** 추가(`internal/excel`): 7케이스(기기 1/3/5/7 + 과제 3/5/7) 각각 생성된 .xlsx의 QR 앵커(from col/row + colOff/rowOff + ext)가 **계산된 박스 중앙**과 일치하는지 단언. 1cm 음수 오프셋 케이스 포함.
- 회귀: `go test ./...` 전부 green 유지.

### 엣지 케이스
- **1cm**: 박스 폭(~63px) < QR(75px) → `offX` 음수 → 대칭 중앙(양옆 ~6px씩 넘침). 의도된 동작. 테스트로 고정.
- 세로: 박스 높이는 75px보다 커서(기기 10행×, 과제 11행×) 음수 아님. 정상 중앙.

---

## 기능 2 — 로고 (Svelte / CSS)

### 현재 동작
`web/frontend/src/App.svelte`가 `logo.png`를 `<img>`로 표시하고, 로드 실패 시에만 `<span class="logo-text">inno.N</span>` 폴백. 배지 `.company-logo` 배경은 `--surface-color`(테마 가변).

### 목표 동작
이미지를 버리고 **텍스트 배지를 상시 표시**. 흰 필(pill) 배경 + 파란 굵은 inno.N.

### 코드 변경
- `App.svelte`: `import logoUrl`·`logoFailed` 상태·`<img>`·`{#if logoFailed}` 분기 제거 → `.company-logo` 안에 `<span class="logo-text">inno.N</span>`만 렌더.
- `style.css`:
  - `.company-logo` 배경을 **흰색 고정**(`#ffffff`) — 다크모드에서도 흰 배경. 필 모양/테두리/그림자 유지.
  - `.logo-text` 파란(`#3b82f6`) 굵은 글자 유지(이미 존재).
  - `.company-logo-img` 규칙 삭제.
- `web/frontend/src/assets/logo.png` 파일 + import 삭제.

### 테스트/검증
- `npm run build` 성공(svelte-check 0/0).
- 빌드 후 `/browse`로 localhost 렌더 확인 — 흰 배지 + 파란 inno.N, 다크모드 토글 시 배지 배경 흰색 유지.

---

## 검증 (End-to-End)

1. `make build` → 단일 바이너리.
2. **QR 중앙**: 7케이스 .xlsx 생성, 중앙 정렬 테스트 green + LibreOffice/육안으로 QR이 하단 박스 가운데 위치 확인.
3. **로고**: `/browse goto localhost:PORT` → 배지 흰 배경/파란 글자, 다크모드 유지.
4. 전체 `go test ./...` green, 비-QR 골든 패리티 유지.

## 비목표 (YAGNI)
- QR 크기 가변(축소) — 1cm은 75px 고정 + 대칭 넘침으로 처리.
- 레거시 Python 앱 변경 — 은퇴 예정, 손대지 않음.
- 로고 SVG/신규 에셋 제작 — 텍스트 배지로 충분.
