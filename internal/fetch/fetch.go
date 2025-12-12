// Package fetch provides HTTP fetching utilities for P2KB.
package fetch

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides HTTP fetching with configurable timeouts.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// Option configures the Client.
type Option func(*Client)

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithBaseURL sets the base URL for requests.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// NewClient creates a new HTTP client with default settings.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main/",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Fetch retrieves content from a URL.
func (c *Client) Fetch(path string) ([]byte, error) {
	url := c.baseURL + path
	return c.FetchURL(url)
}

// FetchURL retrieves content from an absolute URL.
func (c *Client) FetchURL(url string) ([]byte, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// FetchGzip retrieves and decompresses gzipped content.
func (c *Client) FetchGzip(path string) ([]byte, error) {
	url := c.baseURL + path
	return c.FetchGzipURL(url)
}

// FetchGzipURL retrieves and decompresses gzipped content from an absolute URL.
func (c *Client) FetchGzipURL(url string) ([]byte, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	data, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to read gzipped content: %w", err)
	}

	return data, nil
}

// Head performs a HEAD request to check if a resource exists.
func (c *Client) Head(path string) (bool, error) {
	url := c.baseURL + path
	return c.HeadURL(url)
}

// HeadURL performs a HEAD request to an absolute URL.
func (c *Client) HeadURL(url string) (bool, error) {
	resp, err := c.httpClient.Head(url)
	if err != nil {
		return false, fmt.Errorf("HTTP HEAD failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
