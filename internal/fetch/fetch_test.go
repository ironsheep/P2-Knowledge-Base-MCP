package fetch_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/ironsheep/p2kb-mcp/internal/fetch"
	"github.com/ironsheep/p2kb-mcp/internal/kbtest"
	"github.com/ironsheep/p2kb-mcp/internal/logging"
)

// captureLog sends the standard logger to a buffer at level l for the test.
func captureLog(t *testing.T, l logging.Level) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	prev := logging.SetLevel(l)
	t.Cleanup(func() {
		logging.SetLevel(prev)
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

func TestDebugLogsOneLinePerRequest(t *testing.T) {
	r := kbtest.NewRemote(t)
	r.Put("a.yaml", "abcde")
	c := r.Client()
	buf := captureLog(t, logging.Debug)

	_, _ = c.Fetch("a.yaml", false)
	_, _ = c.Fetch("a.yaml", true)
	_, _ = c.Fetch("missing.yaml", false)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("log lines = %d, want 3:\n%s", len(lines), buf)
	}
	for i, want := range []string{
		"fetch GET " + c.URL("a.yaml") + " bust=false status=200 bytes=5 duration=",
		"fetch GET " + c.URL("a.yaml") + " bust=true status=200 bytes=5 duration=",
		"fetch GET " + c.URL("missing.yaml") + " bust=false status=404 bytes=0 duration=",
	} {
		if !strings.HasPrefix(lines[i], want) {
			t.Errorf("line %d = %q, want prefix %q", i, lines[i], want)
		}
	}
}

func TestFetchIsSilentAboveDebug(t *testing.T) {
	r := kbtest.NewRemote(t)
	r.Put("a.yaml", "a")
	buf := captureLog(t, logging.DefaultLevel)
	_, _ = r.Client().Fetch("a.yaml", false)
	if buf.Len() != 0 {
		t.Errorf("fetch logged at the default level: %q", buf)
	}
}

func TestNewUsesDefaultBaseWhenUnset(t *testing.T) {
	t.Setenv(fetch.BaseURLEnv, "")
	c := fetch.New()
	if got := c.BaseURL(); got != fetch.DefaultBaseURL {
		t.Errorf("BaseURL() = %q, want %q", got, fetch.DefaultBaseURL)
	}
	want := "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main/deliverables/ai/p2kb-index.json.gz"
	if got := c.URL(fetch.IndexPath); got != want {
		t.Errorf("URL(IndexPath) = %q, want %q", got, want)
	}
}

func TestNewReadsBaseFromEnv(t *testing.T) {
	r := kbtest.NewRemote(t)
	r.Put("deliverables/ai/P2/a.yaml", "a: 1\n")
	t.Setenv(fetch.BaseURLEnv, r.URL())
	c := fetch.New()

	if _, err := c.FetchGzip(fetch.IndexPath, false); err != nil {
		t.Fatalf("FetchGzip(IndexPath): %v", err)
	}
	body, err := c.Fetch("deliverables/ai/P2/a.yaml", false)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(body) != "a: 1\n" {
		t.Errorf("body = %q, want %q", body, "a: 1\n")
	}
	if r.Hits(fetch.IndexPath) != 1 || r.Hits("deliverables/ai/P2/a.yaml") != 1 {
		t.Errorf("hits: index %d, content %d; want 1 and 1",
			r.Hits(fetch.IndexPath), r.Hits("deliverables/ai/P2/a.yaml"))
	}
}

func TestTrailingSlashIsTrimmed(t *testing.T) {
	r := kbtest.NewRemote(t)
	r.Put("a.yaml", "a")
	for _, base := range []string{r.URL() + "/", r.URL() + "//"} {
		c := fetch.NewWithBase(base)
		if strings.HasSuffix(c.BaseURL(), "/") {
			t.Errorf("BaseURL() for %q = %q, want no trailing slash", base, c.BaseURL())
		}
		if _, err := c.Fetch("a.yaml", false); err != nil {
			t.Fatalf("Fetch via %q: %v", base, err)
		}
	}
	for _, req := range r.Requests("a.yaml") {
		if req.URL.Path != "/a.yaml" {
			t.Errorf("requested path %q, want /a.yaml", req.URL.Path)
		}
	}
	if got := r.Hits("a.yaml"); got != 2 {
		t.Errorf("hits on a.yaml = %d, want 2 (a '//' path would miss)", got)
	}
}

func TestBustAddsUniqueQueryAndNoCacheHeaders(t *testing.T) {
	r := kbtest.NewRemote(t)
	r.Put("a.yaml", "a")
	c := r.Client()
	for i := 0; i < 2; i++ {
		if _, err := c.Fetch("a.yaml", true); err != nil {
			t.Fatalf("Fetch(bust): %v", err)
		}
	}
	reqs := r.Requests("a.yaml")
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(reqs))
	}
	seen := map[string]bool{}
	for _, req := range reqs {
		ts := req.URL.Query().Get("t")
		if ts == "" {
			t.Errorf("busted request %q has no ?t=", req.URL.String())
		}
		seen[ts] = true
		if cc := req.Header.Get("Cache-Control"); cc != "no-cache, no-store, must-revalidate" {
			t.Errorf("Cache-Control = %q", cc)
		}
		if p := req.Header.Get("Pragma"); p != "no-cache" {
			t.Errorf("Pragma = %q", p)
		}
	}
	if len(seen) != 2 {
		t.Errorf("?t= values repeated across busted requests: %v", seen)
	}
}

func TestNoBustSendsPlainRequest(t *testing.T) {
	r := kbtest.NewRemote(t)
	r.Put("a.yaml", "a")
	if _, err := r.Client().Fetch("a.yaml", false); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	req := r.Requests("a.yaml")[0]
	if req.URL.RawQuery != "" {
		t.Errorf("plain request has query %q", req.URL.RawQuery)
	}
	if cc := req.Header.Get("Cache-Control"); cc != "" {
		t.Errorf("plain request has Cache-Control %q", cc)
	}
	if p := req.Header.Get("Pragma"); p != "" {
		t.Errorf("plain request has Pragma %q", p)
	}
}

func TestFetchGzipDecompresses(t *testing.T) {
	r := kbtest.NewRemote(t)
	r.SetIndex(kbtest.Index{System: kbtest.System{Version: "9.9.9"}})
	data, err := r.Client().FetchGzip(fetch.IndexPath, false)
	if err != nil {
		t.Fatalf("FetchGzip: %v", err)
	}
	if !strings.Contains(string(data), `"version":"9.9.9"`) {
		t.Errorf("decompressed index = %s, want version 9.9.9", data)
	}
}

func TestErrorsNameTheURL(t *testing.T) {
	r := kbtest.NewRemote(t)
	c := r.Client()

	_, err := c.Fetch("missing.yaml", false)
	if err == nil || !strings.Contains(err.Error(), c.URL("missing.yaml")) || !strings.Contains(err.Error(), "404") {
		t.Errorf("non-200 error = %v, want HTTP 404 naming %s", err, c.URL("missing.yaml"))
	}

	r.Put(fetch.IndexPath, "not gzip")
	_, err = c.FetchGzip(fetch.IndexPath, false)
	if err == nil || !strings.Contains(err.Error(), c.URL(fetch.IndexPath)) {
		t.Errorf("corrupt gzip error = %v, want one naming %s", err, c.URL(fetch.IndexPath))
	}

	r.Close()
	_, err = c.Fetch("a.yaml", false)
	if err == nil || !strings.Contains(err.Error(), c.URL("a.yaml")) {
		t.Errorf("network error = %v, want one naming %s", err, c.URL("a.yaml"))
	}
}
