package respview

import (
	"bytes"
	"image"
	_ "image/gif" // the formats a response can be previewed in
	_ "image/jpeg"
	_ "image/png"
)

// Picture is a response body that is an image.
type Picture struct {
	Image         image.Image
	Format        string // "png", "jpeg" or "gif"
	Width, Height int
}

// Decode reads body as an image; ok is false when it is not one of the
// formats the standard library reads. A body is sniffed, not trusted to
// its Content-Type, so a mislabeled picture still shows.
func Decode(body []byte) (Picture, bool) {
	if len(body) < 8 {
		return Picture{}, false
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > 64<<20 {
		return Picture{}, false
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return Picture{}, false
	}
	return Picture{Image: img, Format: format, Width: cfg.Width, Height: cfg.Height}, true
}
