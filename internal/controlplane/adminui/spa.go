package adminui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed webui
var spaFS embed.FS

type spaFileSystem struct {
	inner http.FileSystem
}

func (s *spaFileSystem) Open(name string) (http.File, error) {
	f, err := s.inner.Open(name)
	if err != nil && os.IsNotExist(err) {
		return s.inner.Open("index.html")
	}
	return f, err
}

func SPAHandler() http.Handler {
	subFS, err := fs.Sub(spaFS, "webui")
	if err != nil {
		panic("adminui: embedded webui directory not found - run `make ui-copy` before building")
	}
	wrapper := &spaFileSystem{inner: http.FS(subFS)}
	fileServer := http.FileServer(wrapper)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/admin")
		if path == "" {
			path = "/index.html"
		}
		r.URL.Path = path
		fileServer.ServeHTTP(w, r)
	})
}
