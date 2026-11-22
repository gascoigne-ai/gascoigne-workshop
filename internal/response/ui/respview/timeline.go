package respview

import (
	"fmt"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/textfmt"
)

// Timeline is the Timeline pane: the total, the phases the connection
// went through and the size, with a note when the body was cut.
func Timeline(response *model.Response) string {
	var sb strings.Builder
	line := func(name string, duration time.Duration) {
		if duration > 0 {
			fmt.Fprintf(&sb, "%-11s%s\n", name, textfmt.Duration(duration))
		}
	}
	fmt.Fprintf(&sb, "%-11s%s\n", "Total", textfmt.Duration(response.Duration))
	line("DNS", response.Timing.DNS)
	line("Connect", response.Timing.Connect)
	line("TLS", response.Timing.TLS)
	line("First byte", response.Timing.FirstByte)
	line("Transfer", response.Timing.Transfer)
	fmt.Fprintf(&sb, "%-11s%s", "Size", textfmt.Size(response.Size))
	if response.Truncated {
		sb.WriteString("\n\nThe body was longer than the limit and is cut here.")
	}
	return sb.String()
}
