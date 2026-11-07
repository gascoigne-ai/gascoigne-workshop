package port

// BodyPreview shows a response body in the user's browser.
type BodyPreview interface {
	// Show serves body as contentType for a while and opens it.
	Show(body []byte, contentType string) error
}
