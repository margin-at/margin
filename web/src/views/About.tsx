import React from "react";
import { useStore } from "@nanostores/react";
import { useTranslation } from "react-i18next";
import "../i18n";

import { $theme, cycleTheme } from "../store/theme";
import { $user } from "../store/auth";
import {
  MessageSquareText,
  Highlighter,
  Bookmark,
  FolderOpen,
  Keyboard,
  PanelRight,
  MousePointerClick,
  Shield,
  Users,
  Chrome,
  ArrowRight,
  Github,
  ExternalLink,
  Hash,
  Coffee,
  Heart,
  Eye,
  Sun,
  Moon,
  Monitor,
  Check,
  Lock,
} from "lucide-react";
import { AppleIcon, TangledIcon } from "../components/common/Icons";
import { FaFirefox, FaEdge } from "react-icons/fa";

export default function About() {
  const { t } = useTranslation();
  const theme = useStore($theme);
  const user = useStore($user);

  const [browser] = React.useState<
    "chrome" | "firefox" | "edge" | "safari" | "other"
  >(() => {
    if (typeof navigator === "undefined") return "other";
    const ua = navigator.userAgent;
    if (/Edg\//i.test(ua)) return "edge";
    if (/Firefox/i.test(ua)) return "firefox";
    if (/^((?!chrome|android).)*safari/i.test(ua)) return "safari";
    if (/Chrome/i.test(ua)) return "chrome";
    return "other";
  });

  const extensionLink =
    browser === "firefox"
      ? "https://addons.mozilla.org/en-US/firefox/addon/margin/"
      : browser === "edge"
        ? "https://microsoftedge.microsoft.com/addons/detail/margin/nfjnmllpdgcdnhmmggjihjbidmeadddn"
        : "https://chromewebstore.google.com/detail/margin/cgpmbiiagnehkikhcbnhiagfomajncpa";

  const ExtensionIcon =
    browser === "firefox" ? FaFirefox : browser === "edge" ? FaEdge : Chrome;
  const extensionLabel =
    browser === "firefox"
      ? "Firefox"
      : browser === "edge"
        ? "Edge"
        : browser === "safari"
          ? "Chrome"
          : "Chrome";

  const [isScrolled, setIsScrolled] = React.useState(false);

  React.useEffect(() => {
    const handleScroll = () => {
      setIsScrolled((prev) =>
        prev ? window.scrollY > 10 : window.scrollY > 50,
      );
    };
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const features = [
    {
      icon: MessageSquareText,
      title: t("about.features.annotations.title"),
      description: t("about.features.annotations.description"),
    },
    {
      icon: Highlighter,
      title: t("about.features.highlights.title"),
      description: t("about.features.highlights.description"),
    },
    {
      icon: Bookmark,
      title: t("about.features.bookmarks.title"),
      description: t("about.features.bookmarks.description"),
    },
    {
      icon: FolderOpen,
      title: t("about.features.collections.title"),
      description: t("about.features.collections.description"),
    },
    {
      icon: Users,
      title: t("about.features.socialDiscovery.title"),
      description: t("about.features.socialDiscovery.description"),
    },
    {
      icon: Hash,
      title: t("about.features.tagsSearch.title"),
      description: t("about.features.tagsSearch.description"),
    },
  ];

  return (
    <div className="min-h-screen bg-surface-100 dark:bg-surface-900">
      <nav
        className={`sticky top-0 z-50 transition-[background-color,border-color,backdrop-filter] duration-200 ease-out ${
          isScrolled
            ? "bg-white/75 dark:bg-surface-900/75 backdrop-blur-xl border-b border-surface-200/60 dark:border-surface-800/60"
            : "bg-transparent border-b border-transparent"
        }`}
      >
        <div className="max-w-6xl mx-auto px-5 sm:px-8 h-16 flex items-center justify-between gap-4">
          <div className="flex items-center gap-7">
            <a href="/" className="group flex items-center gap-2.5">
              <img
                src="/logo.svg"
                alt="Margin"
                className="w-7 h-7 transition-transform group-hover:rotate-[-4deg]"
              />
              <span className="font-display font-bold text-lg tracking-tight text-surface-900 dark:text-white">
                Margin
              </span>
            </a>

            {user && (
              <div className="hidden md:flex items-center gap-5">
                <a
                  href="/home"
                  className="text-[13px] font-medium text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors"
                >
                  {t("nav.feed")}
                </a>
                <a
                  href="/discover"
                  className="text-[13px] font-medium text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors"
                >
                  {t("nav.discover")}
                </a>
              </div>
            )}
          </div>

          <div className="flex items-center gap-1 sm:gap-2">
            {!user && (
              <a
                href="/login"
                className="text-[13px] font-medium text-surface-600 dark:text-surface-300 hover:text-surface-900 dark:hover:text-white transition-colors px-3 py-1.5"
              >
                {t("nav.signIn")}
              </a>
            )}
            <a
              href={extensionLink}
              target="_blank"
              rel="noopener noreferrer"
              className="group inline-flex items-center gap-1.5 text-[13px] font-semibold pl-3 pr-3.5 py-1.5 bg-surface-900 dark:bg-white text-white dark:text-surface-900 rounded-lg hover:bg-surface-800 dark:hover:bg-surface-100 transition-colors"
            >
              <ExtensionIcon size={14} />
              <span className="hidden sm:inline">
                {t("about.nav.getExtension")}
              </span>
              <span className="sm:hidden">{t("about.nav.install")}</span>
            </a>
          </div>
        </div>
      </nav>

      <section className="max-w-6xl mx-auto px-6 pt-16 pb-16 md:pt-24 md:pb-20">
        <div className="inline-flex items-center gap-1.5 px-1 py-1 rounded-full bg-white dark:bg-surface-800 border border-surface-200 dark:border-surface-700/60 mb-8 shadow-sm">
          <div className="flex items-center -space-x-1.5">
            <a
              href="https://github.com/margin-at"
              target="_blank"
              rel="noreferrer"
              className="relative z-10 flex items-center justify-center w-7 h-7 rounded-full bg-surface-50 dark:bg-surface-900 text-surface-600 hover:text-surface-900 dark:text-surface-400 dark:hover:text-white border-2 border-white dark:border-surface-800 shadow-sm transition-transform hover:z-20 hover:scale-110"
              title="GitHub"
            >
              <Github size={13} />
            </a>
            <a
              href="https://tangled.org/margin.at/margin"
              target="_blank"
              rel="noreferrer"
              className="relative z-0 flex items-center justify-center w-7 h-7 rounded-full bg-surface-50 dark:bg-surface-900 border-2 border-white dark:border-surface-800 shadow-sm transition-transform hover:z-20 hover:scale-110"
              title="Tangled"
            >
              <TangledIcon
                size={14}
                className="text-surface-600 dark:text-surface-400"
              />
            </a>
          </div>
          <span className="pr-3 pl-1 text-xs font-semibold text-surface-600 dark:text-surface-300">
            {t("about.hero.openSource")}
          </span>
        </div>

        <h1 className="font-display text-5xl md:text-7xl lg:text-[5.5rem] font-bold tracking-tight text-surface-900 dark:text-white leading-[1.02] mb-8 max-w-4xl">
          {t("about.hero.headline")} <br className="hidden sm:block" />
          <span className="text-primary-600 dark:text-primary-400">
            {t("about.hero.headlineAccent")}
          </span>
        </h1>

        <p className="text-lg md:text-xl text-surface-500 dark:text-surface-400 max-w-2xl leading-relaxed mb-10">
          {t("about.hero.descriptionPre")}{" "}
          <a
            href="https://atproto.com"
            target="_blank"
            rel="noreferrer"
            className="text-surface-800 dark:text-surface-200 hover:text-primary-600 dark:hover:text-primary-400 border-b border-surface-300 dark:border-surface-600 hover:border-primary-400 transition-colors font-medium"
          >
            {t("about.hero.atProtocol")}
          </a>
          {t("about.hero.descriptionPost")}
        </p>

        <div className="flex flex-col sm:flex-row items-start gap-3">
          <a
            href={user ? "/home" : "/login"}
            className="group inline-flex items-center gap-2 px-6 py-3 bg-surface-900 dark:bg-white text-white dark:text-surface-900 rounded-xl font-semibold hover:bg-surface-800 dark:hover:bg-surface-200 transition-all duration-200 text-[15px] shadow-sm"
          >
            {user ? t("about.hero.openApp") : t("about.hero.getStarted")}
            <ArrowRight
              size={16}
              className="transition-transform group-hover:translate-x-0.5"
            />
          </a>
          <a
            href={extensionLink}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-3 text-surface-700 dark:text-surface-200 hover:text-surface-900 dark:hover:text-white bg-white dark:bg-surface-800 hover:bg-surface-50 dark:hover:bg-surface-700 border border-surface-200 dark:border-surface-700 rounded-xl font-semibold transition-all duration-200 text-[15px]"
          >
            <ExtensionIcon size={16} />
            {t("about.hero.installFor", { browser: extensionLabel })}
          </a>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-6 pb-20 md:pb-28">
        <div className="relative">
          <div
            aria-hidden
            className="absolute inset-x-12 top-12 bottom-0 bg-gradient-to-b from-primary-100/40 to-transparent dark:from-primary-900/20 dark:to-transparent rounded-3xl blur-2xl"
          />

          <div className="relative rounded-2xl border border-surface-200 dark:border-surface-700/60 bg-white dark:bg-surface-800 shadow-2xl shadow-surface-900/10 overflow-hidden">
            <div className="flex items-center gap-3 px-4 py-3 border-b border-surface-100 dark:border-surface-700/60 bg-surface-50/50 dark:bg-surface-900/40">
              <div className="flex gap-1.5">
                <div className="w-3 h-3 rounded-full bg-red-400/70" />
                <div className="w-3 h-3 rounded-full bg-yellow-400/70" />
                <div className="w-3 h-3 rounded-full bg-green-400/70" />
              </div>
              <div className="flex-1 max-w-md mx-auto bg-white dark:bg-surface-800 border border-surface-200 dark:border-surface-700/60 rounded-md px-3 py-1 flex items-center justify-center gap-1.5">
                <Lock size={10} className="text-surface-400" />
                <span className="text-[11px] font-mono text-surface-500 dark:text-surface-400 truncate">
                  essays.example.com/marginalia
                </span>
              </div>
              <div className="flex items-center gap-2 pl-2">
                <img src="/logo.svg" alt="" className="w-5 h-5" />
                <span className="text-[11px] font-mono font-semibold text-primary-600 dark:text-primary-400">
                  3
                </span>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-[1fr_280px] gap-0 md:gap-8">
              <article className="px-6 sm:px-10 py-8 md:py-10 border-b md:border-b-0 md:border-r border-surface-100 dark:border-surface-700/60">
                <div className="mb-6">
                  <h3 className="font-display font-bold text-2xl md:text-3xl text-surface-900 dark:text-white mb-2 tracking-tight">
                    The Reader's Margin
                  </h3>
                  <div className="flex items-center gap-3 text-xs text-surface-400 dark:text-surface-500 font-mono">
                    <span>Lena Park</span>
                    <span>·</span>
                    <span>12 min</span>
                  </div>
                </div>

                <div className="space-y-2.5">
                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-full" />
                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[94%]" />
                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[88%]" />

                  <div className="text-[15px] text-surface-700 dark:text-surface-300 leading-[1.7] py-1">
                    When you mark up the page,{" "}
                    <span className="text-surface-900 dark:text-white font-medium underline decoration-yellow-400 dark:decoration-yellow-500/80 [text-decoration-thickness:2px] underline-offset-[3px] [text-decoration-skip-ink:none]">
                      the page begins to read you back
                    </span>
                    .
                  </div>

                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[82%]" />

                  <div className="pt-3" />

                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[91%]" />
                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-full" />

                  <div className="text-[15px] text-surface-700 dark:text-surface-300 leading-[1.7] py-1">
                    A note in the margin is{" "}
                    <span className="text-surface-900 dark:text-white font-medium underline decoration-primary-500 dark:decoration-primary-400 [text-decoration-thickness:2px] underline-offset-[3px] [text-decoration-skip-ink:none]">
                      a conversation with the future you
                    </span>{" "}
                    — and now, with everyone else who reads.
                  </div>

                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[85%]" />
                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[70%]" />

                  <div className="pt-3" />

                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[93%]" />
                  <div className="text-[15px] text-surface-700 dark:text-surface-300 leading-[1.7] py-1">
                    The library was once shared in pencil and{" "}
                    <span className="text-surface-900 dark:text-white font-medium underline decoration-yellow-400 dark:decoration-yellow-500/80 [text-decoration-thickness:2px] underline-offset-[3px] [text-decoration-skip-ink:none]">
                      whispered footnotes
                    </span>
                    .
                  </div>
                  <div className="h-2.5 bg-surface-200 dark:bg-surface-700/70 rounded-full w-[78%]" />
                </div>
              </article>

              <aside className="px-6 sm:px-6 py-8 md:py-10 bg-surface-50/40 dark:bg-surface-900/30">
                <div className="flex items-center justify-between mb-4">
                  <span className="text-[10px] font-mono uppercase tracking-[0.12em] text-surface-400 dark:text-surface-500">
                    Annotations
                  </span>
                  <span className="text-[10px] font-mono text-surface-400 dark:text-surface-500">
                    3
                  </span>
                </div>

                <div className="space-y-3">
                  <div className="bg-white dark:bg-surface-800 rounded-lg border border-surface-200/70 dark:border-surface-700/60 p-3">
                    <div className="flex items-center gap-1.5 mb-2">
                      <div className="w-5 h-5 rounded-full bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center text-white text-[9px] font-bold">
                        S
                      </div>
                      <span className="text-[11px] font-semibold text-surface-700 dark:text-surface-300 truncate">
                        @scan.margin.cafe
                      </span>
                      <span className="text-[10px] text-surface-400 ml-auto font-mono shrink-0">
                        2m
                      </span>
                    </div>
                    <p className="text-[12px] text-surface-600 dark:text-surface-400 leading-snug">
                      this is the whole thesis tbh
                    </p>
                    <div className="flex items-center gap-3 mt-2 pt-2 border-t border-surface-100 dark:border-surface-700/60">
                      <span className="inline-flex items-center gap-1 text-[10px] text-surface-400">
                        <Heart size={9} /> 4
                      </span>
                      <span className="inline-flex items-center gap-1 text-[10px] text-surface-400">
                        <MessageSquareText size={9} /> 1
                      </span>
                    </div>
                  </div>

                  <div className="bg-white dark:bg-surface-800 rounded-lg border border-surface-200/70 dark:border-surface-700/60 p-3">
                    <div className="flex items-center gap-1.5 mb-2">
                      <div className="w-5 h-5 rounded-full bg-gradient-to-br from-emerald-400 to-emerald-600 flex items-center justify-center text-white text-[9px] font-bold">
                        M
                      </div>
                      <span className="text-[11px] font-semibold text-surface-700 dark:text-surface-300 truncate">
                        @maya.margin.cafe
                      </span>
                      <span className="text-[10px] text-surface-400 ml-auto font-mono shrink-0">
                        1h
                      </span>
                    </div>
                    <p className="text-[12px] text-surface-600 dark:text-surface-400 leading-snug">
                      saving this for the design book
                    </p>
                  </div>

                  <div className="bg-white dark:bg-surface-800 rounded-lg border border-surface-200/70 dark:border-surface-700/60 p-3">
                    <div className="flex items-center gap-1.5 mb-2">
                      <div className="w-5 h-5 rounded-full bg-surface-200 dark:bg-surface-700 flex items-center justify-center">
                        <Highlighter
                          size={10}
                          className="text-surface-500 dark:text-surface-400"
                        />
                      </div>
                      <span className="text-[11px] font-semibold text-surface-700 dark:text-surface-300">
                        highlight
                      </span>
                      <span className="text-[10px] text-surface-400 ml-auto font-mono">
                        3h
                      </span>
                    </div>
                    <p className="text-[12px] text-surface-500 dark:text-surface-500 italic leading-snug">
                      "whispered footnotes"
                    </p>
                  </div>
                </div>
              </aside>
            </div>
          </div>
        </div>
      </section>

      <section className="border-t border-surface-200 dark:border-surface-800 bg-white dark:bg-surface-900/40">
        <div className="max-w-6xl mx-auto px-6 py-24 md:py-32">
          <div className="max-w-2xl mb-16 md:mb-20">
            <h2 className="font-display text-4xl md:text-5xl lg:text-6xl font-bold tracking-tight text-surface-900 dark:text-white mb-5 leading-[1.05]">
              {t("about.features.title")}
            </h2>
            <p className="text-lg text-surface-500 dark:text-surface-400 leading-relaxed">
              {t("about.features.subtitle")}
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-12 gap-y-14">
            {features.map((feature) => (
              <div key={feature.title}>
                <div className="flex items-center gap-2.5 mb-3">
                  <feature.icon
                    size={16}
                    className="text-primary-500 dark:text-primary-400 flex-shrink-0"
                  />
                  <h3 className="font-display font-semibold text-xl text-surface-900 dark:text-white tracking-tight">
                    {feature.title}
                  </h3>
                </div>
                <p className="text-[15px] text-surface-500 dark:text-surface-400 leading-relaxed">
                  {feature.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="border-t border-surface-200/60 dark:border-surface-800/60">
        <div className="max-w-6xl mx-auto px-6 py-24 md:py-28">
          <div className="grid grid-cols-1 lg:grid-cols-[1.3fr_1fr] gap-14 lg:gap-20 items-start">
            <div>
              <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-white dark:bg-surface-800 text-surface-600 dark:text-surface-400 text-xs font-medium mb-5 border border-surface-200/60 dark:border-surface-700/60">
                <ExtensionIcon size={13} />
                {t("about.extension.badge")}
              </div>
              <h2 className="font-display text-4xl md:text-5xl font-bold tracking-tight text-surface-900 dark:text-white mb-5 leading-[1.05]">
                {t("about.extension.title")}{" "}
                <span className="text-surface-400 dark:text-surface-500">
                  {t("about.extension.titleLine2")}
                </span>
              </h2>
              <p className="text-lg text-surface-500 dark:text-surface-400 leading-relaxed mb-12 max-w-lg">
                {t("about.extension.description")}
              </p>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-10 gap-y-8">
                {(
                  [
                    { icon: Eye, key: "inlineOverlay" },
                    { icon: MousePointerClick, key: "contextMenu" },
                    { icon: Keyboard, key: "keyboard" },
                    { icon: PanelRight, key: "sidePanel" },
                  ] as const
                ).map(({ icon: Icon, key }) => (
                  <div key={key}>
                    <div className="flex items-center gap-2.5 mb-2">
                      <Icon
                        size={15}
                        className="text-primary-500 dark:text-primary-400"
                      />
                      <h4 className="font-display font-semibold text-base text-surface-900 dark:text-white tracking-tight">
                        {t(`about.extension.features.${key}.title`)}
                      </h4>
                    </div>
                    <p className="text-sm text-surface-500 dark:text-surface-400 leading-relaxed">
                      {t(`about.extension.features.${key}.description`)}
                    </p>
                  </div>
                ))}
              </div>
            </div>

            <div className="lg:sticky lg:top-24">
              <div className="rounded-2xl bg-surface-50 dark:bg-surface-800/50 border border-surface-200/60 dark:border-surface-700/60 p-6">
                <div className="text-[10px] font-mono uppercase tracking-[0.12em] text-surface-400 dark:text-surface-500 mb-4">
                  Install
                </div>
                <div className="flex flex-col gap-2">
                  <a
                    href={extensionLink}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center justify-between gap-2 px-4 py-3 bg-surface-900 dark:bg-white text-white dark:text-surface-900 rounded-lg font-medium text-sm transition-all hover:opacity-90"
                  >
                    <span className="inline-flex items-center gap-2">
                      <ExtensionIcon size={15} />
                      {t("about.hero.installFor", {
                        browser: extensionLabel,
                      })}
                    </span>
                    <ExternalLink size={13} />
                  </a>
                  {browser !== "chrome" && (
                    <a
                      href="https://chromewebstore.google.com/detail/margin/cgpmbiiagnehkikhcbnhiagfomajncpa"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center justify-between gap-2 px-4 py-2.5 bg-white dark:bg-surface-800 text-surface-700 dark:text-surface-200 rounded-lg font-medium text-sm transition-all hover:bg-surface-100 dark:hover:bg-surface-700 border border-surface-200/80 dark:border-surface-700/80"
                    >
                      <span className="inline-flex items-center gap-2">
                        <Chrome size={14} />
                        Chrome
                      </span>
                      <ExternalLink size={12} />
                    </a>
                  )}
                  {browser !== "firefox" && (
                    <a
                      href="https://addons.mozilla.org/en-US/firefox/addon/margin/"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center justify-between gap-2 px-4 py-2.5 bg-white dark:bg-surface-800 text-surface-700 dark:text-surface-200 rounded-lg font-medium text-sm transition-all hover:bg-surface-100 dark:hover:bg-surface-700 border border-surface-200/80 dark:border-surface-700/80"
                    >
                      <span className="inline-flex items-center gap-2">
                        <FaFirefox size={14} />
                        Firefox
                      </span>
                      <ExternalLink size={12} />
                    </a>
                  )}
                  {browser !== "edge" && (
                    <a
                      href="https://microsoftedge.microsoft.com/addons/detail/margin/nfjnmllpdgcdnhmmggjihjbidmeadddn"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center justify-between gap-2 px-4 py-2.5 bg-white dark:bg-surface-800 text-surface-700 dark:text-surface-200 rounded-lg font-medium text-sm transition-all hover:bg-surface-100 dark:hover:bg-surface-700 border border-surface-200/80 dark:border-surface-700/80"
                    >
                      <span className="inline-flex items-center gap-2">
                        <FaEdge size={14} />
                        Edge
                      </span>
                      <ExternalLink size={12} />
                    </a>
                  )}
                  <a
                    href="https://www.icloud.com/shortcuts/1e33ebf52f55431fae1e187cfe9738c3"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center justify-between gap-2 px-4 py-2.5 bg-white dark:bg-surface-800 text-surface-700 dark:text-surface-200 rounded-lg font-medium text-sm transition-all hover:bg-surface-100 dark:hover:bg-surface-700 border border-surface-200/80 dark:border-surface-700/80"
                  >
                    <span className="inline-flex items-center gap-2">
                      <AppleIcon size={14} />
                      {t("about.extension.iosShortcut")}
                    </span>
                    <ExternalLink size={12} />
                  </a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="border-t border-surface-200/60 dark:border-surface-800/60 bg-white dark:bg-surface-900/40">
        <div className="max-w-6xl mx-auto px-6 py-24 md:py-32">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-14 lg:gap-20 items-center">
            <div>
              <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-surface-100 dark:bg-surface-800 text-surface-600 dark:text-surface-400 text-xs font-medium mb-5 border border-surface-200/60 dark:border-surface-700/60">
                <Shield size={13} />
                {t("about.protocol.badge")}
              </div>
              <h2 className="font-display text-4xl md:text-5xl font-bold tracking-tight text-surface-900 dark:text-white mb-5 leading-[1.05]">
                {t("about.protocol.title")}
              </h2>
              <p className="text-lg text-surface-500 dark:text-surface-400 leading-relaxed mb-10 max-w-lg">
                {t("about.protocol.descriptionPre")}{" "}
                <a
                  href="https://atproto.com"
                  target="_blank"
                  rel="noreferrer"
                  className="text-primary-600 dark:text-primary-400 hover:underline font-medium"
                >
                  {t("about.hero.atProtocol")}
                </a>
                {t("about.protocol.descriptionPost")}
              </p>
              <ul className="space-y-4">
                {([0, 1, 2, 3] as const).map((i) => (
                  <li key={i} className="flex items-start gap-3">
                    <div className="mt-0.5 flex-shrink-0 w-5 h-5 rounded-md bg-primary-100 dark:bg-primary-950/60 flex items-center justify-center">
                      <Check
                        size={12}
                        className="text-primary-600 dark:text-primary-400"
                        strokeWidth={2.5}
                      />
                    </div>
                    <span className="text-[15px] text-surface-700 dark:text-surface-300 leading-relaxed">
                      {t(`about.protocol.point${i}`)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>

            <div className="rounded-2xl bg-surface-900 dark:bg-surface-950 p-5 md:p-6 text-sm font-mono shadow-2xl shadow-surface-900/10 border border-surface-800">
              <div className="flex items-center gap-2 mb-4 pb-3 border-b border-surface-800">
                <div className="flex gap-1">
                  <div className="w-2 h-2 rounded-full bg-surface-700" />
                  <div className="w-2 h-2 rounded-full bg-surface-700" />
                  <div className="w-2 h-2 rounded-full bg-surface-700" />
                </div>
                <div className="text-xs text-surface-500 ml-2">lexicon</div>
                <div className="text-xs text-primary-400 px-2 py-0.5 rounded bg-primary-400/10 ml-auto">
                  at.margin.note
                </div>
              </div>
              <div className="space-y-1 text-[13px] leading-relaxed">
                <span className="text-surface-500">{"{"}</span>
                <div className="pl-4">
                  <span className="text-green-400">"$type"</span>
                  <span className="text-surface-400">: </span>
                  <span className="text-amber-400">"at.margin.note"</span>
                  <span className="text-surface-400">,</span>
                </div>
                <div className="pl-4">
                  <span className="text-green-400">"motivation"</span>
                  <span className="text-surface-400">: </span>
                  <span className="text-amber-400">"highlighting"</span>
                  <span className="text-surface-400">,</span>
                </div>
                <div className="pl-4">
                  <span className="text-green-400">"body"</span>
                  <span className="text-surface-400">: {"{"}</span>
                </div>
                <div className="pl-8">
                  <span className="text-green-400">"value"</span>
                  <span className="text-surface-400">: </span>
                  <span className="text-amber-400">"Great insight..."</span>
                </div>
                <div className="pl-4">
                  <span className="text-surface-400">{"}"}</span>
                  <span className="text-surface-400">,</span>
                </div>
                <div className="pl-4">
                  <span className="text-green-400">"target"</span>
                  <span className="text-surface-400">: {"{"}</span>
                </div>
                <div className="pl-8">
                  <span className="text-green-400">"source"</span>
                  <span className="text-surface-400">: </span>
                  <span className="text-sky-400">"https://..."</span>
                  <span className="text-surface-400">,</span>
                </div>
                <div className="pl-8">
                  <span className="text-green-400">"selector"</span>
                  <span className="text-surface-400">: {"{"}</span>
                </div>
                <div className="pl-12">
                  <span className="text-green-400">"type"</span>
                  <span className="text-surface-400">: </span>
                  <span className="text-amber-400">"TextQuoteSelector"</span>
                  <span className="text-surface-400">,</span>
                </div>
                <div className="pl-12">
                  <span className="text-green-400">"exact"</span>
                  <span className="text-surface-400">: </span>
                  <span className="text-amber-400">"selected text"</span>
                </div>
                <div className="pl-8">
                  <span className="text-surface-400">{"}"}</span>
                </div>
                <div className="pl-4">
                  <span className="text-surface-400">{"}"}</span>
                </div>
                <span className="text-surface-500">{"}"}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="bg-primary-600 dark:bg-primary-700">
        <div className="max-w-6xl mx-auto px-6 py-24 md:py-32">
          <h2 className="font-display text-4xl md:text-6xl lg:text-7xl font-bold text-white mb-6 max-w-3xl tracking-tight leading-[1.02]">
            {t("about.cta.title")}
          </h2>
          <p className="text-primary-100 text-lg max-w-xl mb-12 leading-relaxed">
            {t("about.cta.description")}
          </p>
          <div className="flex flex-col sm:flex-row items-start gap-3 flex-wrap">
            <a
              href={user ? "/home" : "/login"}
              className="group inline-flex items-center gap-2 px-7 py-3.5 bg-white text-primary-700 hover:bg-primary-50 rounded-xl font-semibold transition-colors text-[15px] shadow-lg shadow-primary-900/20"
            >
              {user ? t("about.hero.openApp") : t("about.cta.signIn")}
              <ArrowRight
                size={16}
                className="transition-transform group-hover:translate-x-0.5"
              />
            </a>
            <a
              href="https://github.com/margin-at"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 px-7 py-3.5 text-white/90 hover:text-white border border-white/30 hover:border-white/50 hover:bg-white/5 rounded-xl font-semibold transition-colors text-[15px]"
            >
              <Github size={16} />
              {t("about.cta.viewGitHub")}
            </a>
            <a
              href="https://tangled.org/margin.at/margin"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 px-7 py-3.5 text-white/90 hover:text-white border border-white/30 hover:border-white/50 hover:bg-white/5 rounded-xl font-semibold transition-colors text-[15px]"
            >
              <TangledIcon size={16} />
              {t("about.cta.viewTangled")}
            </a>
          </div>
          <div className="mt-16 pt-8 border-t border-white/15 flex items-center gap-6 flex-wrap">
            <a
              href="https://ko-fi.com/scan"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-primary-100 hover:text-white transition-colors text-sm font-medium"
            >
              <Coffee size={15} />
              Ko-fi
            </a>
            <a
              href="https://opencollective.com/margin"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-primary-100 hover:text-white transition-colors text-sm font-medium"
            >
              <Heart size={15} />
              Open Collective
            </a>
          </div>
        </div>
      </section>

      <footer className="border-t border-surface-200/60 dark:border-surface-800/60">
        <div className="max-w-6xl mx-auto px-6 py-8">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-2.5">
              <img
                src="/logo.svg"
                alt="Margin"
                className="w-5 h-5 opacity-60"
              />
              <span className="text-sm text-surface-400 dark:text-surface-500">
                {t("sidebar.copyright")}
              </span>
            </div>
            <div className="flex items-center gap-5 text-sm text-surface-400 dark:text-surface-500 flex-wrap">
              {user && (
                <a
                  href="/home"
                  className="hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
                >
                  {t("nav.feed")}
                </a>
              )}
              <a
                href="/privacy"
                className="hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
              >
                {t("about.footer.privacy")}
              </a>
              <a
                href="/terms"
                className="hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
              >
                {t("about.footer.terms")}
              </a>
              <a
                href="/brand"
                className="hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
              >
                Brand
              </a>
              <a
                href="https://github.com/margin-at"
                target="_blank"
                rel="noreferrer"
                className="hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
              >
                GitHub
              </a>
              <a
                href="https://tangled.org/margin.at/margin"
                target="_blank"
                rel="noreferrer"
                className="hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
              >
                Tangled
              </a>
              <a
                href="mailto:hello@margin.at"
                className="hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
              >
                {t("about.footer.contact")}
              </a>
              <div className="w-px h-4 bg-surface-200 dark:bg-surface-700 ml-1" />
              <button
                onClick={cycleTheme}
                title={
                  theme === "light"
                    ? t("nav.themeLight")
                    : theme === "dark"
                      ? t("nav.themeDark")
                      : t("nav.themeSystem")
                }
                className="flex items-center gap-1.5 hover:text-surface-600 dark:hover:text-surface-300 transition-colors"
              >
                {theme === "light" ? (
                  <Sun size={15} />
                ) : theme === "dark" ? (
                  <Moon size={15} />
                ) : (
                  <Monitor size={15} />
                )}
                <span className="hidden sm:inline capitalize">{theme}</span>
              </button>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
