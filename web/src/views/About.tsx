import React from "react";
import { useStore } from "@nanostores/react";
import { useTranslation } from "react-i18next";
import "../i18n";

import { $user } from "../store/auth";
import StaticFooter from "../components/common/StaticFooter";
import {
  ArrowRight,
  ArrowUpRight,
  Chrome,
  Github,
  ChevronDown,
  Copy,
  Check,
  FolderOpen,
  Users,
  Hash,
  Eye,
  MousePointerClick,
  Keyboard,
  PanelRight,
  AtSign,
  Database,
  Braces,
  Code2,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { AppleIcon, TangledIcon } from "../components/common/Icons";
import { FaFirefox, FaEdge, FaSafari } from "react-icons/fa";
import { lexicons } from "virtual:lexicons";
const noteLexiconText = lexicons["at.margin.note"];

function LexiconBlock() {
  const [copied, setCopied] = React.useState(false);
  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(noteLexiconText);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard unavailable */
    }
  };
  return (
    <div className="mt-5 rounded-md bg-surface-950 border border-surface-800 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2 border-b border-surface-800 text-[11px] font-mono text-surface-500">
        <span>at.margin.note</span>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={onCopy}
            className="inline-flex items-center gap-1 hover:text-surface-300 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60 rounded-sm"
            aria-label={copied ? "Copied" : "Copy lexicon"}
          >
            {copied ? <Check size={11} /> : <Copy size={11} />}
            {copied ? "Copied" : "Copy"}
          </button>
          <a
            href="https://github.com/paddinglabs/margin/blob/main/lexicons/at/margin/note.json"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1 hover:text-surface-300 transition-colors"
          >
            Source
            <ArrowUpRight size={11} />
          </a>
        </div>
      </div>
      <pre className="text-[12.5px] leading-[1.6] font-mono p-5 overflow-auto max-h-[28rem] text-surface-300">
        <code
          dangerouslySetInnerHTML={{
            __html: colorizeJson(noteLexiconText),
          }}
        />
      </pre>
    </div>
  );
}

function colorizeJson(raw: string): string {
  const escaped = raw
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
  return escaped.replace(
    /("(?:\\.|[^"\\])*"\s*:?)|\b(true|false|null)\b|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g,
    (m, str, kw, num) => {
      if (str) {
        const isKey = /:\s*$/.test(str);
        return `<span class="${isKey ? "text-emerald-300" : "text-amber-300"}">${str}</span>`;
      }
      if (kw) {
        return `<span class="text-sky-300">${kw}</span>`;
      }
      if (num) {
        return `<span class="text-violet-300">${num}</span>`;
      }
      return m;
    },
  );
}

type Browser = "chrome" | "firefox" | "edge" | "safari" | "other";

function detectBrowser(): Browser {
  if (typeof navigator === "undefined") return "other";
  const ua = navigator.userAgent;
  if (/Edg\//i.test(ua)) return "edge";
  if (/Firefox/i.test(ua)) return "firefox";
  if (/^((?!chrome|android).)*safari/i.test(ua)) return "safari";
  if (/Chrome/i.test(ua)) return "chrome";
  return "other";
}

const STORE_LINKS = {
  chrome:
    "https://chromewebstore.google.com/detail/margin/cgpmbiiagnehkikhcbnhiagfomajncpa",
  firefox: "https://addons.mozilla.org/en-US/firefox/addon/margin/",
  edge: "https://microsoftedge.microsoft.com/addons/detail/margin/nfjnmllpdgcdnhmmggjihjbidmeadddn",
  safari: "https://apps.apple.com/us/app/margin-for-safari/id6773549512",
  ios: "https://www.icloud.com/shortcuts/1e33ebf52f55431fae1e187cfe9738c3",
} as const;

export default function About() {
  const { t } = useTranslation();
  const user = useStore($user);
  const [browser] = React.useState<Browser>(detectBrowser);

  const primaryStore: keyof typeof STORE_LINKS =
    browser === "firefox"
      ? "firefox"
      : browser === "edge"
        ? "edge"
        : browser === "safari"
          ? "safari"
          : "chrome";
  const PrimaryIcon =
    primaryStore === "firefox"
      ? FaFirefox
      : primaryStore === "edge"
        ? FaEdge
        : primaryStore === "safari"
          ? FaSafari
          : Chrome;
  const primaryLabel =
    primaryStore === "firefox"
      ? "Firefox"
      : primaryStore === "edge"
        ? "Edge"
        : primaryStore === "safari"
          ? "Safari"
          : "Chrome";

  const capabilities: { icon: LucideIcon; title: string; desc: string }[] = [
    {
      icon: FolderOpen,
      title: t("about.features.collections.title"),
      desc: t("about.features.collections.description"),
    },
    {
      icon: Users,
      title: t("about.features.socialDiscovery.title"),
      desc: t("about.features.socialDiscovery.description"),
    },
    {
      icon: Hash,
      title: t("about.features.tagsSearch.title"),
      desc: t("about.features.tagsSearch.description"),
    },
    {
      icon: Eye,
      title: t("about.extension.features.inlineOverlay.title"),
      desc: t("about.extension.features.inlineOverlay.description"),
    },
    {
      icon: MousePointerClick,
      title: t("about.extension.features.contextMenu.title"),
      desc: t("about.extension.features.contextMenu.description"),
    },
    {
      icon: Keyboard,
      title: t("about.extension.features.keyboard.title"),
      desc: t("about.extension.features.keyboard.description"),
    },
    {
      icon: PanelRight,
      title: t("about.extension.features.sidePanel.title"),
      desc: t("about.extension.features.sidePanel.description"),
    },
  ];

  const [navHidden, setNavHidden] = React.useState(false);
  React.useEffect(() => {
    let last = typeof window === "undefined" ? 0 : window.scrollY;
    const onScroll = () => {
      const cur = window.scrollY;
      if (cur < 24) {
        setNavHidden(false);
      } else if (cur > last + 4) {
        setNavHidden(true);
      } else if (cur < last - 4) {
        setNavHidden(false);
      }
      last = cur;
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const heroPrimaryHref = user ? "/home" : "/login";
  const heroPrimaryLabel = user
    ? t("about.hero.openApp")
    : t("about.hero.getStarted");

  return (
    <div className="min-h-screen bg-surface-100 dark:bg-surface-900 text-surface-900 dark:text-surface-100 antialiased selection:bg-yellow-200 selection:text-surface-900 dark:selection:bg-yellow-300/40 dark:selection:text-white">
      <nav
        className={`sticky top-3 sm:top-4 z-50 px-3 sm:px-5 transition-transform duration-300 ease-out ${
          navHidden ? "-translate-y-[150%]" : "translate-y-0"
        }`}
      >
        <div className="max-w-[28rem] mx-auto flex items-center justify-between gap-3 bg-white dark:bg-surface-800 ring-1 ring-surface-200 dark:ring-white/10 rounded-full pl-3.5 pr-2 py-2 shadow-md shadow-surface-900/5 dark:shadow-surface-900/15">
          <a href="/" className="group flex items-center">
            <img
              src="/logo.svg"
              alt="Margin"
              className="w-6 h-6 transition-transform duration-300 ease-out group-hover:rotate-[-6deg]"
            />
          </a>

          <div className="flex items-center gap-1">
            {!user && (
              <a
                href="/login"
                className="text-[13px] font-medium text-surface-600 dark:text-surface-300 hover:text-surface-900 dark:hover:text-white transition-colors px-3 py-1.5 rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60"
              >
                {t("nav.signIn")}
              </a>
            )}
            <a
              href={STORE_LINKS[primaryStore]}
              target="_blank"
              rel="noopener noreferrer"
              className="group inline-flex items-center gap-1.5 text-[13px] font-semibold pl-3 pr-3.5 py-1.5 bg-[#027bff] text-white rounded-full hover:bg-[#026ae0] transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#027bff]/60"
            >
              <PrimaryIcon size={13} />
              <span className="hidden sm:inline">
                {t("about.nav.getExtension")}
              </span>
              <span className="sm:hidden">{t("about.nav.install")}</span>
            </a>
          </div>
        </div>
      </nav>

      <header className="relative">
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 pt-16 md:pt-24 pb-16 md:pb-24 text-center">
          <h1 className="font-display font-semibold tracking-[-0.025em] leading-[1.02] text-surface-900 dark:text-white text-[clamp(2.25rem,6.5vw,5rem)] max-w-[20ch] mx-auto">
            {t("about.landing.hero.headPart1")}{" "}
            {t("about.landing.hero.headMargins")}
            <br className="hidden sm:block" />{" "}
            {t("about.landing.hero.headPart2")}{" "}
            {t("about.landing.hero.headInternet")}
            {t("about.landing.hero.headEnd")}
          </h1>

          <p className="mt-8 mx-auto max-w-[40rem] text-[17px] md:text-[19px] leading-[1.6] text-surface-600 dark:text-surface-300">
            {t("about.landing.hero.lede")}
          </p>

          <div className="mt-10 flex flex-wrap justify-center items-center gap-3">
            <a
              href={STORE_LINKS[primaryStore]}
              target="_blank"
              rel="noopener noreferrer"
              className="group inline-flex items-center gap-2 px-5 py-3 text-[14.5px] font-semibold bg-[#027bff] text-white rounded-full hover:bg-[#026ae0] transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#027bff]/60 ring-1 ring-[#027bff]/30 shadow-sm shadow-[#027bff]/20"
            >
              <PrimaryIcon size={15} />
              {t("about.hero.installFor", { browser: primaryLabel })}
              <ArrowRight
                size={14}
                className="transition-transform duration-200 ease-out group-hover:translate-x-0.5"
              />
            </a>
            <a
              href={heroPrimaryHref}
              className="group inline-flex items-center gap-2 px-5 py-3 text-[14.5px] font-medium text-surface-700 dark:text-surface-200 bg-white dark:bg-surface-800 rounded-full border border-surface-200 dark:border-surface-700/60 hover:border-surface-300 dark:hover:border-surface-600 hover:text-surface-900 dark:hover:text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60"
            >
              {heroPrimaryLabel}
              <ArrowRight
                size={14}
                className="transition-transform duration-200 ease-out group-hover:translate-x-0.5 opacity-70"
              />
            </a>
          </div>
        </div>

        <div className="max-w-[68rem] mx-auto px-5 sm:px-8">
          <div className="h-px bg-surface-200 dark:bg-surface-800/80" />
        </div>
      </header>

      <DemoStrip />

      <section className="border-t border-surface-200 dark:border-surface-800">
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-20 md:py-28">
          <div className="max-w-[42rem] mb-12 md:mb-16">
            <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-surface-500 dark:text-surface-400 mb-4">
              {t("about.landing.capabilities.heading")}
            </p>
            <p className="font-display text-[1.75rem] md:text-[2.25rem] leading-[1.15] tracking-[-0.02em] text-surface-900 dark:text-white">
              {t("about.landing.capabilities.subheading")}
            </p>
          </div>
          <dl className="border-t border-surface-200 dark:border-surface-800">
            {capabilities.map((c) => {
              const Icon = c.icon;
              return (
                <div
                  key={c.title}
                  className="grid grid-cols-1 sm:grid-cols-[16rem_minmax(0,1fr)] gap-2 sm:gap-10 py-5 sm:py-6 border-b border-surface-200 dark:border-surface-800"
                >
                  <dt className="flex items-center gap-3 font-display font-semibold text-[1.05rem] tracking-tight text-surface-900 dark:text-white">
                    <span
                      aria-hidden
                      className="flex-shrink-0 inline-flex items-center justify-center w-9 h-9 rounded-lg bg-primary-100 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300 ring-1 ring-primary-200/60 dark:ring-primary-500/15"
                    >
                      <Icon size={17} strokeWidth={1.75} />
                    </span>
                    {c.title}
                  </dt>
                  <dd className="text-[15px] leading-[1.65] text-surface-600 dark:text-surface-400 max-w-[40rem] sm:pt-1.5">
                    {c.desc}
                  </dd>
                </div>
              );
            })}
          </dl>
        </div>
      </section>

      <section className="border-t border-surface-200 dark:border-surface-800">
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-20 md:py-28">
          <div className="grid grid-cols-1 lg:grid-cols-[1fr_15rem] gap-10 lg:gap-14">
            <div className="max-w-[40rem]">
              <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-surface-500 dark:text-surface-400 mb-4">
                {t("about.protocol.badge")}
              </p>
              <h2 className="font-display font-semibold text-[2rem] md:text-[2.5rem] leading-[1.1] tracking-[-0.02em] text-surface-900 dark:text-white mb-5">
                {t("about.landing.protocol.heading")}
              </h2>
              <p className="text-[16px] md:text-[17px] leading-[1.7] text-surface-600 dark:text-surface-300 mb-7">
                {t("about.landing.protocol.lede")}
              </p>
              <ul className="text-[15px] leading-[1.65] text-surface-700 dark:text-surface-300 border-t border-surface-200 dark:border-surface-800">
                {(
                  [
                    [AtSign, "point0"],
                    [Database, "point1"],
                    [Braces, "point2"],
                    [Code2, "point3"],
                  ] as const
                ).map(([Icon, key]) => (
                  <li
                    key={key}
                    className="flex items-start gap-3.5 py-3.5 border-b border-surface-200 dark:border-surface-800"
                  >
                    <span
                      aria-hidden
                      className="flex-shrink-0 mt-0.5 inline-flex items-center justify-center w-7 h-7 rounded-md bg-primary-100 dark:bg-primary-950/50 text-primary-700 dark:text-primary-300"
                    >
                      <Icon size={14} strokeWidth={1.75} />
                    </span>
                    <span>{t(`about.protocol.${key}`)}</span>
                  </li>
                ))}
              </ul>

              <details className="mt-10 group">
                <summary className="cursor-pointer inline-flex items-center gap-1.5 text-[13px] font-mono text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors list-none">
                  <ChevronDown
                    size={13}
                    className="transition-transform duration-200 group-open:rotate-180"
                  />
                  {t("about.landing.protocol.viewLexicon")}
                </summary>
                <LexiconBlock />
              </details>
            </div>

            <aside className="lg:pt-12 flex flex-col gap-3 self-start">
              <a
                href="https://github.com/paddinglabs"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center justify-between gap-3 text-[13px] font-medium text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors py-1.5 border-b border-surface-200 dark:border-surface-800"
              >
                <span className="inline-flex items-center gap-2">
                  <Github size={13} />
                  GitHub
                </span>
                <ArrowUpRight size={12} className="opacity-60" />
              </a>
              <a
                href="https://tangled.org/margin.at/margin"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center justify-between gap-3 text-[13px] font-medium text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors py-1.5 border-b border-surface-200 dark:border-surface-800"
              >
                <span className="inline-flex items-center gap-2">
                  <TangledIcon size={13} />
                  Tangled
                </span>
                <ArrowUpRight size={12} className="opacity-60" />
              </a>
              <a
                href="https://atproto.com"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center justify-between gap-3 text-[13px] font-medium text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors py-1.5 border-b border-surface-200 dark:border-surface-800"
              >
                <span>{t("about.hero.atProtocol")}</span>
                <ArrowUpRight size={12} className="opacity-60" />
              </a>
            </aside>
          </div>
        </div>
      </section>

      <section className="border-t border-surface-200 dark:border-surface-800">
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-14 md:py-20">
          <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-surface-500 dark:text-surface-400 mb-5">
            Install
          </p>
          <p className="font-display text-[1.4rem] md:text-[1.75rem] leading-[1.4] tracking-tight text-surface-900 dark:text-white max-w-[44rem]">
            {t("about.landing.install.lede")}{" "}
            <InstallLink
              href={STORE_LINKS.chrome}
              icon={<Chrome size={18} />}
              label="Chrome"
              active={primaryStore === "chrome"}
            />
            ,{" "}
            <InstallLink
              href={STORE_LINKS.firefox}
              icon={<FaFirefox size={17} />}
              label="Firefox"
              active={primaryStore === "firefox"}
            />
            ,{" "}
            <InstallLink
              href={STORE_LINKS.edge}
              icon={<FaEdge size={17} />}
              label="Edge"
              active={primaryStore === "edge"}
            />
            ,{" "}
            <InstallLink
              href={STORE_LINKS.safari}
              icon={<FaSafari size={17} />}
              label="Safari"
              active={primaryStore === "safari"}
            />
            ,{" "}
            <span className="text-surface-500 dark:text-surface-400">
              {t("about.landing.install.or")}
            </span>{" "}
            <InstallLink
              href={STORE_LINKS.ios}
              icon={<AppleIcon size={17} />}
              label={t("about.extension.iosShortcut")}
              active={false}
            />
            .
          </p>
        </div>
      </section>

      <section className="border-t border-surface-200 dark:border-surface-800">
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-24 md:py-36">
          <h2 className="font-display font-semibold tracking-[-0.025em] leading-[1.02] text-[clamp(2.25rem,6vw,4.75rem)] text-surface-900 dark:text-white max-w-[42rem]">
            {t("about.landing.closer.line")}
          </h2>
          <div className="mt-10 flex items-baseline gap-x-8 gap-y-3 flex-wrap">
            <a
              href={heroPrimaryHref}
              className="group inline-flex items-center gap-1.5 text-[15px] font-semibold text-surface-900 dark:text-white border-b-2 border-surface-900 dark:border-white pb-0.5 hover:text-primary-700 dark:hover:text-primary-300 hover:border-primary-700 dark:hover:border-primary-300 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60 rounded-sm"
            >
              {user
                ? t("about.landing.closer.open")
                : t("about.landing.closer.signIn")}
              <ArrowRight
                size={15}
                className="transition-transform duration-200 ease-out group-hover:translate-x-0.5"
              />
            </a>
            <a
              href="https://github.com/paddinglabs"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 text-[14px] font-medium text-surface-500 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors"
            >
              <Github size={13} />
              {t("about.cta.viewGitHub")}
            </a>
          </div>
        </div>
      </section>

      <StaticFooter />
    </div>
  );
}

const DEMO_STYLES = `
  @keyframes mg-select {
    0%, 8%        { width: 0%; }
    24%, 38%      { width: 100%; }
    48%           { width: 100%; opacity: 1; }
    54%, 100%     { width: 100%; opacity: 0; }
  }
  @keyframes mg-underline {
    0%, 42%       { transform: scaleX(0); }
    56%, 100%     { transform: scaleX(1); }
  }
  @keyframes mg-menu-in {
    0%            { opacity: 0; transform: translateY(4px) scale(0.98); }
    10%, 55%      { opacity: 1; transform: translateY(0) scale(1); }
    65%, 100%     { opacity: 0; transform: translateY(-2px) scale(0.99); }
  }
  @keyframes mg-menu-press {
    0%, 38%       { background-color: var(--mg-press-base); }
    44%           { background-color: var(--mg-press-active); }
    52%, 100%     { background-color: var(--mg-press-base); }
  }
  .mg-press-target {
    --mg-press-base: rgb(239 246 255);
    --mg-press-active: rgb(191 219 254);
  }
  [data-theme="dark"] .mg-press-target,
  .dark .mg-press-target {
    --mg-press-base: rgba(23 37 84 / 0.55);
    --mg-press-active: rgba(30 58 138 / 0.75);
  }
  @keyframes mg-toast-in {
    0%, 60%       { opacity: 0; transform: translateY(8px); }
    72%, 96%      { opacity: 1; transform: translateY(0); }
    100%          { opacity: 0; transform: translateY(-4px); }
  }
  @media (prefers-reduced-motion: reduce) {
    .mg-anim, .mg-anim * { animation: none !important; }
    .mg-final-on { opacity: 1 !important; transform: none !important; }
    .mg-final-width { width: 100% !important; }
  }
`;

function DemoStrip() {
  const { t } = useTranslation();
  return (
    <section className="border-t border-surface-200 dark:border-surface-800">
      <style dangerouslySetInnerHTML={{ __html: DEMO_STYLES }} />
      <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-16 md:py-24">
        <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-surface-500 dark:text-surface-400 mb-10">
          {t("about.landing.demo.eyebrow")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-5 md:gap-6">
          <DemoCard caption={t("about.landing.demo.card1Caption")}>
            <DemoFrameHighlight
              sample={t("about.landing.demo.card1Sample")}
              mark={t("about.landing.demo.card1Mark")}
            />
          </DemoCard>
          <DemoCard caption={t("about.landing.demo.card2Caption")}>
            <DemoFrameAnnotate
              sample={t("about.landing.demo.card2Sample")}
              mark={t("about.landing.demo.card2Mark")}
            />
          </DemoCard>
          <DemoCard caption={t("about.landing.demo.card3Caption")}>
            <DemoFrameBookmark
              url={t("about.landing.demo.card3Url")}
              saved={t("about.landing.demo.card3Saved")}
            />
          </DemoCard>
        </div>
      </div>
    </section>
  );
}

function DemoCard({
  caption,
  children,
}: {
  caption: string;
  children: React.ReactNode;
}) {
  return (
    <figure className="flex flex-col gap-4">
      <div className="rounded-xl border border-surface-200 dark:border-surface-700/60 bg-white dark:bg-surface-800 overflow-hidden h-[15rem] sm:h-[16rem]">
        {children}
      </div>
      <figcaption className="text-[13.5px] leading-[1.55] text-surface-600 dark:text-surface-400 text-balance">
        {caption}
      </figcaption>
    </figure>
  );
}

function DemoFrameHighlight({
  sample,
  mark,
}: {
  sample: string;
  mark: string;
}) {
  const idx = sample.indexOf(mark);
  const before = idx >= 0 ? sample.slice(0, idx) : sample;
  const after = idx >= 0 ? sample.slice(idx + mark.length) : "";
  return (
    <div className="mg-anim relative h-full px-6 sm:px-7 py-8 flex items-center">
      <p className="text-[15.5px] sm:text-[16px] leading-[1.7] text-surface-700 dark:text-surface-300">
        {before}
        <span className="relative inline">
          <span
            aria-hidden
            className="mg-final-width absolute left-0 top-[0.05em] bottom-[-0.05em] bg-yellow-300/40 dark:bg-yellow-300/25 rounded-[2px] pointer-events-none"
            style={{
              animation: "mg-select 6s cubic-bezier(0.22,1,0.36,1) infinite",
              willChange: "width, opacity",
            }}
          />
          <span
            aria-hidden
            className="absolute left-0 right-0 bottom-[-3px] h-[2px] origin-left bg-[#fbbf24] mg-final-on"
            style={{
              animation: "mg-underline 6s cubic-bezier(0.22,1,0.36,1) infinite",
              willChange: "transform",
            }}
          />
          <span className="relative font-medium text-surface-900 dark:text-white">
            {mark}
          </span>
        </span>
        {after}
      </p>
    </div>
  );
}

function DemoFrameAnnotate({ sample, mark }: { sample: string; mark: string }) {
  const idx = sample.indexOf(mark);
  const before = idx >= 0 ? sample.slice(0, idx) : sample;
  const after = idx >= 0 ? sample.slice(idx + mark.length) : "";
  const accent = "#60a5fa";
  return (
    <div className="relative h-full px-6 sm:px-7 py-7 flex flex-col justify-between gap-5">
      <p className="text-[15.5px] sm:text-[16px] leading-[1.7] text-surface-700 dark:text-surface-300">
        {before}
        <span
          className="text-surface-900 dark:text-white font-medium"
          style={{
            textDecoration: "underline",
            textDecorationColor: accent,
            textDecorationThickness: "2px",
            textUnderlineOffset: "3px",
            textDecorationSkipInk: "none",
          }}
        >
          {mark}
        </span>
        {after}
      </p>

      <div className="rounded-xl border border-surface-200 dark:border-surface-700/60 bg-surface-50 dark:bg-surface-900/60 p-3.5">
        <div className="flex items-start gap-2.5">
          <span
            aria-hidden
            className="w-7 h-7 rounded-full bg-gradient-to-br from-primary-400 to-primary-600 flex-shrink-0"
          />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-2">
              <span className="h-2 rounded-full bg-surface-300 dark:bg-surface-600 w-20" />
              <span className="h-1.5 rounded-full bg-surface-200 dark:bg-surface-700 w-7" />
            </div>
            <div
              className="text-[12px] italic px-2.5 py-1.5 rounded-r mb-2 border-l-2 text-surface-700 dark:text-surface-300"
              style={{
                borderColor: accent,
                backgroundColor: `${accent}14`,
              }}
            >
              &ldquo;{mark}&rdquo;
            </div>
            <div className="space-y-1.5">
              <span className="block h-1.5 rounded-full bg-surface-200 dark:bg-surface-700 w-[88%]" />
              <span className="block h-1.5 rounded-full bg-surface-200 dark:bg-surface-700 w-full" />
              <span className="block h-1.5 rounded-full bg-surface-200 dark:bg-surface-700 w-[60%]" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function DemoFrameBookmark({ url, saved }: { url: string; saved: string }) {
  return (
    <div className="mg-anim relative h-full flex flex-col">
      <div className="flex items-center gap-3 px-3.5 py-2.5 border-b border-surface-200 dark:border-surface-700/60 bg-surface-50 dark:bg-surface-900/40">
        <div className="flex gap-1.5">
          <span className="w-2.5 h-2.5 rounded-full bg-surface-300 dark:bg-surface-600" />
          <span className="w-2.5 h-2.5 rounded-full bg-surface-300 dark:bg-surface-600" />
          <span className="w-2.5 h-2.5 rounded-full bg-surface-300 dark:bg-surface-600" />
        </div>
        <div className="flex-1 bg-white dark:bg-surface-900 border border-surface-200 dark:border-surface-700/60 rounded px-3 py-1 text-[11px] font-mono text-surface-500 dark:text-surface-400 truncate text-center">
          {url}
        </div>
        <span
          aria-hidden
          className="relative inline-flex items-center justify-center w-7 h-7 rounded-md hover:bg-surface-200/60 dark:hover:bg-surface-700/40"
        >
          <img src="/logo.svg" alt="" className="w-4 h-4" />
          <span
            aria-hidden
            className="mg-final-on absolute -right-0.5 -top-0.5 w-2 h-2 rounded-full bg-primary-600 dark:bg-primary-500 ring-2 ring-white dark:ring-surface-800"
            style={{
              animation: "mg-toast-in 6s cubic-bezier(0.22,1,0.36,1) infinite",
            }}
          />
        </span>
      </div>

      <div className="relative flex-1 px-6 py-5 overflow-hidden">
        <div className="space-y-2 w-full">
          <div className="h-1.5 rounded-full bg-surface-200 dark:bg-surface-700/70 w-[92%]" />
          <div className="h-1.5 rounded-full bg-surface-200 dark:bg-surface-700/70 w-full" />
          <div className="h-1.5 rounded-full bg-surface-200 dark:bg-surface-700/70 w-[78%]" />
          <div className="h-1.5 rounded-full bg-surface-200 dark:bg-surface-700/70 w-[88%]" />
          <div className="h-1.5 rounded-full bg-surface-200 dark:bg-surface-700/70 w-[64%]" />
        </div>

        <div
          aria-hidden
          className="mg-final-on absolute left-8 top-4 flex items-start gap-0"
          style={{
            animation: "mg-menu-in 6s cubic-bezier(0.22,1,0.36,1) infinite",
          }}
        >
          <ul className="rounded-md border border-surface-200 dark:border-surface-700 bg-white dark:bg-surface-800 shadow-md py-1 min-w-[8.5rem] text-[11.5px] text-surface-700 dark:text-surface-200">
            <li className="px-3 py-1 text-surface-400 dark:text-surface-500">
              Back
            </li>
            <li className="px-3 py-1 text-surface-400 dark:text-surface-500">
              Reload
            </li>
            <li className="my-1 h-px bg-surface-200 dark:bg-surface-700" />
            <li className="relative px-3 py-1 flex items-center gap-2 bg-primary-600 text-white">
              <img
                src="/logo.svg"
                alt=""
                className="w-3 h-3 brightness-0 invert"
              />
              Margin
              <span className="ml-auto text-white/80">▸</span>
            </li>
            <li className="px-3 py-1 text-surface-400 dark:text-surface-500">
              Inspect
            </li>
          </ul>
          <ul className="-ml-px rounded-md border border-surface-200 dark:border-surface-700 bg-white dark:bg-surface-800 shadow-md py-1 min-w-[10rem] text-[11.5px] text-surface-700 dark:text-surface-200">
            <li
              className="mg-press-target px-3 py-1 text-primary-900 dark:text-primary-100"
              style={{
                animation:
                  "mg-menu-press 6s cubic-bezier(0.22,1,0.36,1) infinite",
              }}
            >
              Bookmark page
            </li>
            <li className="px-3 py-1 text-surface-400 dark:text-surface-500">
              Highlight selection
            </li>
            <li className="px-3 py-1 text-surface-400 dark:text-surface-500">
              Annotate selection
            </li>
          </ul>
        </div>

        <span
          aria-hidden
          className="mg-final-on absolute right-4 bottom-4 inline-flex items-center gap-1.5 rounded-full bg-primary-600 dark:bg-primary-500 text-white text-[11px] font-medium px-2.5 py-1 shadow-sm"
          style={{
            animation: "mg-toast-in 6s cubic-bezier(0.22,1,0.36,1) infinite",
          }}
        >
          <svg
            viewBox="0 0 24 24"
            className="w-3 h-3"
            fill="none"
            stroke="currentColor"
            strokeWidth="3"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M5 12 L10 17 L19 7" />
          </svg>
          {saved}
        </span>
      </div>
    </div>
  );
}

function InstallLink({
  href,
  icon,
  label,
  active,
}: {
  href: string;
  icon: React.ReactNode;
  label: string;
  active: boolean;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={`group inline-flex items-baseline gap-1.5 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60 rounded-sm ${
        active
          ? "text-surface-900 dark:text-white font-semibold underline decoration-2 decoration-primary-600 dark:decoration-primary-400 underline-offset-[6px]"
          : "text-surface-700 dark:text-surface-300 hover:text-surface-900 dark:hover:text-white underline decoration-1 decoration-surface-300 dark:decoration-surface-700 underline-offset-[6px] hover:decoration-primary-600 dark:hover:decoration-primary-400"
      }`}
    >
      <span className="translate-y-[3px]" aria-hidden>
        {icon}
      </span>
      {label}
    </a>
  );
}
