---
title: Native Windows acceptance checklist
description: Interactive acceptance checks that must run on a real Windows desktop.
---

Cross-compilation is not acceptance. Run these checks on a non-elevated Windows desktop with a disposable API credential and one explicitly selected STT route. Record browser results, Windows compilation, Windows interactive acceptance, and packaged macOS acceptance separately. Windows evidence does not establish macOS behavior.

- Confirm one mark-only tray icon, one main window with Settings and Connections panes, plus reusable About and Transcription details windows. Switch Windows between light and dark modes and inspect the purpose-drawn tray family at 100%, 125%, 150%, and 200% scale; the waveform and caret must remain distinct without a miniature application tile. Confirm the executable, taskbar, Explorer, installer, uninstaller, Installed apps, and Start Menu retain the tiled application family. During microphone and stored-file workflows, confirm the tray tooltip and its disabled status/detail rows distinguish recording, VAD silence, checkpoints, transcription, cleanup, upload/streaming, cancellation, success, attention, and failure without displaying transcript text, file identity, model, endpoint, or raw errors. Show/Hide Freehand, Settings, About, active cancellation, available transcript copy, and authoritative Quit must remain state-correct. Repeated Settings actions must reveal the main window at General after resolving an active draft; About must focus/reuse its existing native window. A second launch must reveal the main window; left-click opens the compact dictation panel and right-click opens the native menu. Closing the main window first resolves any unsaved configuration draft, then hides the workspace. The native menu must not offer a recording-start action.
- Confirm the credential persists in Windows Credential Manager and never appears in SQLite, legacy JSON, logs, process arguments, events, or returned renderer DTOs. Confirm the transient password draft clears after save and when settings are left or hidden.
- Enumerate the default and an explicit microphone; record, stop, cancel, reach the configured duration bound, unplug a device, and deny microphone privacy permission. After at least five seconds idle, confirm a shortcut does not publish Recording until native capture has actually started and that a prepared device is reused safely.
- In Settings, confirm each shortcut row explains its required or optional forms before capture. Capture modifier-plus-letter, Space, F1/F11, and an unmodified programmable F13-F24 key; verify live keycaps, Escape cancellation, restore-default/clear behavior, saved state, and consistent spoken names. Confirm incomplete, F12, unsupported, duplicate-Freehand, Windows-owned, and attempted toggle/show-swap outcomes are distinct and preserve every prior binding. Repeat modifier ordering with Win/Super aliases, one representative AltGr layout, and Sticky Keys/Filter Keys both enabled and disabled; ordinary input and injected input must not trigger capture actions.
- Exercise hold-to-talk press/release, key repeat, modifier-order permutations, release of the primary and each required modifier, injected input, reconfiguration failure, hook teardown, logoff/session interruption, and application shutdown. With automatic stop enabled, speak and then remain silent beyond its threshold; confirm the recording remains active until key release while VAD feedback and trimming still work. Confirm normal input is never suppressed.
- In direct-input mode, dictate short, long, multiline, and non-ASCII text into Notepad, a Chromium text field, VS Code or another editor, a terminal, and an Office-style rich-text target. Ordinary results should appear in one immediate update; long results should complete without fixed-delay stepping, truncation, or broken surrogate pairs. Change the top-level window or focused child during transcription and during a long insertion, and restart the target process; unsafe delivery must stop before the next dispatch and remain recoverable through **Copy transcript**, never continue into the new target. Confirm direct-input logs contain only unit/batch counts, duration, strategy, and bounded failure stage. In manual-copy mode, confirm a completed run returns successfully with a visible Copy action and performs no insertion or clipboard mutation even when the original target is still focused. Confirm clipboard paste is visible but disabled in Settings; no automatic path may refocus, paste, or overwrite the clipboard.
- Exercise cancellation and shutdown during capture and HTTP, an oversized/malformed response, a failed endpoint, an elevated target (UIPI), and clipboard contention.
- Exercise local VAD in every supported mode and with VAD disabled. Verify the silence indicator debounce, trimmed and untrimmed boundaries, speech padding, auto-stop waiting/armed/countdown/cancelled-by-speech behavior, manual stop during a countdown, silence splitting disabled, pause-aware checkpoints, hard segment limits, and final partial segments. Checkpoint text must remain ordered, the final insertion must contain every completed segment exactly once, and cancellation must zero/discard unfinished audio.
- Choose each supported stored-audio type through the Go-owned native picker; exercise picker cancellation, completed and streamed responses, buffered SSE, unsupported-stream fallback, upload-limit failure, cancellation, explicit copy, raw/processed history, Unicode/long paths, removable and network-backed files, and mutual exclusion with microphone recording. After selection, test disappearance, replacement, size/modification changes, and a direct symbolic link; each unsafe change must require reselection without upload. The full path and audio bytes must never enter renderer state, binding arguments, events, or logs.
- Enable post-processing with a separately configured endpoint and credential. Confirm the workspace and native overlay move from Transcribing to a visually distinct Post-processing phase only when processing is enabled. Confirm successful cleanup, empty output, timeout, HTTP failure, and credential failure; every failure must preserve and deliver the raw transcript and record visible `fallback_raw` history state.
- Confirm history retains nothing while disabled. Enable it, complete enough inserted and copy-required Unicode dictations to overflow the visible history area, verify the newest-first outcome/time/character metadata and independent scrolling, explicitly copy an entry, remove one entry without disturbing the others, then confirm Clear history, disabling history, and quit/relaunch each leave no entries. Exercise oversized processed output and confirm the raw transcript remains with visible memory-limit fallback details; an oversized raw result must not be retained. No history action may focus another application or copy automatically.
- Confirm startup enable/disable writes only the app-owned HKCU Run value and survives sign-out/sign-in. A `--startup` launch must remain resident with all application windows hidden; a normal direct or second ordinary launch must reveal the main window.
- In Windows light and dark modes, confirm the main, About, and Transcription details windows are opaque and legible by default. Enable Mica, save, and confirm the running windows do not change material and show a restart requirement; after tray Quit and relaunch, confirm Mica reaches the main header/status strip, the Settings navigation/action bar, and the About content/action bar without reducing control contrast. Disable it and repeat back to the opaque default. On systems without Mica, confirm any opted-in Wails fallback remains legible and opt-out remains fully opaque.
- Use the page Settings cog to open Voice, Audio file, and Text to speech options beside the workflow. Select a connection and model, toggle cleanup, switch between Custom instruction and S1-mini, and exercise the trained S1-mini controls. Contextual Save must apply edits in place; failed saves must preserve the draft and leave backend-applied configuration unchanged. First-run setup controls still save valid changes immediately. Endpoint, authentication, and credential edits belong to the Connections rail page. In Cleanup options, verify custom-instruction validation/reset/persistence and the read-only S1 instruction/control-line preview. Confirm changed selections invalidate stale metadata, immediate actions cannot race or apply an unrelated draft, and leaving or closing configuration protects unsaved edits.
- Confirm the test build creates exactly one passive native overlay and no overlay WebView. Across first show, repeated state updates, hide, and live appearance changes, record the foreground top-level HWND, focused child HWND, overlay HWND, thread ID, process ID, and process creation time before and after; the target identities and overlay HWND/thread must remain unchanged. Verify Notepad and one Chromium application retain the caret, the overlay is absent from Alt+Tab/taskbar, clicks pass through or are ignored, and placement stays within the active monitor work area at 100%/150%/200% DPI. Exercise minimum/default/maximum size, opacity, top-distance, and glow while idle and recording; confirm proportional scaling, bounded translucency, zero glow, correct work-area-relative placement, no entrance replay, and no extra level tap. Confirm the overlay moves distinctly through speech, stabilized silence, resumed speech, and toggle-recording countdown states, that the countdown rail shrinks to the coordinator deadline, and that hold-to-talk never shows a countdown. Turn **Settings → Overlay → Show status overlay** off and confirm the native surface and its thread/resources close immediately while dictation continues normally; turn it on while idle and while recording and confirm one surface returns with the saved appearance and current state. Repeated cycles must not leak windows, taps, GDI objects, or threads, and disabled startup must create no overlay resources. A genuine user focus change must still produce copy-required behavior. If any overlay action changes focus, disable the overlay; never weaken target validation or restore focus programmatically.
- From **Settings → Overlay**, run the native preview through every curated layout, anchor, visibility, surface, visualizer, and motion policy. Draft changes must update the same surface without saving; discard/close must restore the applied configuration; preview must also work when the applied overlay is off and release the temporary surface afterward. Start dictation during preview and confirm immediate preemption. With taskbars on each screen edge and mixed DPI monitors, confirm each new recording selects the target monitor's work area once and does not chase later focus. Toggle Windows Animation Effects and a contrast theme while visible: decorative motion must follow the system, countdown time must remain live, and contrast must force an opaque system-colour surface with no glow. Inspect Detailed in every state and confirm it contains only fixed product/phase/instruction labels, a normalized shortcut, bounded elapsed time, and a bounded checkpoint count—never transcript, filename, endpoint/model, credential, prompt, provider error, or other user content.
- Exercise the main workspace, its global Settings and contextual options, About, and Transcription details with only the keyboard. Settings navigation and contextual tabs must support Arrow/Home/End with a visible focus indicator. Alt+F4 and the workspace Close button must offer Save, Discard, or Keep editing for a dirty configuration draft. Keep editing and failed saves must retain it; a successful close-time Save commits before hiding the workspace, and Discard restores applied values. Hiding clears credential drafts. Reopening must load the newest Go-owned settings, and immediate setup actions must not mutate configuration while another editor owns an unsaved draft. About must reuse and focus one native window and close from either Alt+F4, native chrome, or its footer. Transcription details opens in its own reusable window without trapping focus in the main window. With Narrator, confirm phase changes and action results are announced once without reading transcript content unexpectedly. Toggle Windows **Animation Effects** while the native overlay is visible: decorative WebView and overlay motion must stop and resume without recreating the overlay, while the overlay's automatic-stop countdown continues to advance. Repeat in a Windows contrast theme and confirm every focused control has a visible focus indicator.
- While microphone or stored-audio work is active, save changed endpoint, model, authentication, and credential values. Confirm the save succeeds for the next operation while the active run keeps its complete start-of-run profile and never sends a new credential to the old endpoint.
- Quit during microphone preparation, active recording, stored-file upload, transcription/cleanup, speech generation, playing/paused speech, and an open Save Audio dialog. Verify process exit and tray/hotkey removal. A dialog completing after Quit must not start an export, and late inference must not insert text, publish a result, or begin playback. With a deliberately slow export destination, Stop/Clear must remain responsive; an interrupted export may be partial. Record device teardown separately from service deadline errors: workflow wait budgets cover locks and cleanup, but do not prove that every OS call or Wails shutdown hook is interruptible. Use the opt-in native audio check documented in [testing](../../development/testing/#shutdown-and-cancellation-acceptance) before interactive review.
- Confirm the WebView is denied microphone, camera, geolocation, notification, and clipboard-read permissions. A renderer binding call must not be able to select an arbitrary audio path that was not granted through the native picker.

Connection tests use only configured health/model metadata routes; qualified
voice discovery may also read `/audio/voices`. Do not invoke model inventories.
Live workflow checks use only the deliberately selected endpoint and model.

For workspace appearance changes, check light and dark modes at compact and
desktop widths and 100%/150%/200% DPI. The selected input mode and
open contextual options must remain identifiable, and
history labels and result actions must stay readable without clipping. Verify
keyboard focus separately from selection, including Windows contrast themes.
Check that neutral black/white surfaces and the brand-blue accent extend to Settings, About,
the Connections page, native captions, and the passive overlay. Verify the local
heading font loads without a network request, and that idle, recording,
processing, copy recovery, and error states retain usable controls.

## Main workspace title bar

### Windows

- Confirm one integrated workspace title bar, with the Freehand mark and
  File/View/Help menus on the left and minimize, maximize/restore, and close
  controls at the far right beside the layout buttons. About, Transcription
  details, and standalone Process output must retain their native frames.
- Drag only intended title-bar surfaces, double-click to maximize/restore,
  drag down from maximized, and resize every edge and corner. Repeat on mixed-DPI
  monitors at 100%, 125%, 150%, and 200%, including negative monitor coordinates
  and taskbars on different edges. The content, cursor, and resize target must
  stay aligned; restored bounds must remain usable.
- On supported Windows 11/WebView2 installations, hover maximize to open native
  Snap Layouts, choose a layout, and restore. Exercise Win+Arrow, caption
  double-click, and taskbar restore; the maximize/restore glyph and accessible
  name must follow actual native state. If composition hosting falls back,
  verify ordinary caption buttons, dragging, and resizing still work; record
  Snap availability separately.
- Click and keyboard-activate File/View/Help, command search, and every layout
  control. They must perform their action without starting a window drag.
  Check hover, focus, pressed state, and contrast in light/dark and Mica modes.
- Use Close, Alt+F4, and the caption system menu with clean settings and with
  dirty global, contextual, and connection drafts. Save must commit before
  hiding; Discard restores applied values; Keep editing and failed saves retain
  the draft. Hiding clears credential drafts and displayed workspace output and
  disables its reader. Reopen from
  the tray and a second launch. Quit must still exit and stop owned services.

### macOS

Run these checks separately in a packaged app on the supported macOS
architectures; browser padding fixtures and a Windows build do not satisfy them.

- Confirm native traffic lights remain at the left of the integrated title bar,
  vertically aligned within the existing 44px header when using the compact
  native toolbar. Confirm a visible gap before the Freehand mark, and that the
  mark, title, and controls never overlap the buttons. Check normal and compact
  sizes, Retina scaling, and light/dark modes.
  The existing macOS application menu must remain available.
- Move the window from intended drag surfaces and operate nearby command,
  menu, and layout controls without accidental dragging. Check the system's
  configured double-click action: zoom, minimize, or no action.
- Use the green traffic light to enter and exit fullscreen, reveal the
  auto-hidden toolbar, move between Spaces, and restore the normal window.
  Verify traffic-light padding, controls, and content remain reachable through
  each transition. Test red close with the same Save/Discard/Keep editing and
  failed-save cases, then reopen through the Dock and menu-bar icon.
- Recheck the native frames and close behavior of About, Transcription details,
  and standalone Process output. Record these results independently of Windows.

## Managed local runtime

Use an isolated test user or explicit test data root for destructive cases.
Do not delete personal runtime installations or pull every catalog model.

Check host-aware recommendations, warm-up, startup progress, and the
process-output viewer on Windows and macOS using the cases below. Deterministic
tests and browser fixtures do not establish native inference or permission acceptance.
On macOS, test NeMo and llama.cpp on each claimed architecture; use a packaged
app for permissions. Verify Metal on Apple Silicon, CPU on Intel, the llama.cpp
13.3 minimum, and unavailable managed whisper.cpp with manual connections intact.
Exercise immediate embedded display, standalone viewer consent, selection-copy,
and teardown on both platforms.

- Start with no manual connections. Open Local runtime, install the official
  binary, and verify that browsing its speech catalog does not download weights
  or load a model. Download only the explicitly selected Nemotron 3.5 model.
- Select the built-in Connection referencing the instance for Voice, and enable
  realtime in Voice settings. Wait for Running, then use a recording
  shortcut from a disposable editor. Check provisional captions, authoritative
  finals, cancellation, Unicode, and changed-focus copy recovery. Repeat with
  realtime off and with a selected audio file using its independently selected
  Connection. Cleanup remains independently
  configured; local recognition must not be described as local cleanup.
- Confirm the listener is on `127.0.0.1` only and has no LAN-facing socket. Verify
  the actual selected backend on supported GPU and CPU-only hardware; successful
  CPU execution is not GPU acceptance.
- Cancel and retry an installation/download, interrupt network access, and test
  insufficient disk space in the isolated root. Progress must remain responsive
  and must not invent percentages for indeterminate work. Reopening runtime management
  must show the backend's current operation, not start a duplicate job.
- Stop/start repeatedly. Quit during model pull, startup, and active streaming;
  confirm the server and model-manager descendants exit. Force-close the test
  app and verify Job Object cleanup on Windows or lifetime-pipe/process-group
  cleanup on macOS. A separate manually started NeMo server
  must remain untouched.
- Preserve a manual connection with a disposable key before selecting a managed
  Connection. Confirm local requests carry neither that key nor its custom headers.
  On local failure, confirm no request reaches the manual server. Select the
  manual Connection explicitly and check it still works. Stop a files-only
  runtime and confirm independently configured manual Voice remains usable.
- For llama.cpp and whisper.cpp, stop the runtime and switch CPU to NVIDIA CUDA
  without removing models or changing Connections. Cancel an isolated replacement
  and confirm the old backend still starts; then complete the switch and restart
  Freehand to confirm its selection persists. Exercise only an explicitly chosen
  model on the GPU, including concurrent NeMo and S1-mini if that is the selected
  workflow. Record actual GPU use and driver/GPU compatibility separately from
  installation, CLI help, and browser results. Switch back to CPU and verify that
  model data and task selections remain intact.
- On supported NVIDIA and CPU-only Windows hosts, open a new llama.cpp or
  whisper.cpp installation and inspect the recommendation before accepting it.
  Unknown/unsupported device-0 or driver metadata must recommend CPU; compatible
  device 0 must recommend the pinned CUDA 12.4 recipe. Confirm explicit CPU
  selection works and that simply opening the choice downloads/starts nothing.
  Existing installations must remain unchanged on reopen/restart or changes in
  available GPU memory; do not infer compatibility from free VRAM.
- Start only the selected downloaded model on CPU and GPU. Observe verification,
  launch, readiness, and loading/warm-up phases with phase-specific elapsed time,
  not a fabricated percentage. On GPU, verify llama.cpp/NeMo built-in warm-up and
  CUDA whisper.cpp's one-second synthetic-silence startup request before Running.
  Confirm there is no microphone capture, history/cleanup result, catalog-model
  invocation, or remote request. Repeat with saved start-at-launch intent.
  Measure startup and first/subsequent selected-model requests separately; do not
  infer a latency improvement from a health response or a CUDA binary label.
- Cancel during verification and warm-up; exercise a slow or failed startup.
  Launch/readiness/warm-up share 120 seconds, followed by up to four seconds of
  owned-process drain; prelaunch hashing is cancellable. Confirm no Running
  endpoint is published on failure and no replacement is admitted while an owned
  child remains alive. Quit during warm-up with multiple providers active and
  check process descendants against the separate application shutdown bound.
- Open **View output** from Local runtime or workflow runtime controls while startup is
  active. Repeated opens use the shared **Runtime output** bottom panel and display
  available output immediately. Opening the panel without an explicit target selects
  a running runtime, or the first installed runtime if none is running. Choose a
  runtime tab by pointer and keyboard; navigating between workflows, Connections,
  Local runtime, and History must retain the visible viewer and its target.
  Hide the panel, change its panel tab, enter global Settings, hide the workspace
  or document, and change the runtime target in turn. Each must disable the old
  reader, clear displayed text, and reject late reads. The hidden viewer must not
  enable access. Reopening immediately reads the retained target's available tail;
  it must not restore stale renderer data. Removing the selected runtime must not
  silently choose another. In the standalone **Process output** window, verify
  **Show output** is still required on each opening or runtime switch and that no
  raw output is retrieved or displayed beforehand.
  Use non-sensitive data only, including synthetic HTML/terminal-control text:
  only SGR colors/styles, carriage return, backspace, and CSI K line erasure may
  render as terminal controls. OSC clipboard,
  links, titles, terminal queries, and mode changes must have no side effects.
  Test colors and carriage-return progress split across output chunks.
- With enough synthetic output to exceed the bounded tail, confirm older text
  is discarded and renderer memory does not grow without bound. Follow and
  search must work; collection continues when Follow is off. **Clear** drops
  the captured tail and terminal state without changing the process.
  Closing revokes retrieval but preserves private capture until Clear, the next
  start attempt, removal, or Quit. Check retained output after process exit,
  generation isolation on restart, and no export, file, event, application
  log, or crash-report output path. Explicit Copy selection must copy only the
  selected displayed text, never automatically or before standalone consent.
  After an ordinary subsequent llama.cpp start, verify normal info/warning/error
  output reaches the visible embedded viewer directly and the standalone viewer
  after Show output on
  CPU and CUDA, with `--log-verbosity 3 --log-colors on`, not `--log-disable` or
  trace/debug logging. Check the runtime working directory for no new log or
  prompt files; inherited logging/config overrides must remain excluded.
  Use only explicitly selected models and non-sensitive requests. Normal logs
  may still contain sensitive text; an empty tail is not proof of startup failure.
  The opt-in pinned CPU ZIP regression checks a real parser warning through the
  owned launcher plus no new temporary-installation files, without loading a
  model. It does not replace native startup/request, CUDA, or viewer acceptance.
- Check viewer focus, keyboard navigation, light/dark appearance, mixed DPI,
  scrolling, and shutdown with real Wails windows. Opening/closing/clearing or
  pausing the viewer must never start, stop, restart, or orphan the runtime.
- Confirm destructive actions explain their scope and protect active work.
  Remove an inactive downloaded model, then remove the runtime and its owned
  model data. Other installations, connections, credentials, and source files
  must remain. Restart after removal and confirm nothing launches or downloads.

## Windows tray panel

- Left-click repeatedly: reuse one 360 × 500 panel with no taskbar entry. Check
  the 8-DIP spacing, taskbar overflow, supported taskbar edges, and mixed-DPI
  monitors. Escape, focus loss, Alt+F4, and Main/Settings navigation hide the
  panel without cancelling work. Right-click must dismiss it before opening the
  native menu, including authoritative Quit.
- Use the connection and model pickers by mouse and keyboard. Verify committed
  changes, metadata-only discovery, failed-save feedback, and disabled controls
  while work or a conflicting Settings draft is active.
- Start and stop from the panel with Notepad and a browser field as destinations,
  then repeat with other Freehand windows visible and with the tray overflow
  open. Observe the actual foreground window and focused control after native
  dismissal and before recording begins. No automatic activation or cached
  target may substitute for the normal recording-start capture. Change focus
  during processing and verify explicit-copy recovery rather than insertion
  into the new target. Check latest-result Copy and failed clipboard feedback.
- Browser fixtures verify command ordering and presentation only. Run these
  native focus checks separately; neither a successful build nor a mock hide
  proves which application Windows focuses after the panel closes.

## SQLite settings acceptance

Use a separate test user or isolated data directories, not personal settings.
Confirm first-run defaults and an empty connection catalog while alpha files and
credentials remain untouched. Check save/reopen persistence, new-key replacement
and clear, and restart reconciliation of startup and shortcuts. Test a denied write
and a deliberately newer/corrupt fixture: the workspace must expose recovery,
new jobs must stop, and existing captured profiles must remain coherent. Retry
a repaired current database, explicitly reset and reconfigure connections/keys,
and restore a new-lineage backup with the app fully closed. Inspect only synthetic credential accounts and confirm no secret reaches
the database, backups, renderer snapshots, or logs. Native temporary-database
tests and a successful executable build do not replace these interactive checks.

<span id="qualified-live-dictation-and-connection-windows"></span>

## Qualified live dictation and connection editing

- Configure Voice and Audio file on different connections. Voice uses the same
  connection/model/profile when switching between completed and qualified realtime
  mode; Audio file retains its independent selection.
- Speak with captions on: latest words appear in a single row at normal DPI and
  125/150/200% scaling. Long text stays within the fixed strip; Unicode remains
  readable. The overlay never takes focus or intercepts input.
- Stop: authoritative final text replaces the preview before cleanup/insertion.
  Change focus before stopping: insertion must fail closed and offer final copy.
- Cancel, disconnect the server, remove the microphone, and quit during capture
  or finalization: no partial insertion/history, no replay, bounded shutdown.
- Turn captions off and turn live mode off separately. Ordinary overlay layouts
  and the saved completed-transcription VAD/checkpoint behavior return.
- Open Add/Edit from Connections and task controls: each reveals the Connections
  rail page in the main workspace. Repeated requests preserve an unfinished draft.
  Rail, palette, Done, runtime links, native close, and Escape protect unsaved
  edits across global Settings, Connections, and contextual options. Hide/save clears
  credential input; hide also clears displayed runtime output, disables its reader,
  and stops shortcut capture and overlay preview. Connection creation's Save and return resumes the originating
  task only after success. Global and contextual Save apply in place. Conflicting
  stale edits are rejected.

## Flat workspace and settings surfaces

- At 560 × 560 and normal desktop sizes, check Voice, Audio file, and Text to speech in light and dark appearance. Recording, playback, toolbars, and footer actions must remain visible without horizontal overflow.
- Resize the editor/bottom-panel divider by pointer and keyboard, hide and restore it, and navigate Recent/Runtime output/Diagnostics with Arrow/Home/End keys. Below 560px height, verify the bottom panel hides and restores its selected tab and visibility preference when room returns. Global Settings must use the full editor height and restore those same panel preferences on return. In narrow windows, use the title-bar sidebar toggles and page Settings cogs; confirm overlay sidebars close with Escape or their backdrop and all controls remain reachable.
- In global Settings, Connections, and workflow options, verify groups align with headings, disclosures reveal all their controls, and long pages still scroll to the final option. Selected rows, input boundaries, and keyboard focus must remain distinguishable in both themes and Windows contrast mode.
