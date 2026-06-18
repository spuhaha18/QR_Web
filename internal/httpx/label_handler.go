package httpx

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

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
		if errors.Is(err, label.ErrValidation) {
			return errJSON(c, fiber.StatusBadRequest, label.ValidationMessage(err))
		}
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}

	// 2. doc_count (already int via ParseLabelRequest).
	docCount := lbl.DocCount()

	// QR image files.
	qrFiles := form.File["qr_images"]

	// qr_order parse.
	var qrOrder []int
	rawOrder := fields["qr_order"]
	if rawOrder == "" {
		rawOrder = "[]"
	}
	if err := json.Unmarshal([]byte(rawOrder), &qrOrder); err != nil {
		return errJSON(c, fiber.StatusBadRequest, "qr_order 형식이 올바르지 않습니다.")
	}

	// 3. len(qr_files) == doc_count.
	if len(qrFiles) != docCount {
		return errJSON(c, fiber.StatusBadRequest,
			fmt.Sprintf("QR 이미지 수가 권수와 다릅니다 (받음: %d, 권수: %d)", len(qrFiles), docCount))
	}

	// 4. len(qr_files) <= MAX_QR_FILES.
	if len(qrFiles) > s.cfg.MaxQRFiles {
		return errJSON(c, fiber.StatusBadRequest,
			fmt.Sprintf("QR 이미지는 최대 %d개까지 허용됩니다.", s.cfg.MaxQRFiles))
	}

	// 5. qr_order length / range / duplicate.
	if len(qrOrder) != docCount {
		return errJSON(c, fiber.StatusBadRequest, "qr_order 길이가 권수와 다릅니다.")
	}
	if !isPermutation(qrOrder, docCount) {
		return errJSON(c, fiber.StatusBadRequest, "qr_order에 중복이나 범위 초과 인덱스가 있습니다.")
	}

	// 6. per-file size + valid PNG.
	fileBytes := make([][]byte, len(qrFiles))
	for i, fh := range qrFiles {
		if fh.Size > s.cfg.MaxQRFileSize {
			return errJSON(c, fiber.StatusBadRequest,
				fmt.Sprintf("QR 이미지 크기가 2MB를 초과합니다: %s", fh.Filename))
		}
		f, err := fh.Open()
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
		}
		raw, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
		}
		if int64(len(raw)) > s.cfg.MaxQRFileSize {
			return errJSON(c, fiber.StatusBadRequest,
				fmt.Sprintf("QR 이미지 크기가 2MB를 초과합니다: %s", fh.Filename))
		}
		if !imaging.ValidatePNGBytes(raw) {
			return errJSON(c, fiber.StatusBadRequest,
				fmt.Sprintf("유효하지 않은 PNG 이미지입니다: %s", fh.Filename))
		}
		fileBytes[i] = raw
	}

	// 7. reorder by qr_order -> generate -> stream.
	ordered := make([][]byte, docCount)
	for sheetIdx, srcIdx := range qrOrder {
		ordered[sheetIdx] = fileBytes[srcIdx]
	}

	data, filename, err := s.gen.CreateLabelExcel(docType, binderSize, lbl, ordered)
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
		if errors.Is(err, label.ErrValidation) {
			return errJSON(c, fiber.StatusBadRequest, label.ValidationMessage(err))
		}
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}

	// Server-side QR generation: one per sheet, 1-based sheet index.
	total := lbl.DocCount()
	qrPNGs := make([][]byte, total)
	for i := 0; i < total; i++ {
		png, err := qr.CreateQRPNG(lbl.QRPayload(i+1, total))
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
		}
		qrPNGs[i] = png
	}

	data, filename, err := s.gen.CreateLabelExcel(docType, binderSize, lbl, qrPNGs)
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
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Set(fiber.HeaderContentType, xlsxContentType)
	return c.Send(data)
}

// isPermutation reports whether order is exactly a permutation of [0, n)
// (sorted(order) == range(n)): correct length, in range, no duplicates.
func isPermutation(order []int, n int) bool {
	if len(order) != n {
		return false
	}
	cp := append([]int(nil), order...)
	sort.Ints(cp)
	for i := 0; i < n; i++ {
		if cp[i] != i {
			return false
		}
	}
	return true
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
