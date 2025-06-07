// Vite config for React + Go WASM (WASM threads ready)
// Ensures COOP/COEP headers for cross-origin isolation (WASM threads, SharedArrayBuffer)
// See: https://web.dev/coop-coep/
// If you see a type error for 'http', run: yarn add -D @types/node

import { defineConfig } from 'vite';
import type { PluginOption } from 'vite';
import type { ServerResponse } from 'node:http';
import react from '@vitejs/plugin-react';

// --- WASM Threads: Required headers for SharedArrayBuffer and threading ---
// See: https://web.dev/coop-coep/ and Go WASM docs
const coopCoepHeaders = [
  ['Cross-Origin-Opener-Policy', 'same-origin'],
  ['Cross-Origin-Embedder-Policy', 'require-corp']
];

// Plugin to set COOP/COEP headers for all responses (HTML, JS, WASM, etc.)
function coopCoepPlugin(): PluginOption {
  return {
    name: 'set-coop-coep-headers',
    configureServer(server) {
      server.middlewares.use((_, res: ServerResponse, next) => {
        for (const [key, value] of coopCoepHeaders) {
          res.setHeader(key, value);
        }
        next();
      });
    }
  };
}

export default defineConfig({
  plugins: [react(), coopCoepPlugin()],
  server: {
    port: 5173,
    cors: true
  },
  build: {
    outDir: 'dist',
    assetsDir: ''
  }
});
