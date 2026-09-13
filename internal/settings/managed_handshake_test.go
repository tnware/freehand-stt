package settings

import (
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"testing"
)

func TestManagedSaveCannotRaceOlderPublication(t *testing.T) {
	s, _, _, _ := transactionalService(false)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	WithManagedRuntimes(nil, func([]managedruntime.Instance) { close(entered); <-release; _ = s.GetSettings() })(s)
	go func() { _, err := s.SaveSettings(request(s.current(), "")); done <- err }()
	<-entered
	err := SaveManagedInstances(s, []managedruntime.Instance{testInstance()})
	close(release)
	if err == nil {
		t.Fatal("overlapping publication admitted")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(s.current().ManagedRuntimes) != 0 {
		t.Fatal("rejected inventory became authoritative")
	}
}
