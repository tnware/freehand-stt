package config

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"net/url"
	"strconv"
)

func ManagedContract(s Settings, id string, role compatibility.Role) (managedruntime.Instance, managedruntime.Contract, error) {
	for _, i := range s.ManagedRuntimes {
		if i.ID == id {
			c, e := managedruntime.Qualify(i.Provider, i.ModelForRole(role), role)
			return i, c, e
		}
	}
	return managedruntime.Instance{}, managedruntime.Contract{}, errors.New("managed runtime reference is unavailable")
}
func validateManagedTransport(url string, insecure bool, auth AuthenticationMode, health string, headers map[string]string) error {
	if url != "" || insecure || auth != AuthenticationModeNone || health != "" || len(headers) != 0 {
		return errors.New("managed runtime cannot contain manual transport or credentials")
	}
	return nil
}
func validateResolvedManagedTransport(baseURL string, auth AuthenticationMode, health string, headers map[string]string) error {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return errors.New("managed recording requires a resolved loopback endpoint")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return errors.New("managed recording requires a resolved loopback port")
	}
	if auth != AuthenticationModeNone || health != "" || len(headers) != 0 {
		return errors.New("managed recording cannot use manual credentials or headers")
	}
	return nil
}

func validateManagedReferences(s Settings) error {
	check := func(id string, role compatibility.Role, model string, profile modelprofile.ID, backend compatibility.ID) error {
		if id == "" {
			return nil
		}
		i, c, e := ManagedContract(s, id, role)
		if e != nil {
			return e
		}
		if model != i.ModelForRole(role) || profile != c.ModelProfile || backend != c.CompatibilityProfile {
			return errors.New("managed task model and profiles must match its runtime")
		}
		return nil
	}
	if e := check(s.ManagedInstanceID, compatibility.Transcription, s.Model, s.ModelProfile, s.CompatibilityProfile); e != nil {
		return e
	}
	v := s.VoiceTranscription
	if e := check(v.ManagedInstanceID, compatibility.Transcription, v.Model, v.ModelProfile, v.CompatibilityProfile); e != nil {
		return e
	}
	if v.ManagedInstanceID != "" && v.Realtime {
		if _, _, e := ManagedContract(s, v.ManagedInstanceID, compatibility.Realtime); e != nil {
			return e
		}
	}
	p := s.PostProcessing
	if e := check(p.ManagedInstanceID, compatibility.PostProcessing, p.Model, modelprofile.ID(p.Preset), p.CompatibilityProfile); e != nil {
		return e
	}
	t := s.TextToSpeech
	return check(t.ManagedInstanceID, compatibility.Speech, t.Model, t.ModelProfile, t.CompatibilityProfile)
}
