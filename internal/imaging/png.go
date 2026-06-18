// Package imaging holds image validation helpers. png.go replaces the Pillow
// verify()+format=='PNG' check from utils.validate_qr_image_bytes.
package imaging

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
)

// pngSignature is the 8-byte PNG magic number. A valid PNG always starts with
// it; we reject early so JPEG/garbage/empty inputs never reach the parser.
var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// ValidatePNGBytes reports whether data is a structurally valid PNG image.
//
// It mirrors utils.validate_qr_image_bytes, which uses Pillow's Image.verify().
// verify() walks the PNG chunk structure and checks each chunk's CRC, but does
// NOT fully decompress the pixel stream. We must match that leniency exactly:
// real-world QR PNGs (e.g. from some encoders) carry extra bytes in IDAT beyond
// what the IHDR dimensions require. Go's image/png.Decode rejects those with
// "too much pixel data", but Pillow accepts them and the legacy app embeds them
// fine — so a full decode here would wrongly reject valid user uploads.
//
// Instead we validate the chunk framing: signature, then a sequence of
// [len][type][data][crc] chunks with matching CRC32, starting at IHDR and
// ending at IEND. This accepts extra-IDAT-data PNGs while still rejecting
// JPEG/garbage/empty (no signature) and truncated/corrupt PNGs (a chunk whose
// declared length runs past the buffer, a bad CRC, or a missing IEND).
func ValidatePNGBytes(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if !bytes.HasPrefix(data, pngSignature) {
		return false
	}

	rest := data[len(pngSignature):]
	sawIHDR := false
	sawIEND := false
	first := true

	for len(rest) > 0 {
		// Each chunk: 4-byte big-endian length, 4-byte type, data, 4-byte CRC.
		if len(rest) < 12 { // 4 (len) + 4 (type) + 0 data + 4 (crc) minimum
			return false
		}
		length := binary.BigEndian.Uint32(rest[0:4])
		ctype := rest[4:8]

		// Per the PNG spec the first chunk must be IHDR.
		if first && string(ctype) != "IHDR" {
			return false
		}
		first = false

		// Guard against overflow and truncation: the chunk must fit entirely
		// (type+data+crc) within the remaining buffer.
		chunkEnd := 8 + uint64(length) + 4
		if chunkEnd > uint64(len(rest)) {
			return false // truncated / corrupt
		}

		// CRC covers the chunk type + data.
		wantCRC := binary.BigEndian.Uint32(rest[8+length : 8+length+4])
		if crc32.ChecksumIEEE(rest[4:8+length]) != wantCRC {
			return false // corrupt chunk
		}

		switch string(ctype) {
		case "IHDR":
			if sawIHDR { // duplicate IHDR -> malformed
				return false
			}
			sawIHDR = true
		case "IEND":
			sawIEND = true
		}

		rest = rest[chunkEnd:]
		if sawIEND {
			break // IEND must be the terminal chunk
		}
	}

	// A valid PNG starts with IHDR and ends with IEND.
	return sawIHDR && sawIEND
}
