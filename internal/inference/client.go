package inference

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

const maxResponse = 1 << 20

// Client is the application's focused OpenAI-compatible inference transport.
// It intentionally exposes only the STT, post-processing, and metadata
// capabilities consumed by this desktop client.
type Client struct {
	HTTP                 *http.Client
	profile              compatibility.ID
	transcriptionOptions compatibility.TranscriptionOptions
	cleanupOptions       compatibility.CleanupOptions
}

// WithCompatibility captures the selection alongside the caller's existing
// settings/credential snapshot while sharing only the concurrency-safe HTTP client.
func (c *Client) WithCompatibility(id compatibility.ID) *Client {
	copy := *c
	copy.profile = id
	return &copy
}

func (c *Client) contract(role compatibility.Role) (compatibility.Contract, error) {
	contract, err := compatibility.Resolve(c.profile, role)
	if err != nil {
		return compatibility.Contract{}, &Error{Kind: "invalid_settings", Message: err.Error()}
	}
	return contract, nil
}

func New() *Client {
	return &Client{HTTP: &http.Client{CheckRedirect: denyRedirect, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, MaxIdleConns: 4, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 10 * time.Second}}}
}

// denyRedirect leaves the original response available for the route's bounded
// HTTP error handling. Never replay credentials, audio, or text, even on the
// same origin; endpoint configuration must name the final URL.
func denyRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

func endpoint(base, suffix string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(strings.TrimSuffix(u.Path, "/"), suffix)
	return u.String(), nil
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
