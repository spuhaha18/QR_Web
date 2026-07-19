# PDF 라벨 출력 (xlsx 완전 대체)

- 날짜: 2026-07-20
- 상태: 설계 승인됨
- 결정: 접근 A — Go PDF 라이브러리로 서버에서 직접 렌더

## 목표

라벨 다운로드 형식을 xlsx에서 **PDF로 완전 대체**한다.

1. 라벨의 물리 크기는 현재 xlsx를 Excel에서 **100% 배율로 인쇄했을 때의 실측 크기와 동일**해야 한다 (열너비/행높이에서 환산한 mm 기준).
2. 종이 절약: 현재는 1페이지 1라벨 → **A4 세로 한 페이지에 여러 라벨**을 배치한다.
3. 과제 라벨의 보조 표(P20:T24)는 메인 라벨과 **별도 조각**으로 함께 패킹한다.
4. 라벨 조각 사이에는 **약 5mm 간격 + 회색 점선 재단선**을 넣는다.

## 결정 사항 (Q&A 확정)

| 항목 | 결정 |
|------|------|
| 출력 형식 | PDF 완전 대체 (xlsx 다운로드 제거) |
| 용지 | A4 세로 (210×297mm) |
| 크기 기준 | Excel 100% 배율 환산 mm (geometry.go 공식 재사용) |
| 과제 보조 표 | 독립 조각으로 같은 페이지에 패킹 |
| 재단 가이드 | 5mm 간격 + 회색 점선 |
| 렌더 방식 | Go PDF 라이브러리 (go-pdf/fpdf 또는 signintech/gopdf) 벡터 직접 렌더 |
| 폰트 | 라틴/숫자 = Times New Roman, 한글 = 바탕체. 사용자가 원본 파일 제공(times.ttf/timesbd.ttf + batang.ttc). 폰트 크기(12/16/20/13pt)는 기존 유지 |

## 크기 산출 (단일 소스)

기존 `internal/excel/geometry.go`의 OOXML 환산 공식을 재사용하고 px/pt→mm로 확장한다.

- px→mm: `px × 25.4 / 96` (Excel 열너비는 96dpi 픽셀 기준)
- pt→mm: `pt × 25.4 / 72` (행높이는 포인트)

조각 크기:

- **메인 라벨 폭** = colWidthToPx(0.375)×2 + colWidthToPx(binderColW)×12 → mm.
  바인더 크기(1/3/5/7cm)별 열너비(0.75/1.0/1.25/1.875)에 따라 달라진다.
- **메인 라벨 높이** = 행 1–18 높이 합 (rowHeights 테이블) ≈ 154mm. 바인더 무관 공통.
- **과제 보조 표** = P20:T24 영역 환산 ≈ 36×29mm (열 P/T=0.375, Q/S=8.13, R=34.88; 행 20–24 = 2.25/48/34.5/27.75/2.25pt).

정확한 mm 값은 구현 시 환산 함수의 스냅샷 테스트로 고정한다.

## 배치 (packer)

- A4 세로, 페이지 여백 10mm, 조각 간 간격 5mm.
- **행 우선(shelf) 배치**: 조각을 순서대로 현재 행에 좌→우로 놓고, 폭이 넘치면 다음 행, 높이가 넘치면 다음 페이지.
- 요청당 문서 N개 → 메인 라벨 N개(i/N 표기 각각) + 과제면 보조 표 N개 조각. 메인 먼저, 보조 표는 이어서 패킹 (남는 하단 공간 활용).
- 각 조각 외곽에 회색(#999 계열) 점선 재단선을 조각 경계에서 그린다.

## 라벨 렌더러

- 셀 그리드 → mm 좌표 테이블(열 x-오프셋, 행 y-오프셋)을 만들어 기존 레이아웃 명세를 그대로 재현:
  - 테두리: thin ≈ 0.2mm, medium ≈ 0.5mm 선.
  - 병합 셀(B2:M6 등) 텍스트: 병합 영역 중앙 정렬(기존 global alignment와 동일).
  - 셀 값/폰트 의도는 기존 `label.Label.CellValues()` / `CellFonts()` 도메인 계층을 그대로 소비 — 도메인 코드는 변경 없음.
- QR: 기존 `internal/qr` PNG 생성 재사용. 75px ≈ 19.84mm 정사각형을 QR 박스(장비 B8:M17, 과제 B7:M17) 중앙에 배치. `DocType.Layout()`의 QRBoxTopRow/BottomRow 재사용.
- i/N: 시트 복제 대신 라벨 조각을 N개 렌더하며 `Layout().CountCells`(B5, 과제는 S23) 값만 교체.

### 폰트

- 기본 폰트 Times New Roman(times.ttf, bold는 timesbd.ttf), 한글 글리프는 바탕체 폴백 — go-pdf/fpdf의 `SetFallbackFonts` 사용 (혼합 문자열에서 글리프 단위 폴백).
- 폰트 파일은 사용자 제공(사내 Windows의 times.ttf/timesbd.ttf + batang.ttc). `internal/pdf/fonts/`에 두고 `go:embed`로 바이너리에 포함.
- batang.ttc는 TTC 컬렉션 — fpdf가 TTC를 못 읽으면 첫 face를 TTF로 추출해 사용 (구현 시 확인).
- 바탕체는 bold face가 없음 — 한글 굵게는 Excel과 동일하게 synthetic bold(외곽선 보강) 또는 regular로 렌더. 구현 시 시각 비교로 결정.

## 구조 변경

- `internal/pdf` 신설:
  - `geometry.go` — Excel 단위→mm 환산, 조각 크기 계산
  - `renderer.go` — 라벨 1조각 / 보조 표 1조각 렌더
  - `packer.go` — A4 shelf 배치 + 재단선
- `internal/httpx/label_handler.go` — excel 대신 pdf 호출. `Content-Type: application/pdf`, 파일명 `.pdf` (기존 `GenerateTimestampFilename(docNumber, "pdf")`).
- 프론트(Svelte): 다운로드 MIME/확장자 처리 확인 외 변경 최소.
- `internal/excel` 및 관련 패리티 자산: PDF 전환 완료·검증 후 삭제. (전환 커밋에서는 유지, 후속 정리 커밋에서 제거.)

## 에러 처리

- 폰트 임베드 실패/PDF 쓰기 실패는 기존 500 경로와 동일하게 처리.
- 입력 검증(DocType, BinderSize, Label 파싱)은 기존 도메인 계층 그대로 — 변경 없음.

## 테스트

1. **환산 스냅샷**: 바인더 4종 × 문서 2종 조각 크기(mm) 고정값 테스트.
2. **패커 속성 테스트**: 조각 겹침 없음, 페이지 경계(여백 포함) 내 배치, 조각 순서 보존.
3. **렌더 스모크**: 샘플 요청으로 PDF 생성 → 유효한 PDF 헤더/페이지 수 확인.
4. **시각 확인(수동)**: 샘플 PDF 인쇄 후 실측 — 기존 xlsx 100% 인쇄물과 크기 비교.

## 제외 (YAGNI)

- xlsx/PDF 선택 옵션 UI — 완전 대체이므로 없음.
- 용지 크기 선택, 라벨 회전 최적화, 여백 사용자 설정 — 필요 시 후속.
