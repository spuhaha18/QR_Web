package httpx

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"

	"qrweb/internal/imaging"
	"qrweb/internal/label"
	"qrweb/internal/qr"
)

// handleCreateLabelPaste ports POST /create_label (paste mode, multipart).
// The client supplies one QR PNG per document copy plus a qr_order permutation;
// the server reorders the bytes and embeds them. The .xlsx is streamed back
// directly (no temp file). Validation order and Korean error strings are
// preserved byte-for-byte from app.py.
func (s *Server) handleCreateLabelPaste(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return errJSON(c, fiber.StatusBadRequest, "잘못된 요청 형식입니다.")
	}

	// Flatten text fields to map[string]string for ParseLabelRequest.
	fields := map[string]string{}
	for k, v := range form.Value {
		if len(v) > 0 {
			fields[k] = v[0]
		}
	}

	// 1. doc_type / binder_size / required fields.
	lbl, docType, binderSize, err := label.ParseLabelRequest(fields, fields["doc_type"], fields["binder_size"])
	if err != nil {
		return fail(c, err)
	}

	// 2. doc_count (already int via ParseLabelRequest).
	docCount := lbl.DocCount()

	// Transport: parse qr_order JSON and read the uploaded bytes. Domain rules
	// (count, permutation, size, PNG validity, reorder) live in the label
	// package's QR image intake; the handler only moves bytes across the wire.
	var qrOrder []int
	rawOrder := fields["qr_order"]
	if rawOrder == "" {
		rawOrder = "[]"
	}
	if err := json.Unmarshal([]byte(rawOrder), &qrOrder); err != nil {
		return errJSON(c, fiber.StatusBadRequest, "qr_order 형식이 올바르지 않습니다.")
	}

	qrFiles := form.File["qr_images"]
	uploads := make([]label.QRUpload, len(qrFiles))
	for i, fh := range qrFiles {
		f, err := fh.Open()
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
		}
		raw, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
		}
		uploads[i] = label.QRUpload{Name: fh.Filename, Bytes: raw}
	}

	limits := label.QRIntakeLimits{MaxFiles: s.cfg.MaxQRFiles, MaxFileSize: s.cfg.MaxQRFileSize}
	qrSet, err := label.BuildQRImageSet(uploads, qrOrder, docCount, limits, imaging.ValidatePNGBytes)
	if err != nil {
		return fail(c, err)
	}

	data, filename, err := s.gen.CreateLabelExcel(docType, binderSize, lbl, qrSet)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}

	s.log.Info("Paste-mode label generated: %s for %s", filename, c.IP())
	return sendXLSX(c, data, filename)
}

// handleCreateLabelAuto ports POST /api/create_label (auto mode, JSON). The
// server generates one QR per sheet from the label payload, embeds them, and
// returns the .xlsx. To preserve the existing success-shaped JSON contract
// (test asserts success==true) while dropping the temp-file download_url, the
// workbook bytes are returned inline as base64.
func (s *Server) handleCreateLabelAuto(c *fiber.Ctx) error {
	var body map[string]any
	if err := json.Unmarshal(c.Body(), &body); err != nil || len(body) == 0 {
		// Empty object {} matches Python app.py's `not data` → same message/status.
		return errJSON(c, fiber.StatusBadRequest, "잘못된 JSON 데이터입니다.")
	}

	fields := jsonFields(body)

	lbl, docType, binderSize, err := label.ParseLabelRequest(fields, fields["doc_type"], fields["binder_size"])
	if err != nil {
		return fail(c, err)
	}

	// Server-side QR generation: one per sheet (1-based index), via the domain
	// intake with the qr package wired in as the renderer.
	renderQR := func(payload string) ([]byte, error) {
		qrText, err := qr.NewQRText(payload)
		if err != nil {
			return nil, err
		}
		return qr.CreateQRPNG(qrText)
	}
	qrSet, err := label.BuildAutoQRImageSet(lbl, renderQR)
	if err != nil {
		return fail(c, err)
	}

	data, filename, err := s.gen.CreateLabelExcel(docType, binderSize, lbl, qrSet)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}

	s.log.Info("Auto-mode label generated: %s for %s", filename, c.IP())
	return c.JSON(fiber.Map{
		"success":      true,
		"message":      "라벨이 성공적으로 생성되었습니다.",
		"filename":     filename,
		"file_base64":  base64.StdEncoding.EncodeToString(data),
		"content_type": xlsxContentType,
	})
}

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// sendXLSX streams the workbook bytes as an attachment.
func sendXLSX(c *fiber.Ctx, data []byte, filename string) error {
	c.Set("Content-Disposition", contentDisposition(filename))
	c.Set(fiber.HeaderContentType, xlsxContentType)
	return c.Send(data)
}

// contentDisposition builds an RFC 6266 attachment header that survives
// non-ASCII filenames (e.g. Korean document numbers). HTTP header values are
// ISO-8859-1, so raw UTF-8 in filename="..." is mis-decoded by browsers into
// mojibake. We emit an ASCII fallback plus a filename*=UTF-8” percent-encoded
// form, which modern clients prefer — matching Flask send_file's behavior.
func contentDisposition(filename string) string {
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s",
		asciiFallbackName(filename), rfc5987Encode(filename))
}

// asciiFallbackName replaces non-ASCII (and quote/backslash) runes with '_' so
// the legacy filename="..." token is a safe ASCII string.
func asciiFallbackName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 && r != '"' && r != '\\' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if out := b.String(); out != "" {
		return out
	}
	return "label.xlsx"
}

// rfc5987Encode percent-encodes a UTF-8 string per RFC 5987 ext-value: every
// byte that is not an attr-char is %HH-escaped.
func rfc5987Encode(s string) string {
	const upperhex = "0123456789ABCDEF"
	const attrChars = "!#$&+-.^_`|~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			strings.IndexByte(attrChars, c) >= 0 {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(upperhex[c>>4])
			b.WriteByte(upperhex[c&0x0f])
		}
	}
	return b.String()
}

// jsonFields flattens a decoded JSON object to map[string]string the way
// ParseLabelRequest expects (numbers/bools stringified; matches Flask's
// request.form.get semantics where JSON values are coerced via str()).
func jsonFields(body map[string]any) map[string]string {
	out := make(map[string]string, len(body))
	for k, v := range body {
		switch t := v.(type) {
		case string:
			out[k] = t
		case float64:
			// JSON numbers decode to float64; render integers without ".0".
			if t == float64(int64(t)) {
				out[k] = fmt.Sprintf("%d", int64(t))
			} else {
				out[k] = fmt.Sprintf("%v", t)
			}
		case bool:
			out[k] = fmt.Sprintf("%v", t)
		case nil:
			out[k] = ""
		default:
			out[k] = fmt.Sprintf("%v", t)
		}
	}
	return out
}
