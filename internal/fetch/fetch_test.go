package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", c.httpClient.Timeout)
	}
}

func TestWithTimeout(t *testing.T) {
	c := NewClient(WithTimeout(60 * time.Second))
	if c.httpClient.Timeout != 60*time.Second {
		t.Errorf("timeout = %v, want 60s", c.httpClient.Timeout)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient(WithBaseURL("https://example.com/"))
	if c.baseURL != "https://example.com/" {
		t.Errorf("baseURL = %q, want https://example.com/", c.baseURL)
	}
}

func TestFetchURL(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test content"))
	}))
	defer ts.Close()

	c := NewClient()
	data, err := c.FetchURL(ts.URL)
	if err != nil {
		t.Fatalf("FetchURL failed: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("data = %q, want 'test content'", string(data))
	}
}

func TestFetchURLError(t *testing.T) {
	// Create test server that returns 404
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	c := NewClient()
	_, err := c.FetchURL(ts.URL)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestFetch(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test/path.yaml" {
			_, _ = w.Write([]byte("yaml content"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL + "/"))
	data, err := c.Fetch("test/path.yaml")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if string(data) != "yaml content" {
		t.Errorf("data = %q, want 'yaml content'", string(data))
	}
}

func TestHeadURL(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("method = %q, want HEAD", r.Method)
		}
		if r.URL.Path == "/exists" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := NewClient()

	// Test existing resource
	exists, err := c.HeadURL(ts.URL + "/exists")
	if err != nil {
		t.Fatalf("HeadURL failed: %v", err)
	}
	if !exists {
		t.Error("expected exists=true for /exists")
	}

	// Test non-existing resource
	exists, err = c.HeadURL(ts.URL + "/notexists")
	if err != nil {
		t.Fatalf("HeadURL failed: %v", err)
	}
	if exists {
		t.Error("expected exists=false for /notexists")
	}
}

func TestHead(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test/exists.yaml" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL + "/"))

	exists, err := c.Head("test/exists.yaml")
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}
	if !exists {
		t.Error("expected exists=true")
	}
}

func TestFetchGzipURL(t *testing.T) {
	// Skip this test in short mode as it would need a proper gzip response
	if testing.Short() {
		t.Skip("skipping gzip test in short mode")
	}

	// Create test server with gzipped content
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This would need actual gzip data
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewClient()
	_, err := c.FetchGzipURL(ts.URL)
	// This will fail because the response isn't actually gzipped
	// In real tests, we'd provide proper gzipped data
	if err == nil {
		t.Log("Note: FetchGzipURL test needs proper gzipped data")
	}
}

func TestClientOptions(t *testing.T) {
	// Test multiple options
	c := NewClient(
		WithTimeout(45*time.Second),
		WithBaseURL("https://custom.example.com/"),
	)

	if c.httpClient.Timeout != 45*time.Second {
		t.Errorf("timeout = %v, want 45s", c.httpClient.Timeout)
	}
	if c.baseURL != "https://custom.example.com/" {
		t.Errorf("baseURL = %q, want https://custom.example.com/", c.baseURL)
	}
}
