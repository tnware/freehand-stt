import { test, expect } from "./fixtures";

for (const width of [560, 1156]) {
  for (const source of ["voice", "file", "history"]) {
    test(`${source} Listen responds immediately and recovers at ${width}px`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 680 });
      await page.goto("/tests/browser/app/?view=workspace&history=expansion&listen-pending");
      if (source === "file")
        await page.getByRole("button", { name: "Show file result", exact: true }).click();
      if (source === "history" && width < 900)
        await page.getByRole("button", { name: "History · 2", exact: true }).click();
      const target =
        source === "history"
          ? page.locator(".history-entry").first()
          : page.getByRole("region", { name: "Current result", exact: true });
      const listen = target.getByRole("button", {
        name: source === "history" ? "Listen to transcript" : "Listen",
        exact: true,
      });
      const before = (await listen.boundingBox())!;
      await listen.click();
      const pending = target.getByRole("button", {
        name: "Preparing speech for this transcript",
        exact: true,
      });
      await expect(pending).toBeDisabled();
      expect((await pending.boundingBox())!.width).toBe(before.width);
      await pending.evaluate((button: HTMLButtonElement) => button.click());
      await expect(page.getByText("Listen requests: 1", { exact: true })).toBeVisible();
      await expect(target.getByRole("textbox").first()).toBeVisible();
      const copy =
        source === "history"
          ? target.getByRole("button", { name: "Copy transcript", exact: true })
          : target.getByRole("button", { name: "Copy", exact: true });
      await expect(copy).toBeEnabled();
      await page.screenshot({ path: info.outputPath(`listen-${source}-${width}.png`) });
      await page.getByRole("button", { name: "Reject listen", exact: true }).click();
      await expect(listen).toBeEnabled();
      await listen.click();
      await expect(page.getByText("Listen requests: 2", { exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Accept listen", exact: true }).click();
      await expect(pending).toBeDisabled();
      for (const other of await page
        .getByRole("button", { name: /^Listen(?: to transcript| to audio file transcript)?$/ })
        .all()) {
        await expect(other).toBeDisabled();
      }
      await page.getByRole("button", { name: "Finish generation", exact: true }).click();
      await expect(listen).toBeEnabled();
      await expect(page.getByText("Listen requests: 2", { exact: true })).toBeVisible();
    });
  }
}
