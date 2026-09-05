package filetranscription

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	MaxAudioFileBytes      int64 = 2 << 30
	maxFileTranscriptBytes       = 8 << 20
	fileProgressInterval         = 100 * time.Millisecond
	DeltaEvent                   = "file-transcription:delta"
)

// FileTranscriptionPhase identifies the current stored-audio operation stage.
type FileTranscriptionPhase string

const (
	FileTranscriptionEmpty      FileTranscriptionPhase = "empty"
	FileTranscriptionSelected   FileTranscriptionPhase = "selected"
	FileTranscriptionUploading  FileTranscriptionPhase = "uploading"
	FileTranscriptionProcessing FileTranscriptionPhase = "processing"
	FileTranscriptionStreaming  FileTranscriptionPhase = "streaming"
	FileTranscriptionCompleted  FileTranscriptionPhase = "completed"
	FileTranscriptionFailed     FileTranscriptionPhase = "failed"
	FileTranscriptionCancelling FileTranscriptionPhase = "cancelling"
)

// FileTranscriptionStatus is the bounded renderer snapshot for a Go-owned
// native file selection. It never contains the full path or audio bytes.
type FileTranscriptionStatus struct {
	Generation           uint64                 `json:"generation"`
	Phase                FileTranscriptionPhase `json:"phase"`
	FileName             string                 `json:"fileName,omitempty"`
	FileSize             int64                  `json:"fileSize,omitempty"`
	BytesUploaded        int64                  `json:"bytesUploaded,omitempty"`
	Streaming            bool                   `json:"streaming"`
	Buffered             bool                   `json:"buffered"`
	StreamingUnavailable bool                   `json:"streamingUnavailable"`
	StreamingNotice      string                 `json:"streamingNotice,omitempty"`
	Transcript           string                 `json:"transcript,omitempty"`
	TranscriptRevision   uint64                 `json:"transcriptRevision"`
	Message              string                 `json:"message,omitempty"`
	CanStart             bool                   `json:"canStart"`
	CanCancel            bool                   `json:"canCancel"`
	CanCopy              bool                   `json:"canCopy"`
}

// FileTranscriptionDelta is the incremental renderer contract for streamed
// transcript text. The complete status remains available as a binding snapshot
// and is emitted once more when the operation reaches a terminal phase.
type FileTranscriptionDelta struct {
	Generation uint64 `json:"generation"`
	Revision   uint64 `json:"revision"`
	Text       string `json:"text"`
}

func init() {
	application.RegisterEvent[FileTranscriptionDelta](DeltaEvent)
}

type fileStreamingCapabilityKey struct {
	Profile  compatibility.ID
	Endpoint string
	Model    string
}

const fileStreamingUnavailableNotice = "Streaming is unavailable for this endpoint and model. New requests will use completed transcripts."

var supportedAudioExtensions = map[string]struct{}{
	".flac": {}, ".mp3": {}, ".mp4": {}, ".mpeg": {}, ".mpga": {},
	".m4a": {}, ".ogg": {}, ".wav": {}, ".webm": {},
}

// audioFileSelection is an app-owned capability created only after the native
// picker returns. It never crosses the Wails bridge. The retained identity and
// metadata let StartFileTranscription reject a path that disappeared, changed,
// or was replaced between selection and upload.
type audioFileSelection struct {
	path    string
	name    string
	size    int64
	modTime time.Time
	info    os.FileInfo
}

type transcriptProcessor interface {
	ProcessWithCredential(context.Context, config.PostProcessingSettings, string, string) (postprocess.Result, error)
}

type Service struct {
	settings                 settings.Source
	profiles                 settings.ProfileSource
	client                   *inference.Client
	processor                transcriptProcessor
	history                  *history.Store
	input                    insertion.Platform
	pickAudioFile            func() (string, error)
	fileChanged              func(FileTranscriptionStatus)
	fileDelta                func(FileTranscriptionDelta)
	activity                 *activity.Coordinator
	logger                   *slog.Logger
	fileMu                   sync.Mutex
	fileStatus               FileTranscriptionStatus
	fileSelection            *audioFileSelection
	fileChoosing             bool
	fileGeneration           uint64
	fileCancel               context.CancelFunc
	fileDone                 chan struct{}
	fileLastPublish          time.Time
	fileTranscript           strings.Builder
	fileTranscriptRevision   uint64
	fileTranscriptLimitHit   bool
	fileStreamingUnsupported map[fileStreamingCapabilityKey]string
	lifecycleMu              sync.RWMutex
	rootContext              context.Context
	rootCancel               context.CancelFunc
	workers                  sync.WaitGroup
	closed                   atomic.Bool
}

func NewService(source settings.Source, profiles settings.ProfileSource, client *inference.Client, processor transcriptProcessor, transcripts *history.Store, input insertion.Platform, pickAudioFile func() (string, error), changed func(FileTranscriptionStatus), delta func(FileTranscriptionDelta), admission *activity.Coordinator, logger *slog.Logger) *Service {
	if admission == nil {
		admission = activity.New(activity.Sources{})
	}
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &Service{settings: source, profiles: profiles, client: client, processor: processor, history: transcripts, input: input, pickAudioFile: pickAudioFile, fileChanged: changed, fileDelta: delta, activity: admission, logger: logger.With("component", "file-transcription")}
}

// snapshotFileStatusLocked materializes the accumulated transcript only at a
// real status boundary or an explicit snapshot request. Streaming deltas never
// copy the growing transcript into FileTranscriptionStatus.
func (s *Service) snapshotFileStatusLocked() FileTranscriptionStatus {
	status := s.fileStatus
	status.Transcript = s.fileTranscript.String()
	status.TranscriptRevision = s.fileTranscriptRevision
	return status
}

func (s *Service) resetFileTranscriptLocked(text string) {
	s.fileTranscript.Reset()
	_, _ = s.fileTranscript.WriteString(text)
	s.fileTranscriptRevision = 0
	s.fileTranscriptLimitHit = false
	s.fileStatus.Transcript = text
	s.fileStatus.TranscriptRevision = 0
}

func (s *Service) replaceFileTranscriptLocked(text string) {
	s.fileTranscript.Reset()
	_, _ = s.fileTranscript.WriteString(text)
	s.fileTranscriptRevision++
	s.fileStatus.Transcript = text
	s.fileStatus.TranscriptRevision = s.fileTranscriptRevision
}

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return diagnostics.DiscardLogger()
}

func (s *Service) current() config.Settings {
	if s.settings == nil {
		return config.Default()
	}
	return s.settings.Current()
}

func (s *Service) captureRequestProfile() (settings.RequestProfile, error) {
	if s.profiles == nil {
		return settings.RequestProfile{}, errors.New("transcription profile is unavailable")
	}
	return s.profiles.Capture()
}

func (s *Service) operationContext() (context.Context, context.CancelFunc) {
	s.lifecycleMu.RLock()
	root := s.rootContext
	s.lifecycleMu.RUnlock()
	if root == nil {
		root = context.Background()
	}
	ctx, cancel := context.WithCancel(root)
	if s.closed.Load() {
		cancel()
	}
	return ctx, cancel
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closed.Store(false)
	s.lifecycleMu.Lock()
	s.rootContext, s.rootCancel = context.WithCancel(ctx)
	s.lifecycleMu.Unlock()
	return nil
}

func (s *Service) ServiceShutdown() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	s.activity.Close()
	s.lifecycleMu.Lock()
	cancelRoot := s.rootCancel
	s.rootCancel = nil
	s.lifecycleMu.Unlock()
	if cancelRoot != nil {
		cancelRoot()
	}
	s.fileMu.Lock()
	cancelFile := s.fileCancel
	s.fileMu.Unlock()
	if cancelFile != nil {
		cancelFile()
	}
	done := make(chan struct{})
	go func() { s.workers.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("file transcription shutdown exceeded the service deadline")
	}
}

func ApplySettings(service *Service, cfg config.Settings) {
	if service != nil {
		service.refreshFileStreamingCapability(cfg)
	}
}

func Active(service *Service) bool {
	return service != nil && service.fileTranscriptionActive()
}

func fileStreamingKey(cfg config.Settings) fileStreamingCapabilityKey {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return fileStreamingCapabilityKey{Profile: compatibility.Effective(cfg.CompatibilityProfile), Endpoint: cfg.BaseURL, Model: cfg.Model}
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return fileStreamingCapabilityKey{Profile: compatibility.Effective(cfg.CompatibilityProfile), Endpoint: parsed.String(), Model: cfg.Model}
}

func connectionServer(requestedURL string) string {
	parsed, err := url.Parse(requestedURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func (s *Service) applyFileStreamingCapabilityLocked(status *FileTranscriptionStatus, cfg config.Settings) {
	reason := ""
	if s.fileStreamingUnsupported != nil {
		reason = s.fileStreamingUnsupported[fileStreamingKey(cfg)]
	}
	status.StreamingUnavailable = reason != ""
	if reason != "" {
		status.StreamingNotice = fileStreamingUnavailableNotice
	} else {
		status.StreamingNotice = ""
	}
}

func (s *Service) refreshFileStreamingCapability(cfg config.Settings) {
	s.fileMu.Lock()
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return
	}
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
}

func (s *Service) rememberFileStreamingUnsupported(generation uint64, cfg config.Settings, reason, fallbackOutcome string, completedMode bool) {
	s.fileMu.Lock()
	if s.fileStreamingUnsupported == nil {
		s.fileStreamingUnsupported = make(map[fileStreamingCapabilityKey]string)
	}
	s.fileStreamingUnsupported[fileStreamingKey(cfg)] = reason
	if s.fileStatus.Generation == generation {
		s.fileStatus.StreamingUnavailable = true
		s.fileStatus.StreamingNotice = fileStreamingUnavailableNotice
		if completedMode {
			s.fileStatus.Streaming = false
		}
	}
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Warn("audio file streaming unsupported",
		"generation", generation,
		"server", connectionServer(cfg.BaseURL),
		"reason", reason,
		"fallback_outcome", fallbackOutcome,
	)
}

// TryFileStreamingAgain clears the remembered unsupported-stream capability
// for the active endpoint and model so the user may explicitly retry it.
func (s *Service) TryFileStreamingAgain() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	cfg := s.current()
	s.fileMu.Lock()
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return errors.New("wait for the active file transcription to finish")
	}
	delete(s.fileStreamingUnsupported, fileStreamingKey(cfg))
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Info("audio file streaming capability reset", "server", connectionServer(cfg.BaseURL))
	return nil
}

// CurrentFileTranscription returns the latest stored-audio operation snapshot.
func (s *Service) CurrentFileTranscription() FileTranscriptionStatus {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	status := s.snapshotFileStatusLocked()
	if status.Phase == "" {
		status.Phase = FileTranscriptionEmpty
	}
	return status
}

// PlaybackTranscript is an ordinary Go collaboration boundary. It keeps the
// retained file transcript out of renderer arguments while allowing the TTS
// feature to read the completed result when in-memory history is disabled.
func PlaybackTranscript(service *Service) (string, error) {
	if service == nil {
		return "", errors.New("audio file transcription is unavailable")
	}
	service.fileMu.Lock()
	defer service.fileMu.Unlock()
	if service.fileStatus.Phase != FileTranscriptionCompleted || service.fileTranscript.Len() == 0 {
		return "", errors.New("no completed audio file transcript is available")
	}
	return service.fileTranscript.String(), nil
}

// ChooseAudioFile opens the native file picker in Go and retains the selected
// path as a backend-only capability. The renderer receives only bounded file
// metadata and cannot grant an arbitrary filesystem path to itself.
func (s *Service) ChooseAudioFile() (result FileTranscriptionStatus, err error) {
	started := time.Now()
	outcome := "selected"
	s.log().Info("audio file selection started")
	defer func() {
		if err != nil {
			outcome = "failed"
		}
		attrs := []any{"duration_ms", time.Since(started).Milliseconds(), "outcome", outcome}
		switch {
		case err != nil:
			attrs = append(attrs, "error_kind", diagnostics.ErrorKind(err))
			s.log().Warn("audio file selection failed", attrs...)
		case outcome == "cancelled":
			s.log().Info("audio file selection cancelled", attrs...)
		default:
			s.log().Info("audio file selection completed", attrs...)
		}
	}()
	if s.closed.Load() {
		return FileTranscriptionStatus{}, errors.New("application is shutting down")
	}
	s.fileMu.Lock()
	if s.fileChoosing {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("the audio file picker is already open")
	}
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("cancel the active file transcription before choosing another file")
	}
	s.fileChoosing = true
	picker := s.pickAudioFile
	s.fileMu.Unlock()
	defer func() {
		s.fileMu.Lock()
		s.fileChoosing = false
		s.fileMu.Unlock()
	}()

	if picker == nil {
		return FileTranscriptionStatus{}, errors.New("the native audio file picker is unavailable")
	}
	path, err := picker()
	if err != nil {
		return FileTranscriptionStatus{}, errors.New("the native audio file picker could not be opened")
	}
	if strings.TrimSpace(path) == "" {
		outcome = "cancelled"
		return s.CurrentFileTranscription(), nil
	}
	return s.selectAudioFile(path)
}

// selectAudioFile converts a native-picker result into an app-owned selection.
// It is deliberately unexported so Wails cannot bind a renderer-controlled
// path argument. Tests use it to exercise the post-picker validation boundary.
func (s *Service) selectAudioFile(path string) (FileTranscriptionStatus, error) {
	selection, err := inspectAudioFile(path)
	if err != nil {
		return FileTranscriptionStatus{}, err
	}

	cfg := s.current()
	s.fileMu.Lock()
	if s.closed.Load() {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("application is shutting down")
	}
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return FileTranscriptionStatus{}, errors.New("cancel the active file transcription before choosing another file")
	}
	s.fileGeneration++
	s.fileSelection = selection
	s.fileStatus = FileTranscriptionStatus{
		Generation: s.fileGeneration,
		Phase:      FileTranscriptionSelected,
		FileName:   selection.name,
		FileSize:   selection.size,
		CanStart:   true,
	}
	s.resetFileTranscriptLocked("")
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	return status, nil
}

func inspectAudioFile(path string) (*audioFileSelection, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || !filepath.IsAbs(path) {
		return nil, errors.New("the native file picker returned an invalid selection")
	}
	if _, ok := supportedAudioExtensions[strings.ToLower(filepath.Ext(path))]; !ok {
		return nil, errors.New("supported audio types are FLAC, MP3, MP4, MPEG, MPGA, M4A, OGG, WAV, and WebM")
	}
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return nil, errors.New("the selected audio file is unavailable")
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("choose the audio file itself rather than a symbolic link")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("the selected audio file could not be opened")
	}
	info, err := file.Stat()
	_ = file.Close()
	if err != nil {
		return nil, errors.New("the selected audio file is unavailable")
	}
	if !os.SameFile(pathInfo, info) {
		return nil, errors.New("the selected audio file changed while it was being inspected")
	}
	if err := validateAudioFileInfo(info); err != nil {
		return nil, err
	}
	return &audioFileSelection{path: path, name: info.Name(), size: info.Size(), modTime: info.ModTime(), info: info}, nil
}

func validateAudioFileInfo(info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return errors.New("the selected path is not a regular file")
	}
	if info.Size() <= 0 {
		return errors.New("the selected audio file is empty")
	}
	if info.Size() > MaxAudioFileBytes {
		return fmt.Errorf("audio files must be no larger than %d GiB", MaxAudioFileBytes>>30)
	}
	return nil
}

func (selection *audioFileSelection) open() (*os.File, os.FileInfo, error) {
	pathInfo, err := os.Lstat(selection.path)
	if err != nil {
		return nil, nil, errors.New("the selected audio file is no longer available; choose it again")
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return nil, nil, errors.New("the selected audio file was replaced by a symbolic link; choose it again")
	}
	file, err := os.Open(selection.path)
	if err != nil {
		return nil, nil, errors.New("the selected audio file could not be opened")
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, errors.New("the selected audio file is unavailable")
	}
	if err := validateAudioFileInfo(info); err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if !os.SameFile(pathInfo, info) || !os.SameFile(selection.info, info) || info.Size() != selection.size || !info.ModTime().Equal(selection.modTime) {
		_ = file.Close()
		return nil, nil, errors.New("the selected audio file changed; choose it again")
	}
	return file, info, nil
}

// ClearAudioFile releases the current backend-owned audio-file selection.
func (s *Service) ClearAudioFile() error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	cfg := s.current()
	s.fileMu.Lock()
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return errors.New("cancel the active file transcription before clearing it")
	}
	if s.fileChoosing {
		s.fileMu.Unlock()
		return errors.New("finish choosing an audio file before clearing it")
	}
	s.fileGeneration++
	s.fileSelection = nil
	s.fileStatus = FileTranscriptionStatus{Generation: s.fileGeneration, Phase: FileTranscriptionEmpty}
	s.resetFileTranscriptLocked("")
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	return nil
}

// StartFileTranscription uploads the selected file using completed or streamed
// response mode while keeping the file path and bytes out of the renderer.
func (s *Service) StartFileTranscription(stream bool) error {
	if s.closed.Load() {
		return errors.New("application is shutting down")
	}
	release, err := s.activity.BeginFileTranscription()
	if err != nil {
		return err
	}
	defer release()
	s.fileMu.Lock()
	if s.closed.Load() {
		s.fileMu.Unlock()
		return errors.New("application is shutting down")
	}
	if filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		return errors.New("an audio file is already being transcribed")
	}
	if s.fileChoosing {
		s.fileMu.Unlock()
		return errors.New("finish choosing an audio file before starting transcription")
	}
	selection := s.fileSelection
	s.fileMu.Unlock()
	if selection == nil {
		return errors.New("choose an audio file first")
	}

	profile, err := s.captureRequestProfile()
	if err != nil {
		return err
	}
	defer func() {
		profile.STTCredential = ""
		profile.PostProcessingCredential = ""
	}()
	cfg := profile.Settings
	file, info, err := selection.open()
	if err != nil {
		return err
	}
	key := profile.STTCredential
	processingKey := profile.PostProcessingCredential
	ctx, cancel := s.operationContext()
	done := make(chan struct{})

	s.fileMu.Lock()
	if s.closed.Load() || s.fileSelection != selection || filePhaseActive(s.fileStatus.Phase) {
		s.fileMu.Unlock()
		cancel()
		_ = file.Close()
		key = ""
		processingKey = ""
		return errors.New("the selected audio file changed before transcription started")
	}
	s.fileGeneration++
	generation := s.fileGeneration
	streamingUnavailable := false
	if s.fileStreamingUnsupported != nil {
		_, streamingUnavailable = s.fileStreamingUnsupported[fileStreamingKey(cfg)]
	}
	effectiveStream := stream && !streamingUnavailable
	s.fileCancel = cancel
	s.fileDone = done
	s.fileLastPublish = time.Time{}
	s.workers.Add(1)
	s.fileStatus = FileTranscriptionStatus{
		Generation: generation,
		Phase:      FileTranscriptionUploading,
		FileName:   info.Name(),
		FileSize:   info.Size(),
		Streaming:  effectiveStream,
		Message:    "Uploading audio",
		CanCancel:  true,
	}
	s.resetFileTranscriptLocked("")
	s.applyFileStreamingCapabilityLocked(&s.fileStatus, cfg)
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Info("audio file transcription started", "generation", generation, "server", connectionServer(cfg.BaseURL), "bytes", info.Size(), "streaming_requested", stream, "streaming_effective", effectiveStream, "timeout_seconds", cfg.FileTranscriptionTimeoutSeconds, "post_processing_timeout_seconds", cfg.PostProcessing.TimeoutSeconds)

	go func() {
		defer s.workers.Done()
		s.runFileTranscription(ctx, generation, file, info.Size(), key, processingKey, cfg, effectiveStream, done)
	}()
	return nil
}

func (s *Service) runFileTranscription(ctx context.Context, generation uint64, file *os.File, size int64, key, processingKey string, cfg config.Settings, stream bool, done chan struct{}) {
	started := time.Now()
	startedAt := started.UTC()
	responseMode := history.HistoryResponseCompleted
	if stream {
		responseMode = history.HistoryResponseStreamed
	}
	details := history.HistoryRunDetails{
		Source:                history.HistorySourceAudioFile,
		StartedAt:             startedAt,
		Server:                history.SanitizedServer(cfg.BaseURL),
		Route:                 "/audio/transcriptions",
		AuthenticationMode:    string(cfg.AuthenticationMode),
		Model:                 cfg.Model,
		Language:              cfg.Language,
		ResponseMode:          responseMode,
		InsertionMode:         insertion.ManualCopy,
		FileName:              filepath.Base(file.Name()),
		FileSize:              size,
		RequestTimeoutSeconds: cfg.FileTranscriptionTimeoutSeconds,
		Processing:            history.NewProcessingDetails(cfg.PostProcessing, ""),
	}
	var uploadMilliseconds atomic.Int64
	defer func() {
		s.fileMu.Lock()
		if s.fileDone == done {
			s.fileDone = nil
		}
		s.fileMu.Unlock()
		close(done)
	}()
	defer file.Close()
	defer func() {
		key = ""
		processingKey = ""
	}()

	transcribe := func(streaming bool) (inference.TranscriptionResult, string, error) {
		unsupportedReason := ""
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(cfg.FileTranscriptionTimeoutSeconds)*time.Second)
		defer requestCancel()
		result, err := s.client.WithCompatibility(cfg.CompatibilityProfile).WithTranscriptionOptions(cfg.TranscriptionOptions).TranscribeFile(requestCtx, cfg.BaseURL, cfg.Model, cfg.Language, key, cfg.Headers, filepath.Base(file.Name()), size, io.LimitReader(file, size), streaming, inference.FileTranscriptionCallbacks{
			UploadProgress: func(sent, _ int64) { s.updateFileUpload(generation, sent, false) },
			UploadComplete: func() {
				uploadMilliseconds.Store(time.Since(started).Milliseconds())
				s.fileUploadComplete(generation, streaming)
			},
			Delta:             func(delta string) { s.appendFileDelta(generation, delta) },
			StreamBuffered:    func() { s.fileStreamBuffered(generation) },
			StreamUnsupported: func(reason string) { unsupportedReason = reason },
		})
		return result, unsupportedReason, err
	}

	transcription, unsupportedReason, err := transcribe(stream)
	text := transcription.Text
	s.fileMu.Lock()
	partialText := ""
	transcriptLimitHit := s.fileStatus.Generation == generation && s.fileTranscriptLimitHit
	if s.fileStatus.Generation == generation && (transcriptLimitHit || diagnostics.ErrorKind(err) == "response_too_large") {
		partialText = s.fileTranscript.String()
	}
	s.fileMu.Unlock()
	if transcriptLimitHit {
		text = partialText
		err = &inference.Error{Kind: "response_too_large", Message: "transcript exceeded Freehand's 8 MiB safety limit; partial text was preserved"}
		s.log().Warn("audio file transcript safety limit reached", "generation", generation, "limit_bytes", maxFileTranscriptBytes, "partial_bytes", len(partialText))
	} else if partialText != "" && diagnostics.ErrorKind(err) == "response_too_large" {
		// The SSE reader bounds the complete wire response as well as the text
		// accumulator. Preserve already accepted deltas when framing overhead
		// reaches that limit first.
		text = partialText
	}
	if stream && unsupportedReason != "" && err == nil {
		details.ResponseMode = history.HistoryResponseCompleted
		details.StreamFallbackReason = unsupportedReason
		s.rememberFileStreamingUnsupported(generation, cfg, unsupportedReason, "completed_response_used", true)
	}
	var unsupported *inference.FileStreamUnsupportedError
	if stream && errors.As(err, &unsupported) {
		details.StreamFallbackReason = unsupported.Reason
		if unsupported.PartialText != "" {
			text = unsupported.PartialText
			s.rememberFileStreamingUnsupported(generation, cfg, unsupported.Reason, "partial_result_preserved", false)
		} else {
			// An unreadable stream may already represent completed inference.
			// Remember the capability, but require an explicit new user action
			// before submitting the file again, even after parameter rejection.
			s.rememberFileStreamingUnsupported(generation, cfg, unsupported.Reason, "explicit_retry_required", true)
		}
	}
	details.Transcription = history.NewResponseDetails(transcription.Metadata)
	totalTranscriptionMilliseconds := time.Since(started).Milliseconds()
	details.UploadMilliseconds = uploadMilliseconds.Load()
	details.TranscriptionMilliseconds = max(0, totalTranscriptionMilliseconds-details.UploadMilliseconds)
	s.fileMu.Lock()
	details.Buffered = s.fileStatus.Generation == generation && s.fileStatus.Buffered
	if text != "" && err != nil && s.fileStatus.Generation == generation {
		s.replaceFileTranscriptLocked(text)
	}
	s.fileMu.Unlock()
	details.Processing = history.NewProcessingDetails(cfg.PostProcessing, text)

	processingFallback := false
	processingFallbackKind := ""
	historyID := uint64(0)
	if text != "" && s.history != nil {
		historyID = s.history.Begin(text, history.HistoryTranscribed, details.Processing.Requested, time.Now().UTC(), details)
	}
	if err == nil && text != "" && cfg.PostProcessing.Enabled {
		s.fileMu.Lock()
		if s.fileStatus.Generation == generation {
			s.fileStatus.Phase = FileTranscriptionProcessing
			s.replaceFileTranscriptLocked(text)
			s.fileStatus.Message = "Post-processing transcript"
			status := s.snapshotFileStatusLocked()
			changed := s.fileChanged
			s.fileMu.Unlock()
			s.publishFileStatus(changed, status)
		} else {
			s.fileMu.Unlock()
		}

		processingStarted := time.Now()
		var processingResult postprocess.Result
		var processingErr error
		if s.processor == nil {
			processingErr = errors.New("post-processing is unavailable")
		} else {
			processingResult, processingErr = s.processor.ProcessWithCredential(ctx, cfg.PostProcessing, text, processingKey)
		}
		processing := postprocess.Resolve(ctx, text, processingResult, processingErr, processingStarted)
		details = s.history.FinalizeProcessing(historyID, text, processing, details)
		// Cancellation may arrive while history finalizes a failed attempt.
		// Keep the final fallback-admission check with the live workflow owner.
		if ctxErr := ctx.Err(); processing.Err != nil && errors.Is(ctxErr, context.Canceled) {
			err = ctxErr
		}
		processingFallback = processing.Fallback() && err == nil
		if processingFallback {
			processingFallbackKind = diagnostics.ErrorKind(processing.Err)
			s.log().Warn("audio file post-processing fell back to raw transcript", "generation", generation, "error_kind", processingFallbackKind)
		}
		text = processing.Text
	}

	s.fileMu.Lock()
	if s.fileStatus.Generation != generation {
		s.fileMu.Unlock()
		return
	}
	s.fileCancel = nil
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.fileStatus.Phase = FileTranscriptionSelected
			s.fileStatus.Message = "Transcription cancelled"
			s.replaceFileTranscriptLocked("")
			s.fileStatus.CanStart = true
			s.fileStatus.CanCancel = false
			s.fileStatus.CanCopy = false
		} else {
			s.fileStatus.Phase = FileTranscriptionFailed
			if text != "" {
				if diagnostics.ErrorKind(err) == "response_too_large" {
					s.fileStatus.Message = "Transcript reached the 8 MiB safety limit; partial text preserved"
				} else {
					s.fileStatus.Message = "Stream ended after a partial transcript"
				}
			} else {
				s.fileStatus.Message = err.Error()
			}
			s.fileStatus.CanStart = true
			s.fileStatus.CanCancel = false
			s.fileStatus.CanCopy = s.fileTranscript.Len() != 0
		}
	} else {
		s.fileStatus.Phase = FileTranscriptionCompleted
		s.fileStatus.BytesUploaded = s.fileStatus.FileSize
		s.replaceFileTranscriptLocked(text)
		if processingFallback {
			s.fileStatus.Message = "Transcription complete; post-processing failed, using raw text"
			if processingFallbackKind == "timeout" {
				s.fileStatus.Message = "Transcription complete; post-processing timed out, using raw text"
			} else if processingFallbackKind == "incomplete_response" {
				s.fileStatus.Message = "Transcription complete; post-processing reached the output limit, using raw text"
			}
		} else if text == "" {
			s.fileStatus.Message = "No speech detected"
		} else {
			s.fileStatus.Message = "Transcription complete"
		}
		s.fileStatus.CanStart = true
		s.fileStatus.CanCancel = false
		s.fileStatus.CanCopy = text != ""
	}
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	if historyID != 0 && s.history != nil {
		outcome := history.HistoryTranscribed
		if err != nil {
			details.ErrorKind = history.ErrorKind(err)
			details.Processing.DeliveredCharacters = utf8.RuneCountInString(text)
			outcome = history.HistoryFailed
			if errors.Is(ctx.Err(), context.Canceled) {
				outcome = history.HistoryCancelled
			}
		} else {
			details.Processing.DeliveredCharacters = utf8.RuneCountInString(text)
		}
		s.history.Finalize(historyID, outcome, details, time.Now().UTC(), false)
	}

	s.publishFileStatus(changed, status)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.log().Info("audio file transcription cancelled", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "outcome", "cancelled")
		} else {
			s.log().Error("audio file transcription failed", "generation", generation, "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		}
		return
	}
	s.log().Info("audio file transcription completed", "generation", generation, "characters", utf8.RuneCountInString(text), "duration_ms", time.Since(started).Milliseconds(), "outcome", "transcribed", "response_mode", details.ResponseMode, "stream_fallback_reason", details.StreamFallbackReason)
}

func (s *Service) updateFileUpload(generation uint64, sent int64, force bool) {
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || s.fileStatus.Phase != FileTranscriptionUploading {
		s.fileMu.Unlock()
		return
	}
	s.fileStatus.BytesUploaded = min(sent, s.fileStatus.FileSize)
	now := time.Now()
	if !force && now.Sub(s.fileLastPublish) < fileProgressInterval && sent < s.fileStatus.FileSize {
		s.fileMu.Unlock()
		return
	}
	s.fileLastPublish = now
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
}

func (s *Service) fileUploadComplete(generation uint64, stream bool) {
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || s.fileStatus.Phase != FileTranscriptionUploading {
		s.fileMu.Unlock()
		return
	}
	s.fileStatus.BytesUploaded = s.fileStatus.FileSize
	if stream {
		s.fileStatus.Phase = FileTranscriptionStreaming
		s.fileStatus.Message = "Waiting for transcript"
	} else {
		s.fileStatus.Phase = FileTranscriptionProcessing
		s.fileStatus.Message = "Transcribing audio"
	}
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	responseMode := "completed"
	if stream {
		responseMode = "stream"
	}
	s.log().Info("audio file upload completed", "generation", generation, "response_mode", responseMode)
}

func (s *Service) appendFileDelta(generation uint64, delta string) {
	if delta == "" {
		return
	}
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || (s.fileStatus.Phase != FileTranscriptionStreaming && !(s.fileStatus.Phase == FileTranscriptionProcessing && s.fileStatus.Streaming)) {
		s.fileMu.Unlock()
		return
	}
	if s.fileTranscript.Len()+len(delta) > maxFileTranscriptBytes {
		alreadyReported := s.fileTranscriptLimitHit
		s.fileTranscriptLimitHit = true
		s.fileStatus.Message = "Transcript reached the 8 MiB safety limit; preserving partial text"
		status := s.snapshotFileStatusLocked()
		changed := s.fileChanged
		s.fileMu.Unlock()
		if !alreadyReported {
			s.publishFileStatus(changed, status)
		}
		return
	}
	_, _ = s.fileTranscript.WriteString(delta)
	s.fileTranscriptRevision++
	s.fileStatus.TranscriptRevision = s.fileTranscriptRevision
	s.fileStatus.Message = "Receiving transcript"
	changed := s.fileDelta
	payload := FileTranscriptionDelta{Generation: generation, Revision: s.fileTranscriptRevision, Text: delta}
	s.fileMu.Unlock()
	s.publishFileDelta(changed, payload)
}

func (s *Service) fileStreamBuffered(generation uint64) {
	s.fileMu.Lock()
	if s.fileStatus.Generation != generation || s.fileStatus.Phase != FileTranscriptionStreaming {
		s.fileMu.Unlock()
		return
	}
	s.fileStatus.Phase = FileTranscriptionProcessing
	s.fileStatus.Buffered = true
	s.fileStatus.Message = "Server buffered the streamed response"
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	s.log().Info("audio file stream buffered by server", "generation", generation)
}

// CancelFileTranscription requests cancellation of active stored-audio work.
func (s *Service) CancelFileTranscription() error {
	s.fileMu.Lock()
	if !filePhaseActive(s.fileStatus.Phase) || s.fileCancel == nil {
		s.fileMu.Unlock()
		return nil
	}
	s.fileStatus.Phase = FileTranscriptionCancelling
	s.fileStatus.Message = "Cancelling transcription"
	s.fileStatus.CanCancel = false
	cancel := s.fileCancel
	status := s.snapshotFileStatusLocked()
	changed := s.fileChanged
	s.fileMu.Unlock()
	s.publishFileStatus(changed, status)
	cancel()
	return nil
}

// CopyFileTranscript copies the completed stored-audio transcript.
func (s *Service) CopyFileTranscript() error {
	s.fileMu.Lock()
	text := s.fileTranscript.String()
	canCopy := s.fileStatus.CanCopy
	s.fileMu.Unlock()
	if !canCopy || text == "" {
		return errors.New("no completed file transcript is available")
	}
	return s.input.Copy(context.Background(), text)
}

func (s *Service) fileTranscriptionActive() bool {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	return filePhaseActive(s.fileStatus.Phase)
}

func filePhaseActive(phase FileTranscriptionPhase) bool {
	switch phase {
	case FileTranscriptionUploading, FileTranscriptionProcessing, FileTranscriptionStreaming, FileTranscriptionCancelling:
		return true
	default:
		return false
	}
}

func (s *Service) publishFileStatus(changed func(FileTranscriptionStatus), status FileTranscriptionStatus) {
	if changed != nil && !s.closed.Load() {
		changed(status)
	}
}

func (s *Service) publishFileDelta(changed func(FileTranscriptionDelta), delta FileTranscriptionDelta) {
	if changed != nil && !s.closed.Load() {
		changed(delta)
	}
}
