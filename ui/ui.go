package ui

import (
	"embed"
	"io/fs"
)

var (
	//go:embed all:dist
	assetsFS embed.FS
)

// Assets returns the dist directory
func Assets() fs.FS {
	f, err := fs.Sub(assetsFS, "dist")
	if err != nil {
		panic(err)
	}
	return f
}

// IndexFile returns the content of dist/index.html file
func IndexFile() ([]byte, error) {
	return assetsFS.ReadFile("dist/index.html")
}
