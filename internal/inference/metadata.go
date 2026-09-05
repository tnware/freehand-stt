package inference

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	MaxDiscoveredModels       = 200
	MaxDiscoveredModelIDBytes = 200
)

type MetadataResult struct {
	Reachable           bool
	Probe               string
	RequestedURL        string
	HTTPStatus          int
	LatencyMilliseconds int64
	ErrorKind           string
	ModelPresence       string
	ModelIDs            []string
}

// MetadataTarget preserves the saved health-path convention: the leading slash
// is a required separator, not an instruction to replace the base URL path.
// For example, a /v1 base with /health probes /v1/health.
func MetadataTarget(base, health string) (probe, requestedURL string, err error) {
	probe = "models"
	suffix := "models"
	if health != "" {
		probe = "health"
		suffix = strings.TrimPrefix(health, "/")
	}
	requestedURL, err = endpoint(base, suffix)
	return
}

func (c *Client) TestMetadata(ctx context.Context, base, health, key, model string, headers map[string]string) (result MetadataResult) {
	started := time.Now()
	defer func() { result.LatencyMilliseconds = time.Since(started).Milliseconds() }()
	result.ModelPresence = "unavailable"
	result.Probe, result.RequestedURL, _ = MetadataTarget(base, health)
	if result.RequestedURL == "" {
		result.ErrorKind = "invalid_url"
		return result
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, result.RequestedURL, nil)
	if err != nil {
		result.ErrorKind = "invalid_url"
		return result
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		result.ErrorKind = metadataNetworkErrorKind(err)
		return result
	}
	defer resp.Body.Close()
	result.Reachable = true
	result.HTTPStatus = resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		result.ErrorKind = "response"
		return result
	}
	if len(body) > maxResponse {
		result.ErrorKind = "response_too_large"
		return result
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.ErrorKind = "http"
		return result
	}
	if health != "" {
		return result
	}
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &list) != nil || list.Data == nil {
		result.ErrorKind = "response"
		return result
	}
	result.ModelPresence = "not-listed"
	seen := make(map[string]struct{}, min(len(list.Data), MaxDiscoveredModels))
	for _, value := range list.Data {
		// Do not publish reflected credentials as selectable model IDs.
		if safePeerString(value.ID, key) == "" {
			continue
		}
		if value.ID == model {
			result.ModelPresence = "listed"
		}
		if value.ID == "" || len(value.ID) > MaxDiscoveredModelIDBytes || len(result.ModelIDs) >= MaxDiscoveredModels {
			continue
		}
		if _, exists := seen[value.ID]; exists {
			continue
		}
		seen[value.ID] = struct{}{}
		result.ModelIDs = append(result.ModelIDs, value.ID)
	}
	return result
}

func metadataNetworkErrorKind(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) && networkErr.Timeout() {
		return "timeout"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}
	var verificationErr *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	var hostnameErr x509.HostnameError
	var invalidCert x509.CertificateInvalidError
	var recordHeader tls.RecordHeaderError
	if errors.As(err, &verificationErr) || errors.As(err, &unknownAuthority) || errors.As(err, &hostnameErr) || errors.As(err, &invalidCert) || errors.As(err, &recordHeader) {
		return "tls"
	}
	return "network"
}
