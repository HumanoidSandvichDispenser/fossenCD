// TODO: we could decouple some of this and make a module dedicated for the shadow fs

import { $typst } from '@myriaddreamin/typst.ts';
import compilerModule from '@myriaddreamin/typst-ts-web-compiler/pkg/typst_ts_web_compiler_bg.wasm?url';
import rendererModule from '@myriaddreamin/typst-ts-renderer/pkg/typst_ts_renderer_bg.wasm?url';

let configured = false;

/**
 * Configure the compiler and renderer with the correct WASM modules. This is
 * done lazily but should be done before any other calls to the compiler or
 * renderer.
 */
function configure() {
  if (configured) {
    return;
  }
  configured = true;
  $typst.setCompilerInitOptions({ getModule: () => compilerModule });
  $typst.setRendererInitOptions({ getModule: () => rendererModule });
}

/**
 * Convert a file path to the format expected by the compiler. The compiler
 * expects all paths to be absolute, so we prefix them with a slash if they
 * aren't already.
 */
function vfsPath(file: string): string {
  return file.startsWith('/') ? file : `/${file}`;
}

/**
 * Last synced content for each file. Used to avoid unnecessary re-syncs.
 */
const synced = new Map<string, string>();

/**
 * Make a file's current text available to the compiler. Every opened file in
 * the project is synced.
 */
export async function syncSource(file: string, content: string): Promise<void> {
  configure();
  const path = vfsPath(file);
  if (synced.get(path) === content) {
    return;
  }
  synced.set(path, content);
  await $typst.addSource(path, content);
}

/**
 * Drop a file from the shadow FS (e.g. when it is removed from the project).
 */
export async function dropSource(file: string): Promise<void> {
  configure();
  const path = vfsPath(file);
  if (!synced.has(path)) {
    return;
  }
  synced.delete(path);
  await $typst.unmapShadow(path);
}

/**
 * Clear the entire shadow FS. Useful when switching projects, to avoid stale
 * files lingering.
 */
export async function resetSources(): Promise<void> {
  configure();
  synced.clear();
  await $typst.resetShadow();
}

/**
 * Compile a project file to an SVG string, resolving any imports/includes
 * against the files synced via {@link syncSource}. Rejects if the source fails
 * to compile (the message is the compiler's diagnostic output). NOTE: not
 * incremental and lags like balls when the project is big.
 */
export function renderTypstFile(mainFile: string): Promise<string> {
  configure();
  return $typst.svg({ mainFilePath: vfsPath(mainFile) });
}
