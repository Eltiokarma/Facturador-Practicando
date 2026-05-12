import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      manifest: {
        name: "Facturador",
        short_name: "Facturador",
        description: "Facturador electrónico self-hosted Perú",
        theme_color: "#0f172a",
        background_color: "#0f172a",
        display: "standalone",
        start_url: "/",
        icons: [],
      },
    }),
  ],
  server: {
    host: "0.0.0.0",
    port: 5173,
  },
});
