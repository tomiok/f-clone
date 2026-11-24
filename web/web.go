// Package web holds the templates and static files served by the app.
package web

import "embed"

// FS contains templates and static assets.
//
//go:embed templates static
var FS embed.FS
