package textfmt

import (
	"fmt"
	"time"
)

// Duration renders d as "850ms" below a second, else "1.4s".
func Duration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// Size renders n bytes as "512B", "1.50KB" or "2.00MB".
func Size(n int) string {
	const kb = 1024
	switch {
	case n < kb:
		return fmt.Sprintf("%dB", n)
	case n < kb*kb:
		return fmt.Sprintf("%.2fKB", float64(n)/kb)
	default:
		return fmt.Sprintf("%.2fMB", float64(n)/(kb*kb))
	}
}
