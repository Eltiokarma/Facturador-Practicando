import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.png", "apple-touch-icon.png"],
      manifest: {
        name: "Facturador Self-Hosted Perú",
        short_name: "Facturador",
        description:
          "Facturador electrónico self-hosted para Perú. Emite facturas, boletas, notas y guías directo a SUNAT.",
        theme_color: "#0f172a",
        background_color: "#0f172a",
        display: "standalone",
        display_override: ["window-controls-overlay", "standalone"],
        orientation: "any",
        start_url: "/",
        scope: "/",
        lang: "es-PE",
        categories: ["business", "finance", "productivity"],
        icons: [
          {
            src: "/icons/icon-192.png",
            sizes: "192x192",
            type: "image/png",
            purpose: "any",
          },
          {
            src: "/icons/icon-512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "any",
          },
          {
            src: "/icons/icon-maskable-512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
        ],
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,png,svg,ico}"],
        // Las llamadas a la API NO se cachean — necesitamos respuestas frescas.
        navigateFallbackDenylist: [/^\/api\//],
      },
    }),
  ],
  server: {
    host: "0.0.0.0",
    port: 5173,
  },
});
