import { test, expect } from "./fixtures";

for (const width of [1280, 560]) {
  for (const theme of ["light", "dark"] as const) {
    test(`page headers keep their geometry across navigation at ${width}px in ${theme}`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 820 });
      await page.emulateMedia({ colorScheme: theme });
      await page.goto(
        `/tests/browser/app/?main&setup-ready&history-workbench&runtime&runtime-ready&pickers&theme=${theme}`,
      );
      const navigation = page.getByRole("navigation", {
        name: "Workspace",
        exact: true,
      });
      // Exclude contextual sidebar headings and the mounted, hidden workflow.
      const header = page.locator(
        ".workbench-columns > div [data-page-header]:visible",
      );
      let baseline:
        | { top: number; titleInset: number; titleTop: number }
        | undefined;
      for (const name of [
        "Voice transcription",
        "Audio file",
        "Text to speech",
        "Connections",
        "Local runtime",
        "History",
        "Settings",
        "Voice transcription",
      ]) {
        await navigation.getByRole("button", { name, exact: true }).click();
        await expect(
          navigation.getByRole("button", { name, exact: true }),
        ).toHaveAttribute("aria-current", "page");
        await expect(header, `${name} has one page header`).toHaveCount(1);
        const title = header.getByRole("heading").first();
        await expect(title).toBeInViewport();
        await expect(title).toHaveCSS("font-size", "16px");
        await expect(title).toHaveCSS("font-weight", "600");
        // Settings keeps its existing opaque background while the header sticks.
        if (name !== "Settings") {
          await expect(header).toHaveCSS(
            "background-color",
            "rgba(0, 0, 0, 0)",
          );
        }

        const bounds = (await header.boundingBox())!;
        const titleBounds = (await title.boundingBox())!;
        baseline ??= {
          top: bounds.y,
          titleInset: titleBounds.x - bounds.x,
          titleTop: titleBounds.y - bounds.y,
        };
        expect(bounds.height, `${name} header height`).toBe(44);
        expect(bounds.y, `${name} top edge`).toBe(baseline.top);
        expect(titleBounds.x - bounds.x, `${name} title inset`).toBeCloseTo(
          baseline.titleInset,
          0,
        );
        expect(titleBounds.y - bounds.y, `${name} title baseline`).toBeCloseTo(
          baseline.titleTop,
          0,
        );
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
          `${name} document fits`,
        ).toBe(true);
      }
      await page.screenshot({
        path: info.outputPath(`page-headers-${width}-${theme}.png`),
      });
    });
  }
}
