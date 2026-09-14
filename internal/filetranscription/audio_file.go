package filetranscription

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxAudioFileBytes int64 = 2 << 30

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
