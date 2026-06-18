# 멀티시트 i/N: copy_worksheet → CopySheet

## openpyxl 현재 동작
Sheet 1 완성 후 2..N 복제:
```python
for i in range(2, doc_count + 1):
    dest = wb.copy_worksheet(wb['Sheet 1'])
    dest.title = f"Sheet {i}"
    dest["B5"].value = f"{i}/{doc_count}"
    if doc_type == '2':                # 과제
        dest["S23"].value = f"{i}/{doc_count}"
        dest.print_area = 'A1:T24'
    # 전 셀 center/vcenter/wrap 재적용 + B4 = FONT_TITLE
```

## excelize 재현
excelize에는 `copy_worksheet`가 없다. `CopySheet(fromIdx, toIdx int)` 사용:
```go
src, _ := f.GetSheetIndex("Sheet 1")
for i := 2; i <= docCount; i++ {
    name := fmt.Sprintf("Sheet %d", i)
    to, _ := f.NewSheet(name)
    if err := f.CopySheet(src, to); err != nil { /* 폴백: 전체 레이아웃 재실행 */ }
    f.SetCellValue(name, "B5", fmt.Sprintf("%d/%d", i, docCount))
    if docType == "2" {
        f.SetCellValue(name, "S23", fmt.Sprintf("%d/%d", i, docCount))
        setPrintArea(f, name) // _xlnm.Print_Area
    }
    // 전 셀 alignment + B4 title 스타일 재적용
}
```

## CopySheet 충실도 검증 (parity 위험 #3)
Phase D에서 반드시 확인 — `CopySheet`가 다음을 복제하는가:
- 병합(MergeCell) ✓ 기대
- 열너비/행높이 ✓ 기대
- 셀 스타일(테두리/폰트) ✓ 기대
- 임베드 이미지 — **불확실**. 그래서 QR은 모든 시트 생성 후 시트별로 따로 임베드(현재 코드 순서와 동일).

복제가 무언가 누락하면 **폴백**: CopySheet 대신 각 시트에 전체 레이아웃 함수를 다시 실행.

## print_area
```go
f.SetDefinedName(&excelize.DefinedName{
    Name:     "_xlnm.Print_Area",
    RefersTo: "'Sheet 1'!$A$1:$T$24",
    Scope:    "Sheet 1",
})
```
과제 문서의 각 시트마다 자기 이름 scope로 설정.
