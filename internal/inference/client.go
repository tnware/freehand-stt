package inference

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const maxResponse = 1 << 20

// Client is the application's focused OpenAI-compatible inference transport.
// It intentionally exposes only the STT, post-processing, and metadata
// capabilities consumed by this desktop client.
type Client struct{ HTTP *http.Client }

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
