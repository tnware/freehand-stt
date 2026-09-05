---
title: Maintain backend compatibility
description: Keep the application catalog, public matrix, provider guides, and validation evidence aligned.
---

## Ownership and update path

`internal/compatibility` owns profile IDs, operation scope, availability, routes,
and implemented capabilities. The renderer receives that catalog through the
Settings DTO. The website consumes its generated JSON export; public availability
and the feature matrix must never be maintained as a second set of support flags.

When a contract changes:

1. Implement and validate the bounded request/response behavior in Go. Reject
   wrong-operation and unavailable profiles at the backend boundary.
2. Update the profile catalog and its capability rules. Keep advanced features
   unavailable until their model and server requirements are represented.
3. Run `go generate ./internal/compatibility` from the repository root. Commit
   `site/src/data/compatibility.generated.json` with the implementation.
4. Update editorial copy in `site/src/data/backends.ts` and the relevant backend
   guide under `site/src/content/docs/docs/backends/`. Every catalog profile must
   have a directory entry and a guide or a specific planned-contract anchor.
5. Run the affected Go fixtures and the site build. The Go catalog test rejects
   a stale export; site rendering rejects missing or extra directory entries.
6. Record validation evidence and limitations in the guide and PR. Do not promote
   a source review or fixture result into a claim of live Windows interoperability.

Run `go test ./internal/compatibility` to check the app/site catalog boundary.
Local site-only builds consume the committed export and need no Go runtime.
The Pages workflow also runs the Go catalog check, including for site-only
changes, before publishing the site.

## Evidence to record

For a live setup, record the operation, Freehand revision, server release or
commit when known, model/voice identifier, response format or streaming dialect,
and observed outcome. Explicitly mark unknown versions. Do not publish private
URLs, credentials, transcripts, machine names, or personal file paths.

Separate these kinds of evidence:

- Client contract fixtures, including request fields, errors, truncation, and
  completion semantics.
- Tagged upstream source or documentation with the inspected version.
- Reported live behavior for a particular setup.
- Native interactive acceptance performed on Windows.

Existing Speaches and llama.cpp reports have limited version information. Keep
that limitation visible until a more specific report replaces it. A model list
is metadata and cannot be used as a capability proof.

## Adding a planned profile

Add a stable ID and only the relevant operation entries with availability off
and no implemented capabilities. Explain the concrete missing contract work.
Add a public directory entry and a specific guide anchor, then regenerate the
catalog. Keep the planned state consistent across Settings, the public site,
and technical documentation. Track scheduling and delivery in GitHub issues/PRs,
not in a separate public task checklist.

## Public-page review

Verify desktop and mobile navigation, active-page indication, keyboard focus,
small-screen matrix scrolling, base-path-aware links, provider guide anchors,
and canonical metadata. The directory tracks main-branch behavior; keep that
notice visible so it is not mistaken for a promise about an older release.

## Current optional STT fields

`transcriptionPrompt`, `transcriptionTemperature`, and `transcriptionHotwords`
are explicit capability flags exported to both Settings and the website matrix.
Generic supports the optional common prompt/temperature shape; Speaches also
supports hotwords. New provider contracts must qualify their own field mapping
instead of inheriting hotwords because they advertise an OpenAI-compatible API.
Source evidence for Speaches is v0.8.3 and v0.9.0-rc.3 Whisper paths. Actual model
response to hints still requires a scoped live report. No version-sensitive VAD
or unexposed faster-whisper tuning field is implied by these flags.

## Cleanup capabilities

`cleanupOutputLimit` and `cleanupDisableReasoning` are generated capability flags
shared by Settings and the public matrix. Generic, llama.cpp, and vLLM support the
optional `max_tokens` shape; llama.cpp and vLLM additionally qualify
`reasoning_effort: "none"`. The S1-mini preset always requires thinking disabled,
so a qualified adapter enforces it automatically while Generic relies on the
server. Record runtime/template evidence separately from the model requirement.
The [llama.cpp guide](../../backends/llama-cpp/#source-qualification) pins source
inspection to an exact upstream commit without claiming a minimum version or
universal live compatibility.


## Additional qualified provider profiles

whisper.cpp now supports completed transcription using its native `/inference`
route and server-loaded model. Its default connection probe is `/health` beneath
the configured server root; it has no client model selection or file streaming.
vLLM v0.28.0 supports completed transcription, its own file-stream dialect, and
text cleanup with optional output limits and reasoning-off requests. The
S1-mini preset requires reasoning off through both qualified cleanup profiles.
See the [whisper.cpp guide](../../backends/whisper-cpp/) and
[vLLM guide](../../backends/vllm/) for setup, contract details, and limitations.


## September 2026 provider acceptance environment

Scoped Windows HTTP-adapter acceptance used the public 11-second JFK WAV from
whisper.cpp and fixed invented cleanup text, one selected server/model at a time.
No inventory inference or private audio was used. The provider guides separate
passing request paths from the empty result on the repeated 33-second sample.

- whisper.cpp image digest: `ghcr.io/ggml-org/whisper.cpp@sha256:2285844e0c38744d90eed59ce5b90fe68cd2dfc6ecb07bb0b68b8ff800528be4`, using `ggml-tiny.en.bin`.
- vLLM base image digest: `vllm/vllm-openai@sha256:61fc8a896b0a4fbbbdc063bc4b0dbc25ce98e02b5050c24aeb7830ac02039b14`, reporting version 0.28.0.
- That vLLM image's audio path needed `av==18.1.0`, `scipy==1.18.1`, `soundfile==0.14.0`, and `soxr==1.1.0`; these missing extras were installed without changing its existing CUDA/PyTorch dependencies. This records one image's environment, not a universal dependency recipe.
- On WSL2 with an RTX 5060 Ti, vLLM used `VLLM_USE_V2_MODEL_RUNNER=0`, `--no-async-scheduling`, `--enforce-eager`, `--gpu-memory-utilization 0.35`, `--max-num-seqs 1`, and `--max-num-batched-tokens 2048`.
- STT used `openai/whisper-tiny.en` with `--max-model-len 448`; cleanup used `superwhisper/s1-mini` with `--max-model-len 2048`. These are test context limits, not Freehand defaults or model maximums.

The default V2 runner failed with unavailable UVA on this WSL setup. With the
older runner, asynchronous scheduling left the first transcription stalled;
synchronous scheduling completed the selected short sample. Keep such runtime
settings on the server. Freehand's adapter must not inject GPU or scheduler
configuration into inference requests.

The user subsequently confirmed live Freehand transcription with
`Qwen/Qwen3-ASR-0.6B` on the same vLLM v0.28.0 base. That server used
`--max-model-len 4096` and `--max-num-batched-tokens 4096`, retaining the other
bounded runtime settings above. Health and model metadata were checked before
the user performed inference. This is user-reported interactive transcription
evidence; Qwen-specific file-stream acceptance and latency tuning are not
claimed. No client language or model-specific prompt override was introduced
for the test. First-class language support remains separate follow-up work.

## Provider identity assets

The app, directory, matrix, and provider guides share the SVG collection and
manifest in `branding/providers/`. Follow its README to add a pinned source,
license notice, and asset mapping. This presentation registry does not grant
capabilities or change the generated Go catalog. Use a documented neutral
fallback when no suitable brand asset is available. Icons remain decorative
beside text; availability and connection health are separate signals.

The Svelte and Astro `ProviderIcon` wrappers use the same CSS tile and local
assets. Provider guide frontmatter sets `provider` to the catalog ID, rendered
through Starlight's supported PageTitle override. Full third-party notices ship
in About and [Provider icon credits](../../reference/provider-icons/).

For changes, run the actual Svelte autofixer on edited components, frontend
check/build, and the site build. Check selectors and quick settings in the app,
then backend cards, matrix, docs headings, and guide links on desktop/mobile
and in both docs themes. Verify that icons load without external requests and
that Generic/fallback icons do not imply a branded service or enabled support.
