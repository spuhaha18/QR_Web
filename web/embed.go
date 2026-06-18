// Package web embeds the built Vite/Svelte SPA (web/dist) into the Go binary so
// the server ships as a single static executable with no external asset files.
//
// The build chain (see Makefile / build-embed.md) runs the frontend build into
// web/dist before `go build`, so the //go:embed directive below always has a
// populated dist/ directory. The `all:` prefix includes files whose names begin
// with "." or "_" (Vite hashed asset names are safe, but all: is future-proof).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded SPA file system rooted at web/dist (so paths are
// e.g. "index.html", "assets/index-*.js"). Used by the Fiber filesystem
// middleware to serve the SPA with index.html as both index and SPA fallback.
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Unreachable when dist is embedded; panic surfaces a broken build.
		panic("web: embedded dist subtree missing: " + err.Error())
	}
	return sub
}
