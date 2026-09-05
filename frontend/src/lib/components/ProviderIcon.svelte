<script lang="ts">
  import { providerIdentity } from "../../../../branding/providers";
  import "../../../../branding/providers/provider-icon.css";
  let { profile, size = 24 }: { profile?: string; size?: number } = $props();
  const identity = $derived(providerIdentity(profile));
</script>

<span
  class="provider-icon"
  style={`--provider-icon-size: ${size}px`}
  aria-hidden="true"
  data-provider={profile || "generic"}
>
  <img src={identity.src} alt="" width={size} height={size} draggable="false" />
</span>
