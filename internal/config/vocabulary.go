package config

import (
	"errors"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// VocabularySettings belongs to the user's task, independently of models.
type VocabularySettings struct {
	Terms string  `json:"terms"`
	Voice bool    `json:"voice"`
	Files bool    `json:"files"`
	Boost float64 `json:"boost"`
}

func ValidateVocabulary(v VocabularySettings) error {
	if !utf8.ValidString(v.Terms) || len(v.Terms) > 16384 {
		return errors.New("vocabulary must be at most 16,384 UTF-8 bytes")
	}
	for _, r := range v.Terms {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return errors.New("vocabulary cannot contain control characters")
		}
	}
	if math.IsNaN(v.Boost) || math.IsInf(v.Boost, 0) || v.Boost < 0 || v.Boost > 5 {
		return errors.New("vocabulary strength must be between 0 and 5")
	}
	return nil
}

// VocabularyTerms preserves phrase boundaries and removes exact duplicates.
func VocabularyTerms(text string) string {
	seen := map[string]bool{}
	var phrases []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			phrases = append(phrases, line)
			seen[line] = true
		}
	}
	return strings.Join(phrases, "\n")
}

type VocabularySelection struct {
	Backend      compatibility.ID `json:"backend"`
	ModelProfile modelprofile.ID  `json:"modelProfile"`
	Realtime     bool             `json:"realtime"`
	Context      string           `json:"context"`
}
type VocabularyPreviewRequest struct {
	Vocabulary VocabularySettings  `json:"vocabulary"`
	Voice      VocabularySelection `json:"voice"`
	Files      VocabularySelection `json:"files"`
}
type VocabularySupport struct {
	Mode    string `json:"mode"`
	Problem string `json:"problem"`
}
type VocabularyPreview struct {
	Voice VocabularySupport `json:"voice"`
	Files VocabularySupport `json:"files"`
}

func PreviewVocabulary(v VocabularySettings, s VocabularySelection) VocabularySupport {
	mode := modelprofile.VocabularyMode(s.ModelProfile, s.Backend, s.Realtime)
	result := VocabularySupport{Mode: mode}
	if mode == "" {
		result.Problem = "This model and backend do not support vocabulary hints."
		return result
	}
	if err := ValidateVocabulary(v); err != nil {
		result.Problem = err.Error()
		return result
	}
	terms := VocabularyTerms(v.Terms)
	switch mode {
	case "speech-contexts":
		if err := modelprofile.ValidateNemotron("auto", modelprofile.NemotronOptions{Vocabulary: terms, Boost: v.Boost}); err != nil {
			result.Problem = err.Error()
		}
	case "hotwords":
		if len(terms) > compatibility.MaxTranscriptionHotwordsBytes {
			result.Problem = "This backend accepts at most 2,048 UTF-8 bytes of vocabulary."
		}
	case "prompt":
		if len(vocabularyPrompt(s.Context, terms)) > compatibility.MaxTranscriptionPromptBytes {
			result.Problem = "Context and vocabulary together must fit within 8,192 UTF-8 bytes."
		}
	}
	return result
}

func vocabularyPrompt(context, terms string) string {
	if terms == "" {
		return context
	}
	if strings.TrimSpace(context) == "" {
		return terms
	}
	return context + "\n\n" + terms
}

// WithVocabulary projects into an immutable request snapshot. Unsupported
// adapters omit hints; supported adapters reject excess text rather than truncate.
func WithVocabulary(v Settings, voice bool) (Settings, error) {
	enabled := v.Vocabulary.Files
	s := VocabularySelection{Backend: v.CompatibilityProfile, ModelProfile: v.ModelProfile, Context: v.TranscriptionOptions.Prompt}
	if voice {
		enabled = v.Vocabulary.Voice
		r := v.VoiceTranscription
		s = VocabularySelection{Backend: r.CompatibilityProfile, ModelProfile: r.ModelProfile, Context: r.TranscriptionOptions.Prompt, Realtime: r.Realtime}
	}
	if !enabled || strings.TrimSpace(v.Vocabulary.Terms) == "" {
		return v, nil
	}
	support := PreviewVocabulary(v.Vocabulary, s)
	if support.Mode == "" {
		return v, nil
	}
	if support.Problem != "" {
		return v, errors.New("Vocabulary: " + support.Problem)
	}
	terms := VocabularyTerms(v.Vocabulary.Terms)
	options := &v.TranscriptionOptions
	if voice {
		options = &v.VoiceTranscription.TranscriptionOptions
	}
	switch support.Mode {
	case "hotwords":
		options.Hotwords = terms
	case "prompt":
		options.Prompt = vocabularyPrompt(options.Prompt, terms)
	case "speech-contexts":
		if voice && s.Realtime {
			v.VoiceTranscription.Options = modelprofile.NemotronOptions{Vocabulary: terms, Boost: v.Vocabulary.Boost}
		} else {
			options.Vocabulary = terms
			options.VocabularyBoost = v.Vocabulary.Boost
		}
	}
	return v, nil
}
