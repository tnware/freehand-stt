export type DiffKind = "equal" | "removed" | "added";

export type DiffPart = {
  text: string;
  kind: DiffKind;
};

export type TranscriptComparison = {
  raw: DiffPart[];
  processed: DiffPart[];
  highlighted: boolean;
};

const TOKEN_PATTERN = /\s+|[\p{L}\p{N}_’'-]+|[^\s]/gu;
// History expansion runs on the renderer thread. These limits let long,
// lightly edited transcripts use Myers diffing while making the worst case
// deterministic: at most ~2.6 MiB of trace storage and one million work units.
const MAX_COMPARISON_CHARACTERS = 200_000;
const MAX_COMPARISON_TOKENS = 20_000;
const MAX_EDIT_DISTANCE = 800;
const MAX_WORK_UNITS = 1_000_000;

const tokens = (text: string): string[] => text.match(TOKEN_PATTERN) ?? [];

function append(parts: DiffPart[], text: string, kind: DiffKind) {
  if (!text) return;
  const previous = parts.at(-1);
  if (previous?.kind === kind) {
    previous.text += text;
    return;
  }
  parts.push({ text, kind });
}

type DiffStep = {
  text: string;
  kind: DiffKind;
};

function diagonalValue(vector: Int32Array, distance: number, diagonal: number): number {
  const index = diagonal + distance;
  return index >= 0 && index < vector.length ? vector[index] : -1;
}

/**
 * Returns a shortest edit script using Myers' O((N + M)D) algorithm. Only
 * active diagonals are retained for each edit distance, so trace memory is
 * bounded by MAX_EDIT_DISTANCE rather than transcript length.
 */
function diffTokens(rawTokens: string[], processedTokens: string[]): DiffStep[] | null {
  const rawLength = rawTokens.length;
  const processedLength = processedTokens.length;
  const maximumDistance = Math.min(rawLength + processedLength, MAX_EDIT_DISTANCE);
  const trace: Int32Array[] = [];
  let workUnits = 0;

  for (let distance = 0; distance <= maximumDistance; distance++) {
    const vector = new Int32Array(distance * 2 + 1);

    for (let diagonal = -distance; diagonal <= distance; diagonal += 2) {
      workUnits++;
      if (workUnits > MAX_WORK_UNITS) return null;

      let rawIndex: number;
      if (distance === 0) {
        rawIndex = 0;
      } else {
        const previous = trace[distance - 1];
        if (
          diagonal === -distance ||
          (diagonal !== distance &&
            diagonalValue(previous, distance - 1, diagonal - 1) <
              diagonalValue(previous, distance - 1, diagonal + 1))
        ) {
          rawIndex = diagonalValue(previous, distance - 1, diagonal + 1);
        } else {
          rawIndex = diagonalValue(previous, distance - 1, diagonal - 1) + 1;
        }
      }

      let processedIndex = rawIndex - diagonal;
      while (
        rawIndex < rawLength &&
        processedIndex < processedLength &&
        rawTokens[rawIndex] === processedTokens[processedIndex]
      ) {
        rawIndex++;
        processedIndex++;
        workUnits++;
        if (workUnits > MAX_WORK_UNITS) return null;
      }

      vector[diagonal + distance] = rawIndex;
      if (rawIndex >= rawLength && processedIndex >= processedLength) {
        trace.push(vector);
        return backtrack(rawTokens, processedTokens, trace);
      }
    }

    trace.push(vector);
  }

  return null;
}

function backtrack(
  rawTokens: string[],
  processedTokens: string[],
  trace: Int32Array[],
): DiffStep[] {
  const reversed: DiffStep[] = [];
  let rawIndex = rawTokens.length;
  let processedIndex = processedTokens.length;

  for (let distance = trace.length - 1; distance > 0; distance--) {
    const previous = trace[distance - 1];
    const diagonal = rawIndex - processedIndex;
    const previousDiagonal =
      diagonal === -distance ||
      (diagonal !== distance &&
        diagonalValue(previous, distance - 1, diagonal - 1) <
          diagonalValue(previous, distance - 1, diagonal + 1))
        ? diagonal + 1
        : diagonal - 1;
    const previousRawIndex = diagonalValue(previous, distance - 1, previousDiagonal);
    const previousProcessedIndex = previousRawIndex - previousDiagonal;

    while (rawIndex > previousRawIndex && processedIndex > previousProcessedIndex) {
      reversed.push({ text: rawTokens[rawIndex - 1], kind: "equal" });
      rawIndex--;
      processedIndex--;
    }

    if (rawIndex === previousRawIndex) {
      reversed.push({ text: processedTokens[processedIndex - 1], kind: "added" });
      processedIndex--;
    } else {
      reversed.push({ text: rawTokens[rawIndex - 1], kind: "removed" });
      rawIndex--;
    }
  }

  while (rawIndex > 0 && processedIndex > 0) {
    reversed.push({ text: rawTokens[rawIndex - 1], kind: "equal" });
    rawIndex--;
    processedIndex--;
  }
  while (rawIndex > 0) {
    reversed.push({ text: rawTokens[rawIndex - 1], kind: "removed" });
    rawIndex--;
  }
  while (processedIndex > 0) {
    reversed.push({ text: processedTokens[processedIndex - 1], kind: "added" });
    processedIndex--;
  }

  return reversed.reverse();
}

function fallback(rawText: string, processedText: string): TranscriptComparison {
  return {
    raw: [{ text: rawText, kind: "equal" }],
    processed: [{ text: processedText, kind: "equal" }],
    highlighted: false,
  };
}

/**
 * Produces two independently renderable word-level views. Work is bounded so
 * expanding retained history cannot monopolize the renderer.
 */
export function compareTranscriptText(rawText: string, processedText: string): TranscriptComparison {
  if (rawText === processedText) {
    return {
      raw: [{ text: rawText, kind: "equal" }],
      processed: [{ text: processedText, kind: "equal" }],
      highlighted: true,
    };
  }

  if (rawText.length + processedText.length > MAX_COMPARISON_CHARACTERS) {
    return fallback(rawText, processedText);
  }

  const rawTokens = tokens(rawText);
  const processedTokens = tokens(processedText);
  if (
    rawTokens.length === 0 ||
    processedTokens.length === 0 ||
    rawTokens.length + processedTokens.length > MAX_COMPARISON_TOKENS
  ) {
    return fallback(rawText, processedText);
  }

  const steps = diffTokens(rawTokens, processedTokens);
  if (!steps) return fallback(rawText, processedText);

  const raw: DiffPart[] = [];
  const processed: DiffPart[] = [];

  for (const step of steps) {
    if (step.kind !== "added") append(raw, step.text, step.kind);
    if (step.kind !== "removed") append(processed, step.text, step.kind);
  }

  return { raw, processed, highlighted: true };
}
