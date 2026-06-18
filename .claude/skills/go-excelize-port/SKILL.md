---
name: go-excelize-port
description: openpyxl Excel 생성 코드를 Go excelize로 1:1 포팅할 때 사용. 셀 스타일(테두리/폰트/정렬), 셀 병합, 행높이/열너비, QR 이미지 임베드(75px 앵커), 멀티시트 i/N 복제, print_area를 시각 패리티로 재현하는 매핑 레퍼런스. Excel/.xlsx 생성, 셀 스타일, excelize, 라벨 레이아웃, QR 이미지 임베드 작업 시 반드시 사용.
---

# Go excelize 포팅 (openpyxl 패리티)

현재 `excel_generator.py`의 라벨 레이아웃을 excelize로 재현한다. **목표는 시각 패리티** — 생성 .xlsx가 현재 Python 출력과 셀값·테두리·이미지 위치까지 일치해야 한다.

## 왜 이게 어려운가
openpyxl과 excelize는 스타일 모델이 근본적으로 다르다. openpyxl은 `.border`/`.font`/`.alignment`를 셀별 **독립 속성**으로 누적·덮어쓰기 적용한다. excelize는 셀당 **단일 스타일 ID**(Font+Border+Alignment+Fill 합본)를 `SetCellStyle`로 적용한다. 그래서 incremental하게 옮기면 마지막 적용이 이전을 지운다. **셀별 최종 합성 스타일을 계산해 한 번에 flush**해야 한다.

## 작업 순서 (각 단계 후 골든 비교)
1. 기본 레이아웃: 행높이/열너비/병합
2. 테두리(외곽 medium + 내부 thin + 모서리 특수)
3. 폰트/정렬(전체 used range center/vcenter/wrap)
4. 멀티시트 i/N 복제
5. QR 이미지 임베드

patity-qa가 만든 `testdata/golden/`과 매 단계 구조 비교. **패리티 통과 전 다음 단계 금지.**

## 핵심 규칙 (반드시 지킬 것)
- **시트명 "Sheet 1"(공백 포함).** excelize 기본 "Sheet1"을 `SetSheetName`으로 rename. 멀티시트는 "Sheet 2"... 현재 테스트가 이 이름을 단언함.
- **셀당 합성 스타일.** 레이아웃 중 셀별 border 변(side)들의 **합집합**을 누적한 뒤, (font, alignment, border-combo) 별 고유 스타일 ID를 만들어 flush. 상세: `references/styling-map.md`.
- **이미지 75×75px 절대, one-cell 앵커.** `image.DecodeConfig`로 원본 px 구해 `ScaleX=75/srcW`, `ScaleY=75/srcH`. one-cell(절대) 앵커 + offset 0으로 셀 좌상단 고정. 상세: `references/image-anchor.md`.
- **멀티시트는 `CopySheet`** (openpyxl `copy_worksheet` 대체). QR은 시트 복제 **후** 전체 시트에 임베드(현재 코드 순서와 동일). CopySheet가 병합/치수/스타일 보존하는지 Phase D에서 검증. 상세: `references/multisheet.md`.
- **바인더 레이아웃 테이블**(셀 위치·열너비)은 `references/binder-layout.md`의 Go 구조체로. 7→E9/E8, 5·3→D9/D8, 1→B9·B9. 미지 크기는 3으로 폴백.
- **바이트로 저장**: `f.WriteToBuffer()` → 응답 스트림. 임시파일 없음.

## 셀값/치수 빠른 참조
- 행높이: `1:2.25, 2:27, 3:27, 4:216, 5:40.5, 6:27, 7:27`, 8–17:6.75, 18:2.25. 과제 추가 `20:2.25,21:48,22:34.5,23:27.75,24:2.25`.
- 열너비: A=N=0.375, B–M=바인더별 가변. 과제 추가 Q=8.13,R=34.88,S=8.13,T=0.375, N/O/P=0.375.
- 병합: `B2:M2~B6:M6`; 기기 `+B7:M7`; 과제 `+Q21:S21,Q22:S22`.
- 기기 셀값: B2=eq_number,B3=eq_doc_number,B4=eq_doc_title(FONT_TITLE 16),B5="1/N",B6=eq_doc_department,B7=eq_doc_year(int).
- 과제 셀값: B2~B6 + Q21="[{pjt_number}] {pjt_test_number}",Q22=pjt_doc_title,R23=pjt_doc_writer,S23="1/N". print_area=A1:T24.
- 폰트: FONT_TIMES=Times New Roman 12 bold black, FONT_TITLE=TNR 16 bold(B4). 과제 Q21=TNR20 bold center/wrap, Q22=TNR13 bold center/wrap, R23=TNR13 bold center/wrap, S23=FONT_TIMES.

상세 셀값·QR 페이로드·검증 규칙은 `excel_generator.py`/`document_schema.py` 원본을 오라클로 확인.

## 참조 파일
- `references/styling-map.md` — openpyxl Style/Border/Font/Alignment → excelize 스타일 ID 매핑, 합성 전략
- `references/image-anchor.md` — add_image → AddPictureFromBytes 75px 스케일링/앵커
- `references/multisheet.md` — copy_worksheet → CopySheet, i/N 갱신, print_area
- `references/binder-layout.md` — 바인더 config Go 구조체
