import { useEffect, useRef, useState } from "react";
import { Check, ChevronDown } from "lucide-react";
import { FONT_GROUPS, fontStack, ensureFontLoaded } from "../../lib/fonts";

export default function FontPicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    ensureFontLoaded(value);
  }, [value]);

  useEffect(() => {
    if (!open) return;
    FONT_GROUPS.forEach((g) =>
      g.fonts.forEach((f) => ensureFontLoaded(f.name)),
    );
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node))
        setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="w-full flex items-center justify-between gap-2 text-base px-3 py-2.5 rounded-lg bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-surface-900 dark:text-white cursor-pointer hover:border-surface-300 dark:hover:border-surface-600 transition-colors"
      >
        <span className="truncate" style={{ fontFamily: fontStack(value) }}>
          {value}
        </span>
        <ChevronDown
          size={16}
          className={`shrink-0 text-surface-400 transition-transform duration-200 ${
            open ? "rotate-180" : ""
          }`}
        />
      </button>

      {open && (
        <div
          role="listbox"
          className="absolute z-30 mt-1.5 w-full max-h-72 overflow-y-auto custom-scrollbar rounded-xl border border-surface-200 dark:border-surface-700 bg-white dark:bg-surface-800 shadow-lg py-1.5 animate-scale-in origin-top"
        >
          {FONT_GROUPS.map((group) => (
            <div key={group.label}>
              <div className="px-3 pt-2 pb-1 text-[11px] font-semibold uppercase tracking-wider text-surface-400 dark:text-surface-500">
                {group.label}
              </div>
              {group.fonts.map((f) => {
                const selected = f.name === value;
                return (
                  <button
                    key={f.name}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => {
                      onChange(f.name);
                      setOpen(false);
                    }}
                    className={`w-full flex items-center justify-between gap-2 px-3 py-2 text-left text-lg transition-colors ${
                      selected
                        ? "bg-primary-50 dark:bg-primary-500/10 text-primary-700 dark:text-primary-300"
                        : "text-surface-800 dark:text-surface-100 hover:bg-surface-100 dark:hover:bg-surface-700/50"
                    }`}
                    style={{ fontFamily: fontStack(f.name) }}
                  >
                    <span className="truncate">{f.name}</span>
                    {selected && <Check size={16} className="shrink-0" />}
                  </button>
                );
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
