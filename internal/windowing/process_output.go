package windowing

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ProcessOutputRequest contains navigation only, never captured process text.
type ProcessOutputRequest struct {
	InstanceID string `json:"instanceID"`
}
type ProcessOutputNavigation struct {
	Exists func(string) bool
	Open   func(string) error
	Close  func(string)
}

func ConfigureProcessOutput(s *Service, navigation ProcessOutputNavigation) {
	s.processOutput = navigation
}
func (s *Service) CurrentProcessOutput() ProcessOutputRequest {
	s.requestMu.Lock()
	defer s.requestMu.Unlock()
	return s.outputRequest
}
func (s *Service) setProcessOutput(request ProcessOutputRequest) {
	s.requestMu.Lock()
	s.outputRequest = request
	s.requestMu.Unlock()
}
func (s *Service) OpenProcessOutput(request ProcessOutputRequest) error {
	id := request.InstanceID
	if id == "" || len(id) > 64 || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return errors.New("invalid runtime selection")
	}
	s.navigationMu.Lock()
	defer s.navigationMu.Unlock()
	nav := s.processOutput
	if nav.Exists == nil || !nav.Exists(id) || nav.Open == nil {
		return errors.New("runtime output is unavailable")
	}
	previous := s.CurrentProcessOutput()
	if previous.InstanceID != "" && previous.InstanceID != id && nav.Close != nil {
		nav.Close(previous.InstanceID)
	}
	s.setProcessOutput(request)
	if err := nav.Open(id); err != nil {
		s.setProcessOutput(ProcessOutputRequest{})
		if nav.Close != nil {
			nav.Close(id)
		}
		return err
	}
	return nil
}
func (s *Service) CloseProcessOutput() {
	s.navigationMu.Lock()
	defer s.navigationMu.Unlock()
	previous := s.CurrentProcessOutput()
	s.setProcessOutput(ProcessOutputRequest{})
	if previous.InstanceID != "" && s.processOutput.Close != nil {
		s.processOutput.Close(previous.InstanceID)
	}
}
