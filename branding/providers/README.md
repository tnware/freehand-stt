# Provider identity assets

This is the single vendored SVG collection for the Svelte app and Astro site,
including the backend documentation. `manifest.json` records each source,
pinned revision, license notice, and whether the image is a brand mark, family
mark, or Freehand's own neutral symbol. Neutral symbols are not official logos.
Generic is a protocol, so it never uses OpenAI's brand mark.

`index.ts` resolves profile IDs through the manifest to statically imported local
assets. Both `ProviderIcon` components use it and `provider-icon.css`. Bundlers
package these files locally; there is no runtime CDN, provider request, or icon
package dependency. Unknown IDs receive the Generic symbol. Presentation never
enables a profile or grants a capability; the Go compatibility catalog owns that.

Use an icon next to a visible provider name. Decorative icons have empty alt
text; availability and connection status remain explicit text and controls.
Preserve the source viewBox and proportions inside the shared square tile. The
light tile keeps original dark and color marks legible in both app/site themes.
Do not recolor a brand mark to represent success, failure, or planned support.

To add or update an asset:

1. Find the project's own mark or a licensed icon collection. Pin the source
   revision and retain the full license notice. Record any modifications.
2. Vendor a self-contained SVG. Inspect it for scripts, event handlers, external
   resources, embedded images, and editor metadata. Preserve the artwork;
   use a documented neutral fallback when provenance or reuse is unclear.
3. Add the manifest entry and static import in `index.ts`. Shared marks (such
   as vLLM and vLLM-Omni) reference one file. Never fetch an icon from user input.
4. Add new third-party notices to `notices.ts`, which ships in About and the
   documentation's provider asset credits. Brand names and marks belong to
   their respective owners; inclusion identifies an integration, not endorsement.
5. Check app selectors, quick settings, backend cards/table, docs headings,
   and both light/dark and narrow layouts. Run Svelte checks and both builds.

Original neutral symbols are covered by Freehand's root MIT license. The
third-party license notices in `licenses/` apply to the vendored brand artwork;
trademark rights remain with the respective owners.
