// Package web embeds the built React frontend so the single binary serves the
// SPA with no external assets.
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
