# 스타일 매핑: openpyxl → excelize

## 핵심 차이
| openpyxl | excelize |
|----------|----------|
| 셀별 `.font`/`.border`/`.alignment` 독립 누적 | 셀당 단일 스타일 ID (`*excelize.Style`) |
| 겹치는 border 패스가 이전 border를 덮어씀 | `SetCellStyle(sheet, h, v, styleID)` 한 번에 |
| `Font(name,size,bold,color)` | `Style.Font = &excelize.Font{Family,Size,Bold,Color}` |
| `Side(border_style,color)` | `Style.Border = []excelize.Border{{Type,Color,Style}}` |
| `Alignment(horizontal,vertical,wrap_text)` | `Style.Alignment = &excelize.Alignment{Horizontal,Vertical,WrapText}` |

## 합성 전략 (필수)
openpyxl 코드가 border를 영역별로 여러 번 적용하고(예: `B2:M6` thin → 외곽 `A1:N1` medium top → 모서리 2-side), 마지막에 전체 셀 alignment를 덮어쓴다. excelize는 셀당 하나뿐이므로:

1. 레이아웃 진행 중 **셀별 상태 맵** 유지: `map[cell]{borders map[side]style, font, align}`.
2. 같은 셀에 border가 여러 번 오면 **side별로 누적**(left/right/top/bottom 각각 최종값).
3. 전 셀에 center/vcenter/wrap alignment 적용.
4. 끝나면 셀별로 (font,align,border-set) 시그니처를 키로 `NewStyle` 캐시 → `SetCellStyle`로 flush.

```go
type cellStyle struct {
    sides map[string]string // "left"->"thin","top"->"medium" ...
    font  string            // "times"|"title"|""
    // alignment는 전 셀 동일(center/center/wrap)
}
// 시그니처로 styleID 캐싱: map[string]int (서명→NewStyle 결과)
```

## border style 매핑
- openpyxl `border_style='thin'` → excelize `Style: 1`(thin), `'medium'` → `Style: 2`. color `"000000"`.
- excelize Border Type: `"left"|"right"|"top"|"bottom"`.
```go
excelize.Border{Type: "left", Color: "000000", Style: 1} // thin
excelize.Border{Type: "top",  Color: "000000", Style: 2} // medium
```

## 영역별 border (excel_generator.py 매핑)
- 공통: `B2:M6` thin 전체. 외곽 medium: `A1:A18`(left), `N1:N18`(right), `A1:N1`(top), `A18:N18`(bottom). 모서리 A1/N1/A18/N18은 2-side medium.
- 기기 추가: `B2:M7` thin, `B8:M8` top-thin, `B8:B17` left-thin, `M8:M17` right-thin, `B17:M17` bottom-thin.
- 과제 추가(`_apply_project_borders`): `B7:M7` top, `B7:B17` left, `M7:M17` right, `B17:M17` bottom, B17/M17 2-side. `Q20:S20`/`Q24:S24` top+bottom, `P21:P23`/`T21:T23` left+right, 모서리 P20/T20/P24/T24 2-side, `Q22:S22` thin 전체.

## 폰트 스타일 ID
- `styleTimes`: Font{Family:"Times New Roman",Size:12,Bold:true,Color:"000000"} + center/center/wrap.
- `styleTitle`: Family TNR Size 16 Bold + center/center/wrap (B4 전용).
- 과제: Q21 Size20 Bold, Q22/R23 Size13 Bold, S23=styleTimes — 모두 center/center/wrap.

## 주의
- 미스타일 셀은 openpyxl이 Calibri11로 남김(빈 셀이라 시각 무관). 골든 비교는 의미 단위로.
- `eq_doc_year`는 int로 `SetCellValue` → 숫자 셀(따옴표/텍스트 아님) 확인.
