package inference

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomHealthPathPreservesBaseRelativeTargets(t *testing.T) {
	for _, tc := range []struct{ name, basePath, healthPath, want string }{
		{"origin base", "", "/health", "/health"},
		{"versioned base", "/v1", "/health", "/v1/health"},
		{"trailing slash", "/gateway/v1/", "/ready", "/gateway/v1/ready"},
		{"nested health", "/api", "/health/ready", "/api/health/ready"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, status := range []int{200, 404} {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != http.MethodGet || r.URL.Path != tc.want {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					w.WriteHeader(status)
					_, _ = io.WriteString(w, "health response")
				}))
				result := New().TestMetadata(context.Background(), server.URL+tc.basePath, tc.healthPath, "", "model", nil)
				server.Close()
				wantError := ""
				if status != 200 {
					wantError = "http"
				}
				if calls != 1 || result.Probe != "health" || result.RequestedURL != server.URL+tc.want || result.HTTPStatus != status || result.ErrorKind != wantError || result.ModelPresence != "unavailable" {
					t.Fatalf("result=%+v calls=%d", result, calls)
				}
			}
		})
	}
}
