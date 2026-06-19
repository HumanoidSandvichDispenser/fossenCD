import { $typst } from '@myriaddreamin/typst.ts';
// The compiler and renderer WASM are bundled (not loaded from a CDN) so the
// editor stays self-hostable and works offline once the page has loaded.
import compilerModule from '@myriaddreamin/typst-ts-web-compiler/pkg/typst_ts_web_compiler_bg.wasm?url';
import rendererModule from '@myriaddreamin/typst-ts-renderer/pkg/typst_ts_renderer_bg.wasm?url';

let configured = false;

// Point the global `$typst` instance at our bundled WASM. The instance
// initializes lazily on first use, so this only has to run once before the
// first `svg()` call.
function configure() {
  if (configured) {
    return;
  }
  configured = true;
  $typst.setCompilerInitOptions({ getModule: () => compilerModule });
  $typst.setRendererInitOptions({ getModule: () => rendererModule });
}

/**
 * Compile Typst source to an SVG string. Rejects if the source fails to
 * compile (the message is the compiler's diagnostic output).
 */
export function renderTypstSvg(source: string): Promise<string> {
  configure();
  return $typst.svg({ mainContent: source });
}
