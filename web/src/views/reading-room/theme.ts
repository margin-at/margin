export interface RRNote {
  id: string;
  motivation?: string;
  color?: string;
  created: string;
  body?: { value?: string };
  target?: {
    source?: string;
    title?: string;
    selector?: { exact?: string; prefix?: string; suffix?: string };
  };
  tags?: string[];
}

export interface RRPalette {
  bg: string;
  ink: string;
  muted: string;
  border: string;
  card: string;
  accent: string;
  accentText: string;
  accentTint: string;
  accentOn: string;
}

export type NoteKind = "highlight" | "note" | "bookmark";

function hexToRgb(hex: string): [number, number, number] | null {
  const m = hex.trim().match(/^#?([0-9a-f]{3}|[0-9a-f]{6})$/i);
  if (!m) return null;
  let h = m[1];
  if (h.length === 3)
    h = h
      .split("")
      .map((c) => c + c)
      .join("");
  const n = parseInt(h, 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

function mix(a: string, b: string, t: number): string {
  const ca = hexToRgb(a);
  const cb = hexToRgb(b);
  if (!ca || !cb) return a;
  return `#${ca
    .map((v, i) =>
      Math.round(v + (cb[i] - v) * t)
        .toString(16)
        .padStart(2, "0"),
    )
    .join("")}`;
}

function relLuminance(hex: string): number {
  const rgb = hexToRgb(hex);
  if (!rgb) return 1;
  const [r, g, b] = rgb.map((v) => {
    const s = v / 255;
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrastRatio(a: string, b: string): number {
  const l1 = relLuminance(a);
  const l2 = relLuminance(b);
  return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
}

function ensureContrast(color: string, bg: string, min: number): string {
  const target = relLuminance(bg) < 0.4 ? "#ffffff" : "#0a0a0a";
  let c = color;
  for (let i = 0; i < 12 && contrastRatio(c, bg) < min; i++) {
    c = mix(c, target, 0.14);
  }
  return c;
}

export function buildPalette(bgIn: string, accentIn: string): RRPalette {
  const bg = hexToRgb(bgIn) ? bgIn : "#fcfcfc";
  const rawAccent = hexToRgb(accentIn) ? accentIn : "#3b82f6";
  const dark = relLuminance(bg) < 0.4;
  const inkTarget = dark ? "#ffffff" : "#0a0a0a";
  const ink = mix(bg, inkTarget, 0.92);
  const accent = ensureContrast(rawAccent, bg, 3);
  return {
    bg,
    ink,
    muted: mix(bg, ink, 0.68),
    border: mix(bg, ink, 0.13),
    card: dark ? mix(bg, "#ffffff", 0.05) : mix(bg, "#ffffff", 0.55),
    accent,
    accentText: ensureContrast(rawAccent, bg, 4.5),
    accentTint: mix(bg, rawAccent, 0.12),
    accentOn:
      contrastRatio("#ffffff", accent) >= contrastRatio("#0a0a0a", accent)
        ? "#ffffff"
        : "#0a0a0a",
  };
}

export function noteKind(n: RRNote): NoteKind {
  if (n.motivation === "highlighting") return "highlight";
  if (n.motivation === "bookmarking") return "bookmark";
  return "note";
}

export function textFragmentUrl(
  source?: string,
  selector?: { exact?: string; prefix?: string; suffix?: string },
): string | null {
  if (!source || !selector?.exact) return null;
  const prefix = selector.prefix
    ? encodeURIComponent(selector.prefix) + "-,"
    : "";
  const suffix = selector.suffix
    ? ",-" + encodeURIComponent(selector.suffix)
    : "";
  return `${source}#:~:text=${prefix}${encodeURIComponent(selector.exact)}${suffix}`;
}

const HIGHLIGHT_TONES: Record<string, string> = {
  yellow: "#fef08a",
  green: "#bbf7d0",
  blue: "#bfdbfe",
  red: "#fecaca",
  pink: "#fbcfe8",
  purple: "#e9d5ff",
  orange: "#fed7aa",
};

export function highlightBg(color: string | undefined, bg: string): string {
  const dark = relLuminance(bg) < 0.4;
  if (color) {
    if (color.startsWith("#") && hexToRgb(color)) {
      return dark ? mix(bg, color, 0.3) : mix("#ffffff", color, 0.4);
    }
    if (HIGHLIGHT_TONES[color]) {
      return dark
        ? mix(bg, HIGHLIGHT_TONES[color], 0.2)
        : HIGHLIGHT_TONES[color];
    }
  }
  return dark ? mix(bg, HIGHLIGHT_TONES.yellow, 0.2) : HIGHLIGHT_TONES.yellow;
}

export function highlightInk(bg: string): string {
  return relLuminance(bg) < 0.4 ? "#f5f5f4" : "#1c1917";
}

export function readingRoomNoteUrl(roomHandle: string, noteId: string): string {
  return `/reading-room/${encodeURIComponent(roomHandle)}/note?uri=${encodeURIComponent(noteId)}`;
}
