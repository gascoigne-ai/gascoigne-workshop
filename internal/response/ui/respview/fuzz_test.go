package respview

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func FuzzDecode(f *testing.F) {
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	f.Add(buffer.Bytes())
	f.Add([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"))
	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = Decode(body)
	})
}
