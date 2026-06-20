// Typst compile + render orchestration for the live preview.
//
// The compiler emits a vector-IR artifact, which is loaded into a persistent
// RenderSession that the renderer rasterizes straight to canvas elements.
//
// Two layers stay hot for incremental speed:
//   - Compilation: one compiler plus a shadow FS mutated in place, so comemo
//     memoization keeps warm recompiles cheap.
//   - Rendering: one RenderSession lives for the whole app lifetime, so the
//     renderer diffs against prior state instead of rebuilding every frame.
//
// The session is held for that lifetime by never resolving the callback passed
// to runWithSession, which otherwise frees the session once the callback settles.

import { $typst } from '@myriaddreamin/typst.ts';
import { CompileFormatEnum } from '@myriaddreamin/typst.ts/compiler';
import type { RenderSession, TypstRenderer } from '@myriaddreamin/typst.ts';
import compilerModule from '@myriaddreamin/typst-ts-web-compiler/pkg/typst_ts_web_compiler_bg.wasm?url';
import rendererModule from '@myriaddreamin/typst-ts-renderer/pkg/typst_ts_renderer_bg.wasm?url';
import { measure } from './perf';

let configured = false;

/**
 * Configure the compiler and renderer with the correct WASM modules. Lazy and
 * idempotent; runs before any compiler or renderer use.
 */
export function configure() {
  if (configured) {
    return;
  }
  configured = true;
  $typst.setCompilerInitOptions({ getModule: () => compilerModule });
  $typst.setRendererInitOptions({ getModule: () => rendererModule });
}

/**
 * Convert a file path to the format the compiler expects: an absolute path,
 * prefixed with a slash if it isn't already.
 */
export function vfsPath(file: string): string {
  return file.startsWith('/') ? file : `/${file}`;
}

/**
 * The persistent render session, created once and reused for every render so
 * the renderer can keep its per-document state warm. `null` until the first
 * render forces creation.
 */
let sessionPromise: Promise<{ renderer: TypstRenderer; session: RenderSession }> | null = null;

/**
 * Get the long-lived renderer + session pair, creating it on first use.
 *
 * `runWithSession` frees the session once its callback resolves; the callback
 * here never resolves, so the session lives for the app's lifetime.
 */
export function getSession(): Promise<{ renderer: TypstRenderer; session: RenderSession }> {
  if (sessionPromise === null) {
    sessionPromise = (async () => {
      configure();
      const renderer = await $typst.getRenderer();
      const session = await new Promise<RenderSession>((resolveSession) => {
        void renderer.runWithSession((s) => {
          resolveSession(s);
          // Never resolves: keeps `s` alive instead of letting runWithSession
          // free it. This promise is the session's lifetime.
          return new Promise<void>(() => {});
        });
      });
      return { renderer, session };
    })();
  }
  return sessionPromise;
}

/** Join the compiler's structured diagnostics into a single message string. */
function formatDiagnostics(diags: string[] | undefined): string {
  if (diags === undefined || diags.length === 0) {
    return 'Typst compilation failed';
  }
  return diags.join('\n');
}

/**
 * Compile `mainFile` to a Typst vector-IR artifact, resolving imports/includes
 * against the files in the shadow FS. Rejects with the compiler's diagnostics if
 * the source fails to compile.
 */
export function compileVector(mainFile: string): Promise<Uint8Array> {
  return measure('compile', async () => {
    const compiler = await $typst.getCompiler();
    const out = await compiler.compile({
      mainFilePath: vfsPath(mainFile),
      format: CompileFormatEnum.vector,
      diagnostics: 'unix',
    });
    if (out.result === undefined) {
      throw new Error(formatDiagnostics(out.diagnostics));
    }
    return out.result;
  });
}

/** Upper bound on render resolution, to cap canvas backing-store memory at
 *  high zoom (a single A4 page at this density is ~32M px / ~128MB). */
const MAX_PIXEL_PER_PT = 8;

/**
 * Raster resolution (device pixels per Typst point) for a given display zoom.
 *
 * The page is displayed at `zoom` CSS px per point, and we want ~one device
 * pixel per displayed pixel for crispness, so `pixelPerPt = zoom * dpr`. The
 * renderer draws each page at `pageWidthPt * pixelPerPt` backing pixels and the
 * library's fit transform scales it down by `1/dpr` to the displayed size.
 *
 * Clamped to {@link MAX_PIXEL_PER_PT} to keep zoomed-in backing stores bounded.
 */
export function pixelPerPt(zoom: number): number {
  const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
  return Math.min(Math.max(zoom * dpr, 1), MAX_PIXEL_PER_PT);
}

/** The compile + render-session capability needed to rasterize a document. */
export interface CompilerPort {
  compile(mainFile: string): Promise<Uint8Array>;
  getSession(): Promise<{ renderer: TypstRenderer; session: RenderSession }>;
  pixelPerPt(zoom: number): number;
}

/** {@link CompilerPort} backed by the `$typst` compiler and render session. */
export const compilerPort: CompilerPort = {
  compile: compileVector,
  getSession,
  pixelPerPt,
};
