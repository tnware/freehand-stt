package inference

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranscribeCapturesOptionalResponseMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("X-Request-Id", "req_transcription")
		_, _ = io.WriteString(w, `{
			"text":"captured words",
			"id":"transcription_123",
			"request_id":"body_transcription_request",
			"model":"whisper-1-2026-08-01",
			"provider":"Compatible provider",
			"created":1788200000,
			"language":"en",
			"languages":["en",{"code":"es"}],
			"duration":12.5,
			"service_tier":"default",
			"system_fingerprint":"fp_transcription",
			"usage":{
				"type":"tokens",
				"input_tokens":20,
				"output_tokens":4,
				"total_tokens":24,
				"input_token_details":{"audio_tokens":18,"text_tokens":2}
			},
			"timings":{"prompt_n":20,"prompt_ms":100,"prompt_per_second":200}
		}`)
	}))
	defer server.Close()

	result, err := New().Transcribe(context.Background(), server.URL, "configured-model", "", "", nil, []byte("RIFF"))
	if err != nil {
		t.Fatal(err)
	}
	metadata := result.Metadata
	if result.Text != "captured words" || metadata.RequestID != "body_transcription_request" || metadata.ResponseID != "transcription_123" || metadata.EffectiveModel != "whisper-1-2026-08-01" || metadata.Provider != "Compatible provider" {
		t.Fatalf("result = %#v", result)
	}
	if metadata.RequestCount != 1 || metadata.UsageReportCount != 1 || metadata.CostReportCount != 0 || metadata.PerformanceReportCount != 1 {
		t.Fatalf("report coverage = %#v", metadata)
	}
	if metadata.CreatedAtUnix == nil || *metadata.CreatedAtUnix != 1788200000 || metadata.ServerAudioSeconds == nil || *metadata.ServerAudioSeconds != 12.5 {
		t.Fatalf("timestamps and duration = %#v", metadata)
	}
	if strings.Join(metadata.DetectedLanguages, ",") != "en,es" {
		t.Fatalf("languages = %#v", metadata.DetectedLanguages)
	}
	if metadata.Usage.Type != "tokens" || value(metadata.Usage.InputTokens) != 20 || value(metadata.Usage.OutputTokens) != 4 || value(metadata.Usage.TotalTokens) != 24 || value(metadata.Usage.AudioInputTokens) != 18 || value(metadata.Usage.TextInputTokens) != 2 {
		t.Fatalf("usage = %#v", metadata.Usage)
	}
	if value(metadata.Performance.PromptTokens) != 20 || floatValue(metadata.Performance.PromptMilliseconds) != 100 || floatValue(metadata.Performance.PromptTokensPerSecond) != 200 {
		t.Fatalf("performance = %#v", metadata.Performance)
	}
}

func TestTranscriptionStreamCapturesTerminalMetadata(t *testing.T) {
	stream := strings.NewReader("data: {\"type\":\"transcript.text.delta\",\"delta\":\"Hello \"}\n\n" +
		"data: {\"type\":\"transcript.text.done\",\"text\":\"Hello world.\",\"id\":\"transcript_stream\",\"model\":\"gpt-4o-mini-transcribe\",\"language\":\"en\",\"usage\":{\"type\":\"duration\",\"seconds\":4.75}}\n\n" +
		"data: [DONE]\n\n")

	result, err := readTranscriptionSSE(stream, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Hello world." || result.Metadata.ResponseID != "transcript_stream" || result.Metadata.EffectiveModel != "gpt-4o-mini-transcribe" {
		t.Fatalf("result = %#v", result)
	}
	if result.Metadata.Usage.Type != "duration" || result.Metadata.Usage.AudioSeconds == nil || *result.Metadata.Usage.AudioSeconds != 4.75 || result.Metadata.UsageReportCount != 1 {
		t.Fatalf("usage = %#v", result.Metadata.Usage)
	}
}

func TestChatCompletionCapturesPortableAndProviderMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("X-Request-Id", "header_request")
		_, _ = io.WriteString(w, `{
			"id":"chatcmpl_123",
			"request_id":"body_request",
			"model":"cleanup-model-q4",
			"provider":"Local runtime",
			"created":1788200100,
			"service_tier":"default",
			"system_fingerprint":"fp_cleanup",
			"choices":[{"message":{"content":"Clean words."},"finish_reason":"stop"}],
			"usage":{
				"prompt_tokens":120,
				"completion_tokens":18,
				"total_tokens":138,
				"prompt_tokens_details":{"cached_tokens":24,"cache_write_tokens":6},
				"completion_tokens_details":{"reasoning_tokens":3},
				"cost":0.0012,
				"cost_details":{"upstream_inference_cost":0.0009}
			},
			"timings":{
				"prompt_n":120,
				"prompt_ms":600,
				"prompt_per_token_ms":5,
				"prompt_per_second":200,
				"predicted_n":18,
				"predicted_ms":360,
				"predicted_per_token_ms":20,
				"predicted_per_second":50,
				"cache_n":24
			}
		}`)
	}))
	defer server.Close()

	result, err := New().ChatCompletion(context.Background(), server.URL, "configured-model", "", "instruction", "raw words")
	if err != nil {
		t.Fatal(err)
	}
	metadata := result.Metadata
	if result.Text != "Clean words." || metadata.RequestID != "body_request" || metadata.ResponseID != "chatcmpl_123" || metadata.EffectiveModel != "cleanup-model-q4" || metadata.Provider != "Local runtime" || metadata.FinishReason != "stop" {
		t.Fatalf("result = %#v", result)
	}
	if value(metadata.Usage.InputTokens) != 120 || value(metadata.Usage.OutputTokens) != 18 || value(metadata.Usage.TotalTokens) != 138 || value(metadata.Usage.CachedInputTokens) != 24 || value(metadata.Usage.CacheWriteTokens) != 6 || value(metadata.Usage.ReasoningOutputTokens) != 3 {
		t.Fatalf("usage = %#v", metadata.Usage)
	}
	if floatValue(metadata.Usage.ReportedCost) != 0.0012 || floatValue(metadata.Usage.UpstreamCost) != 0.0009 || metadata.CostReportCount != 1 {
		t.Fatalf("cost = %#v", metadata.Usage)
	}
	if value(metadata.Performance.PromptTokens) != 120 || floatValue(metadata.Performance.PromptTokensPerSecond) != 200 || value(metadata.Performance.GeneratedTokens) != 18 || floatValue(metadata.Performance.GenerationTokensPerSecond) != 50 || value(metadata.Performance.CachedPromptTokens) != 24 || metadata.PerformanceReportCount != 1 {
		t.Fatalf("performance = %#v", metadata.Performance)
	}
}

func TestMalformedOptionalMetadataDoesNotRejectText(t *testing.T) {
	result, err := readTranscriptionJSON(strings.NewReader(`{"text":"still valid","usage":"not an object","duration":"unknown","languages":{"unexpected":true}}`), maxResponse)
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "still valid" || result.Metadata.Reported() || result.Metadata.UsageReportCount != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestResponseMetadataAggregatesCheckpointCoverage(t *testing.T) {
	first := ResponseMetadata{
		RequestID:         "request_one",
		EffectiveModel:    "speech-model",
		DetectedLanguages: []string{"en"},
		Usage: Usage{
			InputTokens:  intPointer(10),
			OutputTokens: intPointer(2),
			TotalTokens:  intPointer(12),
			ReportedCost: floatPointer(0.1),
		},
		Performance: Performance{
			PromptTokens:       intPointer(10),
			PromptMilliseconds: floatPointer(100),
		},
		RequestCount:           1,
		UsageReportCount:       1,
		CostReportCount:        1,
		PerformanceReportCount: 1,
	}
	second := ResponseMetadata{
		RequestID:          "request_two",
		EffectiveModel:     "speech-model",
		DetectedLanguages:  []string{"es"},
		ServerAudioSeconds: floatPointer(3.5),
		RequestCount:       1,
	}

	var aggregate ResponseMetadata
	aggregate.Add(first)
	aggregate.Add(second)
	if aggregate.RequestCount != 2 || aggregate.UsageReportCount != 1 || aggregate.CostReportCount != 1 || aggregate.PerformanceReportCount != 1 {
		t.Fatalf("coverage = %#v", aggregate)
	}
	if aggregate.RequestID != "" || aggregate.EffectiveModel != "speech-model" || strings.Join(aggregate.DetectedLanguages, ",") != "en,es" {
		t.Fatalf("identity = %#v", aggregate)
	}
	if value(aggregate.Usage.TotalTokens) != 12 || floatValue(aggregate.Usage.ReportedCost) != 0.1 || floatValue(aggregate.ServerAudioSeconds) != 3.5 {
		t.Fatalf("totals = %#v", aggregate)
	}
	if floatValue(aggregate.Performance.PromptMillisecondsPerToken) != 10 || floatValue(aggregate.Performance.PromptTokensPerSecond) != 100 {
		t.Fatalf("derived aggregate performance = %#v", aggregate.Performance)
	}
}

func value(number *int64) int64 {
	if number == nil {
		return -1
	}
	return *number
}

func floatValue(number *float64) float64 {
	if number == nil {
		return -1
	}
	return *number
}

func intPointer(value int64) *int64 {
	return &value
}

func floatPointer(value float64) *float64 {
	return &value
}
