import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import vueDevTools from 'vite-plugin-vue-devtools';
import wasm from 'vite-plugin-wasm';

// https://vite.dev/config/
export default defineConfig({
  // wasm() lets Vite consume codemirror-lang-typst's `--target=bundler` wasm
  // parser (ESM .wasm import). The generated loader uses top-level await,
  // which the modern build target and dev server support natively.
  plugins: [vue(), vueDevTools(), wasm()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  optimizeDeps: {
    exclude: ['@sandvichxyz/teamtype-wasm'],
  },
  server: {
    fs: {
      allow: ['..'],
    },
  },
});
