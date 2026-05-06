package data

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/big"
)

// ocrCharset avoids confusable characters (0/O, 1/I/l).
const ocrCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// RandomOCRText generates a random n-character string for OCR/PDF testing.
func RandomOCRText(n int) string {
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(ocrCharset))))
		b[i] = ocrCharset[idx.Int64()]
	}
	return string(b)
}

// GenTestImageBase64 renders text into a 288x90 PNG and returns base64.
func GenTestImageBase64(text string) string {
	const (
		scale = 5
		charW = 5
		charH = 7
		gapX  = 2 // gap between chars in font units
		imgW  = 288
		imgH  = 90
	)

	textW := len(text)*(charW+gapX)*scale - gapX*scale
	padX := (imgW - textW) / 2
	padY := (imgH - charH*scale) / 2

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}

	// fill white
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			img.Set(x, y, white)
		}
	}

	// render each character
	for i, ch := range text {
		glyph, ok := font5x7[byte(ch)]
		if !ok {
			continue
		}
		ox := padX + i*(charW+gapX)*scale
		for row := 0; row < charH; row++ {
			for col := 0; col < charW; col++ {
				if glyph[row]&(1<<uint(charW-1-col)) != 0 {
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							img.Set(ox+col*scale+dx, padY+row*scale+dy, black)
						}
					}
				}
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// GenTestPDFBase64 creates a minimal single-page PDF containing text and returns base64.
// Text must be exactly 8 ASCII characters to keep PDF offsets stable.
func GenTestPDFBase64(text string) string {
	stream := fmt.Sprintf("BT /F1 14 Tf 10 20 Td (%s) Tj ET", text)
	streamLen := len(stream)

	// Build PDF with precise byte offsets for xref
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.0\n")

	off1 := buf.Len()
	buf.WriteString("1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n")

	off2 := buf.Len()
	buf.WriteString("2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n")

	off3 := buf.Len()
	buf.WriteString("3 0 obj<</Type/Page/MediaBox[0 0 300 50]/Parent 2 0 R/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n")

	off4 := buf.Len()
	fmt.Fprintf(&buf, "4 0 obj<</Length %d>>\nstream\n%s\nendstream\nendobj\n", streamLen, stream)

	off5 := buf.Len()
	buf.WriteString("5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n")

	xrefOff := buf.Len()
	buf.WriteString("xref\n0 6\n")
	fmt.Fprintf(&buf, "%010d 65535 f \n", 0)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off1)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off2)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off3)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off4)
	fmt.Fprintf(&buf, "%010d 00000 n \n", off5)
	buf.WriteString("trailer<</Size 6/Root 1 0 R>>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF", xrefOff)

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// font5x7 is a 5-wide 7-tall bitmap font. Each row is a uint8 where
// bit 4 = leftmost pixel, bit 0 = rightmost pixel.
var font5x7 = map[byte][7]uint8{
	'A': {0x04, 0x0A, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D': {0x1C, 0x12, 0x11, 0x11, 0x11, 0x12, 0x1C},
	'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0E},
	'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'J': {0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
	'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'L': {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	'M': {0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11},
	'N': {0x11, 0x11, 0x19, 0x15, 0x13, 0x11, 0x11},
	'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'Q': {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S': {0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E},
	'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V': {0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04},
	'W': {0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11},
	'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'Z': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
	'2': {0x0E, 0x11, 0x01, 0x02, 0x04, 0x08, 0x1F},
	'3': {0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
}
