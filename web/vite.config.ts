/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// The Go app embeds the built UI from internal/webui/ui/dist (see internal/webui).
// `go run ./tools/build dev` runs this dev server next to the Go API and sets DASHCAM_API.
const api = process.env.DASHCAM_API ?? 'http://127.0.0.1:8765';

export default defineConfig({
  plugins: [react()],
  base: './',
  build: {
    outDir: '../internal/webui/ui/dist',
    emptyOutDir: true,
    assetsInlineLimit: 0,
    chunkSizeWarningLimit: 900,
  },
  server: {
    proxy: {
      '/api': api,
      '/media': api,
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
});
