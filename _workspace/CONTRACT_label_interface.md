# 계약: Label 인터페이스 (오케스트레이터 고정)

Phase 2 두 에이전트 공통 계약. 협상 불필요 — 이대로 구현.

```go
package label

// Label: 기기/과제 공통 추상. excel.Generator가 소비.
type Label interface {
    CellValues() map[string]any   // 셀주소→값 (예: "B2"->eq_number, "B5"->"1/N"는 i별로 생성 X — 시트1 기준 "1/{count}")
    QRPayload(i, total int) string // 파이프 구분 페이로드 (자동 모드용)
    DocNumber() string             // 파일명 base (기기=eq_doc_number, 과제=pjt_test_number)
    DocCount() int                 // 시트 수 (eq_doc_count / pjt_doc_count)
    TitleCell() string             // 제목 셀 주소 (둘 다 "B4") — FONT_TITLE 대상
}
```

## 셀값 (CellValues, Sheet 1 기준)
- 기기: B2=eq_number, B3=eq_doc_number, B4=eq_doc_title, B5="1/{count}", B6=eq_doc_department, B7=eq_doc_year(int).
- 과제: B2=pjt_number, B3=pjt_test_number, B4=pjt_doc_title, B5="1/{count}", B6=pjt_doc_writer, Q21="[{pjt_number}] {pjt_test_number}", Q22=pjt_doc_title, R23=pjt_doc_writer, S23="1/{count}".

## QRPayload (파이프 구분, i/total 포함)
- 기기: `eq_number|eq_doc_number|eq_doc_title|eq_doc_department|eq_doc_year|{i}/{total}`
- 과제: `pjt_number|pjt_test_number|pjt_doc_title|pjt_doc_writer|{i}/{total}`

## ParseLabelRequest
```go
func ParseLabelRequest(form map[string]string) (lbl Label, docType string, binder int, err error)
```
- docType "1"=기기, "2"=과제. binder ∈ {1,3,5,7}.
- **과제 + binder==1 → err** "과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다."
- 필수필드 누락 → err (한국어 메시지, app.py 원문 확인).
- SafeIntConversion: isdigit만 변환, 아니면 default, max(1, result). year 기본=현재년.
- ValidateAndCleanInput: strip 후 \n,\r 제거.

## 결정 반영 (§12)
- QR 인코딩 EUC-KR (자동 모드만). dnd 순열배열. 스트리밍(파일 경로 아닌 bytes).
