package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html admin.html static/*
var files embed.FS

func IndexHTML() ([]byte, error) {
	return files.ReadFile("index.html")
}

func AdminHTML() ([]byte, error) {
	return files.ReadFile("admin.html")
}

func StaticFiles() (fs.FS, error) {
	return fs.Sub(files, "static")
}
