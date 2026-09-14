package managedruntime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// This boundary adapter exercises the real HTTP acquisition and worker events;
// it replaces only runtime installation/metadata and the remote file origin.
type streamingAcquisitionAdapter struct {
	serviceAdapter
	root   string
	spec   modelSpec
	client *http.Client
}

func (a *streamingAcquisitionAdapter) Pull(ctx context.Context, id string, progress func(AcquisitionProgress)) error {
	err := acquireModel(ctx, a.root, a.spec, a.client, progress)
	a.downloaded = err == nil
	return err
}

func TestDownloadStreamsProgressBeforeVerifiedTerminal(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "8")
		w.Write([]byte("test"))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Write([]byte("data"))
	}))
	defer srv.Close()
	defer unblock()
	spec := modelSpecs["nemotron-3.5"]
	spec.size = 8
	spec.sha256 = fmt.Sprintf("%x", sha256.Sum256([]byte("testdata")))
	client := &http.Client{Transport: ggmlRewriteTransport{srv.URL, srv.Client().Transport}}
	a := &streamingAcquisitionAdapter{serviceAdapter: serviceAdapter{installed: true}, root: t.TempDir(), spec: spec, client: client}
	events := make(chan Status, 64)
	s := NewService(Options{Preferences: Defaults(), Changed: func(st Status) { events <- st }})
	s.status.Supported = true
	s.adapter = a
	if err := s.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer s.ServiceShutdown()
	waitService(t, s, "installed")
	if err := s.DownloadModel("nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	sawTransfer, sawVerify := false, false
	var operationID uint64
	for {
		select {
		case st := <-events:
			if st.Operation.Kind != "download" {
				continue
			}
			if operationID == 0 {
				operationID = st.Operation.ID
			}
			if st.Operation.ID != operationID {
				t.Fatal("operation identity changed")
			}
			p := st.Acquisition
			if p.Phase == "downloading" && p.Bytes == 4 {
				if p.TotalBytes != 8 || st.Progress != .5 || st.Operation.Outcome != "running" {
					t.Fatalf("bad streaming progress: %+v", st)
				}
				sawTransfer = true
				unblock()
			}
			if p.Phase == "verifying" && st.Operation.Outcome == "running" {
				if st.Progress != -1 {
					t.Fatal("verification cannot claim transfer completion")
				}
				sawVerify = true
			}
			if st.Operation.Outcome == "succeeded" {
				if !sawTransfer || !sawVerify || !st.Models[0].Installed {
					t.Fatalf("premature success: %+v transfer=%v verify=%v", st, sawTransfer, sawVerify)
				}
				return
			}
			if st.Operation.Outcome == "failed" {
				t.Fatalf("download failed: %+v", st)
			}
		case <-deadline:
			t.Fatal("no real streaming progress before transfer completion")
		}
	}
}

type noAcquisitionAdapter struct{ serviceAdapter }

func (a *noAcquisitionAdapter) Pull(context.Context, string, func(AcquisitionProgress)) error {
	return nil
}
func TestDownloadCannotSucceedWithoutInstalledModel(t *testing.T) {
	s := testService(t, &serviceAdapter{installed: true})
	waitService(t, s, "installed")
	s.adapter = &noAcquisitionAdapter{serviceAdapter{installed: true}}
	if err := s.DownloadModel("nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	s.wg.Wait()
	if st := s.GetStatus(); st.Operation.Outcome != "failed" {
		t.Fatalf("adapter return without installed model reported success: %+v", st.Operation)
	}
}

func TestTerminalPublicationReservesAdmission(t *testing.T) {
	s := testService(t, &serviceAdapter{installed: true})
	waitService(t, s, "installed")
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	s.changed = func(st Status) {
		if st.Operation.Kind == "download" && st.Operation.Outcome == "succeeded" {
			close(entered)
			<-release
		}
	}
	if err := s.DownloadModel("nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("no terminal notification")
	}
	if err := s.RefreshCatalog(); err != errBusy {
		t.Fatalf("new operation can replace terminal before publication: %v", err)
	}
}

func TestDownloadPublishesTerminalResult(t *testing.T) {
	a := &serviceAdapter{installed: true}
	s := testService(t, a)
	waitService(t, s, "installed")
	if err := s.DownloadModel("nemotron-3.5"); err != nil {
		t.Fatal(err)
	}
	waitService(t, s, "installed")
	b, err := json.Marshal(s.GetStatus())
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Operation struct {
			ID                   uint64
			Kind, Model, Outcome string
		}
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Operation.ID == 0 || wire.Operation.Kind != "download" || wire.Operation.Model != "nemotron-3.5" || wire.Operation.Outcome != "succeeded" {
		t.Fatalf("missing authoritative download outcome: %s", b)
	}
}
