# D — Phase 3: Excel 코어 (excel-parity-engineer 완료 보고)

## 결과: 14/14 골든 MATCH ✅

`go test ./internal/excel/ -run TestGoldenParity -v` → 14 서브테스트 전부 PASS.
`go vet ./internal/excel/` clean, `go build ./internal/...` 통과, `gofmt -l` clean.

## 구현 파일
- `internal/excel/styles.go` — 합성 스타일 ID 빌더(`styleBuilder`). openpyxl의 셀별 border **replace** 시맨틱을 셀상태 맵으로 재현하고, (font, border-set) 시그니처로 `NewStyle` 캐싱 후 `SetCellStyle` flush. 전 셀 center/center/wrap alignment 동봉.
- `internal/excel/generator.go` — `Generator.CreateLabelExcel(docType string, binder int, lbl label.Label, qrPNGs [][]byte) (data []byte, filename string, err error)`. 시트1 빌드(레이아웃→공통테두리→기기/과제분기→flush) + 멀티시트 CopySheet + QR 임베드 + col-span 확장 후 bytes 반환.
- `internal/excel/colsplit.go` — `expandColSpans`: excelize가 동일폭 인접열을 하나의 `<col min max>`로 coalesce하는 것을 포스트-시리얼라이즈 XML 변환으로 열당 1개 `<col min=x max=x>`로 분해(골든 패리티 위해 필수).
- `internal/excel/parity_test.go` — 14 매트릭스 Go 생성 → `compare_xlsx.py`로 골든 비교(`TestGoldenParity`), paste/project 스모크(`TestExcelPasteMode`/`TestExcelProjectLabel`). capture_golden.py와 동일 입력값 + 동일 수 64x64 검정 PNG.

## 시그니처 확정 통보 (go-backend-engineer)
```go
func (g *Generator) CreateLabelExcel(docType string, binder int, lbl label.Label, qrPNGs [][]byte) (data []byte, filename string, err error)
```
- `qrPNGs[i]` = 시트 i의 QR PNG(paste 모드 순서). `nil`이면 QR 미삽입(auto 모드는 핸들러가 PNG 생성 후 주입).
- filename = `label.GenerateTimestampFilename(lbl.DocNumber(), "xlsx")`.
- 시트명 "Sheet 1"/"Sheet 2"... (공백 포함).

## excelize 함정 & 해결 내역
### ⚠️ 환경: 설치 버전 불일치 (가장 큰 발견)
- 과제 메모는 excelize v2.10.1이라 했으나, 모듈 캐시엔 **v1.4.1(360EntSecGroup-Skylar 구 경로)**만 있었고 go.mod엔 excelize 자체가 없었음. v1.4.1은 `WriteToBuffer`/`CopySheet`/`SetDefinedName`/oneCell positioning 미지원 → 스킬과 비호환.
- **해결**: `go get github.com/xuri/excelize/v2@v2.10.1`로 정식 v2 추가. go.mod에 등록됨. 이후 모든 v2 API 시그니처 godoc 직접 확인 후 사용.

### 위험 #1 — 스타일 합성 (border replace 시맨틱)
- 스킬은 "side별 union 누적"을 제안했으나, 골든을 직접 디코드해보니 openpyxl은 `cell.border = Border(...)`가 **전체 교체**(union 아님). 예: B8은 B8:M8(top)→B8:B17(left) 순서로 최종 `{left}`만, M8은 `{right}`만. 따라서 `setBorderCell`을 **full replace**로 구현, Python 패스 순서 그대로 재생.
- 비교기는 alignment를 비교하지 않지만 시각 패리티 위해 전 styled 셀에 center/center/wrap 적용.

### 위험 #2 — 이미지 앵커
- `AddPictureFromBytes(sheet, cell, *Picture)` (v2). PNG 디코드로 srcW/H 구해 `ScaleX=75/srcW`, `ScaleY=75/srcH`, `Positioning:"oneCell"`, offset 0. 골든 `images` 앵커셀(기기 D9/E9·B9, 과제 D8/E8·B9) 14케이스 전부 일치.

### 위험 #3 — CopySheet 충실도
- v2 `CopySheet(from,to int)`가 **병합·행높이·열너비·셀스타일 모두 보존** 확인됨(t*_n3 멀티시트 6케이스 모두 시트2/3까지 MATCH). 폴백(전체 레이아웃 재실행) 불필요.
- 복제 후 B5(과제는 S23) i/N 덮어쓰기 + 과제는 시트별 print_area(`_xlnm.Print_Area`, 시트 scope) 재설정.

### 추가 함정 — col coalescing (해결의 핵심)
- excelize는 B..M 동일폭을 단일 `<col min=2 max=13>`로 묶어 출력. openpyxl 로더는 이를 첫 열(B) 한 항목으로만 등록 → 비교기 `col_widths`가 `{B:1.0}`로 골든 `{B..M:1.0}`과 불일치(유일했던 MISMATCH 카테고리).
- **해결**: `expandColSpans`로 WriteToBuffer 후 zip 내 `xl/worksheets/sheet*.xml`의 `<col>` 스팬을 열당 1개로 재작성. 순수 문자열 변환(스키마 무변경). 이후 14/14 MATCH.

## 14케이스 MATCH 결과표
| 케이스 | 시트수 | 결과 |
|---|---|---|
| t1_b1_n1 / n3 | 1 / 3 | PASS / PASS |
| t1_b3_n1 / n3 | 1 / 3 | PASS / PASS |
| t1_b5_n1 / n3 | 1 / 3 | PASS / PASS |
| t1_b7_n1 / n3 | 1 / 3 | PASS / PASS |
| t2_b3_n1 / n3 | 1 / 3 | PASS / PASS |
| t2_b5_n1 / n3 | 1 / 3 | PASS / PASS |
| t2_b7_n1 / n3 | 1 / 3 | PASS / PASS |

비교 항목(전부 일치): 시트명/수, 셀값(B7 int 유지), 병합, 열너비(B..M 개별), 행높이, 이미지 앵커셀, 셀별 테두리(변+style), 셀별 폰트.

## 남은 이슈 / 주의
- 없음(14/14 MATCH). 단, **go.mod에 excelize/v2 v2.10.1을 추가했음** — 다른 에이전트/CI는 이 의존성 가정.
- `parity_test.go`의 `TestGoldenParity`는 `.venv/bin/python`+openpyxl 필요. 없으면 자동 Skip. 스모크 테스트는 python 무관.
- auto 모드(qrPNGs 생성)는 httpx 핸들러 Phase 담당. 본 패키지는 paste용 `[][]byte` 주입 인터페이스만 제공(nil 허용).

---

## 정리 작업 (post-Phase3, parity-qa 비차단 지적 제거) — 후처리 제거 + 비교기 col-span 정규화

parity-qa의 `QA_report.md` 비차단 권고(colsplit XML 후처리가 excelize 출력포맷에 silent 결합 →
버전업 시 조용히 깨질 위험)를 제거했다. col-span vs per-col은 Excel 렌더링상 완전 동일하므로,
**산출물을 후처리하는 대신 비교기를 col-span 인식으로 고쳤다.**

### 변경 내역
1. **`internal/excel/colsplit.go` 삭제.** `expandColSpans`/`expandColsBlock`/regex 전부 제거.
   Go 출력에 잔여 참조 없음(`grep` 0건).
2. **`internal/excel/generator.go` 수정.** `CreateLabelExcel`이 `WriteToBuffer().Bytes()`를
   **그대로 반환**(후처리 호출 삭제). excelize 자연 출력 = `<col min=2 max=13>` coalesced span 유지.
   `applyQRCodes`의 expandColSpans 언급 주석도 정리.
3. **`.claude/skills/parity-qa/scripts/compare_xlsx.py` 수정.** `col_widths()` 헬퍼 추가:
   양쪽 워크북의 `ws.column_dimensions` 각 `ColumnDimension`을 `min..max` 범위로 전개해
   **개별 열 letter→너비 맵**으로 정규화 후 비교. 골든(openpyxl per-col)과 candidate(excelize
   coalesced) 둘 다 동일 정규화. 다른 비교 항목(값/병합/행높이/이미지앵커/테두리/폰트)은 불변.

### 재검증 (자연 출력 기준, 후처리 없음)
- `go build ./...` / `go vet ./internal/excel/` / `go test -count=1 ./internal/excel/` 전부 green.
- `TestGoldenParity` 14 서브테스트 전부 **PASS** (수정된 compare_xlsx.py로 비교).
- **negative test로 비교기 견고성 입증**: binder3(width 1.0) 골든 vs binder1(width 0.75) candidate
  비교 시 col_widths 전 열(B..M)에서 차이 검출 → MISMATCH. 정규화는 동등 전개일 뿐, 너비값 비교는
  유지됨(느슨해지지 않음).
- **자연 출력 sheet1.xml 확인**: `<col customWidth="true" max="13" min="2" width="1">` coalesced
  span 그대로(후처리 제거 입증). 비교기가 B..M 개별 열로 펼쳐 골든과 MATCH.
- **멀티시트/print_area/이미지앵커 무영향 확인**: proj_n3 자연 출력에서 시트별 print_area
  `'Sheet N'!$A$1:$T$24`, B5/S23 i/N(1/3·2/3·3/3), 이미지 앵커 col=3(D열) 정상 + golden t2_b5_n3
  대비 compare MATCH. zip 무결성 `unzip -t` No errors.

### 결과: 14/14 재MATCH ✅ (excelize 자연 출력, XML 후처리 제거, 버전 결합 해소)
