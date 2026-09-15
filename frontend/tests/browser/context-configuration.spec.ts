import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

const rail = (page: Page) =>
  page.getByRole("navigation", { name: "Workspace", exact: true });
const area = (page: Page, name: string) =>
  rail(page).getByRole("button", { name, exact: true });
const primary = (page: Page) => page.locator("#workbench-primary-sidebar");
const secondary = (page: Page) => page.locator("#workbench-secondary-sidebar");
const configuration = (page: Page) =>
  secondary(page).locator('[data-pane="configuration"]');
const layoutControl = (page: Page, side: "primary" | "secondary") =>
  page.getByRole("button", {
    name: `Toggle ${side} sidebar`,
    exact: true,
  });
const decision = (page: Page) =>
  page.getByRole("dialog", { name: "Save changes?", exact: true });
const connectionEditor = (page: Page) =>
  page.getByRole("region", { name: "Connection editor", exact: true });
const connectionDecision = (page: Page) =>
  page.getByRole("dialog", { name: "Save connection changes?", exact: true });

async function openApp(page: Page, width = 1280, history = false) {
  await page.setViewportSize({ width, height: 820 });
  await page.goto(
    `/tests/browser/app/?main&setup-ready&workflows${history ? "&history-workbench" : ""}`,
  );
}

async function showSidebar(page: Page, side: "primary" | "secondary") {
  const button = layoutControl(page, side);
  if ((await button.getAttribute("aria-pressed")) !== "true")
    await button.click();
  await expect(button).toHaveAttribute("aria-pressed", "true");
}

async function openVoiceAudio(page: Page) {
  await area(page, "Voice transcription").click();
  await showSidebar(page, "secondary");
  await secondary(page)
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  await expect(configuration(page).locator("#max-duration")).toBeVisible();
}

const workflowCogs = [
  {
    id: "voice",
    page: "Voice transcription",
    cog: "Voice settings",
    title: "Voice transcription",
  },
  {
    id: "file",
    page: "Audio file",
    cog: "Audio file settings",
    title: "Audio-file transcription",
  },
  {
    id: "tts",
    page: "Text to speech",
    cog: "Text to speech settings",
    title: "Text to speech",
  },
] as const;

for (const width of [1280, 560]) {
  test(`workflow settings cogs open local options at ${width}px`, async ({
    page,
  }, info) => {
    await openApp(page, width);
    for (const workflow of workflowCogs) {
      await area(page, workflow.page).click();
      const cog = page.getByRole("button", { name: workflow.cog, exact: true });
      await expect(cog).toBeInViewport();
      await expect(cog.locator("svg")).toBeVisible();
      await page.screenshot({
        path: info.outputPath(`${workflow.id}-settings-cog-${width}.png`),
      });
      await cog.click();
      await expect(
        configuration(page).getByRole("heading", {
          name: workflow.title,
          exact: true,
        }),
      ).toBeVisible();
      await expect(area(page, workflow.page)).toHaveAttribute(
        "aria-current",
        "page",
      );
      await expect(area(page, "Settings")).not.toHaveAttribute(
        "aria-current",
        "page",
      );
      await configuration(page)
        .getByRole("button", { name: "Done", exact: true })
        .click();
      await expect(configuration(page)).toHaveCount(0);
    }
  });
}

test("first-run pages expose settings cogs before transport controls are ready", async ({
  page,
}) => {
  await page.setViewportSize({ width: 560, height: 820 });
  await page.goto("/tests/browser/app/?main&blank&pickers");
  for (const workflow of workflowCogs) {
    await area(page, workflow.page).click();
    const cog = page.getByRole("button", { name: workflow.cog, exact: true });
    await expect(cog).toBeInViewport();
    await cog.click();
    await expect(
      configuration(page).getByRole("heading", {
        name: workflow.title,
        exact: true,
      }),
    ).toBeVisible();
    await expect(area(page, workflow.page)).toHaveAttribute(
      "aria-current",
      "page",
    );
    await configuration(page)
      .getByRole("button", { name: "Done", exact: true })
      .click();
  }
});

test("managed Voice settings cog opens Voice options without runtime actions", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await area(page, "Voice transcription").click();
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  await expect(
    configuration(page).getByRole("heading", {
      name: "Voice transcription",
      exact: true,
    }),
  ).toBeVisible();
  await expect(area(page, "Voice transcription")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(area(page, "Local runtime")).not.toHaveAttribute(
    "aria-current",
    "page",
  );
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("compact inspector reopening closes the primary drawer and keeps its controls interactive", async ({
  page,
}) => {
  await openApp(page, 560);
  await area(page, "Voice transcription").click();
  await showSidebar(page, "secondary");

  await showSidebar(page, "primary");
  await expect(primary(page)).toBeVisible();
  await expect(secondary(page)).toBeHidden();
  await layoutControl(page, "secondary").click();
  await expect(primary(page)).toBeHidden();
  await expect(layoutControl(page, "primary")).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(secondary(page)).toBeVisible();
  const audioTab = secondary(page).getByRole("tab", {
    name: "Audio settings",
    exact: true,
  });
  await audioTab.click();
  await expect(audioTab).toHaveAttribute("aria-selected", "true");
  await configuration(page).locator("#max-duration").focus();
  await expect(configuration(page).locator("#max-duration")).toBeFocused();

  await showSidebar(page, "primary");
  await expect(secondary(page)).toBeHidden();
  await primary(page)
    .getByRole("button", { name: /^Vocabulary/ })
    .click();
  await expect(primary(page)).toBeHidden();
  await expect(layoutControl(page, "primary")).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(secondary(page)).toBeVisible();
  const vocabularyTab = secondary(page).getByRole("tab", {
    name: "Vocabulary settings",
    exact: true,
  });
  await expect(vocabularyTab).toHaveAttribute("aria-selected", "true");
  const terms = configuration(page).getByRole("textbox", {
    name: "Names and phrases",
    exact: true,
  });
  await terms.focus();
  await expect(terms).toBeFocused();
  await audioTab.click();
  await expect(audioTab).toHaveAttribute("aria-selected", "true");
  await expect(configuration(page).locator("#max-duration")).toBeVisible();
});

for (const width of [1280, 560]) {
  test(`workflow options stay beside their owning page at ${width}px`, async ({
    page,
  }, info) => {
    await openApp(page, width);
    await area(page, "Voice transcription").click();
    await showSidebar(page, "secondary");
    await expect(area(page, "Voice transcription")).toHaveAttribute(
      "aria-current",
      "page",
    );
    await expect(area(page, "Settings")).not.toHaveAttribute(
      "aria-current",
      "page",
    );
    await expect(
      page.getByRole("group", { name: "Workspace layout" }).getByRole("button"),
    ).toHaveCount(3);
    const tabs = secondary(page).getByRole("tablist", {
      name: "Context settings",
      exact: true,
    });
    await expect(tabs.getByRole("tab")).toHaveText([
      "Transcription",
      "Audio",
      "Cleanup",
      "Vocabulary",
      "Overlay",
      "Delivery",
    ]);
    await expect(
      tabs.getByRole("tab", {
        name: "Voice transcription settings",
        exact: true,
      }),
    ).toHaveAttribute("aria-selected", "true");
    if (width === 1280)
      await page.screenshot({
        path: info.outputPath("voice-context-wide.png"),
      });

    await area(page, "Audio file").click();
    await expect(configuration(page)).toHaveCount(0);
    await showSidebar(page, "secondary");
    await expect(tabs.getByRole("tab")).toHaveText([
      "Transcription",
      "Cleanup",
      "Vocabulary",
    ]);
    await expect(
      tabs.getByRole("tab", {
        name: "Audio-file transcription settings",
        exact: true,
      }),
    ).toHaveAttribute("aria-selected", "true");
    await expect(
      secondary(page).getByRole("tab", { name: "Audio settings", exact: true }),
    ).toHaveCount(0);

    await area(page, "Text to speech").click();
    await expect(configuration(page)).toHaveCount(0);
    await showSidebar(page, "secondary");
    await expect(
      configuration(page).getByRole("heading", {
        name: "Text to speech",
        exact: true,
      }),
    ).toBeVisible();
    await expect(tabs).toHaveCount(0);
    await expect(
      configuration(page).locator("#saved-connection-speech"),
    ).toBeVisible();
    await area(page, "Voice transcription").click();
    await expect(configuration(page)).toHaveCount(0);
    await showSidebar(page, "secondary");
    await expect(
      tabs.getByRole("tab", {
        name: "Voice transcription settings",
        exact: true,
      }),
    ).toHaveAttribute("aria-selected", "true");
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}

test("closing and changing pages resolve a local draft, including failed save and retry", async ({
  page,
  saves,
}) => {
  await openApp(page);
  await openVoiceAudio(page);
  const duration = configuration(page).locator("#max-duration");
  const original = await duration.inputValue();
  const changed = original === "90" ? "91" : "90";
  await duration.fill(changed);
  await layoutControl(page, "secondary").click();
  await expect(decision(page)).toBeVisible();
  await decision(page)
    .getByRole("button", { name: "Save", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "failure");
  await expect(decision(page).getByRole("alert")).toContainText(
    "Fixture save failed",
  );
  await expect(
    page.getByRole("complementary", { name: "Notifications" }),
  ).toHaveCount(0);
  await decision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(duration).toHaveValue(changed);
  await expect(secondary(page)).toBeVisible();

  await area(page, "Audio file").click();
  await expect(decision(page)).toBeVisible();
  await expect(area(page, "Voice transcription")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await decision(page)
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(area(page, "Audio file")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await openVoiceAudio(page);
  await expect(duration).toHaveValue(original);
  await duration.fill(changed);
  await area(page, "Audio file").click();
  await decision(page)
    .getByRole("button", { name: "Save", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(area(page, "Audio file")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await openVoiceAudio(page);
  await expect(duration).toHaveValue(changed);
});

for (const width of [1280, 560]) {
  test(`shared sidebar options open locally and keep global command routes at ${width}px`, async ({
    page,
    saves,
  }) => {
    await openApp(page, width);
    await area(page, "Voice transcription").click();
    await showSidebar(page, "primary");
    await primary(page)
      .getByRole("button", { name: /^Vocabulary/ })
      .click();
    await expect(area(page, "Voice transcription")).toHaveAttribute(
      "aria-current",
      "page",
    );
    await expect(
      configuration(page).getByRole("heading", {
        name: "Vocabulary",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      configuration(page).getByText("Shared", { exact: true }),
    ).toBeVisible();
    await expect(configuration(page)).toContainText(
      "Changes apply to Voice and audio files.",
    );
    await configuration(page)
      .getByRole("textbox", { name: "Names and phrases", exact: true })
      .fill("Freehand\nShared project name");
    await configuration(page)
      .getByRole("button", { name: "Save", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "success");
    await configuration(page)
      .getByRole("button", { name: "Done", exact: true })
      .click();

    await showSidebar(page, "primary");
    await primary(page)
      .getByRole("button", { name: /^Overlay/ })
      .click();
    await expect(area(page, "Voice transcription")).toHaveAttribute(
      "aria-current",
      "page",
    );
    await expect(
      configuration(page).getByRole("heading", {
        name: "Overlay",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      configuration(page).getByText("Shared", { exact: true }),
    ).toBeVisible();
    await configuration(page)
      .getByRole("button", { name: "Done", exact: true })
      .click();

    await showSidebar(page, "primary");
    await primary(page)
      .getByRole("button", { name: /^Insert into/ })
      .click();
    await expect(
      configuration(page).getByRole("heading", {
        name: "Transcript delivery",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      configuration(page).getByRole("radiogroup", {
        name: "Transcript delivery mode",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      configuration(page).getByText("Shared", { exact: true }),
    ).toBeVisible();
    await expect(
      configuration(page).getByRole("switch", {
        name: "Show window when launched",
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(
      configuration(page).getByRole("radiogroup", {
        name: "Color mode",
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(area(page, "Settings")).not.toHaveAttribute(
      "aria-current",
      "page",
    );

    await area(page, "Audio file").click();
    await showSidebar(page, "primary");
    await primary(page)
      .getByRole("button", { name: /^Vocabulary/ })
      .click();
    await expect(area(page, "Audio file")).toHaveAttribute(
      "aria-current",
      "page",
    );
    await expect(
      configuration(page).getByRole("textbox", {
        name: "Names and phrases",
        exact: true,
      }),
    ).toHaveValue("Freehand\nShared project name");
    await expect(
      configuration(page).getByText("Shared", { exact: true }),
    ).toBeVisible();

    for (const section of ["general", "overlay", "vocabulary"] as const) {
      await openSection(page, section);
      await expect(area(page, "Settings")).toHaveAttribute(
        "aria-current",
        "page",
      );
      await expect(configuration(page)).toHaveCount(0);
      const globalSettings = page.locator('[data-pane="settings"]');
      await expect(globalSettings).toBeVisible();
      if (section === "general") {
        await expect(
          globalSettings.getByRole("switch", {
            name: "Show window when launched",
            exact: true,
          }),
        ).toBeVisible();
        await expect(
          globalSettings.getByRole("radiogroup", {
            name: "Color mode",
            exact: true,
          }),
        ).toBeVisible();
      }
      if (section === "vocabulary")
        await expect(
          globalSettings.getByRole("textbox", {
            name: "Names and phrases",
            exact: true,
          }),
        ).toHaveValue("Freehand\nShared project name");
    }
  });
}

test("global Settings hides the bottom panel without changing its tab or visibility preference", async ({
  page,
}) => {
  await openApp(page);
  await area(page, "Voice transcription").click();
  const toggle = page.getByRole("button", {
    name: "Toggle bottom panel",
    exact: true,
  });
  const panel = page.locator("#workbench-bottom-panel");
  if ((await toggle.getAttribute("aria-pressed")) !== "true")
    await toggle.click();
  await panel.getByRole("tab", { name: "Diagnostics", exact: true }).click();
  await area(page, "Settings").click();
  await expect(panel).toHaveCount(0);
  await expect(toggle).toBeDisabled();
  await page.keyboard.press("Control+j");
  await area(page, "Voice transcription").click();
  await expect(panel).toBeVisible();
  await expect(
    panel.getByRole("tab", { name: "Diagnostics", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  await toggle.click();
  await expect(panel).toHaveCount(0);
  await area(page, "Settings").click();
  await expect(panel).toHaveCount(0);
  await area(page, "Voice transcription").click();
  await expect(panel).toHaveCount(0);
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await toggle.click();
  await expect(
    panel.getByRole("tab", { name: "Diagnostics", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
});

test("a hidden compact inspector reopens its invalid field when navigation save fails", async ({
  page,
  saves,
}) => {
  await openApp(page);
  await openVoiceAudio(page);
  await configuration(page).locator("#max-duration").fill("90");
  await page.setViewportSize({ width: 560, height: 820 });
  await expect(secondary(page)).toBeHidden();
  await area(page, "Audio file").click();
  await decision(page)
    .getByRole("button", { name: "Save", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "invalid-duration");
  await expect(decision(page)).toHaveCount(0);
  await expect(secondary(page)).toBeVisible();
  await expect(configuration(page).locator("#max-duration")).toBeFocused();
  await expect(configuration(page).locator("#max-duration")).toHaveValue("90");
  await expect(area(page, "Voice transcription")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await configuration(page)
    .getByRole("button", { name: "Done", exact: true })
    .click();
  await decision(page)
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(secondary(page)).toBeHidden();
});

test("Connections owns one sidebar inventory and guards the central editor", async ({
  page,
}, info) => {
  await openApp(page);
  await area(page, "Connections").click();
  await showSidebar(page, "primary");
  const inventory = primary(page).getByRole("navigation", {
    name: "Saved connections",
    exact: true,
  });
  await expect(inventory).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Saved connections", exact: true }),
  ).toHaveCount(1);
  await expect(
    page
      .locator('[data-pane="connections"]')
      .getByRole("navigation", { name: "Saved connections", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "Choose a connection", exact: true }),
  ).toBeVisible();
  await inventory.getByRole("button", { name: /Original server/ }).click();
  await expect(connectionEditor(page)).toBeVisible();
  await expect(page.locator("#connection-name")).toHaveValue("Original server");
  await page.screenshot({
    path: info.outputPath("connections-workbench-wide.png"),
  });
  await page.locator("#connection-name").fill("Unfinished server");
  await area(page, "Local runtime").click();
  await expect(connectionDecision(page)).toBeVisible();
  await connectionDecision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(page.locator("#connection-name")).toHaveValue(
    "Unfinished server",
  );
  await area(page, "Local runtime").click();
  await connectionDecision(page)
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(area(page, "Local runtime")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(connectionEditor(page)).toHaveCount(0);
});

test("compact connection requests return to workflow options and hiding clears credentials", async ({
  page,
}) => {
  await openApp(page, 560);
  await area(page, "Voice transcription").click();
  await showSidebar(page, "secondary");
  await configuration(page)
    .getByRole("button", { name: "Edit connection", exact: true })
    .click();
  await expect(area(page, "Connections")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(connectionEditor(page)).toBeVisible();
  await page
    .locator('[data-pane="connections"]')
    .getByRole("button", { name: "Done", exact: true })
    .click();
  await expect(area(page, "Voice transcription")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(configuration(page)).toBeVisible();
  await expect(
    secondary(page).getByRole("tab", {
      name: "Voice transcription settings",
      exact: true,
    }),
  ).toHaveAttribute("aria-selected", "true");

  await configuration(page)
    .getByRole("button", { name: "Edit connection", exact: true })
    .click();
  await page.locator("#connection-auth").click();
  await page.getByRole("option", { name: "API key", exact: true }).click();
  await page
    .locator("#connection-api-key")
    .fill("fixture-transient-context-key");
  await area(page, "History").click();
  await connectionDecision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(page.locator("#connection-api-key")).toHaveValue(
    "fixture-transient-context-key",
  );
  await page.evaluate(() => window.testConnectionWindows.hide());
  await expect(page.locator("#connection-api-key")).toHaveCount(0);
  await expect(connectionEditor(page)).toHaveCount(0);
  await area(page, "Voice transcription").click();
  await showSidebar(page, "secondary");
  await configuration(page)
    .getByRole("button", { name: "Edit connection", exact: true })
    .click();
  await page.locator("#connection-auth").click();
  await page.getByRole("option", { name: "API key", exact: true }).click();
  await expect(page.locator("#connection-api-key")).toHaveValue("");
});

test("History guards its Settings and Details switch while global Settings keeps app-wide sections", async ({
  page,
}) => {
  await openApp(page, 1280, true);
  await area(page, "History").click();
  await showSidebar(page, "secondary");
  const view = secondary(page).getByRole("group", {
    name: "History sidebar view",
    exact: true,
  });
  await view
    .getByRole("button", { name: "History settings", exact: true })
    .click();
  const retention = configuration(page).getByRole("switch", {
    name: "Keep transcript history",
    exact: true,
  });
  await expect(retention).toHaveAttribute("aria-checked", "true");
  await retention.click();
  await view.getByRole("button", { name: "Details", exact: true }).click();
  await expect(decision(page)).toBeVisible();
  await decision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(retention).toHaveAttribute("aria-checked", "false");
  await view.getByRole("button", { name: "Details", exact: true }).click();
  await decision(page)
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(
    secondary(page).getByRole("heading", {
      name: "Transcription details",
      exact: true,
    }),
  ).toBeVisible();
  await expect(configuration(page)).toHaveCount(0);

  await area(page, "Settings").click();
  await showSidebar(page, "primary");
  const settingsNav = primary(page).getByRole("navigation", {
    name: "Settings sections",
    exact: true,
  });
  const sections = settingsNav.locator("[data-settings-section]");
  await expect(sections).toHaveCount(4);
  expect(
    await sections.evaluateAll((nodes) =>
      nodes.map((node) => node.getAttribute("data-settings-section")).sort(),
    ),
  ).toEqual(["general", "overlay", "shortcuts", "vocabulary"]);
  await settingsNav
    .getByRole("textbox", { name: "Find settings", exact: true })
    .fill("Connections");
  await expect(sections).toHaveCount(0);
  await expect(area(page, "Connections")).toBeVisible();
});
