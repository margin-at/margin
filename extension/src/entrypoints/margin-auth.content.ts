import { defineContentScript } from 'wxt/sandbox';

export default defineContentScript({
  matches: ['https://margin.at/*'],
  runAt: 'document_idle',

  main() {
    browser.runtime.onMessage.addListener((message: any) => {
      if (message.type === 'SAFARI_FETCH') {
        const { url, options } = message;
        return fetch(url, { ...options, credentials: 'include' })
          .then(async (res) => ({
            ok: res.ok,
            status: res.status,
            body: await res.text(),
          }))
          .catch((err) => ({ ok: false, status: 0, body: String(err) }));
      }
    });
  },
});
