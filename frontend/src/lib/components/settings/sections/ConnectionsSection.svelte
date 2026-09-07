<script lang="ts">
  import { Action, Purpose, type Connection } from "$bindings/savedconnection";
  import { ID } from "$bindings/compatibility";
  import { AuthenticationMode } from "$lib/state";
  import type { SettingsEditor } from "$lib/stores/editor.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import CompatibilityProfilePicker from "$lib/components/settings/CompatibilityProfilePicker.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Badge } from "$lib/components/ui/badge";
  import { Switch } from "$lib/components/ui/switch";
  import * as Select from "$lib/components/ui/select";
  import * as Dialog from "$lib/components/ui/dialog";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import ConnectionSaveActions from "$lib/components/settings/ConnectionSaveActions.svelte";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";

  let {
    editor,
    activateFor,
    formID,
    externalActions = false,
    error,
    onOpenFeature,
    onBack,
    onSaved,
  }: {
    editor: SettingsEditor;
    activateFor?: Purpose;
    formID?: string;
    externalActions?: boolean;
    error: string;
    onBack: () => void;
    onSaved: () => void;
    onOpenFeature: (purpose: Purpose) => void;
  } = $props();
  const form = $derived(editor.connectionDraft);
  const catalog = $derived(editor.applied?.savedConnections);
  const entries = $derived(catalog?.entries ?? []);
  const busy = $derived(editor.saving || editor.managedConnectionTesting);
  const roles = [
    { id: Purpose.Transcription, label: "Transcription" },
    { id: Purpose.Cleanup, label: "Cleanup" },
    { id: Purpose.Speech, label: "Text to speech" },
  ];
  const roleLabel = (p: Purpose) =>
    roles.find((r) => r.id === p)?.label ?? "Connection";
  const profiles = $derived.by(() => {
    const catalog = editor.applied?.compatibilityProfiles;
    const all = activateFor
      ? ((activateFor === Purpose.Transcription
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
        (!activateFor || supports(activateFor, p.id)),
      description: all.some((x) => x.id === p.id && x.available)
        ? "Uses this backend’s implemented contracts for the features enabled below."
        : p.description,
    }));
  });
  function supports(
    purpose: Purpose,
    profile = form?.details.compatibilityProfile,
  ) {
    const catalog = editor.applied?.compatibilityProfiles;
    const list =
      purpose === Purpose.Transcription
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
    if (!form.uses.includes(Purpose.Transcription)) {
      form.details.healthPath = "";
      form.details.headers = {};
    }
  }
  function setUse(p: Purpose, enabled: boolean) {
    if (!form) return;
    form.uses = enabled ? [...form.uses, p] : form.uses.filter((x) => x !== p);
    if (!form.uses.includes(Purpose.Transcription)) {
      form.details.healthPath = "";
      form.details.headers = {};
    }
  }
  const activeUses = (id: string) =>
    roles.filter((role) => catalog?.selected?.[role.id] === id);
  let deleting = $state<Connection | null>(null);
  let deleteOpen = $state(false);
  let testedID = $state("");
  function changeAuth(value: string) {
    if (!form) return;
    form.details.authenticationMode = value as AuthenticationMode;
    if (value === AuthenticationMode.AuthenticationModeNone)
      form.credentialDraft = "";
  }
  function duplicate(c: Connection) {
    let prefix = c.name;
    while (new TextEncoder().encode(prefix).length > 60)
      prefix = [...prefix].slice(0, -1).join("");
    let n = 1,
      name = `${prefix} copy`;
    while (entries.some((e) => e.name.toLowerCase() === name.toLowerCase()))
      name = `${prefix} copy ${++n}`;
    void editor.changeConnection({
      action: Action.Duplicate,
      id: c.id,
      name,
    });
  }
  async function remove() {
    if (!deleting) return;
    if (
      await editor.changeConnection({
        action: Action.Delete,
        id: deleting.id,
        name: "",
      })
    ) {
      deleteOpen = false;
      deleting = null;
    }
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
</script>

{#if !activateFor}
  <p class="text-xs leading-relaxed text-muted-foreground">
    Save servers here, then select them in a task. Creating or duplicating a
    connection in this library does not activate it.
  </p>
{/if}
{#if form}
  <form
    id={formID}
    onsubmit={async (event) => {
      event.preventDefault();
      if (await editor.saveConnection(activateFor)) onSaved();
    }}
    class="flex min-h-0 flex-col gap-3.5"
  >
    <div
      class="flex min-h-0 flex-col gap-3.5"
      class:overflow-y-auto={activateFor !== undefined}
    >
      {#if !activateFor}
        <div class="flex items-center justify-between">
          <h4 class="text-sm font-semibold">
            {form.creating
              ? "New connection"
              : `Edit ${form.name || "connection"}`}
          </h4>
          <Badge variant="outline">Connection settings</Badge>
        </div>
      {/if}
      {#if !form.creating && activeUses(form.id).length > 0}<p
          class="text-xs text-muted-foreground"
        >
          This connection is in use. Saving updates its connection details for
          new requests; model and feature options stay on their own feature
          pages. This updates every feature using this server.
        </p>{/if}
      {#snippet supportedUses()}
        {#if form}
          <div class="space-y-3 px-5 py-4">
            <h4 class="text-sm font-medium">Used for</h4>
            <p class="text-xs text-muted-foreground">
              Enable only the operations your deployed server provides. Each use
              becomes selectable independently; enabling it here does not
              activate it or verify inference support.
            </p>
            {#each roles as role (role.id)}
              <div class="flex items-center justify-between gap-3">
                <div>
                  <label for={`connection-use-${role.id}`} class="text-sm"
                    >{role.label}</label
                  >
                  {#if !supports(role.id)}<p
                      class="text-xs text-muted-foreground"
                    >
                      Not implemented for this profile.
                    </p>{/if}
                  {#if catalog?.selected?.[role.id] === form.id}<p
                      class="text-xs text-muted-foreground"
                    >
                      In use. Deselect on its feature page before removing.
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
        <ValueRow
          id="connection-name"
          label="Connection name"
          hint="A name you will recognize in the feature selectors."
        >
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
        <CompatibilityProfilePicker
          id="connection-profile"
          bind:value={() => form.details.compatibilityProfile, setProfile}
          {profiles}
        />

        {#if !activateFor}{@render supportedUses()}{/if}
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
        {#if !form.creating}<p
            class="px-5 pb-3 text-xs leading-relaxed text-muted-foreground"
          >
            Changing the base URL or backend profile clears this connection’s
            remembered models and active model choices. Renames and
            authentication changes keep them.
          </p>{/if}
        <SettingRow
          title="Allow insecure HTTP"
          description="Required for an HTTP endpoint, including localhost. HTTPS keeps credentials and requests encrypted in transit."
          >{#snippet control()}<Switch
              checked={form.details.allowInsecureHTTP}
              onCheckedChange={(v) => {
                if (form) form.details.allowInsecureHTTP = v;
              }}
              disabled={busy}
              aria-label="Allow insecure HTTP"
            />{/snippet}</SettingRow
        >
        <ValueRow
          id="connection-auth"
          label="Authentication"
          hint="Choose whether this connection sends an API key."
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
                ><Select.Item value={AuthenticationMode.AuthenticationModeNone}
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
            hint="One stored key shared by this server’s enabled uses. Never returned to this window."
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
        {#if activateFor}
          <details class="border-t border-hairline">
            <summary class="cursor-pointer px-5 py-3 text-sm font-medium"
              >Also use this server for other tasks</summary
            >
            {@render supportedUses()}
          </details>
        {/if}
      </SettingsCard>
      {#if form.uses.includes(Purpose.Transcription)}
        <details class="rounded-xl border border-hairline bg-layer-fill p-4">
          <summary class="cursor-pointer text-sm font-medium"
            >Transcription connection options</summary
          >
          <div class="mt-4 space-y-4">
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
                  class="flex gap-2"
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
        </details>
      {/if}
    </div>
    {#if error}<p role="alert" class="shrink-0 text-sm text-destructive">
        {error}
      </p>{/if}
    {#if !externalActions}
      <ConnectionSaveActions {editor} {activateFor} {onBack} />
    {/if}
  </form>
{:else}
  <div class="flex justify-end">
    <Button
      disabled={busy || editor.runtimeDirty}
      onclick={() => editor.beginConnection()}
      ><PlusIcon />New connection</Button
    >
  </div>
  {#if entries.length === 0}<SettingsCard
      ><div class="space-y-2 px-5 py-6">
        <h4 class="text-sm font-semibold">No connections yet</h4>
        <p class="text-xs text-muted-foreground">
          Create your first connection with a name, server URL, and
          compatibility profile. Model and language choices come next, on the
          feature page.
        </p>
      </div></SettingsCard
    >{/if}
  {#each entries as c (c.id)}
    <SettingsCard
      ><div class="space-y-3 px-5 py-4">
        <div class="flex items-start gap-3">
          <ProviderIcon profile={c.details.compatibilityProfile} />
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-semibold">{c.name}</p>
            <p class="break-all text-xs text-muted-foreground">
              {c.details.baseURL}
            </p>
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          {#each c.uses as p (p)}<Badge
              variant={catalog?.selected?.[p] === c.id
                ? "secondary"
                : "outline"}
              >{roleLabel(p)}{catalog?.selected?.[p] === c.id
                ? " · In use"
                : ""}</Badge
            >{/each}
        </div>
        <div class="flex flex-wrap gap-1.5">
          <Button
            variant="secondary"
            size="sm"
            disabled={busy || editor.runtimeDirty}
            onclick={() => editor.beginConnection(c)}>Edit</Button
          >
          <Button
            variant="ghost"
            size="sm"
            disabled={busy || editor.runtimeDirty || entries.length >= 96}
            onclick={() => duplicate(c)}>Duplicate</Button
          >
          <Button
            variant="ghost"
            size="sm"
            disabled={busy}
            onclick={() => {
              testedID = c.id;
              void editor.testSavedConnection(c.id);
            }}>Test connection</Button
          >
          <Button
            variant="ghost"
            size="sm"
            disabled={busy ||
              editor.runtimeDirty ||
              activeUses(c.id).length > 0}
            onclick={() => {
              deleting = c;
              deleteOpen = true;
            }}>Delete</Button
          >
        </div>
        <div class="flex flex-wrap gap-1">
          {#each c.uses as p (p)}<Button
              variant="link"
              size="sm"
              onclick={() => onOpenFeature(p)}>Open {roleLabel(p)}</Button
            >{/each}
        </div>
        {#if activeUses(c.id).length}<p
            class="text-[11px] text-muted-foreground"
          >
            To delete, choose another connection or None in every feature using
            this server.
          </p>{/if}
        {#if testedID === c.id && editor.managedConnectionTesting}<p
            role="status"
            class="text-xs text-muted-foreground"
          >
            Checking metadata…
          </p>
        {:else if testedID === c.id && editor.managedConnectionResult}<div
            class="border-t border-hairline pt-4"
          >
            <ConnectionDiagnostics result={editor.managedConnectionResult} />
          </div>{/if}
      </div></SettingsCard
    >
  {/each}
  <p class="text-xs text-muted-foreground">
    Tests read health or model-list metadata only. No model is started or
    invoked.
  </p>
{/if}
<Dialog.Root bind:open={deleteOpen}
  ><Dialog.Content
    ><Dialog.Header
      ><Dialog.Title>Delete connection</Dialog.Title><Dialog.Description
        >Delete “{deleting?.name}”? It is not selected by a feature. A key is
        removed only when no saved connection uses it.</Dialog.Description
      ></Dialog.Header
    >{#if error}<p role="alert" class="text-sm text-destructive">
        {error}
      </p>{/if}<Dialog.Footer
      ><Button
        variant="outline"
        disabled={editor.saving}
        onclick={() => (deleteOpen = false)}>Cancel</Button
      ><Button variant="destructive" disabled={editor.saving} onclick={remove}
        >Delete connection</Button
      ></Dialog.Footer
    ></Dialog.Content
  ></Dialog.Root
>
