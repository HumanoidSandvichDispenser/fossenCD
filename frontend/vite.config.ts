import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import vueDevTools from 'vite-plugin-vue-devtools';
import wasm from 'vite-plugin-wasm';

// https://vite.dev/config/
export default defineConfig({
  // wasm() lets Vite consume codemirror-lang-typst's `--target=bundler` wasm
  // parser (ESM .wasm import), and the okayeg-wasm loader's top-level await.
  // The modern build target and dev server support both natively.
  plugins: [vue(), vueDevTools(), wasm()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  optimizeDeps: {
    // a published wasm pkg: keep it out of the pre-bundle so `?url` and the
    // wasm import resolve against the real files, not a flattened copy
    exclude: ['@sandvichxyz/okayeg-wasm'],
  },
  server: {
    fs: {
      allow: ['..'],
    },
  },
});
