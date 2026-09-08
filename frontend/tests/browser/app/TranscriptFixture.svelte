<script lang="ts">
  import CurrentResult from "$lib/components/home/CurrentResult.svelte";
  let generation = $state(1);
  let live = $state(true);
  let count = $state(30);
  const content = $derived(
    Array.from(
      { length: count },
      (_, i) => `Line ${i + 1}: Example transcript for reading and scrolling.`,
    ).join("\n"),
  );
</script>

<div class="flex h-screen flex-col gap-3 bg-background p-4 text-foreground">
  <div class="flex gap-3">
    <button onclick={() => (count += 5)}>Append text</button>
    <button
      onclick={() => {
        count += 5;
        live = false;
      }}>Finalize</button
    >
    <button
      onclick={() => {
        generation++;
        count = 30;
        live = true;
      }}>New recording</button
    >
  </div>
  <CurrentResult
    resultKey={String(generation)}
    mode="voice"
    {live}
    liveFinal=""
    livePartial={live ? content : ""}
    text={live ? "" : content}
    working={live}
    canCopy={!live}
    onCopy={async () => true}
    onClear={() => {
      count = 0;
      generation++;
    }}
  />
</div>
