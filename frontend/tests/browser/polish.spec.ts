import { test, expect } from "./fixtures";

test("speech quick controls save a voice, recover from failure, and retain the composer", async ({
  page,
  saves,
}) => {
  await page.setViewportSize({ width: 650, height: 550 });
  await page.goto("/tests/browser/app/?view=workspace&theme=dark");
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
  await page.getByRole("textbox", { name: "Text to speak", exact: true }).fill("Keep this draft.");
  await page.getByRole("button", { name: "Speech settings", exact: true }).click();
  const voice = page.getByRole("combobox", { name: "Choose voice" });
  await voice.fill("custom-voice");
  await page.keyboard.press("Enter");
  await saves.complete(await saves.waitForStart(), "failure");
  await expect(
    page.getByText(
      "Could not save. Your previous settings are still active. Try the change again.",
    ),
  ).toBeVisible();
  await voice.fill("custom-voice");
  await page.keyboard.press("Enter");
  await saves.complete(await saves.waitForStart(), "success");
  await expect(voice).toHaveValue("custom-voice");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("button", { name: "Speech settings", exact: true })).toBeFocused();
  await expect(page.getByRole("textbox", { name: "Text to speak", exact: true })).toHaveValue(
    "Keep this draft.",
  );
});

test("live transcript follows new text, pauses for reading, and resumes explicitly", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1156, height: 600 });
  await page.goto("/tests/browser/app/?view=transcript");
  const scroll = page.getByRole("region", { name: "Current result" }).locator(".overflow-y-auto");
  const gap = () => scroll.evaluate((el) => el.scrollHeight - el.clientHeight - el.scrollTop);
  await expect.poll(gap).toBeLessThan(2);
  await page.getByRole("button", { name: "Append text" }).click();
  await expect.poll(gap).toBeLessThan(2);
  await scroll.hover();
  await page.mouse.wheel(0, -500);
  await expect(page.getByRole("button", { name: "Jump to latest" })).toBeVisible();
  const top = await scroll.evaluate((el) => el.scrollTop);
  await page.getByRole("button", { name: "Append text" }).click();
  await expect.poll(() => scroll.evaluate((el) => el.scrollTop)).toBe(top);
  await page.getByRole("button", { name: "Finalize" }).click();
  await expect(page.getByRole("textbox", { name: "Current transcript" })).toBeVisible();
  await expect.poll(() => scroll.evaluate((el) => el.scrollTop)).toBe(top);
  await page.getByRole("button", { name: "Jump to latest" }).click();
  await expect.poll(gap).toBeLessThan(2);
  await expect(page.getByRole("button", { name: "Jump to latest" })).toHaveCount(0);
  await page.getByRole("button", { name: "New recording" }).click();
  await expect.poll(gap).toBeLessThan(2);
});

test("speech quick model restores its voice and speed commits roll back on failure", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?view=workspace");
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
  await page.getByRole("button", { name: "Speech settings", exact: true }).click();
  const model = page.getByRole("combobox", { name: "Choose model" });
  await model.fill("speech/alternate");
  await page.keyboard.press("Enter");
  await saves.complete(await saves.waitForStart(), "success");
  await expect(model).toHaveValue("speech/alternate");
  await expect(page.getByRole("combobox", { name: "Choose voice" })).toHaveValue("Alternate voice");
  const slider = page.getByRole("slider");
  const before = Number(await slider.getAttribute("aria-valuenow"));
  await slider.focus();
  await page.keyboard.press("ArrowRight");
  await saves.complete(await saves.waitForStart(), "failure");
  await expect(slider).toHaveAttribute("aria-valuenow", String(before));
  await slider.focus();
  await page.keyboard.press("ArrowRight");
  await saves.complete(await saves.waitForStart(), "success");
  await expect(slider).toHaveAttribute("aria-valuenow", String(before + 0.05));
});
