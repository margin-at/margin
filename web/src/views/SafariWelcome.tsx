import React from "react";
import {
  ArrowRight,
  Settings,
  LayoutGrid,
  KeyRound,
  Search,
  ShieldCheck,
  Globe,
  Puzzle,
  Check,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

const STEPS = [
  {
    title: "Open Safari Settings",
    desc: (
      <>
        Go to{" "}
        <strong className="text-surface-900 dark:text-white">
          Safari {">"} Settings {">"} Extensions
        </strong>
      </>
    ),
  },
  {
    title: "Find Margin in the list",
    desc: (
      <>
        Click{" "}
        <strong className="text-surface-900 dark:text-white">Margin</strong> in
        the left sidebar
      </>
    ),
  },
  {
    title: "Allow on all websites",
    desc: (
      <>
        Click{" "}
        <strong className="text-surface-900 dark:text-white">
          "Always Allow on Every Website..."
        </strong>
      </>
    ),
  },
];

const ANIM = `
  @keyframes sw-btn-pulse {
    0%, 20%   { box-shadow: 0 0 0 0 rgba(2,123,255,0); background: #3a3a3c; color: #e5e5e7; }
    40%        { box-shadow: 0 0 0 4px rgba(2,123,255,0.35); background: #027bff; color: #fff; }
    65%, 85%   { box-shadow: 0 0 0 0 rgba(2,123,255,0); background: #027bff; color: #fff; }
    100%       { box-shadow: 0 0 0 0 rgba(2,123,255,0); background: #3a3a3c; color: #e5e5e7; }
  }
  @media (prefers-reduced-motion: reduce) {
    .sw-anim * { animation: none !important; }
  }
`;

function SafariSettingsDemo() {
  return (
    <div
      className="sw-anim rounded-xl overflow-hidden shadow-2xl"
      style={{ background: "#1c1c1e", border: "1px solid #3a3a3c" }}
    >
      <style dangerouslySetInnerHTML={{ __html: ANIM }} />

      <div
        className="flex items-center px-4 py-3"
        style={{ background: "#2c2c2e", borderBottom: "1px solid #3a3a3c" }}
      >
        <div className="flex gap-1.5">
          <span className="w-3 h-3 rounded-full bg-[#ff5f57]" />
          <span className="w-3 h-3 rounded-full bg-[#febc2e]" />
          <span className="w-3 h-3 rounded-full bg-[#28c840]" />
        </div>
        <span
          className="flex-1 text-center text-[13px] font-medium"
          style={{ color: "#e5e5e7" }}
        >
          Extensions
        </span>
        <div className="w-14" />
      </div>

      <div
        className="flex items-end gap-1 px-3 pt-2 overflow-x-auto"
        style={{ background: "#2c2c2e", borderBottom: "1px solid #3a3a3c" }}
      >
        {(
          [
            { label: "General", Icon: Settings },
            { label: "Tabs", Icon: LayoutGrid },
            { label: "AutoFill", Icon: KeyRound },
            { label: "Search", Icon: Search },
            { label: "Security", Icon: ShieldCheck },
            { label: "Privacy", Icon: Globe },
            { label: "Extensions", Icon: Puzzle, active: true },
          ] as { label: string; Icon: LucideIcon; active?: boolean }[]
        ).map(({ label, Icon, active }) => (
          <div
            key={label}
            className="flex flex-col items-center gap-0.5 px-2 pb-1.5 pt-1 flex-shrink-0 rounded-t"
            style={
              active
                ? {
                    background: "#1c1c1e",
                    borderTop: "1px solid #3a3a3c",
                    borderLeft: "1px solid #3a3a3c",
                    borderRight: "1px solid #3a3a3c",
                    marginBottom: "-1px",
                  }
                : {}
            }
          >
            <Icon
              size={15}
              color={active ? "#027bff" : "#8e8e93"}
              strokeWidth={1.5}
            />
            <span
              className="text-[9px]"
              style={{ color: active ? "#027bff" : "#8e8e93" }}
            >
              {label}
            </span>
          </div>
        ))}
      </div>

      <div className="flex" style={{ height: "13rem" }}>
        <div
          className="w-36 flex-shrink-0 py-2 overflow-y-auto"
          style={{ borderRight: "1px solid #3a3a3c" }}
        >
          <div
            className="px-3 py-1 text-[10px] font-semibold uppercase tracking-wide"
            style={{ color: "#8e8e93" }}
          >
            Installed
          </div>
          <div
            className="flex items-center gap-2 px-3 py-1.5 mx-1 rounded"
            style={{ background: "#027bff" }}
          >
            <div
              className="w-4 h-4 rounded flex-shrink-0 flex items-center justify-center"
              style={{ background: "rgba(255,255,255,0.2)" }}
            >
              <Check size={10} color="white" strokeWidth={3} />
            </div>
            <img
              src="/logo.svg"
              className="w-5 h-5 flex-shrink-0 brightness-0 invert"
              alt=""
            />
            <span className="text-[11px] font-medium text-white truncate">
              Margin
            </span>
          </div>
        </div>

        <div
          className="flex-1 px-4 py-3 overflow-y-auto flex flex-col gap-2"
          style={{ background: "#1c1c1e" }}
        >
          <div className="flex items-start gap-3">
            <img src="/logo.svg" className="w-10 h-10 flex-shrink-0" alt="" />
            <div>
              <div
                className="text-[13px] font-semibold"
                style={{ color: "#e5e5e7" }}
              >
                Margin{" "}
                <span
                  className="font-normal text-[11px]"
                  style={{ color: "#8e8e93" }}
                >
                  1.0.0 from Margin
                </span>
              </div>
              <div
                className="text-[11px] leading-snug mt-0.5"
                style={{ color: "#8e8e93" }}
              >
                Annotate and highlight any webpage.
              </div>
            </div>
          </div>

          <div
            className="text-[11px] font-semibold mt-1"
            style={{ color: "#e5e5e7" }}
          >
            Permissions:
          </div>
          <div className="text-[11px]" style={{ color: "#8e8e93" }}>
            <span style={{ color: "#e5e5e7" }}>
              • Webpage Contents and Browsing History
            </span>
            <br />
            Can read webpage contents when you use the extension.
          </div>

          <div className="flex gap-2 mt-1">
            <button
              className="text-[11px] px-2.5 py-1 rounded text-[#e5e5e7] font-medium"
              style={{ background: "#3a3a3c", border: "none" }}
            >
              Edit Websites...
            </button>
            <button
              className="text-[11px] px-2.5 py-1 rounded font-medium"
              style={{ animation: "sw-btn-pulse 4s ease-in-out infinite" }}
            >
              Always Allow on Every Website...
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function SafariWelcome() {
  return (
    <div className="min-h-screen bg-surface-100 dark:bg-surface-900 text-surface-900 dark:text-surface-100 antialiased">
      <div className="max-w-[68rem] mx-auto px-5 sm:px-8 pt-16 md:pt-24 pb-24">
        <div className="flex items-center gap-2.5 mb-12">
          <img src="/logo.svg" alt="Margin" className="w-6 h-6" />
          <span className="font-display font-semibold text-[15px] tracking-tight">
            Margin
          </span>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 lg:gap-20 items-start">
          <div>
            <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-surface-500 dark:text-surface-400 mb-4">
              Setup
            </p>
            <h1 className="font-display font-semibold tracking-[-0.025em] leading-[1.08] text-[2rem] md:text-[2.75rem] text-surface-900 dark:text-white mb-5">
              One step to get started
            </h1>
            <p className="text-[16px] md:text-[17px] leading-[1.7] text-surface-600 dark:text-surface-300 mb-10">
              Margin needs permission to run on all websites so it can show
              annotations and highlights wherever you browse.
            </p>

            <dl className="border-t border-surface-200 dark:border-surface-800 mb-10">
              {STEPS.map(({ title, desc }, i) => (
                <div
                  key={title}
                  className="flex gap-4 py-5 border-b border-surface-200 dark:border-surface-800"
                >
                  <span
                    aria-hidden
                    className="flex-shrink-0 inline-flex items-center justify-center w-7 h-7 rounded-full bg-primary-100 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 ring-1 ring-primary-200/60 dark:ring-primary-500/15 text-[12px] font-mono font-bold mt-0.5"
                  >
                    {i + 1}
                  </span>
                  <div>
                    <dt className="font-display font-semibold text-[1rem] tracking-tight text-surface-900 dark:text-white mb-1">
                      {title}
                    </dt>
                    <dd className="text-[15px] leading-[1.65] text-surface-600 dark:text-surface-400">
                      {desc}
                    </dd>
                  </div>
                </div>
              ))}
            </dl>

            <a
              href="/"
              className="group inline-flex items-center gap-2 px-5 py-3 text-[14.5px] font-semibold bg-[#027bff] text-white rounded-full hover:bg-[#026ae0] transition-colors ring-1 ring-[#027bff]/30 shadow-sm shadow-[#027bff]/20"
            >
              Take me to Margin
              <ArrowRight
                size={14}
                className="transition-transform duration-200 ease-out group-hover:translate-x-0.5"
              />
            </a>
          </div>

          <div className="lg:pt-12">
            <SafariSettingsDemo />
          </div>
        </div>
      </div>
    </div>
  );
}
