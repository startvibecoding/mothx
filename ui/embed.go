package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var dist embed.FS

// DistFS returns the built Serve Web UI assets embedded in the binary.
func DistFS() fs.FS {
	sub, _ := fs.Sub(dist, "dist")
	return sub
}
