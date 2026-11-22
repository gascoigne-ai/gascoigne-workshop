package response

import "context"

// BodyPreview shows a response body in the user's browser.
type BodyPreview interface {
	// Show serves body as contentType for a while and opens it. Ending
	// ctx stops the opening, not the serving, which has its own lifetime.
	Show(ctx context.Context, body []byte, contentType string) error
}
