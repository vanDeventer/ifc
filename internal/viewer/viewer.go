// Package viewer wraps an IFC model in a self-contained web page that draws it
// with WebGL. Nothing is loaded from the network, so the page works offline and
// can be mailed around as a single file.
package viewer

import (
	_ "embed"
	"strings"
)

//go:embed viewer.html
var page string

// Fragment returns the page body: a <title>, styles, the canvas, the model and
// the script. It is what an artifact host expects, since those supply their own
// document skeleton.
func Fragment(model, title string) string {
	s := strings.ReplaceAll(page, "__IFC_MODEL__", model)
	if title != "" {
		s = strings.ReplaceAll(s, "<title>Cottage</title>", "<title>"+title+"</title>")
		s = strings.ReplaceAll(s, `<h1 id="title">Cottage</h1>`, `<h1 id="title">`+title+`</h1>`)
	}
	return s
}

// Document returns a complete standalone HTML file for opening from disk.
func Document(model, title string) string {
	return `<!doctype html>` + "\n" +
		`<html lang="en">` + "\n" +
		`<head>` + "\n" +
		`<meta charset="utf-8">` + "\n" +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n" +
		`</head>` + "\n" +
		`<body>` + "\n" +
		Fragment(model, title) + "\n" +
		`</body>` + "\n" +
		`</html>` + "\n"
}
