status: todo

# Request body editing

## Goal
Body is read-only and always raw text.

## Scope
- Body pane uses the `TextArea` from task 005 and writes `Request.Body`.
- Body type selector: None, JSON, Text, Form URL-encoded, Multipart (fields only in this task). `Content-Type` is set automatically unless the user adds the header.
- JSON: validity indicator in the pane header and a Format button (`json.Indent`); invalid JSON still sends.
- Form: key/value table (reuses the editable table); encoded on send.

## Acceptance
- proto test: each body type produces the right bytes and content type; a user `Content-Type` header wins.
- ui test: switching the body type keeps the text of the previous type.
- `make check` passes.
