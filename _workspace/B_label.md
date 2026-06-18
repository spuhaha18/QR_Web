# Phase 2 (데이터 계층) — go-backend-engineer 완료 보고

## 구현 파일
- `internal/label/schema.go` — Label 인터페이스, EquipmentLabel/ProjectLabel, ParseLabelRequest, MakeLabel, 헬퍼(SafeIntConversion/ValidateAndCleanInput/GenerateTimestampFilename), ErrValidation 센티넬 + ValidationMessage.
- `internal/label/schema_test.go` — test_document_schema.py + test_label_schema.py 포팅 + utils 헬퍼 테스트.
- `internal/imaging/png.go` — ValidatePNGBytes([]byte) bool (Pillow verify 대체).
- `internal/imaging/png_test.go` — test_utils_qr.py 포팅 (PNG/JPEG/garbage/empty/truncated).
- `internal/qr/qr.go` — CreateQRPNG, CreateQRBase64, EncodeCP949.
- `internal/qr/qr_test.go` — CP949/EUC-KR 골든 패리티 + PNG/base64 스모크.

## 테스트 결과
- `go vet ./internal/label/... ./internal/imaging/... ./internal/qr/...` → OK
- `go test ...` → 3패키지 모두 ok. **테스트 함수 47개 전부 통과, 0 실패.**
- (`go build ./...`는 httpx/config 등 미구현이라 전체는 실패 — 정상. 본 3패키지는 build/test green.)

## Label 인터페이스 최종 시그니처 (계약 그대로)
```go
type Label interface {
    CellValues() map[string]any   // B5 == "1/{count}" (시트1 기준)
    QRPayload(i, total int) string // 파이프 구분, {i}/{total}
    DocNumber() string             // 기기=eq_doc_number, 과제=pjt_test_number
    DocCount() int
    TitleCell() string             // 둘 다 "B4"
}
func ParseLabelRequest(form map[string]string, docType, binderSizeRaw string) (Label, string, int, error)
```
- 구체타입: `EquipmentLabel{EqNumber,EqDocNumber,EqDocTitle,EqDocCount int,EqDocDepartment,EqDocYear int}`,
  `ProjectLabel{PjtNumber,PjtTestNumber,PjtDocTitle,PjtDocWriter,PjtDocCount int}`.
- 둘 다 값 리시버 메서드 → `EquipmentLabel{}`/`ProjectLabel{}`(비포인터)가 Label 만족.

## CP949 / EUC-KR 패리티 결과: **완전 일치 (차이 없음)**
- Go `golang.org/x/text/encoding/korean.EUCKR`는 실제로 Windows-949(MS-949) 구현 → Python `encode('CP949')`와 바이트 동일.
- 검증 문자열(부서명/제목/이름/풀 페이로드/혼합 ASCII)뿐 아니라 **순수 EUC-KR에 없는 MS-949 확장 음절(꽸=0x84,0xc3 / 뷁=0x94,0xee)까지 일치**. 골든 비교 7케이스 통과.
- → §5 위험 #4(CP949 vs EUC-KR) 자동 모드 페이로드 인코딩은 패리티 리스크 해소. 골든은 `qr_test.go cp949Golden`에 박제.

## excel-parity-engineer 주의점
1. **CellValues 값 타입**: B7(eq_doc_year)은 `int`, S23/B5는 `string("1/N")`. map[string]any라 타입 스위치로 SetCellValue 분기 필요(year는 숫자셀, 나머지 문자열). 테스트가 B7 int 단언.
2. **B5는 시트1 고정 "1/{count}"**. 멀티시트 i/N은 CellValues가 아니라 시트 복제 후 generator가 B5(과제는 S23도) 덮어써야 함 — 계약/스펙 §2.4 그대로.
3. **TitleCell()=="B4"** 양쪽 동일 → FONT_TITLE 대상.
4. **DocNumber()**가 파일명 base. 파일명은 `label.GenerateTimestampFilename(lbl.DocNumber(), "xlsx")` 사용 가능(`{base}_{YYYYMMDDhhmmss}.xlsx`).
5. QRPayload(i,total)는 1-based 시트 인덱스 가정(Python sheet_idx 1-based). auto 모드 QR 생성 시 `qr.CreateQRPNG(lbl.QRPayload(i, total))`.
6. 검증 실패는 `errors.Is(err, ErrValidation)`로 400 판별, 원문 메시지는 `label.ValidationMessage(err)`로 추출(jsonify({'error': str(e)}) 패리티).

## 미구현/범위 외 (의도적)
- layout.go(GetQRConfig), httpx, config, excel는 이번 Phase 범위 아님.
- paste 모드 검증(개수/order/2MB)은 httpx 핸들러 Phase 담당. 본 Phase는 PNG 유효성(ValidatePNGBytes)만 제공.
