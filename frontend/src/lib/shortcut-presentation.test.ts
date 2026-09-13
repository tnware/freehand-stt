import { render } from "svelte/server";
import { expect, it } from "vitest";
import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";

it("exposes spoken shortcut names once and hides visual glyphs", () => {
  const { body } = render(ShortcutKeys, {
    props: { value: "Cmd+Alt+Ctrl+Shift+Space", platform: "darwin" },
  });
  expect(body).toContain('role="img"');
  expect(body).toContain(
    'aria-label="Keyboard shortcut: Command plus Option plus Control plus Shift plus Space"',
  );
  expect(body.match(/aria-hidden="true"/g)?.length).toBe(9);
  for (const glyph of ["⌘", "⌥", "⌃", "⇧"]) expect(body).toContain(glyph);
});
