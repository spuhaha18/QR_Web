#!/usr/bin/env python3
"""HTTP E2E: reproduce the 14-case golden matrix via real POST /create_label
(paste mode, multipart) against the running Go binary, then compare each
returned .xlsx against testdata/golden/ with compare_xlsx.py.

Inputs mirror capture_golden.py EQUIP/PROJ exactly + dummy 64x64 black PNGs.
qr_order = [0..n-1] identity (same order capture_golden used → identity).

usage: python e2e_http_matrix.py [BASE_URL] [OUT_DIR]
"""
import io
import json
import os
import sys
import subprocess
import urllib.request

BASE = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:5097"
OUT = sys.argv[2] if len(sys.argv) > 2 else "/tmp/e2e_cand"
ROOT = "/home/spuhaha18/Project/QR_Web"
GOLDEN = os.path.join(ROOT, "testdata", "golden")
CMP = os.path.join(ROOT, ".claude/skills/parity-qa/scripts/compare_xlsx.py")
PY = os.path.join(ROOT, ".venv/bin/python")

os.makedirs(OUT, exist_ok=True)

EQUIP = {
    "eq_number": "EQ-001", "eq_doc_number": "DOC-100", "eq_doc_title": "장비 검교정 기록",
    "eq_doc_count": 1, "eq_doc_department": "품질관리부", "eq_doc_year": 2026,
}
PROJ = {
    "pjt_number": "PJT-7", "pjt_test_number": "T-42", "pjt_doc_title": "안정성 시험 보고서",
    "pjt_doc_writer": "홍길동", "pjt_doc_count": 1,
}


def dummy_png():
    from PIL import Image
    buf = io.BytesIO()
    Image.new("RGB", (64, 64), "black").save(buf, format="PNG")
    return buf.getvalue()


def multipart(fields, files):
    """Build multipart/form-data body. fields: list[(k,v)]. files: list[(k,fname,bytes)]."""
    boundary = "----qae2eBoundary7MA4YWxkTrZu0gW"
    crlf = b"\r\n"
    body = io.BytesIO()
    for k, v in fields:
        body.write(b"--" + boundary.encode() + crlf)
        body.write(f'Content-Disposition: form-data; name="{k}"'.encode() + crlf + crlf)
        body.write(str(v).encode("utf-8") + crlf)
    for k, fname, data in files:
        body.write(b"--" + boundary.encode() + crlf)
        body.write(
            f'Content-Disposition: form-data; name="{k}"; filename="{fname}"'.encode()
            + crlf
        )
        body.write(b"Content-Type: image/png" + crlf + crlf)
        body.write(data + crlf)
    body.write(b"--" + boundary.encode() + b"--" + crlf)
    return body.getvalue(), boundary


def build_matrix():
    m = []
    for binder in (1, 3, 5, 7):
        for count in (1, 3):
            m.append(("1", binder, count, dict(EQUIP, eq_doc_count=count)))
    for binder in (3, 5, 7):
        for count in (1, 3):
            m.append(("2", binder, count, dict(PROJ, pjt_doc_count=count)))
    return m


def main():
    png = dummy_png()
    results = []
    for doc_type, binder, count, data in build_matrix():
        tag = f"t{doc_type}_b{binder}_n{count}"
        fields = [("doc_type", doc_type), ("binder_size", str(binder))]
        for k, v in data.items():
            fields.append((k, v))
        fields.append(("qr_order", json.dumps(list(range(count)))))
        files = [("qr_images", f"qr_{i}.png", png) for i in range(count)]
        body, boundary = multipart(fields, files)
        req = urllib.request.Request(
            f"{BASE}/create_label", data=body, method="POST",
            headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
        )
        try:
            resp = urllib.request.urlopen(req)
            status = resp.status
            cd = resp.headers.get("Content-Disposition", "")
            ct = resp.headers.get("Content-Type", "")
            content = resp.read()
        except urllib.error.HTTPError as e:
            status = e.code
            cd = e.headers.get("Content-Disposition", "")
            ct = e.headers.get("Content-Type", "")
            content = e.read()
        dest = os.path.join(OUT, f"{tag}.xlsx")
        with open(dest, "wb") as f:
            f.write(content)
        sig = content[:2]
        golden = os.path.join(GOLDEN, f"{tag}.xlsx")
        verdict = "?"
        cmpout = ""
        if status == 200 and sig == b"PK":
            p = subprocess.run([PY, CMP, golden, dest], capture_output=True, text=True)
            cmpout = (p.stdout + p.stderr).strip()
            verdict = "MATCH" if p.returncode == 0 else "MISMATCH"
        else:
            verdict = f"HTTP{status}"
            cmpout = content[:300].decode("utf-8", "replace")
        results.append((tag, status, ct, cd, sig == b"PK", verdict, cmpout))
        print(f"{tag}: status={status} sig={'PK' if sig==b'PK' else sig!r} -> {verdict}")
        if verdict not in ("MATCH",):
            print("   ", cmpout[:800])

    n_match = sum(1 for r in results if r[5] == "MATCH")
    print(f"\n=== {n_match}/{len(results)} MATCH ===")
    # sample header echo for contract check
    if results:
        print("sample Content-Type:", results[0][2])
        print("sample Content-Disposition:", results[0][3])
    sys.exit(0 if n_match == len(results) else 1)


if __name__ == "__main__":
    main()
