package postprocess

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/inference"
)

type Processor struct {
	client *inference.Client
	logger *slog.Logger
}

type Result struct {
	Text     string
	Metadata inference.ResponseMetadata
}

func New(client *inference.Client, logger *slog.Logger) *Processor {
	return &Processor{client: client, logger: logger}
}

// ProcessWithCredential uses the operation-scoped credential captured with
// cfg. Callers supply both from one coherent settings snapshot so a later
// credential replacement cannot alter an active request.
func (p *Processor) ProcessWithCredential(ctx context.Context, cfg config.PostProcessingSettings, raw, key string) (Result, error) {
	if err := config.ValidatePostProcessing(cfg); err != nil {
		return Result{}, err
	}
	if p == nil || p.client == nil {
		return Result{}, errors.New("post-processing client is unavailable")
	}

	options := cfg.GenerationOptions
	if cfg.Preset == config.PostProcessingPresetS1Mini {
		contract, err := compatibility.Resolve(cfg.CompatibilityProfile, compatibility.PostProcessing)
		if err != nil {
			return Result{}, err
		}
		// S1-mini requires thinking-disabled generation. Enforce the requirement
		// through a qualified adapter; Generic still requires server configuration.
		options.DisableReasoning = options.DisableReasoning || contract.Capabilities.CleanupDisableReasoning
	}
	systemPrompt, userPrompt := prompts(cfg, raw)
	started := time.Now()
	if p.logger != nil {
		p.logger.Info("transcript post-processing started",
			"server", serverName(cfg.BaseURL),
			"preset", cfg.Preset,
			"input_characters", utf8.RuneCountInString(raw),
			"timeout_seconds", cfg.TimeoutSeconds,
		)
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	completion, err := p.client.WithCompatibility(cfg.CompatibilityProfile).WithCleanupOptions(options).ChatCompletion(requestCtx, cfg.BaseURL, cfg.Model, key, systemPrompt, userPrompt)
	if err == nil && strings.TrimSpace(completion.Text) == "" {
		err = errors.New("post-processing returned an empty transcript")
	}
	if p.logger != nil {
		if err != nil {
			p.logger.Warn("transcript post-processing failed",
				"server", serverName(cfg.BaseURL),
				"preset", cfg.Preset,
				"duration_ms", time.Since(started).Milliseconds(),
				"error_kind", diagnostics.ErrorKind(err),
			)
		} else {
			p.logger.Info("transcript post-processing completed",
				"server", serverName(cfg.BaseURL),
				"preset", cfg.Preset,
				"output_characters", utf8.RuneCountInString(completion.Text),
				"duration_ms", time.Since(started).Milliseconds(),
			)
		}
	}
	return Result{Text: completion.Text, Metadata: completion.Metadata}, err
}

func prompts(cfg config.PostProcessingSettings, raw string) (string, string) {
	if cfg.Preset == config.PostProcessingPresetS1Mini {
		control := fmt.Sprintf("[Styling: %s] [Structure: %s] [Context: %s]", cfg.Styling, cfg.Structure, cfg.Context)
		return S1MiniSystemInstruction, control + "\n" + raw
	}
	return strings.TrimSpace(cfg.SystemPrompt), raw
}

func serverName(baseURL string) string {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://")
	server, _, _ := strings.Cut(trimmed, "/")
	return server
}
