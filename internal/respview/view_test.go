package respview

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func pngEncode(w *bytes.Buffer, m image.Image) error  { return png.Encode(w, m) }
func jpegEncode(w *bytes.Buffer, m image.Image) error { return jpeg.Encode(w, m, nil) }
func gifEncode(w *bytes.Buffer, m image.Image) error  { return gif.Encode(w, m, nil) }

func TestLooksBinary(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want bool
	}{
		{"empty", nil, false},
		{"json", []byte(`{"a":"żółw"}`), false},
		{"nul byte", []byte("abc\x00def"), true},
		{"invalid utf-8", []byte{0xff, 0xfe, 'a'}, true},
		{"png header", []byte("\x89PNG\r\n\x1a\n\x00\x00"), true},
		{"a multi-byte character cut by the 8 KiB sample is still text", append(bytes.Repeat([]byte("a"), 8191), []byte("ż")...), false},
		{"binary after the sample is not seen", append(bytes.Repeat([]byte("a"), 9000), 0), false},
	}
	for _, tt := range tests {
		if got := LooksBinary(tt.in); got != tt.want {
			t.Errorf("%s: looksBinary = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestDecodeFindsPicturesByTheirBytesAndRefusesTheRest(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(1, 1, color.RGBA{255, 0, 0, 255})
	var png, jpg, gf bytes.Buffer
	if err := pngEncode(&png, img); err != nil {
		t.Fatal(err)
	}
	if err := jpegEncode(&jpg, img); err != nil {
		t.Fatal(err)
	}
	if err := gifEncode(&gf, img); err != nil {
		t.Fatal(err)
	}
	for name, b := range map[string]*bytes.Buffer{"png": &png, "jpeg": &jpg, "gif": &gf} {
		p, ok := Decode(b.Bytes())
		if !ok || p.Format != name || p.Width != 3 || p.Height != 2 || p.Image == nil {
			t.Errorf("%s: %+v %v", name, p, ok)
		}
	}
	for name, b := range map[string][]byte{"json": []byte(`{"a":1}`), "empty": nil, "short": []byte("png"), "cut png": png.Bytes()[:20]} {
		if _, ok := Decode(b); ok {
			t.Errorf("%s decoded as a picture", name)
		}
	}
}
