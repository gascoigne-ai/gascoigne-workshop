// Package send is the feature of sending a request and reading the answer:
// the logic that turns text into a request (cURL, MCP server info, GraphQL
// introspection, binary messages) and the port that shows a response body
// in the browser. The protocol clients are in proto, the request editor and
// the response viewer are controllers below this package, and ui holds the
// views. It imports the kernel only.
package send

import "context"

// BodyPreview shows a response body in the user's browser.
type BodyPreview interface {
	// Show serves body as contentType for a while and opens it. Ending
	// ctx stops the opening, not the serving, which has its own lifetime.
	Show(ctx context.Context, body []byte, contentType string) error
}
