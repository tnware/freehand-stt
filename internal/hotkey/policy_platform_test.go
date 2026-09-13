package hotkey

import "testing"

func TestPlatformFunctionKeys(t *testing.T) {
 for _, tc := range []struct{os, value string; kind ShortcutRejectionKind} {
 {"darwin", "Ctrl+F12", ""}, {"darwin", "F20", ""}, {"darwin", "F21", RejectionUnsupported}, {"darwin", "Ctrl+F24", RejectionUnsupported},
 {"windows", "Ctrl+F12", RejectionReserved}, {"windows", "F24", ""},
 } {
  chord, err := parseForPlatform(ToggleRecording, tc.value, tc.os)
  if tc.kind == "" { if err != nil || chord.String() == "" {t.Fatalf("%+v = %#v %v", tc,chord,err)} } else if r,ok := RejectionDetails(err); !ok || r.Kind != tc.kind {t.Fatalf("%+v = %v",tc,err)}
 }
}

func TestPlatformCommandAlias(t *testing.T) {
 for _, tc := range []struct{os string; want Modifier}{{"darwin", Meta}, {"windows", Ctrl}} {
  chord, err := parseForPlatform(ToggleRecording, "CmdOrCtrl+D", tc.os)
  if err != nil || chord.Modifiers != tc.want { t.Fatalf("%s alias = %#v, %v", tc.os, chord, err) }
 }
}
