import type { Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import { test, expect } from "./fixtures";

const browser = (page: Page) =>
  page.getByRole("complementary", { name: "History browser", exact: true });
const transcripts = (page: Page) =>
  browser(page).getByRole("button", {
    name: /^View (Voice|Audio file) transcript from /,
  });
const transcript = (page: Page, text: string) =>
  transcripts(page).filter({ hasText: text });
const reader = (page: Page) =>
  page.getByRole("region", { name: "Selected transcript", exact: true });
const information = (page: Page) =>
  page.getByRole("region", { name: "Run information", exact: true });

function observeNativeDetailsCalls(page: Page) {
  const bindings = readFileSync(
    new URL(
      "../../bindings/github.com/tnware/freehand-stt/internal/history/service.ts",
      import.meta.url,
    ),
    "utf8",
  );
  const methods = new Set(
    ["OpenDetails", "CurrentDetails", "CloseDetails"].map((method) =>
      Number(
        bindings.match(
          new RegExp(`function ${method}\\([^]*?ByID\\((\\d+)`),
        )![1],
      ),
    ),
  );
  let calls = 0;
  page.on("request", (request) => {
    if (
      !request.url().endsWith("/wails/runtime") ||
      request.method() !== "POST"
    )
      return;
    if (methods.has(request.postDataJSON()?.args?.methodID)) calls++;
  });
  return () => calls;
}

async function openHistory(page: Page) {
  await page.goto(
    "/tests/browser/app/?main&setup-ready&history-workbench&theme=dark",
  );
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "History", exact: true })
    .click();
  await expect(reader(page)).toContainText("Prepare the release checklist.");
}

test("History filters and search select the matching reader and recover from no matches", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  const nativeDetailsCalls = observeNativeDetailsCalls(page);
  await openHistory(page);
  const sidebar = browser(page);
  await expect(information(page)).toContainText("fixture/voice-3");
  await expect(information(page)).toContainText("voice-response-3");
  const splitter = page.getByRole("separator", {
    name: "Resize editor and secondary sidebar",
    exact: true,
  });
  const before = (await reader(page).boundingBox())!;
  expect((await information(page).boundingBox())!.x).toBeGreaterThan(before.x);
  await splitter.focus();
  await splitter.press("ArrowLeft");
  await expect
    .poll(async () => (await reader(page).boundingBox())!.width)
    .toBeLessThan(before.width);
  const focusedTranscript = reader(page).getByRole("textbox", {
    name: /^Transcript from /,
  });
  await focusedTranscript.focus();
  await page.setViewportSize({ width: 1000, height: 820 });
  await expect(focusedTranscript).toBeFocused();
  await page.setViewportSize({ width: 1280, height: 820 });
  await expect(focusedTranscript).toBeFocused();
  await expect(transcripts(page)).toHaveCount(3);
  await sidebar.getByRole("button", { name: "Voice", exact: true }).click();
  await expect(
    sidebar.getByRole("button", { name: "Voice", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(transcripts(page)).toHaveCount(2);
  await transcript(page, "Ship the updated voice workflow.").click();
  await expect(reader(page)).toContainText("Ship the updated voice workflow.");
  await expect(information(page)).toContainText("fixture/voice-1");
  await expect(information(page)).toContainText("voice-response-1");

  await sidebar
    .getByRole("button", { name: "Audio files", exact: true })
    .click();
  await expect(transcripts(page)).toHaveCount(1);
  await expect(reader(page)).toContainText("Confirm the microphone settings.");
  await expect(information(page)).toContainText("fixture/file-2");
  await expect(information(page)).toContainText("file-response-2");
  await expect(information(page)).toContainText("Microphone check.wav");
  await expect(
    information(page).getByRole("region", { name: "Usage", exact: true }),
  ).toContainText("33");
  await sidebar
    .getByRole("searchbox", { name: "Search history" })
    .fill("release");
  await expect(transcripts(page)).toHaveCount(0);
  await expect(page.getByText(/No matching transcripts/).first()).toBeVisible();
  await expect(reader(page)).toHaveCount(0);
  await expect(information(page)).toHaveCount(0);

  await sidebar
    .getByRole("button", { name: "All sources", exact: true })
    .click();
  await expect(transcripts(page)).toHaveCount(1);
  await expect(reader(page)).toContainText("Prepare the release checklist.");
  await sidebar.getByRole("searchbox", { name: "Search history" }).fill("");
  await expect(transcripts(page)).toHaveCount(3);
  expect(nativeDetailsCalls()).toBe(0);
});

test("a newly completed transcript preserves the selected reader and deleting it selects another entry", async ({
  page,
}) => {
  await openHistory(page);
  await transcript(page, "Confirm the microphone settings.").click();
  await page.evaluate(() => window.testHistory.appendVoice());
  await expect(transcripts(page)).toHaveCount(4);
  await expect(reader(page)).toContainText("Confirm the microphone settings.");
  await expect(information(page)).toContainText("fixture/file-2");
  await expect(
    transcript(page, "Confirm the microphone settings."),
  ).toHaveAttribute("aria-current", "true");
  await expect(transcripts(page).first()).toContainText(
    "Incoming transcript stays in the sidebar.",
  );

  await reader(page)
    .getByRole("button", { name: "Transcript actions", exact: true })
    .click();
  await page
    .getByRole("menuitem", { name: "Remove from history", exact: true })
    .click();
  await expect(transcripts(page)).toHaveCount(3);
  await expect(
    transcript(page, "Confirm the microphone settings."),
  ).toHaveCount(0);
  await expect(reader(page)).toContainText(
    "Incoming transcript stays in the sidebar.",
  );
  await expect(transcripts(page).first()).toHaveAttribute(
    "aria-current",
    "true",
  );
  await expect(information(page)).toContainText("fixture/voice-4");
});

test("Clear history preserves entries after failure and clears every source on retry", async ({
  page,
}) => {
  await page.setViewportSize({ width: 560, height: 760 });
  await openHistory(page);
  await page
    .getByRole("button", { name: "Toggle primary sidebar", exact: true })
    .click();
  await browser(page)
    .getByRole("button", { name: "Audio files", exact: true })
    .click();
  await expect(transcripts(page)).toHaveCount(1);
  await page.evaluate(() => window.testHistory.failNextClear());
  await browser(page)
    .getByRole("button", { name: "Clear history", exact: true })
    .click();
  await expect(page.getByRole("alert")).toContainText(
    "Fixture history clear failed",
  );
  await expect(page.getByRole("alert")).toBeInViewport();
  await expect(transcripts(page)).toHaveCount(1);
  await page
    .getByRole("button", { name: "Toggle primary sidebar", exact: true })
    .click();
  await expect(reader(page)).toContainText("Confirm the microphone settings.");
  await expect(information(page)).toHaveCount(0);
  await page
    .getByRole("button", { name: "Toggle secondary sidebar", exact: true })
    .click();
  await expect(information(page)).toContainText("fixture/file-2");
  await page.keyboard.press("Escape");
  await page
    .getByRole("button", { name: "Toggle primary sidebar", exact: true })
    .click();
  await browser(page)
    .getByRole("button", { name: "Clear history", exact: true })
    .click();
  await expect(transcripts(page)).toHaveCount(0);
  await expect(
    browser(page).getByRole("button", { name: "Clear history", exact: true }),
  ).toBeDisabled();
  await page
    .getByRole("button", { name: "Toggle primary sidebar", exact: true })
    .click();
  await expect(reader(page)).toHaveCount(0);
  await expect(information(page)).toHaveCount(0);
  await expect(
    page
      .getByRole("region", { name: "Transcript history", exact: true })
      .getByText("No transcripts yet.", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Toggle primary sidebar", exact: true })
    .click();
  await browser(page)
    .getByRole("button", { name: "All sources", exact: true })
    .click();
  await expect(transcripts(page)).toHaveCount(0);
  await page.evaluate(() => window.testHistory.appendVoice());
  await expect(transcripts(page)).toHaveCount(1);
  await transcript(page, "Incoming transcript stays in the sidebar.").click();
  await expect(reader(page)).toContainText(
    "Incoming transcript stays in the sidebar.",
  );
});

test("the selected reader retains comparison and explicit copy actions", async ({
  page,
}) => {
  await openHistory(page);
  await transcript(page, "Ship the updated voice workflow.").click();
  await reader(page)
    .getByRole("button", {
      name: "Compare raw and cleaned transcripts",
      exact: true,
    })
    .click();
  await expect(
    reader(page).getByRole("region", { name: "Raw transcript", exact: true }),
  ).toContainText("um ship the updated voice workflow");
  await expect(
    reader(page).getByRole("region", {
      name: "Cleaned transcript",
      exact: true,
    }),
  ).toContainText("Ship the updated voice workflow.");
  await reader(page)
    .getByRole("button", { name: "Copy raw transcript", exact: true })
    .click();
  await expect(
    reader(page).getByRole("button", {
      name: "Raw transcript copied",
      exact: true,
    }),
  ).toBeVisible();
  await reader(page)
    .getByRole("button", { name: "Transcript actions", exact: true })
    .click();
  await expect(
    page.getByRole("menuitem", { name: "Transcription details", exact: true }),
  ).toHaveCount(0);
  await page.keyboard.press("Escape");
  await transcript(page, "Prepare the release checklist.").click();
  await expect(reader(page)).toContainText("Prepare the release checklist.");
  await expect(
    reader(page).getByRole("region", { name: "Raw transcript", exact: true }),
  ).toHaveCount(0);
});

test("the compact History sidebar opens on demand and closes after selecting a transcript", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 720 });
  await openHistory(page);
  const search = browser(page).getByRole("searchbox", {
    name: "Search history",
  });
  await search.focus();
  await page.setViewportSize({ width: 640, height: 720 });
  const toggle = page.getByRole("button", {
    name: "Toggle primary sidebar",
    exact: true,
  });
  await expect(toggle).toBeFocused();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await expect(browser(page)).toBeHidden();
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await expect(browser(page)).toBeVisible();
  await expect(search).toBeFocused();
  await browser(page)
    .getByRole("button", { name: "Voice", exact: true })
    .click();
  await expect(browser(page)).toBeVisible();
  await transcript(page, "Ship the updated voice workflow.").click();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await expect(browser(page)).toBeHidden();
  await expect(toggle).toBeFocused();
  await expect(reader(page)).toContainText("Ship the updated voice workflow.");
  await toggle.click();
  await expect(
    transcript(page, "Ship the updated voice workflow."),
  ).toHaveAttribute("aria-current", "true");
  await page.keyboard.press("Escape");
  await expect(browser(page)).toBeHidden();
  await expect(toggle).toBeFocused();
  await expect(reader(page)).toBeInViewport();
  await expect(information(page)).toHaveCount(0);
  const detailsToggle = page.getByRole("button", {
    name: "Toggle secondary sidebar",
    exact: true,
  });
  await expect(detailsToggle).toBeEnabled();
  await detailsToggle.click();
  await expect(information(page)).toContainText("fixture/voice-1");
  await page.keyboard.press("Escape");
  await expect(detailsToggle).toBeFocused();
  await expect(information(page)).toHaveCount(0);
  const transcriptScroll = reader(page).locator(".overflow-y-auto").first();
  const text = reader(page).getByRole("textbox", { name: /^Transcript from / });
  await text.focus();
  await text.press("Home");
  await text.press("PageDown");
  await expect
    .poll(() => transcriptScroll.evaluate((element) => element.scrollTop))
    .toBeGreaterThan(0);
  await page.setViewportSize({ width: 1280, height: 720 });
  await expect(text).toBeFocused();
  await expect(information(page)).toBeInViewport();
  expect((await information(page).boundingBox())!.x).toBeGreaterThan(
    (await reader(page).boundingBox())!.x,
  );
  const detailsBefore = await information(page).evaluate(
    (element) => element.scrollTop,
  );
  const transcriptBefore = await transcriptScroll.evaluate(
    (element) => element.scrollTop,
  );
  await information(page).focus();
  await information(page).press("PageDown");
  await expect
    .poll(() => information(page).evaluate((element) => element.scrollTop))
    .toBeGreaterThan(detailsBefore);
  expect(await transcriptScroll.evaluate((element) => element.scrollTop)).toBe(
    transcriptBefore,
  );
});

test("Settings navigation keeps runtime management on the activity rail", async ({
  page,
}) => {
  await openHistory(page);
  await browser(page)
    .getByRole("button", { name: "History settings", exact: true })
    .click();
  const navigation = page.getByRole("navigation", {
    name: "Settings sections",
    exact: true,
  });
  await expect(
    navigation.locator('[data-settings-section="local-runtime"]'),
  ).toHaveCount(0);
  const connections = navigation.locator(
    '[data-settings-section="connections"]',
  );
  const vocabulary = navigation.locator('[data-settings-section="vocabulary"]');
  await connections.focus();
  await connections.press("ArrowDown");
  await expect(vocabulary).toBeFocused();
  await expect(vocabulary).toHaveAttribute("aria-current", "page");
  await expect(page.locator('[data-pane="settings"]')).toBeVisible();
  await navigation
    .getByRole("textbox", { name: "Find settings" })
    .fill("local runtime");
  await expect(
    navigation.locator('[data-settings-section="local-runtime"]'),
  ).toHaveCount(0);
  await expect(
    page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: "Local runtime", exact: true }),
  ).toBeVisible();
});
