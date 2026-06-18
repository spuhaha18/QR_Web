# D — Phase 2: internal/label/layout.go

## 구현 내용
`label_layout.py`의 `get_qr_config`를 1:1 포팅. (`internal/label/layout.go`)

- `type QRConfig struct { ColumnWidth float64; CellPos string }` — Python dict의 `column_width`/`cell_pos` 대응.
- `binderTable map[int]binderEntry` — `_BINDER_QR_CONFIG` 미러:
  - 7 → {1.875, eq E9, pjt E8}
  - 5 → {1.25, eq D9, pjt D8}
  - 3 → {1.0, eq D9, pjt D8}
  - 1 → {0.75, eq B9, pjt B9}
- `defaultBinder = 3` — `_DEFAULT_BINDER_SIZE`.
- `func GetQRConfig(docType string, binder int) QRConfig`:
  - binder 미존재 → 3 폴백.
  - docType `"1"` → equipmentCell, 그 외 → projectCell (Python `'equipment_qr_cell' if doc_type=='1' else 'project_qr_cell'`와 동일).

스코프 준수: layout.go만 작성. schema 관련 타입(Label 인터페이스, EquipmentLabel 등)은 미선언 — 다른 에이전트의 schema.go와 충돌 방지.

## 테스트 결과
`internal/label/layout_test.go` — `tests/test_label_layout.py` 포팅. 함수명 `TestGetQRConfig_*` 접두로 schema_test.go와 충돌 회피.

- `go vet ./internal/label/` : clean (출력 없음).
- `go test ./internal/label/ -run TestGetQRConfig -v` : 8/8 PASS.
  - Equipment/Project 7cm (CellPos + ColumnWidth), 5cm, 3cm, 1cm 양 타입 동일, 미지(99)→3 폴백, ColumnWidth 반환.

schema.go가 아직 패키지에 없지만 layout.go는 자기완결이라 패키지가 standalone으로 컴파일·테스트됨. schema 통합 후 전체 `go test ./internal/label/` 재실행 권장(이름 충돌 없음).

## Phase 3 (internal/excel 생성기)에서 이 config 사용법
QR 임베드 단계에서:
```go
cfg := label.GetQRConfig(docType, binder)
f.SetColWidth(sheet, "B", "M", cfg.ColumnWidth)   // Python ord('B')..ord('N')-1 = B..M 루프 → 범위 1회
// QR PNG는 cfg.CellPos 셀 좌상단 one-cell 앵커, 75x75px 절대크기 (ScaleX=75/srcW, ScaleY=75/srcH)
f.AddPictureFromBytes(sheet, cfg.CellPos, &excelize.Picture{...})
```
- 각 시트(멀티시트 시 CopySheet 복제본 포함)마다 동일 cfg 적용.
- docType은 ParseLabelRequest가 반환하는 "1"/"2" 문자열 그대로 전달.
- CellPos는 시트별로 동일(바인더/타입에만 의존, 시트 인덱스 무관).
