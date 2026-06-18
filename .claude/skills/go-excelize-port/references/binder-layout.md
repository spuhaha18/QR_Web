# 바인더 레이아웃 config (label_layout.py → Go)

## 원본 (label_layout.py)
| binder | column_width | equipment cell | project cell |
|--------|--------------|----------------|--------------|
| 7 | 1.875 | E9 | E8 |
| 5 | 1.25  | D9 | D8 |
| 3 | 1.0   | D9 | D8 |
| 1 | 0.75  | B9 | B9 |

미지 크기 → binder 3 폴백. doc_type '1'=equipment cell, '2'=project cell.
`column_width`는 QR 임베드 시 B–M 열에 적용.

## Go 구현 (internal/label/layout.go)
```go
package label

type QRConfig struct {
    ColumnWidth float64
    CellPos     string
}

type binderEntry struct {
    columnWidth   float64
    equipmentCell string
    projectCell   string
}

var binderTable = map[int]binderEntry{
    7: {1.875, "E9", "E8"},
    5: {1.25, "D9", "D8"},
    3: {1.0, "D9", "D8"},
    1: {0.75, "B9", "B9"},
}

const defaultBinder = 3

// GetQRConfig: docType "1"=equipment, "2"=project. 미지 binder는 3 폴백.
func GetQRConfig(docType string, binder int) QRConfig {
    e, ok := binderTable[binder]
    if !ok {
        e = binderTable[defaultBinder]
    }
    cell := e.equipmentCell
    if docType != "1" {
        cell = e.projectCell
    }
    return QRConfig{ColumnWidth: e.columnWidth, CellPos: cell}
}
```

## 열너비 적용 (excel_generator.py 대응)
QR 임베드 단계에서 B–M 열에 `config.ColumnWidth` 적용:
```go
f.SetColWidth(sheet, "B", "M", cfg.ColumnWidth)
```
(현재 코드는 `ord('B')..ord('N')-1` = B..M 루프. excelize는 범위 한 번에.)

## 테스트 (test_label_layout.py 대응)
바인더 1/3/5/7 × doc_type 1/2 각각 CellPos·ColumnWidth 단언, 미지 크기(예: 99) → 3 값 폴백 단언.
