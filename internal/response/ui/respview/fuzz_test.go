package respview

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func FuzzDecode(fuzz *testing.F) {
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	fuzz.Add(buffer.Bytes())
	fuzz.Add([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"))
	fuzz.Fuzz(func(t *testing.T, body []byte) {
		_, _ = Decode(body)
	})
}
