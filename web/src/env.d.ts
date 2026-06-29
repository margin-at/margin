/// <reference types="astro/client" />

declare module "virtual:lexicons" {
  /** Raw lexicon JSON text, keyed by NSID id (e.g. "at.margin.note"). */
  export const lexicons: Record<string, string>;
}

declare module "virtual:i18n-resources" {
  export const resources: Record<
    string,
    { translation: Record<string, unknown> }
  >;
}

declare module "virtual:i18n-languages" {
  export const languages: { code: string; name: string; nativeName: string }[];
}

declare namespace App {
  interface Locals {
    user: import("./types").UserProfile | null;
  }
}

interface Window {
  __posthog_initialized?: boolean;
  posthog?: import("posthog-js").PostHog;
}
