package managedruntime

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// The OS child boundary reproduces the pinned upstream's actual file lifecycle,
// not imaginary progress output. See testdata/nemo-acquisition-source.md.
func TestNemoAcquisitionMeasuredProgressAndOutcomes(t *testing.T) {
	for _, outcome := range []string{"succeeded", "cancelled", "failed", "corrupt"} {
		t.Run(outcome, func(t *testing.T) {
			root := t.TempDir()
			a, _ := managedAdapterFixture(t, root)
			spec := modelSpecs["nemotron-3.5"]
			originalLaunch := a.launch
			release := make(chan struct{})
			a.launch = func(ctx context.Context, exe string, args []string, dir string, env []string) (processHandle, error) {
				if strings.Join(args, " ") != "--json model pull "+spec.repo {
					return originalLaunch(ctx, exe, args, dir, env)
				}
				if err := os.MkdirAll(spec.directory(root), 0700); err != nil {
					return nil, err
				}
				if err := os.WriteFile(spec.path(root)+".partial", []byte("fix"), 0600); err != nil {
					return nil, err
				}
				p := &fakeProcess{done: make(chan struct{}), closeJob: func() {}}
				p.stdout.Write([]byte("private child URL/token canary"))
				go func() {
					defer close(p.done)
					select {
					case <-ctx.Done():
						p.err = errors.New("private process exit canary")
						return
					case <-release:
					}
					if outcome == "failed" {
						p.err = errors.New("private download failure canary")
						return
					}
					data := []byte("fixture")
					if outcome == "corrupt" {
						data = []byte("corrupt")
					}
					if err := os.WriteFile(spec.path(root)+".partial", data, 0600); err != nil {
						p.err = err
						return
					}
					p.err = os.Rename(spec.path(root)+".partial", spec.path(root))
				}()
				return p, nil
			}
			events := make(chan Status, 64)
			s := newTestWorker(t, false)
			s.changed = func(st workerSnapshot) { events <- st.Status }
			s.status.Supported = true
			s.adapter = a
			if err := s.startup(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer s.ServiceShutdown()
			waitWorker(t, s, "installed")
			if err := s.DownloadModel("nemotron-3.5"); err != nil {
				t.Fatal(err)
			}
			expected := outcome
			if outcome == "corrupt" {
				expected = "failed"
			}
			sawBytes, sawVerify, released := false, false, false
			deadline := time.After(3 * time.Second)
			for {
				select {
				case st := <-events:
					if st.Operation.Kind != "download" {
						continue
					}
					if strings.Contains(st.Error, "canary") || strings.Contains(st.Operation.Error, "canary") {
						t.Fatal("private child data escaped")
					}
					if st.Acquisition.Phase == "downloading" && st.Acquisition.Bytes == 3 && st.Operation.Outcome == "running" {
						if st.Acquisition.TotalBytes != 7 || st.Operation.Outcome != "running" {
							t.Fatalf("bad NeMo progress: %+v", st)
						}
						sawBytes = true
						if !released {
							released = true
							if outcome == "cancelled" {
								if err := s.Cancel(); err != nil {
									t.Fatal(err)
								}
							} else {
								close(release)
							}
						}
					}
					if st.Acquisition.Phase == "verifying" && st.Operation.Outcome == "running" {
						sawVerify = true
					}
					if st.Operation.Outcome != "running" {
						if !sawBytes || st.Operation.Outcome != expected {
							t.Fatalf("bad terminal status: %+v", st)
						}
						if outcome == "succeeded" && (!sawVerify || !st.Models[0].Installed) {
							t.Fatal("unverified success")
						}
						if expected == "failed" && st.Operation.Error == "" {
							t.Fatal("failure has no safe feedback")
						}
						return
					}
				case <-deadline:
					t.Fatal("NeMo did not publish measured progress/outcome")
				}
			}
		})
	}
}
