import { expect, it } from "vitest";
import { ProcessOutputHighlighter } from "./process-output-highlighter";

const plain = (text: string) =>
  text.replace(/\u001b\[(?:31|32|33|34|39)m/g, "");
const request = (status: number) =>
  JSON.stringify({
    event: "http.request",
    method: "POST",
    path: "/v1/audio/transcriptions",
    remote_address: "127.0.0.1",
    request_id: "fixture",
    status,
  }) + "\n";

it("highlights the qualified HTTP status classes and listener event without changing text", () => {
  for (const [status, color] of [
    [200, 32],
    [400, 33],
    [503, 31],
  ]) {
    const input = request(status);
    const output = new ProcessOutputHighlighter().write(input, "stderr");
    expect(plain(output)).toBe(input);
    expect(output).toContain(`\u001b[${color}m${status}`);
    expect(output).toContain("\u001b[39m\n");
  }
  const ready =
    '{"capabilities":["transcription"],"event":"listener.ready","transport":"http","url":"http://127.0.0.1:8080"}\n';
  const output = new ProcessOutputHighlighter().write(ready, "stdout");
  expect(plain(output)).toBe(ready);
  expect(output).toContain("\u001b[32m");
});

it("renders every partial chunk immediately with the same styles at every split", () => {
  const input = request(503) + "warning: no grammar configured\n";
  const expected = new ProcessOutputHighlighter().write(input, "stderr");
  for (let split = 0; split <= input.length; split++) {
    const highlighter = new ProcessOutputHighlighter();
    const before = highlighter.write(input.slice(0, split), "stderr");
    expect(plain(before)).toBe(input.slice(0, split));
    const after = highlighter.write(input.slice(split), "stderr");
    expect(before + after).toBe(expected);
  }
  const highlighter = new ProcessOutputHighlighter();
  expect([...input].map((c) => highlighter.write(c, "stderr")).join("")).toBe(
    expected,
  );
});

it("ignores nested, escaped, unknown and oversized event markers", () => {
  for (const input of [
    '{"detail":{"event":"http.request","status":503}}\n',
    '{"event":"other","status":503}\n',
    '{"event":"http.\\u0072equest","status":503}\n',
    '{"ev\\u0065nt":"http.request","status":503}\n',
    `{"event":"${"x".repeat(100_000)}","status":503}\n`,
  ])
    expect(new ProcessOutputHighlighter().write(input, "stderr")).toBe(input);
});

it("never decodes JSON strings into escapes or interprets severity inside a message", () => {
  const input =
    '{"event":"http.request","path":"error: \\u001b]52;c;anything","extra":{"status":503},"status":200}\n';
  const output = new ProcessOutputHighlighter().write(input, "stderr");
  expect(plain(output)).toBe(input);
  expect(output).not.toContain("\u001b[31m");
  expect(output).toContain("\u001b[32m200");
  expect(output).not.toContain("\u001b]");
});

it("preserves native ANSI and never inserts styling into split escape sequences", () => {
  const highlighter = new ProcessOutputHighlighter();
  const native = "\u001b[35mnative\n" + request(503) + "\u001b[0m";
  const chunks = [native.slice(0, 1), native.slice(1, 4), native.slice(4)];
  expect(chunks.map((part) => highlighter.write(part, "stderr")).join("")).toBe(
    native,
  );

  const later = new ProcessOutputHighlighter();
  const initial = later.write("error: plain ", "stderr");
  expect(initial).toContain("\u001b[31m");
  expect(later.write("\u001b[36mproducer", "stderr")).toBe(
    "\u001b[39m\u001b[36mproducer",
  );
  expect(later.write(request(503), "stderr")).toBe(request(503));
});

it("drops incomplete record context across pipes and resets color at carriage returns", () => {
  const highlighter = new ProcessOutputHighlighter();
  const start = highlighter.write('{"event":"http.request",', "stderr");
  expect(start).toContain("\u001b[34m");
  expect(highlighter.write('"status":503}\n', "stdout")).toBe(
    '\u001b[39m"status":503}\n',
  );
  const next = highlighter.write("error: failed\rprogress", "stderr");
  expect(next).toContain("\u001b[39m\rprogress");
  expect(plain(next)).toBe("error: failed\rprogress");
});
