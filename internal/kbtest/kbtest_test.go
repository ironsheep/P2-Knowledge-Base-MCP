package kbtest

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

// get fetches base/path and returns the status and body.
func get(t *testing.T, r *Remote, path string) (int, string) {
	t.Helper()
	resp, err := http.Get(r.URL() + "/" + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return resp.StatusCode, string(body)
}

// fetchIndex fetches and decodes the served index the way a client does.
func fetchIndex(t *testing.T, r *Remote) Index {
	t.Helper()
	resp, err := http.Get(r.URL() + "/" + IndexPath)
	if err != nil {
		t.Fatalf("GET index: %v", err)
	}
	defer resp.Body.Close()
	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("index is not gzip: %v", err)
	}
	var idx Index
	if err := json.NewDecoder(gr).Decode(&idx); err != nil {
		t.Fatalf("index is not JSON: %v", err)
	}
	return idx
}

func TestRemoteURLHasNoTrailingSlash(t *testing.T) {
	r := NewRemote(t)
	if strings.HasSuffix(r.URL(), "/") {
		t.Errorf("URL() = %q, want no trailing slash", r.URL())
	}
}

func TestRemoteIndexRoundTrips(t *testing.T) {
	r := NewRemote(t)
	want := Index{
		System:         System{Version: "3.5.0"},
		Categories:     map[string][]string{"pasm2_math": {"p2kbPasm2Add"}},
		Files:          map[string]FileEntry{"p2kbPasm2Add": {Path: "a/add.yaml", Mtime: 7, SHA256: "ab"}},
		Aliases:        map[string][]string{"add": {"p2kbPasm2Add"}},
		DeliveryFilter: json.RawMessage(`{"format":1,"remove_fields":["source"],"remove_column0_comments":true}`),
	}
	r.SetIndex(want)
	if got := fetchIndex(t, r); !reflect.DeepEqual(got, want) {
		t.Errorf("served index\n  got  %+v\n  want %+v", got, want)
	}
}

func TestRemoteIndexOmitsAbsentDeliveryFilter(t *testing.T) {
	r := NewRemote(t)
	_, body := get(t, r, IndexPath)
	gr, err := gzip.NewReader(strings.NewReader(body))
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	raw, _ := io.ReadAll(gr)
	if strings.Contains(string(raw), "delivery_filter") {
		t.Errorf("index without a rule serves delivery_filter: %s", raw)
	}
}

func TestRemoteAddFileListsAndServes(t *testing.T) {
	r := NewRemote(t)
	const body = "mnemonic: ADD\n"
	r.AddFile("p2kbPasm2Add", "deliverables/ai/P2/add.yaml", 42, body)
	r.AddFile("p2kbPasm2Sub", "deliverables/ai/P2/sub.yaml", 43, "mnemonic: SUB\n")

	sum := sha256.Sum256([]byte(body))
	want := FileEntry{Path: "deliverables/ai/P2/add.yaml", Mtime: 42, SHA256: hex.EncodeToString(sum[:])}
	idx := fetchIndex(t, r)
	if got := idx.Files["p2kbPasm2Add"]; got != want {
		t.Errorf("index entry = %+v, want %+v", got, want)
	}
	if len(idx.Files) != 2 {
		t.Errorf("index lists %d files, want 2", len(idx.Files))
	}
	if status, got := get(t, r, want.Path); status != http.StatusOK || got != body {
		t.Errorf("GET %s = %d %q, want 200 %q", want.Path, status, got, body)
	}
}

func TestRemoteUnknownPathIs404(t *testing.T) {
	r := NewRemote(t)
	r.Put("known.yaml", "x")
	if status, _ := get(t, r, "unknown.yaml"); status != http.StatusNotFound {
		t.Errorf("unknown path status = %d, want 404", status)
	}
}

func TestRemotePutOverridesIndex(t *testing.T) {
	r := NewRemote(t)
	r.Put(IndexPath, "not gzip")
	if _, body := get(t, r, IndexPath); body != "not gzip" {
		t.Errorf("index body = %q, want the corrupt override", body)
	}
}

func TestRemoteHitsCountPerPath(t *testing.T) {
	r := NewRemote(t)
	r.Put("a.yaml", "a")
	r.Put("b.yaml", "b")
	get(t, r, "a.yaml")
	get(t, r, "a.yaml")
	get(t, r, "b.yaml")
	get(t, r, "missing.yaml")
	for path, want := range map[string]int{"a.yaml": 2, "b.yaml": 1, "missing.yaml": 1, "c.yaml": 0} {
		if got := r.Hits(path); got != want {
			t.Errorf("Hits(%q) = %d, want %d", path, got, want)
		}
	}
}

func TestRemoteRequestsRecordQueryAndHeaders(t *testing.T) {
	r := NewRemote(t)
	r.Put("a.yaml", "a")
	req, _ := http.NewRequest(http.MethodGet, r.URL()+"/a.yaml?t=123", nil)
	req.Header.Set("Pragma", "no-cache")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	reqs := r.Requests("a.yaml")
	if len(reqs) != 1 {
		t.Fatalf("Requests = %d, want 1", len(reqs))
	}
	if got := reqs[0].URL.Query().Get("t"); got != "123" {
		t.Errorf("recorded query t = %q, want 123", got)
	}
	if got := reqs[0].Header.Get("Pragma"); got != "no-cache" {
		t.Errorf("recorded Pragma = %q, want no-cache", got)
	}
}

func TestRemoteSequenceAdvancesThenRepeatsLast(t *testing.T) {
	r := NewRemote(t)
	r.Put("a.yaml", "first", "second")
	for i, want := range []string{"first", "second", "second"} {
		if _, got := get(t, r, "a.yaml"); got != want {
			t.Errorf("request %d = %q, want %q", i, got, want)
		}
	}
	r.Put("a.yaml", "restart", "after")
	if _, got := get(t, r, "a.yaml"); got != "restart" {
		t.Errorf("after a new Put = %q, want the new sequence's first body", got)
	}
}

func TestRemoteGateBlocksUntilReleased(t *testing.T) {
	r := NewRemote(t)
	r.Put("a.yaml", "a")
	g := r.Gate("a.yaml")

	done := make(chan string, 1)
	go func() {
		resp, err := http.Get(r.URL() + "/a.yaml")
		if err != nil {
			done <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		done <- string(body)
	}()

	select {
	case <-g.Arrived():
	case <-time.After(5 * time.Second):
		t.Fatal("request never reached the gate")
	}
	select {
	case got := <-done:
		t.Fatalf("gated request completed before release: %q", got)
	case <-time.After(50 * time.Millisecond):
	}

	if _, body := get(t, r, IndexPath); body == "" {
		t.Error("a gate on one path blocked another path")
	}

	g.Release()
	select {
	case got := <-done:
		if got != "a" {
			t.Errorf("released request = %q, want %q", got, "a")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("gated request did not complete after release")
	}
}

func TestRemoteCloseCausesNetworkError(t *testing.T) {
	r := NewRemote(t)
	url := r.URL()
	r.Close()
	if resp, err := http.Get(url + "/" + IndexPath); err == nil {
		resp.Body.Close()
		t.Error("GET after Close succeeded, want a network error")
	}
}
