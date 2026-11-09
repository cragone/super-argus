import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': '/src',
    },
  },
  build: {
    outDir: '../internal/webui/dist',
    emptyOutDir: true,
  },
  server: {
    // In dev, the Argus server runs separately (`go run ./cmd/argus`); proxy API/SSE
    // calls to it so the app can use relative, same-origin paths in both dev and prod.
    proxy: {
      '/api': 'http://localhost:9100',
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
});
