---
name: parity-qa
description: 마이그레이션 정합성을 점진적으로 검증할 때 사용. 현재 Python 앱으로 골든 .xlsx를 캡처해 Go 출력과 의미 단위 diff(셀값/병합/치수/이미지앵커/스타일), API 응답 JSON과 Svelte fetch 파싱 shape 경계면 교차검증. 각 모듈 완성 직후 검증. QA, 패리티 비교, xlsx diff, 회귀 검증, 경계면 검사 작업 시 반드시 사용.
---

# Parity QA (마이그레이션 정합성 검증)

마이그레이션이 동작을 보존하는지 **점진적으로** 검증한다. 전체 완성 후 1회가 아니라 각 모듈(label→QR→Excel→HTTP→프론트) 완성 직후 실행.

## 두 검증 축
1. **Excel 패리티**: 현재 Python 출력을 오라클로, Go .xlsx와 구조 비교.
2. **경계면 정합성**: Fiber 응답 JSON shape vs Svelte fetch 파싱 코드를 **동시에 읽어** 필드명·타입 불일치 검출. "존재 확인" 아닌 "교차 비교".

## 1단계: 골든 캡처 (오라클 생성)
현재 Python 앱(`.venv`)으로 매트릭스 .xlsx를 `testdata/golden/`에 생성:
- doc_type {1 기기, 2 과제} × binder {1,3,5,7} × {단일, 멀티(예 3권)} × {paste, auto}
- 과제는 binder 1 제외(거부됨).
`scripts/capture_golden.py`로 자동화. 이게 모든 Excel 패리티의 기준.

## 2단계: 의미 단위 diff (XML 완전일치 비대상)
excelize와 openpyxl은 XML 출력이 달라 바이트 비교 무의미. **양쪽을 중립 리더로 읽어 속성 비교**:
- 시트명·시트 수 (특히 "Sheet 1" 공백)
- 셀별 값 (B2~B7, Q21/Q22/R23/S23, B5="i/N")
- 병합 범위 집합
- 열너비(B–M 바인더값, A/N=0.375 등)·행높이(4=216 등)
- 이미지 앵커 셀(바인더별 E9/D9/D8/B9) + 개수(시트당 1)
- 셀별 테두리 변(side)·폰트 존재
`scripts/compare_xlsx.py`(openpyxl로 양쪽 로드) 사용.

## 3단계: 경계면 교차검증
- go-backend의 `_workspace/E_api_contract.md` + Fiber 핸들러 응답 구조를 읽고, Svelte `lib/api.ts`의 fetch 파싱을 **같이** 읽어 비교: FormData 키 일치? 응답 `.xlsx` 바이너리 vs `res.blob()` 가정 일치? 에러 `{error}` JSON shape 일치? Content-Disposition 파싱 정상?
- 흔한 버그: 백엔드 snake_case vs 프론트 camelCase, 응답 형식 변경(download_url→bytes) 미반영.

## 4단계: 회귀 (기존 67 테스트 대응)
Go 테스트가 현재 pytest 동작을 커버하는지 확인(schema/layout/excel/handler). 한국어 에러 문자열·상태코드 단언 존재 확인.

## 원칙
- 불일치는 삭제·은폐 금지. **어느 속성이 다른지 특정**해 해당 담당에게 보고, `_workspace/QA_report.md`에 누적.
- 환경 문제로 검증 불가 시 우회(수동) 후 명시.

## 참조/스크립트
- `references/boundary-bugs.md` — 경계면 버그 패턴 체크리스트
- `scripts/capture_golden.py` — Python 앱으로 골든 .xlsx 생성
- `scripts/compare_xlsx.py` — 두 .xlsx 의미 단위 비교
