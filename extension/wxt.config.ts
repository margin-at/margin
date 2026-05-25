import { defineConfig } from 'wxt';
import { cp, mkdir } from 'fs/promises';
import { existsSync } from 'fs';
import { resolve } from 'path';

export default defineConfig({
  srcDir: 'src',
  modules: ['@wxt-dev/module-react'],
  manifestVersion: 3,
  hooks: {
    'build:manifestGenerated': (wxt, manifest) => {
      if (wxt.config.browser === 'safari') {
        delete (manifest as any).side_panel;
        if (Array.isArray(manifest.permissions)) {
          manifest.permissions = manifest.permissions.filter((p) => p !== 'sidePanel');
        }
      }
    },
    'build:done': async (wxt) => {
      const publicDir = resolve(__dirname, 'public');
      const outDir = wxt.config.outDir;

      if (existsSync(publicDir)) {
        await cp(publicDir, outDir, { recursive: true });
      }

      if (wxt.config.browser !== 'safari') {
        const pdfjsBuildSrc = resolve(__dirname, 'node_modules', 'pdfjs-dist', 'build');
        const pdfjsBuildDest = resolve(outDir, 'pdfjs', 'build');
        await mkdir(pdfjsBuildDest, { recursive: true });
        for (const file of ['pdf.mjs', 'pdf.worker.mjs', 'pdf.sandbox.mjs']) {
          await cp(resolve(pdfjsBuildSrc, file), resolve(pdfjsBuildDest, file));
        }
      }
    },
  },
  manifest: ({ browser }) => {
    const basePermissions = ['storage', 'activeTab', 'tabs', 'cookies', 'contextMenus'];
    const chromePermissions = [...basePermissions, 'sidePanel'];
    const isSafari = browser === 'safari';

    return {
      name: 'Margin',
      description:
        'Annotate and highlight any webpage, with your notes saved to the decentralized AT Protocol.',
      permissions: browser === 'chrome' ? chromePermissions : basePermissions,
      host_permissions: ['https://margin.at/*', '*://*/*'],
      ...(!isSafari && {
        web_accessible_resources: [
          {
            resources: ['pdfjs/*'],
            matches: ['<all_urls>'],
          },
        ],
      }),
      icons: {
        16: '/icons/icon-16.png',
        32: '/icons/icon-32.png',
        48: '/icons/icon-48.png',
        128: '/icons/icon-128.png',
      },
      commands: {
        ...(!isSafari && {
          'toggle-sidebar': {
            suggested_key: {
              default: 'Alt+M',
              mac: 'Alt+M',
            },
            description: 'Toggle Margin sidebar',
          },
        }),
        'annotate-selection': {
          suggested_key: {
            default: 'Alt+A',
            mac: 'Alt+A',
          },
          description: 'Annotate selected text',
        },
        'highlight-selection': {
          suggested_key: {
            default: 'Alt+H',
            mac: 'Alt+H',
          },
          description: 'Highlight selected text',
        },
        'bookmark-page': {
          suggested_key: {
            default: 'Alt+B',
            mac: 'Alt+B',
          },
          description: 'Bookmark current page',
        },
      },
      action: {
        default_title: 'Margin',
        default_popup: 'popup.html',
        default_icon: {
          16: '/icons/icon-16.png',
          32: '/icons/icon-32.png',
          48: '/icons/icon-48.png',
          128: '/icons/icon-128.png',
        },
      },
      ...(browser === 'chrome'
        ? {
            side_panel: {
              default_path: 'sidepanel.html',
            },
          }
        : browser === 'firefox'
        ? {
            sidebar_action: {
              default_title: 'Margin',
              default_panel: 'sidepanel.html',
            },
            browser_specific_settings: {
              gecko: {
                id: 'hello@margin.at',
                strict_min_version: '140.0',
                data_collection_permissions: {
                  required: ['none'],
                },
              },
            },
          }
        : {}),
    };
  },
});
