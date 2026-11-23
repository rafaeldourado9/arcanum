import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// O build vai direto para internal/dashboard/dist, que o binario Go embute.
// Em desenvolvimento (npm run dev), /api, /docs e /health vao para o gateway na 3100.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "../internal/dashboard/dist",
    emptyOutDir: true,
    sourcemap: false,
  },
  server: {
    port: 5180,
    proxy: {
      "/api": "http://localhost:3100",
      "/docs": "http://localhost:3100",
      "/health": "http://localhost:3100",
    },
  },
});
