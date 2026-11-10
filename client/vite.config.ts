import { defineConfig } from 'vite';

// A fixed port (the back allows websockets from it), and /api, HTTP and websocket, proxied to the back: no CORS.
export default defineConfig({
  server: {
    port: 5180,
    strictPort: true,
    proxy: {
      '/api': { target: 'http://localhost:8400', ws: true },
    },
  },
});
