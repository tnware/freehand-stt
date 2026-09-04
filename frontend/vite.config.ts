import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";
import path from "node:path";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true
  },
  resolve: {
    alias: {
      "@": path.resolve("./src"),
      "$lib": path.resolve("./src/lib"),
      "$bindings": path.resolve(
        "./bindings/github.com/tnware/freehand-stt/internal"
      )
    }
  },
  plugins: [tailwindcss(), svelte(), wails("./bindings")]
});
