package theme

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"

// RequestBadge is the short label shown before a request name, in the tree
// and on a tab.
func RequestBadge(protocol request.Protocol, method string) string {
	if info := protocol.Info(); info.Label != "" && info.Method != request.MethodHTTP {
		return info.Label
	}
	if method == "DELETE" {
		return "DEL"
	}
	return method
}
