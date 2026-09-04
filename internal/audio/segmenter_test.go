package audio

import (
	"errors"
	"testing"
)

type decisions struct {
	values []bool
	err    error
}

func (d *decisions) Speech([]int16) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	if len(d.values) == 0 {
		return false, nil
	}
	value := d.values[0]
	d.values = d.values[1:]
	return value, nil
}
func (*decisions) Close() error { return nil }

func testFrame(value byte) []byte {
	frame := make([]byte, VADFrameBytes)
	for i := range frame {
		frame[i] = value
	}
	return frame
}

func TestSegmenterClosesOnlyAfterSustainedSilence(t *testing.T) {
	detector := &decisions{values: []bool{false, true, true, true, false, false}}
	segmenter, err := NewSegmenter(detector, 1, 40)
	if err != nil {
		t.Fatal(err)
	}
	segmenter.targetBytes = 4 * VADFrameBytes
	var chunk []byte
	for i := byte(1); i <= 6; i++ {
		chunk, err = segmenter.Push(testFrame(i))
		if err != nil {
			t.Fatal(err)
		}
		if i < 6 && chunk != nil {
			t.Fatalf("chunk closed on frame %d", i)
		}
	}
	if len(chunk) != 6*VADFrameBytes {
		t.Fatalf("chunk bytes = %d, want %d", len(chunk), 6*VADFrameBytes)
	}
	if chunk[0] != 1 || chunk[len(chunk)-1] != 6 {
		t.Fatal("pre-roll or trailing silence was not preserved")
	}
}

func TestSegmenterFlushesSpeechAndDropsSilence(t *testing.T) {
	speech, _ := NewSegmenter(&decisions{values: []bool{true}}, 1, 1000)
	if _, err := speech.Push(testFrame(7)); err != nil {
		t.Fatal(err)
	}
	if chunk := speech.Flush(); len(chunk) != VADFrameBytes {
		t.Fatalf("short speech flush = %d bytes", len(chunk))
	}

	silence, _ := NewSegmenter(&decisions{values: []bool{false, false}}, 1, 1000)
	_, _ = silence.Push(testFrame(0))
	_, _ = silence.Push(testFrame(0))
	if chunk := silence.Flush(); chunk != nil {
		t.Fatalf("silence produced %d bytes", len(chunk))
	}
}

func TestSegmenterPropagatesDetectorFailure(t *testing.T) {
	want := errors.New("detector failed")
	segmenter, _ := NewSegmenter(&decisions{err: want}, 1, 1000)
	if _, err := segmenter.Push(testFrame(1)); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestSegmenterReportsStabilizedDetectorActivity(t *testing.T) {
	segmenter, err := NewSegmenter(&decisions{values: []bool{
		false, false, false,
		true, true,
		false, false, false,
	}}, 1, 60)
	if err != nil {
		t.Fatal(err)
	}

	for frame := 1; frame <= 8; frame++ {
		if _, err := segmenter.Push(testFrame(byte(frame))); err != nil {
			t.Fatal(err)
		}
		active, known := segmenter.VoiceActivity()
		switch {
		case frame < 3 && known:
			t.Fatalf("activity became known on frame %d", frame)
		case frame == 3 && (!known || active):
			t.Fatalf("frame 3 activity = active:%v known:%v, want silence", active, known)
		case frame == 5 && (!known || !active):
			t.Fatalf("frame 5 activity = active:%v known:%v, want speech", active, known)
		case (frame == 6 || frame == 7) && (!known || !active):
			t.Fatalf("frame %d activity changed before the sustained pause", frame)
		case frame == 8 && (!known || active):
			t.Fatalf("frame 8 activity = active:%v known:%v, want silence", active, known)
		}
	}
}

func TestSpeechSegmenterTrimsToConfiguredPadding(t *testing.T) {
	detector := &decisions{values: []bool{
		false, false, false, false, false,
		true, true, true,
		false, false, false, false, false,
	}}
	segmenter, err := NewSpeechSegmenter(detector, SpeechSegmenterOptions{
		ActivitySilenceMilliseconds: 40,
		TrimSilence:                 true,
		SpeechPaddingMilliseconds:   40,
	})
	if err != nil {
		t.Fatal(err)
	}
	for frame := byte(1); frame <= 13; frame++ {
		if _, err := segmenter.Push(testFrame(frame)); err != nil {
			t.Fatal(err)
		}
	}
	chunk := segmenter.Flush()
	if got, want := len(chunk), 7*VADFrameBytes; got != want {
		t.Fatalf("trimmed bytes = %d, want %d", got, want)
	}
	if chunk[0] != 4 || chunk[len(chunk)-1] != 10 {
		t.Fatalf("trimmed frame range = %d..%d, want 4..10", chunk[0], chunk[len(chunk)-1])
	}
}

func TestSpeechSegmenterPreservesWholeRecordingWhenTrimmingIsOff(t *testing.T) {
	segmenter, err := NewSpeechSegmenter(&decisions{values: []bool{false, false, true, true, false}}, SpeechSegmenterOptions{
		ActivitySilenceMilliseconds: 40,
		SpeechPaddingMilliseconds:   40,
	})
	if err != nil {
		t.Fatal(err)
	}
	for frame := byte(1); frame <= 5; frame++ {
		if _, err := segmenter.Push(testFrame(frame)); err != nil {
			t.Fatal(err)
		}
	}
	chunk := segmenter.Flush()
	if got, want := len(chunk), 5*VADFrameBytes; got != want {
		t.Fatalf("untrimmed bytes = %d, want %d", got, want)
	}
	if chunk[0] != 1 || chunk[len(chunk)-1] != 5 {
		t.Fatal("trimming-off recording did not preserve its complete boundary")
	}
}

func TestSpeechSegmenterAutoStopRequiresConfirmedSpeechAndSustainedSilence(t *testing.T) {
	segmenter, err := NewSpeechSegmenter(&decisions{values: []bool{
		true, false, // A noise blip cannot arm the policy.
		true, true, true,
		false, false, false, false, false,
	}}, SpeechSegmenterOptions{
		ActivitySilenceMilliseconds:       40,
		SpeechPaddingMilliseconds:         40,
		AutoStopSilenceMilliseconds:       100,
		AutoStopMinimumSpeechMilliseconds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	for frame := 1; frame <= 10; frame++ {
		if _, err := segmenter.Push(testFrame(byte(frame))); err != nil {
			t.Fatal(err)
		}
		if frame < 10 && segmenter.AutoStopReady() {
			t.Fatalf("automatic stop triggered early on frame %d", frame)
		}
	}
	if !segmenter.AutoStopReady() {
		t.Fatal("automatic stop did not trigger after confirmed speech and five silence frames")
	}
	enabled, armed, countdown, remaining := segmenter.AutoStopStatus()
	if !enabled || !armed || !countdown || remaining != 0 {
		t.Fatalf("automatic stop status = enabled:%v armed:%v countdown:%v remaining:%d", enabled, armed, countdown, remaining)
	}
}

func TestSpeechSegmenterResumedSpeechCancelsAutoStopCountdown(t *testing.T) {
	segmenter, err := NewSpeechSegmenter(&decisions{values: []bool{
		true, true, true,
		false, false, false,
		true,
	}}, SpeechSegmenterOptions{
		ActivitySilenceMilliseconds:       40,
		SpeechPaddingMilliseconds:         40,
		AutoStopSilenceMilliseconds:       100,
		AutoStopMinimumSpeechMilliseconds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	for frame := 1; frame <= 6; frame++ {
		if _, err := segmenter.Push(testFrame(byte(frame))); err != nil {
			t.Fatal(err)
		}
	}
	_, armed, countdown, remaining := segmenter.AutoStopStatus()
	if !armed || !countdown || remaining != 40 {
		t.Fatalf("countdown before resumed speech = armed:%v countdown:%v remaining:%d", armed, countdown, remaining)
	}
	if _, err := segmenter.Push(testFrame(7)); err != nil {
		t.Fatal(err)
	}
	_, armed, countdown, remaining = segmenter.AutoStopStatus()
	if !armed || countdown || remaining != 100 || segmenter.AutoStopReady() {
		t.Fatalf("countdown after resumed speech = armed:%v countdown:%v remaining:%d ready:%v", armed, countdown, remaining, segmenter.AutoStopReady())
	}
}

func TestFramePipeReframesAndZeroesBuffers(t *testing.T) {
	pipe := NewFramePipe()
	input := make([]byte, VADFrameBytes+17)
	for i := range input {
		input[i] = 9
	}
	if !pipe.WritePCM(input[:101]) || !pipe.WritePCM(input[101:]) {
		t.Fatal("pipe rejected available capacity")
	}
	pipe.Close()

	first := <-pipe.Frames()
	second := <-pipe.Frames()
	if len(first) != VADFrameBytes || len(second) != VADFrameBytes {
		t.Fatalf("frame lengths = %d, %d", len(first), len(second))
	}
	for _, value := range second[:17] {
		if value != 9 {
			t.Fatal("partial frame data was not preserved")
		}
	}
	for _, value := range second[17:] {
		if value != 0 {
			t.Fatal("partial frame was not padded with silence")
		}
	}
	pipe.Release(first)
	pipe.Release(second)
	if _, ok := <-pipe.Frames(); ok {
		t.Fatal("frame channel remained open")
	}
}
