package windowing

import (
	"errors"
	"reflect"
	"testing"
)

func TestProcessOutputNavigationRevokesPreviousViewer(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil)
	var calls []string
	ConfigureProcessOutput(s, ProcessOutputNavigation{
		Exists: func(id string) bool { return id == "llama-cpp" || id == "whisper-cpp" },
		Open:   func(id string) error { calls = append(calls, "open:"+id); return nil },
		Close:  func(id string) { calls = append(calls, "close:"+id) },
	})
	for _, id := range []string{"llama-cpp", "whisper-cpp"} {
		if err := s.OpenProcessOutput(ProcessOutputRequest{InstanceID: id}); err != nil {
			t.Fatal(err)
		}
	}
	if s.CurrentProcessOutput().InstanceID != "whisper-cpp" {
		t.Fatal("viewer selection was not updated")
	}
	s.CloseProcessOutput()
	if s.CurrentProcessOutput().InstanceID != "" {
		t.Fatal("closed viewer retained selection")
	}
	want := []string{"open:llama-cpp", "close:llama-cpp", "open:whisper-cpp", "close:whisper-cpp"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
}

func TestProcessOutputNavigationRejectsUnavailableSelection(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil)
	ConfigureProcessOutput(s, ProcessOutputNavigation{
		Exists: func(id string) bool { return id == "local" },
		Open:   func(string) error { return errors.New("window unavailable") },
	})
	for _, id := range []string{"", "missing", "local"} {
		if err := s.OpenProcessOutput(ProcessOutputRequest{InstanceID: id}); err == nil {
			t.Fatal("invalid or failed navigation succeeded", id)
		}
		if s.CurrentProcessOutput().InstanceID != "" {
			t.Fatal("failed navigation retained access")
		}
	}
}
