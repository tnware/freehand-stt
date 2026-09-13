package hotkey

// The wire key IDs stay compatible with persisted Windows chords. Adapters
// translate physical native keycodes; macOS/Carbon exposes F1 through F20 only.
func primaryForPlatform(key uint32, platform string) bool {
 if platform == "darwin" { return key < 0x84 || key > 0x87 }
 return key != 0x7B
}
