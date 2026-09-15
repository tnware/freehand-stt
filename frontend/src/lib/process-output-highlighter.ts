type Phase =
  | "prefix"
  | "key"
  | "colon"
  | "value"
  | "after"
  | "string"
  | "primitive"
  | "nested"
  | "plain";

const resetColor = "\u001b[39m";

/** Streaming presentation only: never decodes or reformats output text. */
export class ProcessOutputHighlighter {
  #phase: Phase = "prefix";
  #prefix = "";
  #key = "";
  #token = "";
  #keyString = false;
  #escaped = false;
  #escapedToken = false;
  #depth = 0;
  #nestedString = false;
  #record = "";
  #color = "";
  #activeColor = "";
  #stream: string | undefined;
  #lineStarted = false;
  // Once native styling appears, leave all subsequent styling to the producer.
  // This also preserves SGR sequences and styles spanning arbitrary chunks.
  #native = false;

  write(text: string, stream: string): string {
    const output: string[] = [];
    if (this.#stream !== undefined && this.#stream !== stream) {
      this.#resetLine();
      // Interleaved pipes are not continuations of one JSON/log record.
      if (this.#lineStarted) this.#phase = "plain";
    }
    this.#stream = stream;
    for (const character of text) {
      if (character === "\u001b") {
        this.#native = true;
        this.#color = "";
      }
      if (!this.#native) {
        if (character === "\n" || character === "\r") {
          this.#resetLine();
          this.#lineStarted = false;
        } else {
          this.#step(character);
          this.#lineStarted = true;
        }
      }
      if (this.#activeColor !== this.#color) {
        output.push(this.#color ? `\u001b[${this.#color}m` : resetColor);
        this.#activeColor = this.#color;
      }
      output.push(character);
    }
    return output.join("");
  }

  #resetLine() {
    this.#phase = "prefix";
    this.#prefix = "";
    this.#key = "";
    this.#token = "";
    this.#escaped = false;
    this.#escapedToken = false;
    this.#depth = 0;
    this.#nestedString = false;
    this.#record = "";
    this.#color = "";
  }

  #step(character: string): void {
    switch (this.#phase) {
      case "prefix": {
        if (!this.#prefix && (character === " " || character === "\t")) return;
        if (!this.#prefix && character === "{") {
          this.#phase = "key";
          return;
        }
        this.#prefix += character;
        if (/^warning:$/i.test(this.#prefix)) {
          this.#color = "33";
          this.#phase = "plain";
        } else if (/^error:$/i.test(this.#prefix)) {
          this.#color = "31";
          this.#phase = "plain";
        } else if (this.#prefix.length >= 8) {
          this.#phase = "plain";
        }
        return;
      }
      case "key":
        if (character === " " || character === "\t") return;
        if (character === '"') this.#startString(true);
        else this.#phase = "plain";
        return;
      case "colon":
        if (character === " " || character === "\t") return;
        this.#phase = character === ":" ? "value" : "plain";
        return;
      case "value":
        if (character === " " || character === "\t") return;
        if (character === '"') this.#startString(false);
        else if (character === "{" || character === "[") {
          this.#depth = 1;
          this.#phase = "nested";
        } else {
          this.#phase = "primitive";
          if (this.#record === "http.request" && this.#key === "status") {
            this.#color =
              character === "5"
                ? "31"
                : character === "4"
                  ? "33"
                  : /[123]/.test(character)
                    ? "32"
                    : "34";
          }
        }
        return;
      case "string":
        if (this.#escaped) {
          this.#escaped = false;
          return;
        }
        if (character === "\\") {
          this.#escaped = true;
          this.#escapedToken = true;
        } else if (character === '"') {
          const token = this.#escapedToken ? "" : this.#token;
          if (this.#keyString) {
            this.#key = token;
            this.#phase = "colon";
          } else {
            if (
              this.#key === "event" &&
              (token === "http.request" || token === "listener.ready")
            ) {
              this.#record = token;
              this.#color = token === "listener.ready" ? "32" : "34";
            }
            this.#phase = "after";
          }
          this.#token = "";
        } else if (this.#token.length < 32) {
          this.#token += character;
        } else {
          // Large strings still render immediately; never retain their contents.
          this.#escapedToken = true;
        }
        return;
      case "nested":
        if (this.#escaped) this.#escaped = false;
        else if (this.#nestedString && character === "\\") this.#escaped = true;
        else if (character === '"') this.#nestedString = !this.#nestedString;
        else if (!this.#nestedString) {
          if (character === "{" || character === "[") this.#depth++;
          if (character === "}" || character === "]") this.#depth--;
          if (this.#depth === 0) this.#phase = "after";
          if (this.#depth > 32) this.#phase = "plain";
        }
        return;
      case "primitive":
        if (!/[ \t,}]/.test(character)) return;
        this.#phase = "after";
        this.#step(character);
        return;
      case "after":
        if (character === " " || character === "\t") return;
        this.#color = this.#record === "http.request" ? "34" : this.#color;
        this.#phase = character === "," ? "key" : "plain";
        return;
      case "plain":
        return;
    }
  }

  #startString(key: boolean) {
    this.#phase = "string";
    this.#keyString = key;
    this.#token = "";
    this.#escapedToken = false;
    this.#escaped = false;
  }
}
