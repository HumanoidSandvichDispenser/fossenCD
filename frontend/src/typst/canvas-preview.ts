// Display size is driven by an explicit zoom level (an A4 page is 595x842 CSS
// px at 100%). The library fits the canvas to `container.offsetWidth` via a
// CSS transform, so display size is controlled by rendering into a `mount`
// element whose parent `wrap` is sized to `pageWidthPt * zoom`. The library
// makes `mount` 100% of `wrap`, reads that width, and scales accordingly.

import { measure } from './perf';
import type { CompilerPort } from './compiler';
import type { TypstDiagnostic } from './diagnostics';
import type { RenderSession, TypstRenderer } from '@myriaddreamin/typst.ts';

/** Stable identity of the page layout, to detect when a rebuild is required. */
function pagesSignature(pages: { width: number; height: number }[]): string {
  return pages.map((p) => `${Math.round(p.width)}x${Math.round(p.height)}`).join('|');
}

const BACKGROUND = '#ffffff';
// Pre-raster pages within this margin of the viewport so scrolling rarely shows
// a stale page before it repaints.
const ROOT_MARGIN = '300px 0px';

export class CanvasPreview {
  private readonly wrap: HTMLElement;
  private readonly mount: HTMLElement;

  private canvases: HTMLCanvasElement[] = [];
  private pageSig = '';
  private dirty: boolean[] = [];
  private readonly visible = new Set<number>();
  private observer: IntersectionObserver | null = null;
  private destroyed = false;

  constructor(
    private readonly scroller: HTMLElement,
    host: HTMLElement,
    private readonly compiler: CompilerPort,
    private zoom = 1,
  ) {
    this.wrap = document.createElement('div');
    this.wrap.className = 'typst-zoom-wrap';
    this.mount = document.createElement('div');
    this.wrap.appendChild(this.mount);
    host.appendChild(this.wrap);
  }

  /**
   * Set the display zoom (1 = natural point size). Forces the next render to
   * rebuild at the new resolution.
   */
  setZoom(zoom: number): void {
    if (zoom === this.zoom) {
      return;
    }
    this.zoom = zoom;
    this.pageSig = ''; // force a full rebuild at the new pixelPerPt
  }

  /**
   * Compile `mainFile` and update the preview, returning any diagnostics
   * (warnings) the compile emitted. Cheap when the layout and zoom are stable
   * (re-rasters only visible pages); falls back to a full rebuild otherwise.
   */
  async render(mainFile: string): Promise<TypstDiagnostic[]> {
    const { artifact, diagnostics } = await this.compiler.compile(mainFile);
    if (this.destroyed) {
      return diagnostics;
    }
    const { renderer, session } = await this.compiler.getSession();
    if (this.destroyed) {
      return diagnostics;
    }
    await measure('raster', async () => {
      await measure('manip', () => {
        renderer.manipulateData({ renderSession: session, action: 'reset', data: artifact });
      });
      const pages = session.retrievePagesInfo();
      const sig = pagesSignature(pages);

      if (sig !== this.pageSig || this.canvases.length !== pages.length) {
        // First render, layout change, or zoom change: rebuild everything.
        await this.rebuild(renderer, session, pages);
        this.pageSig = sig;
        return;
      }

      // Same layout & zoom: every page's pixels are now stale. Visible pages
      // re-raster now; the rest repaint when scrolled to.
      this.dirty = pages.map(() => true);
      await this.rasterVisible();
    });
    return diagnostics;
  }

  /** Full render of all pages at the current zoom into freshly built canvases. */
  private async rebuild(
    renderer: TypstRenderer,
    session: RenderSession,
    pages: { width: number; height: number }[],
  ): Promise<void> {
    const first = pages[0];
    if (first !== undefined) {
      // Size the wrapper to the page's natural width (pt) times zoom; the
      // library fits the canvas to this, so 1pt renders as `zoom` CSS px,
      // independent of the pane.
      this.wrap.style.width = `${first.width * this.zoom}px`;
    }
    // `renderToCanvas` clears the mount (innerHTML='') and rebuilds every page,
    // which momentarily collapses the scroller to ~0 height and clamps its
    // scrollTop to 0. Save and restore it so a rebuild keeps the scroll position.
    const savedScrollTop = this.scroller.scrollTop;
    await renderer.renderToCanvas({
      renderSession: session,
      container: this.mount,
      pixelPerPt: this.compiler.pixelPerPt(this.zoom),
      backgroundColor: BACKGROUND,
    });
    this.canvases = [...this.mount.querySelectorAll('canvas')];
    this.dirty = pages.map(() => false);
    this.attachObserver();
    this.scroller.scrollTop = savedScrollTop;
  }

  /** Re-raster every currently-visible page that is still dirty. */
  private async rasterVisible(): Promise<void> {
    const pending = [...this.visible].filter((i) => this.dirty[i]).map((i) => this.rasterPage(i));
    await Promise.all(pending);
  }

  /** Draw one page into its existing canvas via the per-page render path. */
  private async rasterPage(index: number): Promise<void> {
    const canvas = this.canvases[index];
    if (canvas === undefined || this.destroyed) {
      return;
    }
    const ctx = canvas.getContext('2d');
    if (ctx === null) {
      return;
    }
    const { renderer, session } = await this.compiler.getSession();
    if (this.destroyed) {
      return;
    }
    await measure('rasterPage', () =>
      renderer.renderCanvas({
        renderSession: session,
        canvas: ctx,
        pageOffset: index,
        pixelPerPt: this.compiler.pixelPerPt(this.zoom),
        backgroundColor: BACKGROUND,
      }),
    );
    this.dirty[index] = false;
  }

  /**
   * Observe the page wrappers so pages scrolled into view get re-rastered if
   * they went stale while off-screen. Re-attached after every full rebuild
   * because the wrappers are recreated.
   */
  private attachObserver(): void {
    this.observer?.disconnect();
    this.visible.clear();
    const wrappers = [...this.mount.querySelectorAll<HTMLElement>('.typst-page')];
    this.observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          const index = Number((entry.target as HTMLElement).dataset.pageIndex);
          if (Number.isNaN(index)) {
            continue;
          }
          if (entry.isIntersecting) {
            this.visible.add(index);
            if (this.dirty[index]) {
              void this.rasterPage(index);
            }
          } else {
            this.visible.delete(index);
          }
        }
      },
      { root: this.scroller, rootMargin: ROOT_MARGIN },
    );
    wrappers.forEach((wrapper, index) => {
      wrapper.dataset.pageIndex = String(index);
      this.observer?.observe(wrapper);
    });
  }

  destroy(): void {
    this.destroyed = true;
    this.observer?.disconnect();
    this.observer = null;
    this.wrap.remove();
  }
}
