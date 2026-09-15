// Package web embeds the control-plane dashboard.
package web

import "embed"

// Files holds the dashboard assets served by the control plane.
//
//go:embed index.html style.css app.js feed.js favicon.svg
var Files embed.FS
