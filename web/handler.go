package web

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/target/goalert/version"
)

// NewHandler creates a new http.Handler that will serve UI files
// using bundled assets or by proxying to urlStr if set.
func NewHandler(urlStr string) (http.Handler, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		// even empty, the URL is valid.
		return nil, errors.Wrap(err, "parse url")
	}
	if u.Host == "" {
		if u.Path == "" {
			return newMemoryHandler("/"), nil
		}
		return newMemoryHandler(u.Path), nil
	}
	// here do host empty, so path !
	return httputil.NewSingleHostReverseProxy(u), nil
}

type memoryHandler struct {
	files     map[string]File
	loadIndex sync.Once
	indexData []byte
	root string
}
type memoryFile struct {
	*bytes.Reader
	name string
	size int
}

func (m *memoryHandler) index() (*memoryFile, error) {
	m.loadIndex.Do(func() {
		f, ok := m.files["src/build/index.html"]
		if !ok {
			return
		}
		const footer = "\n\n<!-- Version: %s -->\n<!-- GitCommit: %s (%s) -->\n<!-- BuildDate: %s -->\n\n</html>"
		stamp := []byte(
			fmt.Sprintf(
				footer,
				version.GitVersion(),
				version.GitCommit(),
				version.GitTreeState(),
				version.BuildDate().UTC().Format(time.RFC3339),
			),
		)
		data := make([]byte, len(f.Data()), len(f.Data())+len(stamp))
		copy(data, f.Data())
		data = bytes.Replace(data, []byte("</html>"), stamp, 1)
		m.indexData = data
	})
	if len(m.indexData) == 0 {
		return nil, errors.New("not found")
	}
	return &memoryFile{
		Reader: bytes.NewReader(m.indexData),
		name: "src/build/index.html",
		size: len(m.indexData),
	}, nil
}

func (m *memoryHandler) Open(file string) (http.File, error) {
	// A root path may not exist, a physical must be at the root.
	file = strings.Replace(file, m.root, "/", 1)
	if file == "/index.html" {
		return m.index()
	}
	if f, ok := m.files["src/build"+file]; ok {
		return &memoryFile{Reader: bytes.NewReader(f.Data()), name: f.Name, size: len(f.Data())}, nil
	}
	// fallback to loading the index page
	return m.index()
}

func (m *memoryFile) Close() error { return nil }
func (m *memoryFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, errors.New("not a directory")
}
func (m *memoryFile) Stat() (os.FileInfo, error) {
	return m, nil
}
func (m *memoryFile) Name() string      { return path.Base(m.name) }
func (m *memoryFile) Size() int64       { return int64(m.size) }
func (m *memoryFile) Mode() os.FileMode { return 0644 }
func (m *memoryFile) ModTime() time.Time {
	if strings.Contains(m.name, "/static/") {
		return time.Time{}
	}
	return time.Now()
}
func (m *memoryFile) IsDir() bool      { return false }
func (m *memoryFile) Sys() interface{} { return nil }

func rootFSFix(root string, h http.Handler) http.Handler {
	if root == "" {
		// security first.
		root = "/"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == root {
			// necessary to avoid redirect loop
			req.URL.Path = path.Join(root, "alerts")
		}
		if strings.Contains(req.URL.Path, "/static/") {
			w.Header().Add("Cache-Control", "public, immutable, max-age=315360000")
		}
		h.ServeHTTP(w, req)
	})
}

func newMemoryHandler(root string) http.Handler {
	m := &memoryHandler{
		files: make(map[string]File, len(Files)),
		root: root,
	}
	for _, f := range Files {
		m.files[f.Name] = f
	}
	go m.index()

	return rootFSFix(root, http.FileServer(m))
}
