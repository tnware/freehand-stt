// Package process owns native child-process trees and listener identity checks.
// Windows uses Job Objects; macOS uses a re-executed lifetime-pipe supervisor.
// It reports bounded output through explicit callbacks and has no dependency on
// runtime providers, model selection, settings, renderer state, or Wails.
package process
