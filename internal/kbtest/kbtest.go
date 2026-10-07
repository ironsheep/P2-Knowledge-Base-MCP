// Package kbtest provides a local stand-in for the P2KB raw-content host, so
// every package's tests can serve an index and content without the network.
package kbtest

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// IndexPath is where the KB publishes its gzipped index, relative to the base URL.
const IndexPath = "deliverables/ai/p2kb-index.json.gz"

// Index mirrors the published index JSON. It is declared here rather than
// borrowed from internal/index so that index's own tests can use Remote
// without an import cycle.
type Index struct {
	System         System               `json:"system"`
	Categories     map[string][]string  `json:"categories"`
	Files          map[string]FileEntry `json:"files"`
	Aliases        map[string][]string  `json:"aliases"`
	DeliveryFilter json.RawMessage      `json:"delivery_filter,omitempty"`
}

// System mirrors the index's "system" block.
type System struct {
	Version string `json:"version"`
}

// FileEntry mirrors one entry of the index's "files" map.
type FileEntry struct {
	Path   string `json:"path"`
	Mtime  int64  `json:"mtime"`
	SHA256 string `json:"sha256,omitempty"`
}

// Remote is an in-memory stand-in for the KB's raw-content host. It serves
// the gzipped index at IndexPath and any other body at its repo-relative
// path; unknown paths get 404. Pass URL() as the base the way P2KB_BASE_URL
// is passed. Safe for concurrent use; closed by t.Cleanup.
type Remote struct {
	srv *httptest.Server

	mu       sync.Mutex
	index    Index
	bodies   map[string][][]byte // path -> sequence; the last body repeats
	next     map[string]int      // path -> position in its sequence
	hits     map[string]int
	requests map[string][]*http.Request
	gates    map[string]*Gate
}

// NewRemote starts a Remote serving an empty index.
func NewRemote(t testing.TB) *Remote {
	t.Helper()
	r := &Remote{
		bodies:   map[string][][]byte{},
		next:     map[string]int{},
		hits:     map[string]int{},
		requests: map[string][]*http.Request{},
		gates:    map[string]*Gate{},
	}
	r.srv = httptest.NewServer(http.HandlerFunc(r.serve))
	t.Cleanup(func() {
		r.mu.Lock()
		for _, g := range r.gates {
			g.Release()
		}
		r.mu.Unlock()
		r.srv.Close()
	})
	r.SetIndex(Index{System: System{Version: "test-1.0"}})
	return r
}

// URL returns the base URL, without a trailing slash.
func (r *Remote) URL() string { return r.srv.URL }

// Close shuts the server down, so later requests fail with a network error.
func (r *Remote) Close() { r.srv.Close() }

// Index returns a copy of the index being served.
func (r *Remote) Index() Index {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.index
}

// SetIndex replaces the index and serves it, gzipped, at IndexPath. Nil maps
// are served as empty objects.
func (r *Remote) SetIndex(idx Index) {
	if idx.Categories == nil {
		idx.Categories = map[string][]string{}
	}
	if idx.Files == nil {
		idx.Files = map[string]FileEntry{}
	}
	if idx.Aliases == nil {
		idx.Aliases = map[string][]string{}
	}
	body := GzipJSON(idx)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.index = idx
	r.bodies[IndexPath] = [][]byte{body}
	r.next[IndexPath] = 0
}

// AddFile serves body at path and lists it in the index under key, with the
// body's sha256, then republishes the index.
func (r *Remote) AddFile(key, path string, mtime int64, body string) {
	sum := sha256.Sum256([]byte(body))
	idx := r.Index()
	files := make(map[string]FileEntry, len(idx.Files)+1)
	for k, v := range idx.Files {
		files[k] = v
	}
	files[key] = FileEntry{Path: path, Mtime: mtime, SHA256: hex.EncodeToString(sum[:])}
	idx.Files = files
	r.SetIndex(idx)
	r.Put(path, body)
}

// Put serves bodies[i] on the i-th request for path after this call, and the
// last body on every request after that. It replaces anything served there,
// the index included, so a test can serve a corrupt index.
func (r *Remote) Put(path string, bodies ...string) {
	seq := make([][]byte, len(bodies))
	for i, b := range bodies {
		seq[i] = []byte(b)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bodies[path] = seq
	r.next[path] = 0
}

// Hits returns how many requests path has received, 404s included.
func (r *Remote) Hits(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hits[path]
}

// Requests returns the requests path has received, in order, so a test can
// assert on query strings and headers.
func (r *Remote) Requests(path string) []*http.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*http.Request(nil), r.requests[path]...)
}

// Gate holds every response for path until the returned gate is released.
func (r *Remote) Gate(path string) *Gate {
	g := &Gate{arrived: make(chan struct{}), release: make(chan struct{})}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gates[path] = g
	return g
}

// Gate blocks responses on one path. See Remote.Gate.
type Gate struct {
	arrived     chan struct{}
	release     chan struct{}
	arriveOnce  sync.Once
	releaseOnce sync.Once
}

// Arrived is closed when the first request reaches the gate.
func (g *Gate) Arrived() <-chan struct{} { return g.arrived }

// Release lets every held and future request through. Safe to call twice.
func (g *Gate) Release() { g.releaseOnce.Do(func() { close(g.release) }) }

func (r *Remote) serve(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/")

	r.mu.Lock()
	r.hits[path]++
	r.requests[path] = append(r.requests[path], req.Clone(req.Context()))
	seq := r.bodies[path]
	n := r.next[path]
	if n < len(seq)-1 {
		r.next[path] = n + 1
	}
	g := r.gates[path]
	r.mu.Unlock()

	if g != nil {
		g.arriveOnce.Do(func() { close(g.arrived) })
		select {
		case <-g.release:
		case <-req.Context().Done():
			return
		}
	}

	if len(seq) == 0 {
		http.NotFound(w, req)
		return
	}
	_, _ = w.Write(seq[n])
}

// GzipJSON returns v marshalled to JSON and gzipped, the index's wire form.
// It panics on error: every caller passes a value that marshals.
func GzipJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(raw); err != nil {
		panic(err)
	}
	if err := gw.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
