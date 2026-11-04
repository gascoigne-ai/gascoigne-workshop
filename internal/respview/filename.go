package respview

import (
	"mime"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// extensions are the file extensions of the content types people save;
// the system's MIME table is the fallback, and its first choice is not
// always the usual one (".ehtml" for HTML).
var extensions = map[string]string{
	"application/json": ".json", "text/html": ".html", "application/xml": ".xml", "text/xml": ".xml",
	"text/plain": ".txt", "text/csv": ".csv", "text/css": ".css", "text/javascript": ".js",
	"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "application/pdf": ".pdf",
}

// SuggestedFileName is a name for saving the body: response.json, .html...
// from the content type.
func SuggestedFileName(r *model.Response) string {
	for _, h := range r.Headers {
		if !strings.EqualFold(h.Name, "Content-Type") {
			continue
		}
		mt, _, err := mime.ParseMediaType(h.Value)
		if err != nil {
			break
		}
		if ext, ok := extensions[mt]; ok {
			return "response" + ext
		}
		if strings.HasSuffix(mt, "+json") {
			return "response.json"
		}
		if exts, _ := mime.ExtensionsByType(mt); len(exts) > 0 {
			return "response" + exts[0]
		}
	}
	return "response.txt"
}
