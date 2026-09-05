package dictation

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A blocked callback makes the write/stop overlap explicit without relying on
// the race detector to catch concurrent access inside a real FramePipe.
type gatedCaptureSink struct {
	entered           chan struct{}
	release           chan struct{}
	closed            chan struct{}
	writeFinished     atomic.Bool
	closedDuringWrite atomic.Bool
	writes            atomic.Int32
	closes            atomic.Int32
}

func (s *gatedCaptureSink) WritePCM([]byte) bool {
	s.writes.Add(1)
	close(s.entered)
	<-s.release
	s.writeFinished.Store(true)
	return true
}

func (s *gatedCaptureSink) Close() {
	if !s.writeFinished.Load() {
		s.closedDuringWrite.Store(true)
	}
	if s.closes.Add(1) == 1 {
		close(s.closed)
	}
}

func TestStreamingCaptureStopWaitsForAdmittedWrite(t *testing.T) {
	for _, operation := range []string{"stop", "cancel", "close"} {
		t.Run(operation, func(t *testing.T) {
			capture := &streamingCapture{}
			sink := &gatedCaptureSink{
				entered: make(chan struct{}),
				release: make(chan struct{}),
				closed:  make(chan struct{}),
			}
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(sink.release) }) }
			defer release()
			if _, err := capture.StartStream(context.Background(), "", 0, sink); err != nil {
				t.Fatal(err)
			}
			written := make(chan bool, 1)
			go func() { written <- capture.write(pcmFrame(1200)) }()
			select {
			case <-sink.entered:
			case <-time.After(time.Second):
				t.Fatal("write did not reach the sink")
			}

			stopping := make(chan struct{})
			stopped := make(chan error, 1)
			var stopReturned atomic.Bool
			go func() {
				close(stopping)
				var err error
				switch operation {
				case "stop":
					_, err = capture.Stop(context.Background())
				case "cancel":
					err = capture.Cancel(context.Background())
				case "close":
					err = capture.Close()
				}
				stopReturned.Store(true)
				stopped <- err
			}()
			select {
			case <-stopping:
			case <-time.After(time.Second):
				t.Fatal("stop operation did not start")
			}
			// The gate holds the admitted write throughout this observation
			// interval. Stop must neither close the sink nor return early.
			select {
			case <-sink.closed:
				t.Error("sink closed while its admitted write was blocked")
			case <-time.After(50 * time.Millisecond):
			}
			if stopReturned.Load() {
				t.Error("stop returned before its admitted write finished")
			}
			release()
			select {
			case accepted := <-written:
				if !accepted {
					t.Error("admitted write was rejected")
				}
			case <-time.After(time.Second):
				t.Fatal("write did not finish after release")
			}
			select {
			case err := <-stopped:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("stop did not finish after the write")
			}
			if sink.closedDuringWrite.Load() {
				t.Error("Close overlapped WritePCM")
			}
			if capture.write(pcmFrame(800)) {
				t.Error("write was admitted after stop")
			}
			if err := capture.Close(); err != nil {
				t.Fatal(err)
			}
			if sink.writes.Load() != 1 || sink.closes.Load() != 1 {
				t.Errorf("sink calls: writes=%d closes=%d; want one each", sink.writes.Load(), sink.closes.Load())
			}
		})
	}
}
