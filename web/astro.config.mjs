// @ts-check
import { defineConfig } from "astro/config";
import react from "@astrojs/react";
import tailwind from "@astrojs/tailwind";
import node from "@astrojs/node";
import { fileURLToPath } from "url";
import { dirname, resolve, join } from "path";
import { readdirSync, existsSync, readFileSync } from "fs";

const __dirname = dirname(fileURLToPath(import.meta.url));

const API_PORT = process.env.API_PORT || 8081;

function i18nResourcesPlugin() {
  const virtualId = "virtual:i18n-resources";
  const resolvedId = "\0" + virtualId;
  return {
    name: "i18n-resources",
    /** @param {string} id */
    resolveId(id) {
      if (id === virtualId) return resolvedId;
    },
    /** @param {string} id */
    load(id) {
      if (id !== resolvedId) return;
      const localesDir = join(__dirname, "public/locales");
      const resources = /** @type {Record<string, unknown>} */ ({});
      readdirSync(localesDir, { withFileTypes: true })
        .filter(
          (d) =>
            d.isDirectory() &&
            existsSync(join(localesDir, d.name, "translation.json")),
        )
        .forEach((d) => {
          const content = readFileSync(
            join(localesDir, d.name, "translation.json"),
            "utf-8",
          );
          resources[d.name] = { translation: JSON.parse(content) };
        });
      return `export const resources = ${JSON.stringify(resources)};`;
    },
  };
}

function i18nLanguagesPlugin() {
  const virtualId = "virtual:i18n-languages";
  const resolvedId = "\0" + virtualId;
  return {
    name: "i18n-languages",
    /** @param {string} id */
    resolveId(id) {
      if (id === virtualId) return resolvedId;
    },
    /** @param {string} id */
    load(id) {
      if (id !== resolvedId) return;
      const localesDir = join(__dirname, "public/locales");
      const languages = readdirSync(localesDir, { withFileTypes: true })
        .filter(
          (d) =>
            d.isDirectory() &&
            existsSync(join(localesDir, d.name, "translation.json")),
        )
        .map((d) => {
          const code = d.name;
          const name =
            new Intl.DisplayNames(["en"], { type: "language" }).of(code) ??
            code;
          const nativeName =
            new Intl.DisplayNames([code], { type: "language" }).of(code) ??
            name;
          return { code, name, nativeName };
        })
        .sort((a, b) => a.name.localeCompare(b.name));
      return `export const languages = ${JSON.stringify(languages)};`;
    },
  };
}

// https://astro.build/config
export default defineConfig({
  output: "server",
  adapter: node({ mode: "standalone" }),
  integrations: [react(), tailwind()],
  security: { checkOrigin: false },
  prefetch: {
    prefetchAll: true,
    defaultStrategy: "viewport",
  },
  vite: {
    plugins: [i18nResourcesPlugin(), i18nLanguagesPlugin()],
    resolve: {
      alias: {
        "@": resolve(__dirname, "src"),
      },
    },
    ssr: {
      external: ["@resvg/resvg-js"],
    },
    build: {
      commonjsOptions: {
        transformMixedEsModules: true,
      },
      chunkSizeWarningLimit: 1000,
    },
    server: {
      proxy: {
        "/api": {
          target: `http://127.0.0.1:${API_PORT}`,
          changeOrigin: true,
        },
        "/auth": {
          target: `http://127.0.0.1:${API_PORT}`,
          changeOrigin: true,
        },
      },
    },
  },
});
