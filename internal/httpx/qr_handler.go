package httpx

import (
	"encoding/json"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"qrweb/internal/qr"
)

// handleQRImage ports GET /api/qr_image/* — returns the QR PNG for the path
// text (<=500 chars). Uses Fiber wildcard so payloads with slashes (e.g. "1/3")
// are captured correctly, matching Flask's <path:qr_text> behaviour.
func (s *Server) handleQRImage(c *fiber.Ctx) error {
	text := c.Params("*")
	// Decode percent-escapes to match Flask's <path:qr_text> (already unescaped).
	if d, err := url.PathUnescape(text); err == nil {
		text = d
	}
	qrText, err := qr.NewQRText(text)
	if err != nil {
		return errJSON(c, fiber.StatusBadRequest, err.Error())
	}

	png, err := qr.CreateQRPNG(qrText)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}
	c.Set(fiber.HeaderContentType, "image/png")
	return c.Send(png)
}

// handleQRImageBase64 ports POST /api/qr_image_base64 — JSON {text} ->
// {success, image_base64, mime_type}.
func (s *Server) handleQRImageBase64(c *fiber.Ctx) error {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return errJSON(c, fiber.StatusBadRequest, "QR 코드 텍스트가 제공되지 않았습니다.")
	}
	qrText, err := qr.NewQRText(body.Text)
	if err != nil {
		return errJSON(c, fiber.StatusBadRequest, err.Error())
	}

	b64, err := qr.CreateQRBase64(qrText)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "서버 오류가 발생했습니다.")
	}
	return c.JSON(fiber.Map{
		"success":      true,
		"image_base64": b64,
		"mime_type":    "image/png",
	})
}
