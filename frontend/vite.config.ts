// Vite config for React + Go WASM (WASM threads ready)
// Ensures COOP/COEP headers for cross-origin isolation (WASM threads, SharedArrayBuffer)
// See: https://web.dev/coop-coep/
// If you see a type error for 'http', run: yarn add -D @types/node

import { defineConfig } from 'vite';
import type { PluginOption } from 'vite';
import type { ServerResponse } from 'node:http';
import react from '@vitejs/plugin-react';
import checker from 'vite-plugin-checker';
import { resolve } from 'path';

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
  plugins: [
    react(),
    coopCoepPlugin(),
    checker({ typescript: true }) // TypeScript type checking overlay
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@protos': resolve(__dirname, './protos')
    }
  },
  server: {
    port: Number(process.env.VITE_PORT) || 5173,
    cors: {
      origin: ['http://localhost:5173', 'http://127.0.0.1:5173'],
      credentials: true
    },
    open: true,
    proxy: {
      // Proxy API requests to Go backend (default: 8080)
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: false
      },
      // Proxy WebSocket requests to ws-gateway (default: 8090)
      '/ws': {
        target: 'ws://localhost:8090',
        ws: true,
        changeOrigin: true
      }
    }
  },
  assetsInclude: ['**/*.wasm'],
  build: {
    outDir: 'dist',
    assetsDir: '',
    sourcemap: true,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name].[hash].js',
        chunkFileNames: 'assets/[name].[hash].js',
        assetFileNames: 'assets/[name].[hash].[ext]'
      }
    }
  }
});
