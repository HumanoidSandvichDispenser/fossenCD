import { ref } from 'vue';
import { compilePdf } from '@/typst/compiler';
import { syncAll } from '@/typst/shadow-fs';
import type { VirtualFs } from '@/vfs';

function pdfName(mainFile: string): string {
  const base = mainFile.split('/').pop() ?? 'document.typ';
  return `${base.replace(/\.typ$/i, '')}.pdf`;
}

function downloadBytes(bytes: Uint8Array, filename: string): void {
  const blob = new Blob([bytes as BlobPart], { type: 'application/pdf' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function useExportPdf() {
  const exporting = ref(false);
  const error = ref<string | null>(null);

  /**
   * Export `mainFile` to PDF, syncing all files in the VFS first. Sets `exporting`
   * to true while the export is in progress, and `error` if it fails.
   */
  async function exportPdf(mainFile: string, vfs: VirtualFs) {
    if (exporting.value) {
      return;
    }
    exporting.value = true;
    error.value = null;
    try {
      await syncAll(vfs);
      const bytes = await compilePdf(mainFile);
      downloadBytes(bytes, pdfName(mainFile));
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      exporting.value = false;
    }
  }

  return { exporting, error, exportPdf };
}
