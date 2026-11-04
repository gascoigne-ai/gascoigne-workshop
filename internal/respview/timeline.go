package respview

import (
	"fmt"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/textfmt"
)

// Timeline is the Timeline pane: the total, the phases the connection
// went through and the size, with a note when the body was cut.
func Timeline(r *model.Response) string {
	var sb strings.Builder
	line := func(name string, d time.Duration) {
		if d > 0 {
			fmt.Fprintf(&sb, "%-11s%s\n", name, textfmt.Duration(d))
		}
	}
	fmt.Fprintf(&sb, "%-11s%s\n", "Total", textfmt.Duration(r.Duration))
	line("DNS", r.Timing.DNS)
	line("Connect", r.Timing.Connect)
	line("TLS", r.Timing.TLS)
	line("First byte", r.Timing.FirstByte)
	line("Transfer", r.Timing.Transfer)
	fmt.Fprintf(&sb, "%-11s%s", "Size", textfmt.Size(r.Size))
	if r.Truncated {
		sb.WriteString("\n\nThe body was longer than the limit and is cut here.")
	}
	return sb.String()
}
