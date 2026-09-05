package inference

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataRequiresModelListButHealthAcceptsOpaqueSuccess(t *testing.T) {
	for _, body := range []string{"<html>gateway</html>", "", "null", "[]", `{}`, `{"data":null}`, `{"data":{}}`, `{"data":[{"id":123}]}`, `{"data":[]} trailing`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet || (r.URL.Path != "/v1/models" && r.URL.Path != "/v1/health") {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			result := New().TestMetadata(context.Background(), server.URL+"/v1", "", "", "speech", nil)
			if !result.Reachable || result.HTTPStatus != 200 || result.ErrorKind != "response" || result.ModelPresence != "unavailable" || len(result.ModelIDs) != 0 {
				t.Fatalf("invalid inventory result = %+v", result)
			}
			health := New().TestMetadata(context.Background(), server.URL+"/v1", "/health", "", "speech", nil)
			if !health.Reachable || health.ErrorKind != "" || health.ModelPresence != "unavailable" {
				t.Fatalf("health = %+v", health)
			}
			if calls != 2 {
				t.Fatalf("requests = %d; metadata checks must not invoke inference or retry", calls)
			}
		})
	}
}

func TestMetadataAcceptsEmptyModelList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{"data":[]}`) }))
	defer server.Close()
	result := New().TestMetadata(context.Background(), server.URL, "", "", "speech", nil)
	if !result.Reachable || result.ErrorKind != "" || result.ModelPresence != "not-listed" || len(result.ModelIDs) != 0 {
		t.Fatalf("result = %+v", result)
	}
}
