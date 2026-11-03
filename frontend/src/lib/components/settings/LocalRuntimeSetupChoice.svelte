<script lang="ts">
  import type { ProviderDescriptor } from "$bindings/managedruntime";
  import type { Role } from "$bindings/compatibility";
  import { Button } from "$lib/components/ui/button";

  let {
    providers,
    role,
    label,
    disabled = false,
    onOpen,
  }: {
    providers: ProviderDescriptor[];
    role: Role;
    label: string;
    disabled?: boolean;
    onOpen: (providerID?: string) => void;
  } = $props();
  const provider = $derived(
    providers.find(
      (provider) =>
        provider.supported &&
        provider.models?.some((model) =>
          model.contracts?.some((contract) => contract.role === role),
        ),
    ),
  );
  const models = $derived(
    provider?.models?.filter((model) =>
      model.contracts?.some((contract) => contract.role === role),
    ) ?? [],
  );
  const model = $derived(
    models.find((model) => model.recommended) ?? models[0],
  );
</script>

<section
  class="space-y-2 border-t border-hairline py-3"
  aria-label={`Local ${label} setup`}
>
  <h3 class="content-section-title">
    {provider ? `Run ${label} locally` : `Local ${label}`}
  </h3>
  <p class="content-meta">
    {provider && model
      ? `${model.name} runs with ${provider.name}. After setup, choose its built-in connection for ${label}.`
      : providers.length
        ? `Local ${label} is unavailable on this platform. Use a compatible server connection.`
        : "Browse local engines to check available models and platform support."}
  </p>
  {#if provider || !providers.length}
    <Button
      variant="outline"
      size="sm"
      {disabled}
      onclick={() => onOpen(provider?.id)}
    >
      {provider ? `Set up local ${label}` : "Browse local engines"}
    </Button>
    <p class="content-meta">
      Setup is explicit. Opening Local runtime does not download models or
      enable this feature.
    </p>
  {/if}
</section>
