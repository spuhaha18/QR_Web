# QR 생성: qrcode(Python) → go-qrcode

## Python 현재 동작 (qr_generator.py)
- qrcode 라이브러리, `version=None`, `make(fit=False)`, `error_correction=ERROR_CORRECT_L`, `box_size=10`, `border=2`, black on white.
- **데이터는 CP949 인코딩 후** QR 생성.
- PNG 출력. Excel 임베드 시 75×75로 강제 리사이즈(원본 모듈 크기 무관).

## Go 구현 (internal/qr/qr.go)
```go
import (
    "github.com/skip2/go-qrcode"
    "golang.org/x/text/encoding/korean"
)

func encodeCP949(s string) ([]byte, error) {
    return korean.EUCKR.NewEncoder().Bytes([]byte(s))
}

func CreateQRPNG(text string) ([]byte, error) {
    payload, err := encodeCP949(text)
    if err != nil { return nil, err }
    q, err := qrcode.New(string(payload), qrcode.Low) // Low = ERROR_CORRECT_L
    if err != nil { return nil, err }
    q.DisableBorder = false
    return q.PNG(256) // 크기는 Excel에서 75로 재스케일되므로 충분히 크게
}

func CreateQRBase64(text string) (string, error) {
    png, err := CreateQRPNG(text)
    if err != nil { return "", err }
    return base64.StdEncoding.EncodeToString(png), nil
}
```

## 인코딩 패리티 위험 (parity-qa 골든 비교)
- Python `CP949` = MS 코드페이지 949(EUC-KR 슈퍼셋, 확장 한글 8,822자 추가). Go `korean.EUCKR` = EUC-KR/Windows-949.
- 현대 상용 한글은 양쪽 일치. 희귀 음절은 다를 수 있어 모듈 수 변동 가능.
- **검증**: 대표 한글 문자열(부서명/제목)로 Go 인코딩 바이트를 Python `'...'.encode('CP949')` 결과와 비교하는 골든 테스트.
- **단, paste 모드는 무관** — 웹폼은 클라이언트 이미지 임베드라 CP949 안 거침. 자동 모드(`/api/create_label`)에서만 영향.

## QR 모듈 크기
`fit=False`+`version=None`의 현재 실제 출력 버전을 경험적으로 캡처(골든). 어차피 75×75 리사이즈되므로 시각 크기는 고정 — 데이터 용량(모듈 밀도)만 영향. go-qrcode는 콘텐츠 기준 자동 사이징.

## 입력 길이 제한
`/api/qr_image/:text` 등은 ≤500자 강제(현행). 초과 시 400 + 한국어 에러.
