import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";

// Serve real UI and generated DTOs without starting the native Wails bridge.
const root = fileURLToPath(new URL("../../", import.meta.url));
export default defineConfig({
  root,
  plugins: [tailwindcss(), svelte()],
  resolve: {
    alias: {
      $lib: `${root}src/lib`,
      $bindings: `${root}bindings/github.com/tnware/freehand-stt/internal`,
    },
  },
  server: { host: "127.0.0.1", port: 9346, strictPort: true },
});
