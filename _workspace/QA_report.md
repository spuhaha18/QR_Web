# QA Report — Parity QA (독립 검증)

## Phase 3: Excel 코어 — 게이트 판정: **PASS** ✅

excel-parity-engineer의 "14/14 MATCH" 자기주장을, 생성자 코드/테스트를 **신뢰하지 않고**
독립적으로(별도 Go 프로그램으로 재생성 + 비교기 + openpyxl 심층점검 + excelize 왕복) 재검증함.
결과: 게이트 PASS. 자기주장이 사실로 확인됨.

검증 환경: go1.26.4, openpyxl 3.1.5, .venv python3.13, excelize/v2 v2.10.1.

---

### 1. 기준선: 생성자 테스트 재실행
- `go test -count=1 ./internal/excel/` → **ok** (3.78s). green 기준선 확인.
- `TestGoldenParity` 14 서브테스트 전부 RUN/PASS, `go vet` clean.

### 2. 독립 재생성 + 비교기 (생성자 parity_test 미사용)
별도 임시 프로그램(`cmd/_qaverify`, 검증 후 삭제)으로 capture_golden.py와 **동일 입력값**
(EQUIP/PROJ + 동수 64x64 검정 PNG)으로 14 매트릭스를 `/tmp/qa_cand/`에 직접 생성 →
`compare_xlsx.py`로 골든과 1:1 비교.

| 케이스 | 시트 | compare_xlsx | 심층점검 |
|---|---|---|---|
| t1_b1_n1 / n3 | 1 / 3 | MATCH / MATCH | PASS / PASS |
| t1_b3_n1 / n3 | 1 / 3 | MATCH / MATCH | PASS / PASS |
| t1_b5_n1 / n3 | 1 / 3 | MATCH / MATCH | PASS / PASS |
| t1_b7_n1 / n3 | 1 / 3 | MATCH / MATCH | PASS / PASS |
| t2_b3_n1 / n3 | 1 / 3 | MATCH / MATCH | PASS / PASS |
| t2_b5_n1 / n3 | 1 / 3 | MATCH / MATCH | PASS / PASS |
| t2_b7_n1 / n3 | 1 / 3 | MATCH / MATCH | PASS / PASS |

**compare_xlsx 14/14 MATCH, 심층점검 14/14 PASS.**

### 3. 심층점검 (compare_xlsx.py가 놓치는 항목, openpyxl 직접)
비교기 약점 보완 — 다음을 골든 vs Go-cand 직접 대조, 14케이스 전부 일치:
- **print_area**: 과제 전 시트 `A1:T24` 존재 일치(기기는 없음, 골든과 동일).
- **이미지 앵커(시트별, sorted 아님)**: 기기 binder별 D9/E9/B9, 과제 D8/E8/B9 — 시트마다 1개, 앵커셀 골든 일치. (compare_xlsx는 sorted라 시트매핑 손실 → 본 점검에서 시트별 매핑까지 확인)
- **B5 시트별 i/N**: 시트1 "1/N", 멀티 시트2 "2/3", 시트3 "3/3" 일치.
- **과제 S23 시트별 i/N**: 시트1 "1/N", 시트2/3 갱신 일치.
- **B7 타입**: 기기 int(2026) 유지(문자열 아님) 일치.
- **폰트 전체속성**(name/size/bold): B2-B6/B7/Q21(20)/Q22(13)/R23(13)/S23 일치.
- **alignment**: center/center/wrap 일치.
- **셀별 테두리 16샘플**: 외곽 medium(A1/N1/A18/N18 코너 2변), B2:M6 thin, B8/M8/B17/M17, 과제 P20/T24/Q22/Q20/P21/T21 일치.

### 4. colsplit.go 후처리 유효성 (의심 지점 #1 집중)
- **zip 무결성**: `unzip -t` 14파일 전부 "No errors detected".
- **openpyxl load**: 14파일 **경고 0개**, 시트명 정상.
- **excelize OpenReader 왕복**(별도 프로그램): 14파일 전부 재오픈 성공, 시트리스트/B5/B7/colwidth(B==M) 정상 판독 → 후처리가 excelize 자체 파싱도 깨지 않음.
- **expandColSpans 정확성**: WriteToBuffer 직후 excelize는 B-M을 `<col min=2 max=13>` 단일 coalesce(raw 덤프로 입증). 후처리 후 sheet XML의 모든 `<col>`이 min==max(14개 개별)로 분해 확인. 너비값(A/N=0.375, B-M=바인더값) 보존.
- **다른 XML 영역 무손상**: regex가 `<cols>.*?</cols>` 블록만 치환. 압축방식(Deflated) 보존. zip 멤버 전부 보존.
- **이미지 dedup**(부수 관찰): excelize는 동일 PNG를 image1.png 1개로 dedup, 3시트 drawing이 모두 `../media/image1.png` 참조 + 각자 앵커(D9 등) 보유. 골든은 image1/2/3.png 3개. **시각/구조 동등**, 디스크 절약일 뿐 — 패리티 영향 없음.
- **inline string vs sharedStrings**(부수 관찰): Go는 sharedStrings.xml 사용, 골든은 inline. 셀값 심층점검 통과 → 의미 동등.

### 5. 의심 지점 #2: border full-replace
styles.go `setBorderCell`이 union 아닌 full-replace 구현. 골든의 openpyxl `cell.border=Border(...)`
전체교체 시맨틱과 일치. 셀별 테두리 16샘플 직접 대조 결과 골든과 정확히 일치(예: B8={left}, M8={right},
외곽 medium, 과제 우측패널 Q22 4변 thin). **올바른 구현.**

---

## colsplit XML 후처리 위험 평가 (정직한 평가)

게이트는 PASS이나, `colsplit.go`의 zip-XML 후처리 접근은 다음 **유지보수 취약성**을 가짐:

1. **excelize 버전 결합**: `expandColSpans`는 excelize가 `<cols>`를 단일 라인에, span 형태로
   직렬화한다는 **현재 출력 포맷에 암묵 의존**. excelize 업그레이드로 출력이 바뀌면(예: `<cols>`와
   `</cols>` 사이 newline 삽입) regex `<cols>.*?</cols>`(non-dotall)가 **silent하게 매칭 실패** →
   분해 안 되고 비교기 MISMATCH 재발. 컴파일 에러 없이 조용히 깨짐.
2. **시각 동등성**: col-span vs per-col은 Excel 렌더링상 **완전 동일**(둘 다 B-M 동일폭). 따라서
   "비교기 끼워맞춤"이 시각 결과를 왜곡하진 않음 — 진짜 시각 동등은 보존됨. 그러나 후처리는
   "openpyxl 로더가 coalesced span을 첫 열에만 등록하는 한계"를 우회하려는 것.
3. **더 깨끗한 대안**: `compare_xlsx.py`를 **col-span 인식**하도록 고쳤다면(coalesced `<col>`을
   읽어 max-min 만큼 펼쳐 비교) 후처리 없이 동일 MATCH 달성 가능 + Go 산출물은 excelize 자연 출력
   유지 → 버전 결합 제거. 즉 **비교기를 고치는 게 산출물을 후처리하는 것보다 깨끗했음**.
   (다만 현재 구현도 zip/excelize/openpyxl 왕복 무손상이 입증되어 기능상 문제는 없음.)

**권고(excel-parity-engineer 대상, FAIL 아님 / 개선 제안)**:
- 단기: `colsplit.go`의 `colsBlockRe`에 `(?s)` 플래그를 추가하고, expandColSpans가
  치환 횟수 0일 때(=`<cols>` 미발견 시 경고/에러) 가드를 두면 silent 실패를 방어 가능.
- 중기: 비교기를 col-span 인식으로 바꿔 후처리 제거 검토.

---

## 발견 이슈 요약
- **게이트 차단 이슈: 없음.** 14/14 의미 동등, 후처리본 유효(zip/openpyxl/excelize 전부 OK).
- **비차단(유지보수)**: colsplit regex의 excelize 출력포맷 silent 결합(위 1번). 담당: excel-parity-engineer.
- 부수 차이(이미지 dedup, sharedStrings)는 의미 동등이라 이슈 아님.
