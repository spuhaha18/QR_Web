package qr

import "errors"

// MaxTextRunes is the maximum QR payload length (in runes). Beyond this the QR
// module density makes the code impractical to scan. This is a QR-domain rule;
// it used to live as a hardcoded check in two HTTP handlers and was not
// enforced on the auto-mode payloads at all.
const MaxTextRunes = 500

// ErrInvalidText is the sentinel a caller matches with errors.Is to map a QR
// text problem to a 400-class response.
var ErrInvalidText = errors.New("invalid qr text")

// TextError carries the exact Korean user-facing message for an invalid QR
// text; its Error() text IS the message and it reports as ErrInvalidText.
type TextError struct {
	Msg string
}

func (e *TextError) Error() string { return e.Msg }

func (e *TextError) Is(target error) bool { return target == ErrInvalidText }

// QRText is a validated QR payload: non-empty and at most MaxTextRunes runes. A
// QRText value cannot exist unless it satisfies those rules, so every QR
// generation path (paste endpoint, base64 endpoint, auto-mode per-sheet
// payloads) goes through the same invariant.
type QRText string

// NewQRText validates s and returns the typed QRText, or a TextError (matching
// ErrInvalidText) carrying the user-facing message.
func NewQRText(s string) (QRText, error) {
	if s == "" {
		return "", &TextError{Msg: "QR 코드 텍스트가 제공되지 않았습니다."}
	}
	if len([]rune(s)) > MaxTextRunes {
		return "", &TextError{Msg: "QR 코드 텍스트가 너무 깁니다 (최대 500자)."}
	}
	return QRText(s), nil
}

// String returns the raw payload text.
func (t QRText) String() string { return string(t) }
