export type FontCategory = "sans" | "serif" | "mono";

interface FontDef {
  name: string;
  google?: string;
  category: FontCategory;
}

interface FontGroup {
  label: string;
  fonts: FontDef[];
}

export const FONT_GROUPS: FontGroup[] = [
  {
    label: "System",
    fonts: [
      { name: "sans-serif", category: "sans" },
      { name: "serif", category: "serif" },
      { name: "monospace", category: "mono" },
    ],
  },
  {
    label: "Sans-serif",
    fonts: [
      { name: "Inter", google: "Inter:wght@400;500;600;700", category: "sans" },
      { name: "Geist", google: "Geist:wght@400;500;600;700", category: "sans" },
      {
        name: "Manrope",
        google: "Manrope:wght@400;500;600;700",
        category: "sans",
      },
      {
        name: "Plus Jakarta Sans",
        google: "Plus+Jakarta+Sans:ital,wght@0,400;0,500;0,600;0,700;1,400",
        category: "sans",
      },
      {
        name: "Space Grotesk",
        google: "Space+Grotesk:wght@400;500;600;700",
        category: "sans",
      },
      { name: "Sora", google: "Sora:wght@400;500;600;700", category: "sans" },
      {
        name: "Figtree",
        google: "Figtree:ital,wght@0,400;0,500;0,600;0,700;1,400",
        category: "sans",
      },
      {
        name: "Outfit",
        google: "Outfit:wght@400;500;600;700",
        category: "sans",
      },
      {
        name: "Schibsted Grotesk",
        google: "Schibsted+Grotesk:ital,wght@0,400;0,500;0,600;0,700;1,400",
        category: "sans",
      },
      {
        name: "Bricolage Grotesque",
        google: "Bricolage+Grotesque:wght@400;500;600;700",
        category: "sans",
      },
    ],
  },
  {
    label: "Serif",
    fonts: [
      {
        name: "Fraunces",
        google: "Fraunces:ital,wght@0,400;0,500;0,600;0,700;1,400",
        category: "serif",
      },
      {
        name: "Newsreader",
        google: "Newsreader:ital,wght@0,400;0,500;0,600;1,400",
        category: "serif",
      },
      {
        name: "Source Serif 4",
        google: "Source+Serif+4:ital,wght@0,400;0,500;0,600;1,400",
        category: "serif",
      },
      {
        name: "Spectral",
        google: "Spectral:ital,wght@0,400;0,500;0,600;1,400",
        category: "serif",
      },
      {
        name: "Lora",
        google: "Lora:ital,wght@0,400;0,500;0,600;1,400",
        category: "serif",
      },
      {
        name: "Libre Baskerville",
        google: "Libre+Baskerville:ital,wght@0,400;0,700;1,400",
        category: "serif",
      },
      {
        name: "EB Garamond",
        google: "EB+Garamond:ital,wght@0,400;0,500;0,600;1,400",
        category: "serif",
      },
      {
        name: "Crimson Pro",
        google: "Crimson+Pro:ital,wght@0,400;0,500;0,600;1,400",
        category: "serif",
      },
      {
        name: "Merriweather",
        google: "Merriweather:ital,wght@0,400;0,700;1,400",
        category: "serif",
      },
      {
        name: "Playfair Display",
        google: "Playfair+Display:ital,wght@0,400;0,500;0,600;1,400",
        category: "serif",
      },
      {
        name: "Instrument Serif",
        google: "Instrument+Serif:ital@0;1",
        category: "serif",
      },
      {
        name: "DM Serif Display",
        google: "DM+Serif+Display:ital@0;1",
        category: "serif",
      },
    ],
  },
  {
    label: "Monospace",
    fonts: [
      {
        name: "JetBrains Mono",
        google: "JetBrains+Mono:ital,wght@0,400;0,500;0,700;1,400",
        category: "mono",
      },
      {
        name: "IBM Plex Mono",
        google: "IBM+Plex+Mono:ital,wght@0,400;0,500;0,600;1,400",
        category: "mono",
      },
      {
        name: "Space Mono",
        google: "Space+Mono:ital,wght@0,400;0,700;1,400",
        category: "mono",
      },
    ],
  },
];

const BY_NAME: Record<string, FontDef> = {};
for (const group of FONT_GROUPS) {
  for (const font of group.fonts) BY_NAME[font.name] = font;
}

const GENERIC_STACKS: Record<FontCategory, string> = {
  sans: 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", Helvetica, Arial, sans-serif',
  serif: 'ui-serif, Georgia, Cambria, "Times New Roman", serif',
  mono: 'ui-monospace, "SF Mono", Menlo, Consolas, monospace',
};

export function fontStack(family?: string): string {
  if (!family) return GENERIC_STACKS.sans;
  const def = BY_NAME[family];
  if (!def) return GENERIC_STACKS.sans;
  if (!def.google) return GENERIC_STACKS[def.category];
  return `"${family}", ${GENERIC_STACKS[def.category]}`;
}

export function ensureFontLoaded(family?: string): void {
  if (!family || typeof document === "undefined") return;
  const def = BY_NAME[family];
  if (!def?.google) return;
  const id = `gfont-${family.replace(/\s+/g, "-").toLowerCase()}`;
  if (document.getElementById(id)) return;
  const link = document.createElement("link");
  link.id = id;
  link.rel = "stylesheet";
  link.href = `https://fonts.googleapis.com/css2?family=${def.google}&display=swap`;
  document.head.appendChild(link);
}
