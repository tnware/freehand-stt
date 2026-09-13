package windowing

import "testing"

func TestOpenManagedRuntimeSettings(t *testing.T) {
 got := ""
 s := NewService(func(section string) { got = section }, nil, nil, nil, nil)
 if err := s.OpenSettings("local-runtime"); err != nil { t.Fatal(err) }
 if got != "local-runtime" { t.Fatalf("opened %q", got) }
}
