<script lang="ts">
  import { onMount } from "svelte";
  import type { Session } from "$lib/stores/session.svelte";
  import { saveConnectionAndContinue } from "$lib/shell-navigation.svelte";
  import { Action, Purpose, type Connection } from "$bindings/savedconnection";
  import {
    connectionSection,
    connectionWorkflows,
  } from "$lib/utils/connectionChoices";
  import ConnectionList from "$lib/components/settings/ConnectionList.svelte";
  import BuiltInConnectionDetails from "$lib/components/settings/BuiltInConnectionDetails.svelte";
  import * as WindowingService from "$bindings/windowing/service";
  import ConnectionSaveActions from "$lib/components/settings/ConnectionSaveActions.svelte";
  import PendingChangesDialog from "$lib/components/settings/PendingChangesDialog.svelte";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  import { connectionStatusLabel } from "$lib/utils/connection";
  import { sectionByID } from "$lib/navigation";
  import CopyIcon from "@lucide/svelte/icons/copy";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import CheckIcon from "@lucide/svelte/icons/check";
  import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  import * as Menu from "$lib/components/ui/dropdown-menu";
  import type { ConnectionManagerRequest } from "$bindings/windowing";

  import ConnectionsSection from "$lib/components/settings/sections/ConnectionsSection.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";

  let {
    session,
    initialRequest,
    onReturn,
    onManageRuntime = () => {
      void WindowingService.OpenSettings("local-runtime").catch((cause) =>
        session.messages.fail(cause),
      );
    },
    onCancelClose = () => {},
  }: {
    session: Session;
    initialRequest: ConnectionManagerRequest;
    onReturn: (purpose?: Purpose) => void;
    onManageRuntime?: () => void;
    onCancelClose?: () => void;
  } = $props();
  let request = $state<ConnectionManagerRequest | null>(null);
  let visible = $state(false);
  let loading = $state(false);
  let discardOpen = $state(false);
  let pendingAction = $state<(() => void) | null>(null);
  let selectedID = $state("");
  let activateFor = $state<Purpose | undefined>();
  let deleteOpen = $state(false);
  const editor = $derived(session.editor);
  const busy = $derived(editor.saving || editor.managedConnectionTesting);
  const selected = $derived(
    editor.applied?.savedConnections.entries?.find(
      (connection) => connection.id === selectedID,
    ),
  );
  const activeUses = $derived(
    connectionWorkflows.filter(
      (role) =>
        editor.applied?.savedConnections.selected?.[role.id] === selectedID,
    ),
  );
  let revision = 0;
  const showingDetails = $derived(
    !!editor.connectionDraft || !!selected?.builtIn,
  );
  const runtimeStatus = $derived(
    session.runtime.statusFor(selected?.details.managedInstanceID ?? ""),
  );
  const runtimeInstance = $derived(
    runtimeStatus?.instance ??
      editor.applied?.managedRuntimes?.find(
        (instance) => instance.id === selected?.details.managedInstanceID,
      ),
  );

  async function prepare() {
    request = initialRequest;
    visible = true;
    activateFor = request.create ? request.purpose || undefined : undefined;
    if (request.create) editor.beginConnection(undefined, activateFor);
    else if (request.id) {
      selectedID = request.id;
      const connection = editor.applied?.savedConnections.entries?.find(
        (c) => c.id === request?.id,
      );
      if (connection) editor.beginConnection(connection);
      else
        session.messages.reportFailure(
          "This connection is no longer available.",
        );
    }
  }
  function clear() {
    revision++;
    loading = false;
    visible = false;
    request = null;
    discardOpen = false;
    pendingAction = null;
    selectedID = "";
    activateFor = undefined;
    deleteOpen = false;
    session.editor.cancelConnectionEdit();
    session.editor.clearCredentialDraft();
  }
  async function close() {
    clear();
    onReturn();
  }
  function navigate(action: () => void) {
    if (busy) return;
    if (editor.connectionDirty) {
      pendingAction = action;
      discardOpen = true;
    } else {
      editor.cancelConnectionEdit();
      action();
    }
  }
  export function blocksReentry() {
    return (
      !!editor.connectionDraft ||
      busy ||
      loading ||
      discardOpen ||
      deleteOpen ||
      pendingAction !== null
    );
  }
  export function requestClose() {
    leave(true);
  }
  function leave(closeWindow: boolean) {
    navigate(() => {
      selectedID = "";
      activateFor = undefined;
      request = null;
      if (closeWindow) void close();
    });
  }
  function select(connection: Connection) {
    if (connection.id === selectedID && editor.connectionDraft) return;
    navigate(() => {
      request = null;
      selectedID = connection.id;
      activateFor = undefined;
      editor.beginConnection(connection);
    });
  }
  function add() {
    navigate(() => {
      request = null;
      selectedID = "";
      activateFor = undefined;
      editor.beginConnection();
    });
  }
  function discard() {
    const action = pendingAction;
    pendingAction = null;
    discardOpen = false;
    editor.cancelConnectionEdit();
    action?.();
  }
  async function saveAndContinue() {
    await saveConnectionAndContinue(
      (purpose) => editor.saveConnection(purpose),
      activateFor,
      discard,
    );
  }
  async function openWorkflow(purpose: Purpose) {
    clear();
    onReturn(purpose);
  }
  function use(purpose: Purpose) {
    const id = selectedID;
    navigate(() => {
      void (async () => {
        if (
          editor.applied?.savedConnections.selected?.[purpose] === id ||
          (await editor.changeConnection({
            action: Action.Select,
            purpose,
            id,
            name: "",
          }))
        )
          await openWorkflow(purpose);
        else if (selected) editor.beginConnection(selected);
      })();
    });
  }
  function saved(purpose?: Purpose, id?: string) {
    if (id) selectedID = id;
    if (purpose) void openWorkflow(purpose);
    else if (initialRequest.purpose) {
      clear();
      onReturn();
    } else if (selected) editor.beginConnection(selected);
  }
  function duplicate() {
    if (!selected || selected.builtIn) return;
    const connection = selected;
    navigate(() => {
      let prefix = connection.name;
      while (new TextEncoder().encode(prefix).length > 60)
        prefix = [...prefix].slice(0, -1).join("");
      let n = 1,
        name = `${prefix} copy`;
      while (
        editor.applied?.savedConnections.entries?.some(
          (entry) => entry.name.toLowerCase() === name.toLowerCase(),
        )
      )
        name = `${prefix} copy ${++n}`;
      void (async () => {
        if (
          !(await editor.changeConnection({
            action: Action.Duplicate,
            id: connection.id,
            name,
          }))
        ) {
          if (selected) editor.beginConnection(selected);
          return;
        }
        const copy = editor.applied?.savedConnections.entries?.find(
          (entry) => entry.name === name,
        );
        if (copy) {
          selectedID = copy.id;
          activateFor = undefined;
          editor.beginConnection(copy);
        }
      })();
    });
  }
  async function remove() {
    if (
      !selected ||
      selected.builtIn ||
      activeUses.length ||
      editor.connectionDirty
    )
      return;
    const id = selected.id;
    editor.cancelConnectionEdit();
    if (
      await editor.changeConnection({ action: Action.Delete, id, name: "" })
    ) {
      selectedID = "";
      deleteOpen = false;
    } else if (selected) editor.beginConnection(selected);
  }
  onMount(() => {
    void prepare();
    void session.runtime.load();
    return clear;
  });
</script>

<svelte:window
  onkeydown={(event) => {
    if (
      event.key === "Escape" &&
      !event.defaultPrevented &&
      !discardOpen &&
      !deleteOpen
    ) {
      event.preventDefault();
      leave(true);
    }
  }}
/>

<div
  class="flex min-h-0 flex-1 flex-col overflow-hidden bg-transparent text-foreground"
>
  <header
    class="flex shrink-0 items-center justify-between gap-4 border-b border-hairline px-4 py-3"
  >
    <div class="flex min-w-0 items-center gap-3">
      {#if showingDetails}<Button
          variant="ghost"
          size="sm"
          disabled={busy}
          onclick={() => leave(false)}><ArrowLeftIcon />All connections</Button
        >{/if}
      <div class="min-w-0">
        <h1 class="truncate text-lg font-semibold">Connections</h1>
        <p class="mt-0.5 text-xs text-secondary-foreground">
          Saved servers and built-in runtimes
        </p>
      </div>
    </div>
    <Button variant="outline" disabled={busy} onclick={() => leave(true)}
      >Done</Button
    >
  </header>
  {#if loading}<p class="p-5 text-sm text-muted-foreground">
      Loading connections…
    </p>
  {:else if visible && editor.applied}
    {#if !editor.connectionDraft && session.messages.error}<p
        role="alert"
        class="px-4 py-2 text-sm text-destructive"
      >
        {session.messages.error}
      </p>{/if}
    <div class="manager-body" class:editing={showingDetails}>
      <aside class="connection-list bg-background">
        <ConnectionList
          catalog={editor.applied.savedConnections}
          instances={session.runtime.instances}
          selected={selectedID}
          creating={editor.connectionDraft?.creating}
          {busy}
          onSelect={select}
          onAdd={add}
        />
      </aside>
      {#if showingDetails}<section
          aria-label={selected?.builtIn
            ? "Connection details"
            : "Connection editor"}
          class="flex min-h-0 min-w-0 flex-1 flex-col"
        >
          <div
            class="flex shrink-0 items-center justify-between gap-3 border-b border-hairline px-5 py-3"
          >
            <h2 class="truncate text-sm font-semibold">
              {editor.connectionDraft?.creating
                ? "New connection"
                : (selected?.name ?? "Edit connection")}
            </h2>
            {#if selected}<div class="flex items-center gap-1">
                <Menu.Root
                  ><Menu.Trigger disabled={busy}>
                    {#snippet child({ props })}<Button
                        {...props}
                        size="sm"
                        variant="soft">Use for…</Button
                      >{/snippet}
                  </Menu.Trigger><Menu.Content
                    align="end"
                    class="w-80 max-w-[calc(100vw-24px)] p-1.5"
                  >
                    {#each connectionWorkflows.filter( (role) => selected?.uses?.includes(role.id) ) as role (role.id)}
                      {@const Icon = sectionByID(role.section).icon}
                      {@const current =
                        editor.applied.savedConnections.selected?.[role.id] ===
                        selected.id}
                      <Menu.Item
                        onSelect={() => use(role.id)}
                        class="gap-3 rounded-md px-3 py-2.5"
                      >
                        <Icon />
                        <span class="flex-1 whitespace-nowrap"
                          >{role.label}</span
                        >
                        <span
                          class="flex size-4 shrink-0 items-center justify-center text-primary"
                        >
                          {#if current}<CheckIcon /><span class="sr-only"
                              >Current connection</span
                            >{/if}
                        </span>
                      </Menu.Item>
                    {/each}
                  </Menu.Content></Menu.Root
                >
                {#if !selected.builtIn}<Menu.Root
                    ><Menu.Trigger disabled={busy}>
                      {#snippet child({ props })}<Button
                          {...props}
                          size="icon-sm"
                          variant="ghost"
                          aria-label="Connection actions"
                          ><EllipsisIcon /></Button
                        >{/snippet}
                    </Menu.Trigger><Menu.Content
                      align="end"
                      class="w-56 max-w-[calc(100vw-24px)] p-1.5"
                    >
                      <Menu.Item
                        class="gap-3 rounded-md px-3 py-2.5"
                        disabled={(editor.applied.savedConnections.entries
                          ?.length ?? 0) >= 96}
                        onSelect={duplicate}><CopyIcon />Duplicate</Menu.Item
                      >
                      <Menu.Separator />
                      <Menu.Item
                        class="gap-3 rounded-md px-3 py-2.5"
                        variant="destructive"
                        disabled={!!activeUses.length || editor.connectionDirty}
                        onSelect={() => {
                          deleteOpen = true;
                        }}
                      >
                        <Trash2Icon /><span class="flex-1">Delete</span>
                        {#if activeUses.length}<span
                            class="text-xs text-muted-foreground">In use</span
                          >
                          <span class="sr-only"
                            >Choose another connection in its active workflows
                            before deleting.</span
                          >
                        {:else if editor.connectionDirty}<span
                            class="text-xs text-muted-foreground"
                            >Unsaved edits</span
                          >{/if}
                      </Menu.Item>
                    </Menu.Content></Menu.Root
                  >{/if}
              </div>{/if}
          </div>
          <main
            class="connection-fields space-y-4 min-h-0 flex-1 overflow-y-auto overscroll-contain p-4"
          >
            {#if selected?.builtIn}
              <BuiltInConnectionDetails
                connection={selected}
                instance={runtimeInstance}
                status={runtimeStatus}
                providers={session.runtime.providers}
                {busy}
                {onManageRuntime}
                onWorkflow={(purpose) => void openWorkflow(purpose)}
              />
            {:else}<ConnectionsSection
                {editor}
                runtime={session.runtime}
                bind:activateFor
                chooseWorkflow={!request?.purpose || !request.create}
                formID="connection-editor"
                externalActions
                error={session.messages.error}
                onBack={() => leave(false)}
                onSaved={saved}
              />{/if}
            {#if selected}<details
                class="rounded-xl border border-hairline bg-card p-3"
              >
                <summary
                  class="cursor-pointer rounded-sm text-xs font-medium text-accent-text focus-visible:outline-ring"
                  >Connection check · {connectionStatusLabel(
                    editor.savedConnectionChecks[selected.id] ?? null,
                  )}</summary
                >
                <div class="mt-3 space-y-3">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={busy}
                    onclick={() => editor.testSavedConnection(selected.id)}
                    >{editor.managedConnectionTesting
                      ? "Checking…"
                      : "Check connection"}</Button
                  >
                  <p class="text-xs text-muted-foreground">
                    Checks metadata without running a model.
                  </p>
                  {#if editor.savedConnectionCheckErrors[selected.id]}<p
                      role="alert"
                      class="text-xs text-destructive"
                    >
                      {editor.savedConnectionCheckErrors[selected.id]}
                    </p>{/if}
                  {#if editor.savedConnectionChecks[selected.id]}<ConnectionDiagnostics
                      result={editor.savedConnectionChecks[selected.id]}
                    />{/if}
                </div>
              </details>{/if}
          </main>
          {#if !selected?.builtIn}<footer
              class="shrink-0 border-t border-hairline bg-layer-fill px-4 py-3"
            >
              <ConnectionSaveActions
                {editor}
                {activateFor}
                formID="connection-editor"
                onBack={() => leave(false)}
              />
            </footer>{/if}
        </section>{/if}
    </div>
  {:else if session.messages.error}<div class="p-5">
      <p role="alert" class="text-sm text-destructive">
        {session.messages.error}
      </p>
      <Button class="mt-3" variant="outline" onclick={prepare}>Try again</Button
      >
    </div>{/if}
</div>
<PendingChangesDialog
  open={discardOpen}
  {busy}
  title="Save connection changes?"
  description="Your edits will be kept until you save or discard them."
  error={session.messages.error}
  errorClass="text-sm text-destructive"
  onKeepEditing={() => {
    discardOpen = false;
    pendingAction = null;
    onCancelClose();
  }}
  onDiscard={discard}
  onSave={saveAndContinue}
/>
<Dialog.Root bind:open={deleteOpen}>
  <Dialog.Content
    ><Dialog.Header
      ><Dialog.Title>Delete connection?</Dialog.Title><Dialog.Description
        >Remove “{selected?.name}” and its unused stored credential.</Dialog.Description
      ></Dialog.Header
    >
    {#if session.messages.error}<p
        role="alert"
        class="text-sm text-destructive"
      >
        {session.messages.error}
      </p>{/if}
    <Dialog.Footer
      ><Button
        variant="outline"
        disabled={busy}
        onclick={() => {
          deleteOpen = false;
        }}>Cancel</Button
      ><Button variant="destructive" disabled={busy} onclick={remove}
        >Delete connection</Button
      ></Dialog.Footer
    >
  </Dialog.Content>
</Dialog.Root>

<style>
  .manager-body {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }
  .connection-list {
    width: 100%;
    min-height: 0;
  }
  .editing .connection-list {
    display: none;
  }
  .connection-fields :global([role="group"]) {
    padding-top: 0.65rem;
    padding-bottom: 0.65rem;
    gap: 0.4rem;
  }
  @media (min-width: 760px) {
    .editing .connection-list {
      display: block;
      width: 270px;
      flex-shrink: 0;
      border-right: 1px solid var(--hairline);
    }
  }
</style>
