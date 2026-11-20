## JSON view

`jsonview.Lines(body)` pretty-prints valid JSON with two-space indent and returns one `Line` per
display row, each a list of `Span{Text, Kind}` (punctuation, key, string, number, literal, plain).
Lines that open an object or array are `Foldable`, which the response pane uses for its fold arrows.
Invalid JSON and other text come back as plain lines, so the pane renders any body.
