package tts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type writeFunc func([]byte) (int, error)

func (f writeFunc) Write(b []byte) (int, error) { return f(b) }

func TestCancelledExportDoesNotTruncateExistingDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.wav")
	original := []byte("existing user file")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := writeAudioFile(ctx, path, []byte("new audio")); !errors.Is(err, context.Canceled) {
		t.Fatalf("save = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("cancelled export modified the destination")
	}
}
func TestExportChecksCancellationBetweenWrites(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	writer := writeFunc(func(b []byte) (int, error) { calls++; cancel(); return len(b), nil })
	if err := writeAudio(ctx, writer, make([]byte, 128*1024)); !errors.Is(err, context.Canceled) {
		t.Fatalf("write = %v", err)
	}
	if calls != 1 {
		t.Fatalf("continued writing after cancellation: %d", calls)
	}
}
func TestExportRejectsShortWrites(t *testing.T) {
	writer := writeFunc(func(b []byte) (int, error) { return len(b) - 1, nil })
	if err := writeAudio(context.Background(), writer, []byte{1, 2}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("write = %v", err)
	}
}
func TestExportWritesExactWAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.wav")
	wav := bytes.Repeat([]byte{1, 2, 3, 4}, 20000)
	if err := writeAudioFile(context.Background(), path, wav); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, wav) {
		t.Fatal("export did not preserve WAV bytes")
	}
}
