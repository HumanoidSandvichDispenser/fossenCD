import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import vueDevTools from 'vite-plugin-vue-devtools';
import wasm from 'vite-plugin-wasm';

// OKAYEG_WASM_LOCAL points at a locally built okayeg-wasm `pkg` dir (from
// `wasm-pack build`). When set, Vite resolves `@sandvichxyz/okayeg-wasm` there
// instead of the installed registry package, so a local wasm build can be used
// without publishing. It lives OUTSIDE node_modules on purpose: the frontend
// container reinstalls deps on start (`bun install`), which would otherwise
// overwrite anything overlaid inside node_modules.
const okayegWasmLocal = process.env.OKAYEG_WASM_LOCAL;

// https://vite.dev/config/
export default defineConfig({
  // wasm() lets Vite consume codemirror-lang-typst's `--target=bundler` wasm
  // parser (ESM .wasm import), and the okayeg-wasm loader's top-level await.
  // The modern build target and dev server support both natively.
  plugins: [vue(), vueDevTools(), wasm()],
  resolve: {
    alias: [
      { find: '@', replacement: fileURLToPath(new URL('./src', import.meta.url)) },
      ...(okayegWasmLocal
        ? [
            // A local wasm-pack pkg has the flat layout and no wasm-url
            // module, so route the subpath import straight at the binary.
            {
              find: '@sandvichxyz/okayeg-wasm/wasm-url',
              replacement: `${okayegWasmLocal}/okayeg_wasm_bg.wasm?url`,
            },
            { find: '@sandvichxyz/okayeg-wasm', replacement: okayegWasmLocal },
          ]
        : []),
    ],
  },
  optimizeDeps: {
    // a published wasm pkg: keep it out of the pre-bundle so `?url` and the
    // wasm import resolve against the real files, not a flattened copy
    exclude: ['@sandvichxyz/okayeg-wasm'],
  },
  server: {
    fs: {
      // allow the project tree, plus the local wasm build when aliased in
      allow: ['..', ...(okayegWasmLocal ? [okayegWasmLocal] : [])],
    },
  },
});
