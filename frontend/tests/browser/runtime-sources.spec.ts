import { test, expect } from "./fixtures";

for (const provider of ["llama-cpp", "whisper-cpp", "nemo-speech-cpp"]) {
  test(`${provider} sources can be inspected without installation`, async ({
    page,
  }) => {
    await page.goto(`/tests/browser/app/?runtime&runtime-provider=${provider}`);
    await page
      .getByRole("button", { name: "Local runtime", exact: true })
      .click();
    const upstream =
      provider === "nemo-speech-cpp"
        ? "NVIDIA/NeMo-Speech.cpp"
        : `ggml-org/${provider === "llama-cpp" ? "llama.cpp" : "whisper.cpp"}`;
    await page.getByRole("button", { name: /^Binary download source/ }).click();
    await expect(
      page.getByRole("link", { name: upstream, exact: true }),
    ).toHaveAttribute("href", `https://github.com/${upstream}`);
    await page.getByText("Binary download details", { exact: true }).click();
    await expect(
      page.getByText("fixture-cpu.zip", { exact: true }),
    ).toBeVisible();
    const models = page.getByRole("region", {
      name: "Model catalog",
      exact: true,
    });
    await expect(
      page.getByRole("heading", { name: "Models", exact: true }),
    ).toBeVisible();
    const rows = models.getByRole("article");
    await expect(rows).toHaveCount(provider === "nemo-speech-cpp" ? 2 : 1);
    for (const row of await rows.all()) {
      await expect(
        row.getByRole("button", { name: "Get", exact: true }),
      ).toBeDisabled();
      await expect(
        row.getByRole("button", { name: "Select", exact: true }),
      ).toBeDisabled();
      await row.getByRole("button", { name: /^Model details:/ }).click();
    }
    if (provider === "nemo-speech-cpp") {
      await expect(
        models.getByText("Source: NeMo’s built-in model manager"),
      ).toHaveCount(2);
      await expect(models.getByRole("link")).toHaveCount(0);
    } else {
      const repository =
        provider === "llama-cpp"
          ? "superwhisper/s1-mini-GGUF"
          : "ggerganov/whisper.cpp";
      await expect(
        models.getByRole("link", { name: repository }),
      ).toHaveAttribute("href", `https://huggingface.co/${repository}`);
      await models.getByText("Model download details", { exact: true }).click();
      await expect(
        models.getByText("fixture-revision", { exact: true }),
      ).toBeVisible();
    }
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  });
}

test("binary preview details follow the explicit backend choice without downloading", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-provider=llama-cpp");
  await page
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Download and install", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: /^Binary download source/ }).click();
  await page.getByText("Binary download details", { exact: true }).click();
  await expect(
    page.getByText("fixture-cuda.zip", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("fixture-cpu.zip", { exact: true })).toHaveCount(
    0,
  );
  await page.getByRole("button", { name: "CPU", exact: true }).click();
  await expect(
    page.getByText("fixture-cpu.zip", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("fixture-cuda.zip", { exact: true })).toHaveCount(
    0,
  );
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "GetBinaryOptions:llama-cpp",
  ]);
});
