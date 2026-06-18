#!/usr/bin/env python3
"""두 .xlsx를 의미 단위로 비교한다 (XML 바이트 비교 아님).

excelize와 openpyxl은 다른 XML을 내므로, 라벨에 의미 있는 속성만 비교한다:
시트명/수, 셀값, 병합 범위, 열너비, 행높이, 이미지 앵커 셀+개수, 셀별 테두리/폰트.

usage: python compare_xlsx.py golden.xlsx candidate.xlsx
종료코드 0 = 일치, 1 = 불일치(차이 출력).
"""
import sys
from openpyxl import load_workbook


def sheet_facts(ws):
    facts = {}
    facts["title"] = ws.title
    facts["merges"] = sorted(str(m) for m in ws.merged_cells.ranges)
    facts["col_widths"] = {
        k: round(v.width, 4) for k, v in ws.column_dimensions.items() if v.width is not None
    }
    facts["row_heights"] = {
        k: round(v.height, 4) for k, v in ws.row_dimensions.items() if v.height is not None
    }
    # 셀값 (값 있는 셀만)
    values = {}
    fonts = {}
    borders = {}
    for row in ws.iter_rows():
        for cell in row:
            if cell.value is not None:
                values[cell.coordinate] = cell.value
            f = cell.font
            if f and (f.bold or (f.name and f.name.lower() == "times new roman")):
                fonts[cell.coordinate] = (f.name, f.size, bool(f.bold))
            b = cell.border
            sides = {s: getattr(b, s).style for s in ("left", "right", "top", "bottom")
                     if getattr(b, s) and getattr(b, s).style}
            if sides:
                borders[cell.coordinate] = sides
    facts["values"] = values
    facts["fonts"] = fonts
    facts["borders"] = borders
    # 이미지 앵커 셀 + 개수
    imgs = []
    for img in getattr(ws, "_images", []):
        anchor = getattr(img, "anchor", None)
        try:
            cell = f"{chr(65 + anchor._from.col)}{anchor._from.row + 1}"
        except Exception:
            cell = str(anchor)
        imgs.append(cell)
    facts["images"] = sorted(imgs)
    return facts


def compare(a_path, b_path):
    a = load_workbook(a_path)
    b = load_workbook(b_path)
    diffs = []
    if a.sheetnames != b.sheetnames:
        diffs.append(f"sheet names: {a.sheetnames} != {b.sheetnames}")
    for name in a.sheetnames:
        if name not in b.sheetnames:
            continue
        fa, fb = sheet_facts(a[name]), sheet_facts(b[name])
        for key in fa:
            if fa[key] != fb[key]:
                diffs.append(f"[{name}] {key}:\n  golden={fa[key]}\n  cand  ={fb[key]}")
    return diffs


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(__doc__)
        sys.exit(2)
    diffs = compare(sys.argv[1], sys.argv[2])
    if diffs:
        print(f"MISMATCH ({len(diffs)} differences):")
        for d in diffs:
            print(" -", d)
        sys.exit(1)
    print("MATCH")
    sys.exit(0)
