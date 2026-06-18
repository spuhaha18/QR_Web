#!/usr/bin/env python3
"""현재 Python 앱으로 골든 .xlsx 매트릭스를 생성한다 (Go 패리티 오라클).

프로젝트 루트에서 .venv 활성화 후 실행. ExcelLabelGenerator를 직접 호출해
(doc_type × binder × 단일/멀티) 조합의 .xlsx를 testdata/golden/에 저장한다.

usage: python capture_golden.py [output_dir]   # 기본 testdata/golden
주의: paste 모드는 실제 QR PNG가 필요 — 더미 PNG를 생성해 사용한다.
auto 모드 골든은 별도(앱 /api/create_label 경유)로 캡처 권장.
"""
import os
import sys
import io

# 프로젝트 루트를 import path에 추가 (이 스크립트가 .claude/skills/... 하위에 있으므로)
ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", ".."))
sys.path.insert(0, ROOT)

from excel_generator import ExcelLabelGenerator  # noqa: E402


def dummy_png(n=64):
    from PIL import Image
    buf = io.BytesIO()
    Image.new("RGB", (n, n), "black").save(buf, format="PNG")
    return buf.getvalue()


def write_dummies(folder, count):
    paths = []
    for i in range(count):
        p = os.path.join(folder, f"qr_{i}.png")
        with open(p, "wb") as f:
            f.write(dummy_png())
        paths.append(p)
    return paths


EQUIP = {
    "eq_number": "EQ-001", "eq_doc_number": "DOC-100", "eq_doc_title": "장비 검교정 기록",
    "eq_doc_count": 1, "eq_doc_department": "품질관리부", "eq_doc_year": 2026,
}
PROJ = {
    "pjt_number": "PJT-7", "pjt_test_number": "T-42", "pjt_doc_title": "안정성 시험 보고서",
    "pjt_doc_writer": "홍길동", "pjt_doc_count": 1,
}


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, "testdata", "golden")
    os.makedirs(out, exist_ok=True)
    tmp = os.path.join(out, "_tmp")
    os.makedirs(tmp, exist_ok=True)
    gen = ExcelLabelGenerator(out)

    matrix = []
    for binder in (1, 3, 5, 7):
        for count in (1, 3):
            matrix.append(("1", binder, count, dict(EQUIP, eq_doc_count=count)))
    for binder in (3, 5, 7):  # 과제는 1cm 제외
        for count in (1, 3):
            matrix.append(("2", binder, count, dict(PROJ, pjt_doc_count=count)))

    for doc_type, binder, count, data in matrix:
        paths = write_dummies(tmp, count)
        filepath, filename = gen.create_label_excel(doc_type, binder, data, qr_image_paths=paths)
        tag = f"t{doc_type}_b{binder}_n{count}"
        dest = os.path.join(out, f"{tag}.xlsx")
        os.replace(filepath, dest)
        print(f"captured {dest}")

    print(f"\nDone. {len(matrix)} golden files in {out}")


if __name__ == "__main__":
    main()
