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
6. Record validation evidence and limitations in contributor documentation and the PR. Do not promote
   a source review or fixture result into a claim of live Windows interoperability.

Run `go test ./internal/compatibility` to check the app/site catalog boundary.
Local site-only builds consume the committed export and need no Go runtime.
The Pages workflow also runs the Go catalog check, including for site-only
changes, before publishing the site.

## Backend and model documentation

Keep server APIs under `docs/backends/` and dedicated model behavior under
`docs/models/`. The public `/backends/` directory compares server operations;
`/models/` explains the controls each dedicated profile adds in Freehand.
NeMo-Speech.cpp is a backend, Nemotron is a model, and Qwen3-ASR belongs under
model profiles alongside S1-mini.

`internal/modelprofile` remains the authority for dedicated profiles and their
backend intersections. Update the editorial summaries in `site/src/data/models.ts`
and the corresponding model guides when these contracts change. Do not present
Generic examples, such as Whisper or Kokoro, as additional dedicated profiles.
Link model guides to backend installation and backend guides to model controls.
Keep test reports and qualification history here rather than on product cards.

When moving a guide, update internal links and the explicit Starlight sidebar,
and preserve its old URL with a base-path-aware Astro redirect. Check the Models
and Backends navigation on desktop and mobile, cross-links, and redirects after
building the static site. No inference is needed for this review.

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
The [llama.cpp source record](#llama-cpp-source-qualification) pins source
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
No inventory inference or private audio was used. The acceptance records below separate passing request paths from the empty
result on the repeated 33-second sample.

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
for the test. Language selection has since gained explicit provider mapping and S1-mini admission policy; see the [language guide](../../guides/languages/). This does not extend that earlier live acceptance to other languages.

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

## Archived provider qualification notes

These records were moved from the public setup guides to keep implementation
and acceptance history in contributor documentation. They describe the scope
of the original checks, not new acceptance runs.

### Generic — original coverage note

**Evidence:** automated client contract fixtures. No universal server or model
compatibility is implied.

### Speaches — original coverage note

**Evidence:** automated client contract fixtures plus a reported working
Whisper-family STT and Kokoro TTS setup. The runtime version for that reported
setup was not recorded; it does not establish compatibility for every model
in the server inventory.

### llama.cpp — original coverage note

**Evidence:** automated client chat-contract fixtures plus a reported working
S1-mini setup. The runtime version for that report was not recorded. Other
models and server versions need their own qualification.

### Speaches — versions and streaming

The compatibility audit inspected **v0.8.3** and **v0.9.0-rc.3**:

- The older transcription route emits a JSON text segment per SSE event and
  completes by closing the response.
- The inspected newer Whisper executor emits typed delta/done events. Freehand
  requires final text for a typed stream; EOF alone is insufficient.
- The inspected binary speech path can produce compatible PCM16 WAV. Freehand
  does not consume speech SSE events or perform progressive playback.

These are source and client-fixture qualifications, not live tests of every
release or executor. The RC label is retained here intentionally. Actual model
and voice IDs, decoder support, and speed behavior depend on the served setup.

### Speaches — voice discovery sources

Voice discovery is qualified against the
[Speaches metadata routes](https://github.com/speaches-ai/speaches/blob/fc50e7133c175bae320eed6e0db9a342fcb21837/src/speaches/routers/models.py)
and local fixtures. This does not claim live acceptance of every Speaches release.

### Speaches — recognition control coverage

The inspected **v0.8.3** transcription route accepts `prompt`, `hotwords`, and
`temperature` and forwards them to faster-whisper. **v0.9.0-rc.3** forwards the
same controls in both its completed and streaming Whisper executor paths.
This qualification covers the Whisper-family contract, not every Speaches
executor or all model behavior. These new fields have request fixtures and
source evidence; the earlier working-setup report does not establish live
acceptance of the new controls.

<span id="llama-cpp-source-qualification"></span>

### llama.cpp — source qualification

The adapter was inspected against upstream llama.cpp commit
[`6a1a922d2699`](https://github.com/ggml-org/llama.cpp/commit/6a1a922d269908a29cbd4b49c27e6a8e7fd10fae):

- [Server schema](https://github.com/ggml-org/llama.cpp/blob/6a1a922d269908a29cbd4b49c27e6a8e7fd10fae/tools/server/server-schema.cpp) accepts `max_tokens` as a generation-limit alias.
- [Chat request parser](https://github.com/ggml-org/llama.cpp/blob/6a1a922d269908a29cbd4b49c27e6a8e7fd10fae/tools/server/server-common.cpp) maps `reasoning_effort: "none"` to thinking disabled before applying the model template.

This records source and client-fixture evidence, not a minimum supported release
or a live test of every model/template. The reported S1-mini runtime's exact
build is still unknown. Its launch log shows Jinja enabled and recognition of
the newer reasoning option, which alone does not prove the HTTP override works.

### Kokoro-FastAPI — sample and source coverage

A manually requested sample against a deployment reporting API **0.6.0** returned
68 voice entries and a playable-format 24 kHz mono PCM16 WAV for `kokoro` with
`af_heart` at speed 1.0. Freehand's real request adapter and WAV decoder handled
that response on Windows. This verifies the response format, not speaker-device
playback or every voice, model alias, speed, image tag, or server version.

Local fixtures cover both voice-list shapes, Speaches isolation, authentication,
malformed and oversized metadata, redirects, and the Kokoro buffering override.
The upstream router was inspected at
[`5fb71ea`](https://github.com/remsky/Kokoro-FastAPI/blob/5fb71ea6e75379f95dee0f4a42c12152f4ea0e1a/api/src/routers/openai_compatible.py).

### whisper.cpp — source and Windows adapter acceptance

Source qualification pins whisper.cpp
[`52a939a2a762`](https://github.com/ggml-org/whisper.cpp/blob/52a939a2a762224e255d366c1182b2af4dd1a032/examples/server/server.cpp).
Client fixtures cover prefixed routing, health defaults/overrides, omitted
model fields, bounded multipart upload, hints, and completed-only behavior.
This is not a claim that every build, audio format, or model has been tested
interactively on Windows.

#### Scoped live acceptance — 2026-09-05

Freehand's Windows Go adapters completed both microphone-request and file-upload
paths against the existing CUDA image (`sha256:2c42506808d7546ea3440c0053dd6543373cc4252c525b7972ab96554a533837`)
using `ggml-tiny.en.bin` and the public 11-second whisper.cpp JFK sample. The
server's `/health` probe succeeded. This exercises HTTP/inference behavior from
Windows; interactive capture, focus-safe insertion, and every file format were
not part of that fixed-sample run. The source pin above records inspected
contract evidence independently of the tested image digest.

### vLLM — initial transcription observations

`Qwen/Qwen3-ASR-0.6B` was confirmed working by a user in the native Freehand
application with vLLM v0.28.0. Use the exact model ID exposed by your deployment.
The [upstream Qwen3-ASR guide](https://docs.vllm.ai/projects/recipes/en/latest/Qwen/Qwen3-ASR.html)
documents the transcription API used by this profile.

The initial English-only `openai/whisper-tiny.en` test checkpoint had limitations:
automatic language detection returned HTTP 500, and some speech returned empty
text even with explicit English. Its passing fixed sample is limited transport
evidence, not a recommendation for dictation. Freehand preserves your language
choice; it does not silently force English for other models.

### vLLM — WSL runner observations

On the tested WSL2/RTX 5060 Ti system, the default V2 runner failed with
`UVA is not available`. Setting `VLLM_USE_V2_MODEL_RUNNER=0` selected the installed
older runner. Whisper's encoder also required `--max-num-batched-tokens 2048`
even with `--max-model-len 448` and one concurrent request. With that older
runner, the short sample stalled under asynchronous scheduling;
`--no-async-scheduling` allowed completed and streamed requests to finish. These are scoped
server setup observations, not settings Freehand sends in its API requests.

### vLLM — source and runtime acceptance

Client fixtures cover requests, framing, multiple audio chunks, usage, failures,
cancellation, credential reflection, and required S1-mini reasoning off. Source
and fixture evidence do not establish interoperability for every earlier
release or model/template. Record actual server versions and models when doing
live acceptance.

On September 5, 2026, the native Windows client HTTP adapter passed the public
11-second whisper.cpp JFK sample against v0.28.0 with
`openai/whisper-tiny.en`: completed microphone-shaped upload, completed file,
and streamed file (22 nonempty deltas). A 33-second repetition of that sample
returned two empty, successfully terminated server chunks; it is recorded as
an empty model result, not successful long-file recognition. Multiple-chunk
text assembly and incomplete-stream recovery are covered by fixtures. These
checks do not establish microphone capture, focus-safe insertion, recognition
quality, or support for every audio format.

On the same date, the user confirmed successful live transcription in Freehand
with **vLLM v0.28.0 and `Qwen/Qwen3-ASR-0.6B`**. GPU access and GPU model/cache
allocation were verified. The observed request was slow under the conservative
WSL test configuration; latency tuning remains separate. This confirmation does
not establish Qwen file streaming, all languages/formats, or exhaustive native
focus-safety acceptance.

The same v0.28.0 runtime also passed a fixed S1-mini cleanup request using
`superwhisper/s1-mini` and the native Windows processing adapter, with the
optional custom-model reasoning switch off: the S1-mini preset still enforced
`reasoning_effort: "none"`. A one-token output limit produced the expected
incomplete-response error rather than accepting truncated cleanup text.

### Qwen3-ASR — integration qualification

This integration targets vLLM 0.28.0 and the original
`Qwen/Qwen3-ASR-1.7B` checkpoint at revision `7278e1e70fe206f11671096ffdd38061171dd6e5`. It does not automatically
qualify other runtimes, modified weights, quantizations, or a future vLLM protocol.
The completed API and realtime endpoint were exercised with synthetic English
audio through Docker/WSL2, including Freehand's Go realtime adapter. This is
transport evidence; native microphone, overlay, and insertion acceptance is
separate. Runtime performance and recognition quality belong to the chosen
model and deployment.

### Planned profiles — implementation requirements

Before enabling any dedicated profile, define the supported operation and
request/response shape, distinguish model-specific fields, cover malformed and
incomplete responses, and document applicable upload/audio limits. Preserve
metadata-only tests, credential isolation, cancellation, and no automatic
inference replay. An unimplemented advanced feature remains unavailable even
when another operation on the same provider is supported.

Contributors can use the [maintenance guide](../../development/backend-compatibility/)
for the exact update and validation process.

### Original protocol acceptance summary

Qualification evidence for the Speaches stream formats comes from the audit's
v0.8.3 and v0.9.0-rc.3 source comparison. Profile fixtures cover these response
shapes and the shared llama.cpp text request; they do not establish live
compatibility with every release or model. Existing user-tested integrations
remain distinct from automated fixture coverage.

#### Validated implementations

Freehand targets capability-specific OpenAI-compatible routes rather than requiring one particular server. Compatibility claims use three evidence levels:

- **Validated** — exercised end to end in the native Windows application.
- **Contract-compatible** — the client implements the documented route and
  shape, but a named backend is not claimed as tested.
- **Unsupported or unknown** — a required route is absent, or available evidence
  is insufficient to make a compatibility claim.

The following combinations have been exercised end to end in the Windows app:

| Freehand capability                       | Tested backend                                                    | Compatible route                | Evidence                        |
| ----------------------------------------- | ----------------------------------------------------------------- | ------------------------------- | ------------------------------- |
| Microphone and stored-file speech to text | [Speaches](https://github.com/speaches-ai/speaches)               | `POST /v1/audio/transcriptions` | Validated in native Windows use |
| Text to speech                            | [Speaches](https://github.com/speaches-ai/speaches)               | `POST /v1/audio/speech`         | Validated in native Windows use |
| S1-mini transcript post-processing        | [llama.cpp](https://github.com/ggml-org/llama.cpp) `llama-server` | `POST /v1/chat/completions`     | Validated in native Windows use |

These are known-working implementations, not product dependencies or an exhaustive compatibility list. “OpenAI-compatible” does not guarantee that a server implements every optional audio and chat route, so Freehand configures and tests STT, TTS, and post-processing independently. A project name or model listing is never sufficient evidence to claim end-to-end support.

See [Connect a speech server](../../guides/connect-a-server/) for topology,
configuration, and failure guidance.
