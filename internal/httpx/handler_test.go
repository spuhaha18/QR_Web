package httpx

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qrweb/internal/config"
	"qrweb/internal/logging"
)

// newTestServer builds a Server with default config and a no-op logger.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.Load()
	cfg.LogFile = "" // Default() logger writes only to stdout when Init not called
	return New(cfg, logging.Default())
}

// makePNG returns valid PNG bytes (passes imaging.ValidatePNGBytes).
func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.Black)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

// makeJPEG returns valid JPEG bytes (fails imaging.ValidatePNGBytes).
func makeJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("jpeg encode: %v", err)
	}
	return buf.Bytes()
}

type qrFile struct {
	field    string
	filename string
	content  []byte
}

// buildMultipart constructs a multipart body with the given text fields and
// files, returning the body and content type.
func buildMultipart(t *testing.T, fields map[string]string, files []qrFile) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	for _, f := range files {
		fw, err := w.CreateFormFile(f.field, f.filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fw.Write(f.content); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return body, w.FormDataContentType()
}

func baseEqFields() map[string]string {
	return map[string]string{
		"doc_type":          "1",
		"binder_size":       "3",
		"eq_number":         "MC-001",
		"eq_doc_number":     "DOC-001",
		"eq_doc_title":      "테스트",
		"eq_doc_count":      "2",
		"eq_doc_department": "개발",
		"eq_doc_year":       "2026",
	}
}

func postMultipart(t *testing.T, s *Server, path string, fields map[string]string, files []qrFile) *httptest.ResponseRecorder {
	t.Helper()
	body, ct := buildMultipart(t, fields, files)
	req := httptest.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", ct)
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	rec := httptest.NewRecorder()
	rec.Code = resp.StatusCode
	for k, vv := range resp.Header {
		for _, v := range vv {
			rec.Header().Add(k, v)
		}
	}
	b, _ := io.ReadAll(resp.Body)
	rec.Body = bytes.NewBuffer(b)
	return rec
}

func decodeError(t *testing.T, body []byte) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	if s, ok := m["error"].(string); ok {
		return s
	}
	return ""
}

func TestCreateLabel_CorrectNFiles_ReturnsXLSX(t *testing.T) {
	s := newTestServer(t)
	png := makePNG(t)
	fields := baseEqFields()
	fields["qr_order"] = "[0,1]"
	files := []qrFile{
		{"qr_images", "qr0.png", png},
		{"qr_images", "qr1.png", png},
	}
	rec := postMultipart(t, s, "/create_label", fields, files)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "spreadsheetml.sheet") {
		t.Errorf("Content-Type = %q, want xlsx", ct)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("PK")) {
		t.Errorf("body is not a ZIP/xlsx (first bytes: %x)", rec.Body.Bytes()[:4])
	}
}

func TestGetLogs_NegativeLines_NoPanic(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Load()
	cfg.LogFile = logPath
	s := New(cfg, logging.Default())

	// lines=-1 must not produce a negative slice index panic (recover -> 500).
	req := httptest.NewRequest("GET", "/api/logs?lines=-1", nil)
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 (negative lines must not panic)", resp.StatusCode)
	}
}

func TestCreateLabel_KoreanDocNumber_RFC5987Filename(t *testing.T) {
	s := newTestServer(t)
	png := makePNG(t)
	fields := baseEqFields()
	fields["eq_doc_count"] = "1"
	fields["eq_doc_number"] = "한글" // filename base becomes "한글_<ts>.xlsx"
	fields["qr_order"] = "[0]"
	files := []qrFile{{"qr_images", "qr0.png", png}}
	rec := postMultipart(t, s, "/create_label", fields, files)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	cd := rec.Header().Get("Content-Disposition")
	// RFC 6266: non-ASCII names must use filename*=UTF-8'' with percent-encoding.
	// 한 -> %ED%95%9C, 글 -> %EA%B8%80.
	if !strings.Contains(cd, "filename*=UTF-8''%ED%95%9C%EA%B8%80") {
		t.Errorf("Content-Disposition missing RFC5987 encoded name: %q", cd)
	}
	// The header value must be ASCII-only (no raw UTF-8 bytes that browsers
	// would mis-decode as Latin-1).
	for i := 0; i < len(cd); i++ {
		if cd[i] >= 0x80 {
			t.Errorf("Content-Disposition contains non-ASCII byte 0x%02x: %q", cd[i], cd)
			break
		}
	}
}

func TestCreateLabel_FileCountMismatch_400(t *testing.T) {
	s := newTestServer(t)
	png := makePNG(t)
	fields := baseEqFields() // count=2
	fields["qr_order"] = "[0]"
	files := []qrFile{{"qr_images", "qr0.png", png}} // only 1 file
	rec := postMultipart(t, s, "/create_label", fields, files)

	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	msg := decodeError(t, rec.Body.Bytes())
	if !strings.Contains(msg, "권수") {
		t.Errorf("error = %q, want containing 권수", msg)
	}
}

func TestCreateLabel_QROrderLengthMismatch_400(t *testing.T) {
	s := newTestServer(t)
	png := makePNG(t)
	fields := baseEqFields()
	fields["qr_order"] = "[0]" // 2 files but order len 1
	files := []qrFile{
		{"qr_images", "qr0.png", png},
		{"qr_images", "qr1.png", png},
	}
	rec := postMultipart(t, s, "/create_label", fields, files)
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := decodeError(t, rec.Body.Bytes()); got != "qr_order 길이가 권수와 다릅니다." {
		t.Errorf("error = %q", got)
	}
}

func TestCreateLabel_QROrderOutOfRange_400(t *testing.T) {
	s := newTestServer(t)
	png := makePNG(t)
	fields := baseEqFields()
	fields["qr_order"] = "[0,5]" // 5 out of range
	files := []qrFile{
		{"qr_images", "qr0.png", png},
		{"qr_images", "qr1.png", png},
	}
	rec := postMultipart(t, s, "/create_label", fields, files)
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeError(t, rec.Body.Bytes()); got != "qr_order에 중복이나 범위 초과 인덱스가 있습니다." {
		t.Errorf("error = %q", got)
	}
}

func TestCreateLabel_QROrderDuplicate_400(t *testing.T) {
	s := newTestServer(t)
	png := makePNG(t)
	fields := baseEqFields()
	fields["qr_order"] = "[0,0]" // duplicate
	files := []qrFile{
		{"qr_images", "qr0.png", png},
		{"qr_images", "qr1.png", png},
	}
	rec := postMultipart(t, s, "/create_label", fields, files)
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeError(t, rec.Body.Bytes()); got != "qr_order에 중복이나 범위 초과 인덱스가 있습니다." {
		t.Errorf("error = %q", got)
	}
}

func TestCreateLabel_NonPNG_400(t *testing.T) {
	s := newTestServer(t)
	fields := baseEqFields()
	fields["eq_doc_count"] = "1"
	fields["qr_order"] = "[0]"
	files := []qrFile{{"qr_images", "bad.jpg", makeJPEG(t)}}
	rec := postMultipart(t, s, "/create_label", fields, files)
	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := decodeError(t, rec.Body.Bytes()); !strings.Contains(got, "유효하지 않은 PNG 이미지입니다") {
		t.Errorf("error = %q", got)
	}
}

func TestAPICreateLabel_AutoGenerates_200(t *testing.T) {
	s := newTestServer(t)
	payload := map[string]any{
		"doc_type":          "1",
		"binder_size":       3,
		"eq_number":         "MC-001",
		"eq_doc_number":     "DOC-001",
		"eq_doc_title":      "테스트",
		"eq_doc_count":      1,
		"eq_doc_department": "개발",
		"eq_doc_year":       2026,
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/create_label", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		bb, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200 (body=%s)", resp.StatusCode, bb)
	}
	bb, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(bb, &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	if m["success"] != true {
		t.Errorf("success = %v, want true", m["success"])
	}
	if _, ok := m["file_base64"].(string); !ok {
		t.Errorf("missing file_base64")
	}
}

func TestAPICreateLabel_ProjectBinder1_400(t *testing.T) {
	s := newTestServer(t)
	payload := map[string]any{
		"doc_type":        "2",
		"binder_size":     1,
		"pjt_number":      "P-1",
		"pjt_test_number": "T-1",
		"pjt_doc_title":   "제목",
		"pjt_doc_writer":  "작성자",
		"pjt_doc_count":   1,
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/create_label", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	bb, _ := io.ReadAll(resp.Body)
	if got := decodeError(t, bb); got != "과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다." {
		t.Errorf("error = %q", got)
	}
}

func TestAPIHealth_200(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/health", nil)
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	bb, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(bb, &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	if m["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", m["status"])
	}
}

func TestQRImageBase64_200(t *testing.T) {
	s := newTestServer(t)
	b, _ := json.Marshal(map[string]string{"text": "MC-001|DOC-001|테스트|개발|2026|1/1"})
	req := httptest.NewRequest("POST", "/api/qr_image_base64", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	bb, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(bb, &m)
	if m["success"] != true || m["mime_type"] != "image/png" {
		t.Errorf("unexpected body: %s", bb)
	}
	if _, ok := m["image_base64"].(string); !ok {
		t.Errorf("missing image_base64")
	}
}

func TestQRImageBase64_TooLong_400(t *testing.T) {
	s := newTestServer(t)
	long := strings.Repeat("가", 501)
	b, _ := json.Marshal(map[string]string{"text": long})
	req := httptest.NewRequest("POST", "/api/qr_image_base64", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	bb, _ := io.ReadAll(resp.Body)
	if got := decodeError(t, bb); got != "QR 코드 텍스트가 너무 깁니다 (최대 500자)." {
		t.Errorf("error = %q", got)
	}
}

func TestQRImage_PNG_200(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/api/qr_image/MC-001", nil)
	resp, err := s.App().Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
}
