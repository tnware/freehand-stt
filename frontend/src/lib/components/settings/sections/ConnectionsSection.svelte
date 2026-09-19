<script lang="ts">
  import Disclosure from "$lib/components/common/Disclosure.svelte";
  import { platformPresentation } from "$lib/platform";
  import {
    connectionWorkflows,
    managedConnectionSupports,
  } from "$lib/utils/connectionChoices";
  import { getContext, onMount } from "svelte";
  import { session } from "$lib/stores/session.svelte";
  import { SETTINGS_NAVIGATION, type SettingsSectionID } from "$lib/navigation";
  import * as WindowingService from "$bindings/windowing/service";
  const navigate = getContext<
    ((section: SettingsSectionID) => void) | undefined
  >(SETTINGS_NAVIGATION);
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  onMount(() => {
    void runtime.load();
  });
  async function openRuntimes() {
    if (navigate) navigate("local-runtime");
    else
      try {
        await WindowingService.OpenSettings("local-runtime");
      } catch (cause) {
        session.messages.fail(cause);
      }
  }
  import { Purpose } from "$bindings/savedconnection";
  import { ID } from "$bindings/compatibility";
  import { AuthenticationMode } from "$lib/state";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import CompatibilityProfilePicker from "$lib/components/settings/CompatibilityProfilePicker.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import * as Select from "$lib/components/ui/select";
  import ConnectionSaveActions from "$lib/components/settings/ConnectionSaveActions.svelte";

  let {
    editor,
    runtime = session.runtime,
    activateFor = $bindable(),
    chooseWorkflow = false,
    formID,
    externalActions = false,
    error,
    onBack,
    onSaved,
  }: {
    editor: SettingsEditor;
    runtime?: ManagedRuntimeState;
    activateFor?: Purpose;
    chooseWorkflow?: boolean;
    formID?: string;
    externalActions?: boolean;
    error: string;
    onBack: () => void;
    onSaved: (purpose?: Purpose, id?: string) => void;
  } = $props();
  const form = $derived(editor.connectionDraft);
  const catalog = $derived(editor.applied?.savedConnections);
  const managed = $derived(form?.details.managedInstanceID != null);
  const instance = $derived(
    runtime.statusFor(form?.details.managedInstanceID ?? "")?.instance,
  );
  function setTarget(value: string) {
    if (value === "manual") editor.setConnectionTarget(undefined);
    else editor.setConnectionTarget("");
    if (form && managed) form.uses = [];
  }
  function chooseInstance(value: string) {
    editor.setConnectionTarget(value);
    if (!form) return;
    const selected = runtime.statusFor(value)?.instance;
    form.uses = form.uses.filter((purpose) =>
      managedConnectionSupports(selected, runtime.providers, purpose),
    );
    if (
      activateFor &&
      !managedConnectionSupports(selected, runtime.providers, activateFor)
    )
      activateFor = undefined;
    if (activateFor && !form.uses.includes(activateFor))
      form.uses = [...form.uses, activateFor];
    if (!form.uses.length) {
      const first = roles.find((role) =>
        managedConnectionSupports(selected, runtime.providers, role.id),
      );
      if (first) form.uses = [first.id];
    }
  }
  const busy = $derived(editor.saving || editor.managedConnectionTesting);
  const roles = connectionWorkflows;
  function chooseUse(value: string) {
    activateFor =
      value === "save-only"
        ? undefined
        : roles.find((role) => role.id === value)?.id;
    if (activateFor && form && !form.uses.includes(activateFor))
      form.uses = [...form.uses, activateFor];
  }
  const roleLabel = (p: Purpose) =>
    roles.find((r) => r.id === p)?.label ?? "Connection";
  const profiles = $derived.by(() => {
    const catalog = editor.applied?.compatibilityProfiles;
    const all =
      activateFor && !chooseWorkflow
        ? ((activateFor === Purpose.Voice
            ? catalog?.transcription
            : activateFor === Purpose.Transcription
              ? catalog?.transcription
              : activateFor === Purpose.Cleanup
                ? catalog?.postProcessing
                : catalog?.speech) ?? [])
        : [
            ...(catalog?.transcription ?? []),
            ...(catalog?.postProcessing ?? []),
            ...(catalog?.speech ?? []),
          ];
    return [...new Map(all.map((p) => [p.id, p])).values()].map((p) => ({
      ...p,
      available:
        all.some((x) => x.id === p.id && x.available) &&
        (!activateFor || chooseWorkflow || supports(activateFor, p.id)),
      description: all.some((x) => x.id === p.id && x.available)
        ? "Uses this backend’s implemented contracts for the features enabled below."
        : p.description,
    }));
  });
  function supports(
    purpose: Purpose,
    profile = form?.details.compatibilityProfile,
  ) {
    if (managed)
      return managedConnectionSupports(instance, runtime.providers, purpose);
    const catalog = editor.applied?.compatibilityProfiles;
    const list =
      purpose === Purpose.Voice
        ? catalog?.transcription
        : purpose === Purpose.Transcription
          ? catalog?.transcription
          : purpose === Purpose.Cleanup
            ? catalog?.postProcessing
            : catalog?.speech;
    return !!list?.some((p) => p.id === profile && p.available);
  }
  function setProfile(profile: ID) {
    if (!form) return;
    form.details.compatibilityProfile = profile;
    form.uses = form.uses.filter((p) => supports(p, profile));
    if (form.creating && !form.uses.length) {
      const first = roles.find((role) => supports(role.id, profile));
      if (first) form.uses = [first.id];
    }
    if (activateFor && !supports(activateFor, profile))
      activateFor =
        form.uses[0] ?? roles.find((role) => supports(role.id, profile))?.id;
    if (
      !form.uses.includes(Purpose.Transcription) &&
      !form.uses.includes(Purpose.Voice)
    ) {
      form.details.healthPath = "";
      form.details.headers = {};
    }
  }
  function setUse(p: Purpose, enabled: boolean) {
    if (!form) return;
    form.uses = enabled ? [...form.uses, p] : form.uses.filter((x) => x !== p);
    if (
      !form.uses.includes(Purpose.Transcription) &&
      !form.uses.includes(Purpose.Voice)
    ) {
      form.details.healthPath = "";
      form.details.headers = {};
    }
  }
  function changeAuth(value: string) {
    if (!form) return;
    if (
      value !== AuthenticationMode.AuthenticationModeNone &&
      value !== AuthenticationMode.AuthenticationModeAPIKey
    )
      return;
    form.details.authenticationMode = value;
    if (value === AuthenticationMode.AuthenticationModeNone)
      form.credentialDraft = "";
  }
  function renameHeader(old: string, next: string) {
    if (!form || old === next) return;
    const headers = { ...form.details.headers };
    const value = headers[old];
    delete headers[old];
    headers[next] = value;
    form.details.headers = headers;
  }
  function removeHeader(key: string) {
    if (!form) return;
    const headers = { ...form.details.headers };
    delete headers[key];
    form.details.headers = headers;
  }
  function addHeader() {
    if (!form) return;
    let key = "X-Custom-Header",
      n = 1;
    while (key in (form.details.headers ?? {})) key = `X-Custom-Header-${++n}`;
    form.details.headers = { ...form.details.headers, [key]: "" };
  }
  const native = $derived(platformPresentation(editor.applied?.platform));
</script>

{#if form}
  <form
    id={formID}
    onsubmit={async (event) => {
      event.preventDefault();
      const purpose = activateFor;
      const name = form.name.trim();
      if (
        managed &&
        (!instance ||
          !form.uses.length ||
          form.uses.some((use) => !supports(use)))
      ) {
        session.messages.reportInfo(
          "Choose a runtime with a qualified model for each selected use.",
        );
        return;
      }
      if (await editor.saveConnection(purpose))
        onSaved(
          purpose,
          editor.applied?.savedConnections.entries?.find(
            (connection) => connection.name === name,
          )?.id,
        );
    }}
    class="flex min-h-0 flex-col gap-3"
  >
    <div class="flex min-h-0 flex-col gap-3">
      {#snippet supportedUses()}
        {#if form}
          <div class="space-y-3 py-3">
            <h4 class="text-[13px] font-medium">Used for</h4>
            <p class="text-xs text-muted-foreground">
              Choose where this connection appears. Each workflow keeps its own
              model and options.
            </p>
            {#each roles as role (role.id)}
              <div class="flex items-center justify-between gap-3">
                <div>
                  <label for={`connection-use-${role.id}`} class="text-[13px]"
                    >{role.label}</label
                  >
                  {#if !supports(role.id)}<p
                      class="text-xs text-muted-foreground"
                    >
                      Unavailable with this backend.
                    </p>{/if}
                  {#if catalog?.selected?.[role.id] === form.id}<p
                      class="text-xs text-muted-foreground"
                    >
                      Currently selected.
                    </p>{/if}
                </div>
                <Switch
                  id={`connection-use-${role.id}`}
                  checked={form.uses.includes(role.id)}
                  onCheckedChange={(enabled) => setUse(role.id, enabled)}
                  disabled={busy ||
                    role.id === activateFor ||
                    !supports(role.id) ||
                    catalog?.selected?.[role.id] === form.id}
                  aria-label={`Use for ${role.label}`}
                />
              </div>
            {/each}
          </div>
        {/if}
      {/snippet}
      <SettingsCard>
        <ValueRow id="connection-name" label="Connection name">
          {#snippet control()}<ValueInput
              id="connection-name"
              bind:value={form.name}
              maxlength={80}
              required
              disabled={busy}
              mono={false}
              placeholder="For example, Office speech server"
            />{/snippet}
        </ValueRow>
        {#if !form.creating && managed}<ValueRow
            id="connection-target"
            label="Connection target"
          >
            {#snippet control()}
              <Select.Root
                type="single"
                value={managed ? "managed" : "manual"}
                onValueChange={setTarget}
                disabled={busy}
              >
                <Select.Trigger id="connection-target" class="w-full"
                  >{managed
                    ? "Managed local runtime"
                    : "Manual server"}</Select.Trigger
                >
                <Select.Content
                  ><Select.Item value="manual">Manual server</Select.Item
                  ><Select.Item value="managed"
                    >Managed local runtime</Select.Item
                  ></Select.Content
                >
              </Select.Root>
            {/snippet}
          </ValueRow>{/if}
        {#if managed}
          <ValueRow
            id="connection-instance"
            label="Runtime instance"
            hint="The runtime owns its model and endpoint. No URL or credentials are stored in this connection."
          >
            {#snippet control()}
              <Select.Root
                type="single"
                value={form.details.managedInstanceID ?? ""}
                onValueChange={chooseInstance}
                disabled={busy || runtime.loading}
              >
                <Select.Trigger id="connection-instance" class="w-full"
                  >{instance?.name ??
                    (form.details.managedInstanceID
                      ? "Unavailable runtime"
                      : "Choose a runtime")}</Select.Trigger
                >
                <Select.Content>
                  {#each runtime.instances as row (row.instance.id)}
                    <Select.Item value={row.instance.id}
                      >{row.instance.name} · {row.status.state ||
                        "Unavailable"}</Select.Item
                    >
                  {/each}
                  {#if form.details.managedInstanceID && !instance}<Select.Item
                      value={form.details.managedInstanceID}
                      >{form.details.managedInstanceID} · Unavailable</Select.Item
                    >{/if}
                </Select.Content>
              </Select.Root>
            {/snippet}
          </ValueRow>
          <div class="space-y-2 py-3">
            <p class="text-xs text-muted-foreground">
              {instance
                ? `Selected model: ${instance.model}. Stopping this runtime keeps the connection selected; it never falls back to a server.`
                : "This saved alias refers to an unavailable runtime. Select an existing runtime to recover it, or switch to a manual server."}
            </p>
            {#if runtime.error}<p role="alert" class="text-xs text-destructive">
                {runtime.error}
              </p>{/if}
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onclick={openRuntimes}>Manage local runtimes…</Button
            >
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={busy || runtime.loading}
              onclick={() => void runtime.load()}>Refresh instances</Button
            >
          </div>
        {:else}
          <CompatibilityProfilePicker
            id="connection-profile"
            bind:value={() => form.details.compatibilityProfile, setProfile}
            {profiles}
          />

          <ValueRow
            id="connection-url"
            label="Base URL"
            hint={form.details.compatibilityProfile === ID.WhisperCPP
              ? "Native whisper.cpp server root, without /v1 or /inference."
              : "The server’s OpenAI-compatible /v1 base URL."}
          >
            {#snippet control()}<ValueInput
                id="connection-url"
                type="url"
                bind:value={form.details.baseURL}
                required
                disabled={busy}
                placeholder="https://server.example/v1"
                spellcheck={false}
              />{/snippet}
          </ValueRow>
          {#if form.details.baseURL
            .trim()
            .toLowerCase()
            .startsWith("http://") || form.details.allowInsecureHTTP}<SettingRow
              title="Allow HTTP for this connection"
              description="Required when the server URL starts with http://."
              >{#snippet control()}<Switch
                  checked={form.details.allowInsecureHTTP}
                  onCheckedChange={(v) => {
                    if (form) form.details.allowInsecureHTTP = v;
                  }}
                  disabled={busy}
                  aria-label="Allow insecure HTTP"
                />{/snippet}</SettingRow
            >{/if}
          <ValueRow id="connection-auth" label="Authentication"
            >{#snippet control()}<Select.Root
                type="single"
                value={form.details.authenticationMode}
                onValueChange={changeAuth}
                disabled={busy}
                ><Select.Trigger id="connection-auth" class="w-full"
                  >{form.details.authenticationMode ===
                  AuthenticationMode.AuthenticationModeAPIKey
                    ? "API key"
                    : "None"}</Select.Trigger
                ><Select.Content
                  ><Select.Item
                    value={AuthenticationMode.AuthenticationModeNone}
                    >None</Select.Item
                  ><Select.Item
                    value={AuthenticationMode.AuthenticationModeAPIKey}
                    >API key</Select.Item
                  ></Select.Content
                ></Select.Root
              >{/snippet}</ValueRow
          >
          {#if form.details.authenticationMode === AuthenticationMode.AuthenticationModeAPIKey}
            <ValueRow
              id="connection-api-key"
              label="API key"
              hint={`Stored securely in ${native.credentialStore}.`}
              >{#snippet control()}<ValueInput
                  id="connection-api-key"
                  type="password"
                  autocomplete="new-password"
                  maxlength={2048}
                  bind:value={form.credentialDraft}
                  disabled={busy || form.clearCredential}
                  placeholder={form.hasCredential
                    ? "Leave blank to keep the stored key"
                    : "Enter a key if required"}
                  mono={false}
                />{/snippet}</ValueRow
            >
            {#if form.hasCredential}<SettingRow
                title="Remove stored key"
                description="Applies when you save this connection."
                >{#snippet control()}<Switch
                    checked={form.clearCredential}
                    onCheckedChange={(v) => {
                      if (form) {
                        form.clearCredential = v;
                        if (v) form.credentialDraft = "";
                      }
                    }}
                    disabled={busy}
                    aria-label="Remove stored key"
                  />{/snippet}</SettingRow
              >{/if}
          {/if}
        {/if}
        <Disclosure
          title={`Available in ${form.uses.length} ${form.uses.length === 1 ? "workflow" : "workflows"}`}
          open={!form.uses.length}
          class="border-t border-hairline"
        >
          {@render supportedUses()}
        </Disclosure>
        {#if !form.creating}<p class="py-3 text-xs text-muted-foreground">
            Changing the target or backend resets this connection’s model
            choices.
          </p>{/if}
      </SettingsCard>
      {#if chooseWorkflow && form.creating}
        <div class="flex flex-wrap items-center gap-3">
          <label for="connection-start-workflow" class="text-xs font-medium"
            >After saving</label
          >
          <div class="min-w-48 flex-1">
            <Select.Root
              type="single"
              value={activateFor ?? "save-only"}
              onValueChange={chooseUse}
              disabled={busy}
            >
              <Select.Trigger id="connection-start-workflow" class="w-full"
                >{activateFor
                  ? `Set up ${roleLabel(activateFor)}`
                  : "Save for later"}</Select.Trigger
              >
              <Select.Content>
                {#each roles.filter( (role) => supports(role.id), ) as role (role.id)}<Select.Item
                    value={role.id}>Set up {role.label}</Select.Item
                  >{/each}
                <Select.Separator /><Select.Item value="save-only"
                  >Save for later</Select.Item
                >
              </Select.Content>
            </Select.Root>
          </div>
        </div>
      {/if}
      {#if !managed && (form.uses.includes(Purpose.Transcription) || form.uses.includes(Purpose.Voice))}
        <Disclosure
          title="Transcription connection options"
          class="border-t border-hairline"
        >
          <div class="space-y-3">
            <div class="space-y-2">
              <label for="connection-health" class="text-xs font-medium"
                >Custom health path</label
              ><ValueInput
                id="connection-health"
                bind:value={form.details.healthPath}
                placeholder="Optional, for example /health"
                disabled={busy}
              />
              <p class="text-xs text-muted-foreground">
                Appended to the base URL. Leave blank for the profile’s default
                metadata route.
              </p>
            </div>
            <div class="space-y-2">
              <p class="text-xs font-medium">Custom transcription headers</p>
              {#each Object.entries(form.details.headers ?? {}) as [key, value] (key)}<div
                  class="flex items-center gap-2"
                >
                  <ValueInput
                    aria-label="Header name"
                    value={key}
                    onblur={(e) => renameHeader(key, e.currentTarget.value)}
                    disabled={busy}
                  /><ValueInput
                    aria-label={`Value for ${key}`}
                    value={value ?? ""}
                    oninput={(e) => {
                      if (form)
                        form.details.headers = {
                          ...form.details.headers,
                          [key]: e.currentTarget.value,
                        };
                    }}
                    disabled={busy}
                  /><Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onclick={() => removeHeader(key)}
                    disabled={busy}>Remove</Button
                  >
                </div>{/each}<Button
                type="button"
                variant="outline"
                size="sm"
                onclick={addHeader}
                disabled={busy ||
                  Object.keys(form.details.headers ?? {}).length >= 32}
                >Add header</Button
              >
            </div>
          </div>
        </Disclosure>
      {/if}
    </div>
    {#if error}<p role="alert" class="shrink-0 text-[13px] text-destructive">
        {error}
      </p>{/if}
    {#if !externalActions}
      <ConnectionSaveActions {editor} {activateFor} {onBack} />
    {/if}
  </form>
{/if}
