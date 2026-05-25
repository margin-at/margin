import { initContentScript } from '@/utils/overlay';

export default defineUnlistedScript(async () => {
  if (import.meta.env.BROWSER === 'safari') return;

  await waitForPdfTextLayer();

  await initContentScript({
    onInvalidated: (cb: () => void) => {
      window.addEventListener('beforeunload', cb);
    },
  });
});

function waitForPdfTextLayer(): Promise<void> {
  return new Promise((resolve) => {
    if (document.querySelectorAll('.textLayer span').length > 0) {
      resolve();
      return;
    }

    const observer = new MutationObserver(() => {
      if (document.querySelectorAll('.textLayer span').length > 0) {
        observer.disconnect();
        setTimeout(resolve, 500);
      }
    });

    observer.observe(document.body || document.documentElement, {
      childList: true,
      subtree: true,
    });

    setTimeout(() => {
      observer.disconnect();
      resolve();
    }, 10000);
  });
}
