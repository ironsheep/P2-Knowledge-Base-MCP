// Package fetch is the single HTTP fetcher for the P2KB's raw content: the
// index and every content file come through one Client, which owns the base
// URL, the timeout and the one cache-busting implementation.
package fetch

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultBaseURL is the KB's raw-content host, used when P2KB_BASE_URL is unset.
const DefaultBaseURL = "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main"

// BaseURLEnv names the environment variable that replaces DefaultBaseURL.
const BaseURLEnv = "P2KB_BASE_URL"

// IndexPath is the gzipped index, relative to the base URL.
const IndexPath = "deliverables/ai/p2kb-index.json.gz"

// Timeout bounds every request, index and content alike.
const Timeout = 30 * time.Second

// Client fetches repo-relative paths from one base URL.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// New returns a Client whose base is P2KB_BASE_URL when set, else DefaultBaseURL.
func New() *Client {
	return NewWithBase(os.Getenv(BaseURLEnv))
}

// NewWithBase returns a Client for base; an empty base means DefaultBaseURL.
// Trailing slashes are trimmed, so base may be given either way.
func NewWithBase(base string) *Client {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	return &Client{
		httpClient: &http.Client{Timeout: Timeout},
		baseURL:    base,
	}
}

// BaseURL returns the base every path is fetched from, without a trailing slash.
func (c *Client) BaseURL() string { return c.baseURL }

// URL returns the address path is fetched from.
func (c *Client) URL(path string) string {
	return c.baseURL + "/" + strings.TrimLeft(path, "/")
}

// Fetch returns the body at base/path.
//
// When bust is true the request bypasses CDN caches: a unique ?t=<UnixNano>
// query is appended and no-cache headers are set. GitHub's CDN (Fastly)
// ignores client Cache-Control but keys on the query string, so the ?t= is
// what forces a fresh origin fetch. Without bust the request is plain, so the
// CDN edge may answer it.
func (c *Client) Fetch(path string, bust bool) ([]byte, error) {
	url := c.URL(path)
	reqURL := url
	if bust {
		reqURL = fmt.Sprintf("%s?t=%d", url, time.Now().UnixNano())
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}
	if bust {
		req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		req.Header.Set("Pragma", "no-cache")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", url, err)
	}
	return data, nil
}

// FetchGzip is Fetch followed by gzip decompression of the body.
func (c *Client) FetchGzip(path string, bust bool) ([]byte, error) {
	body, err := c.Fetch(path, bust)
	if err != nil {
		return nil, err
	}
	gr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("decompressing %s: %w", c.URL(path), err)
	}
	defer gr.Close()

	data, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("decompressing %s: %w", c.URL(path), err)
	}
	return data, nil
}
