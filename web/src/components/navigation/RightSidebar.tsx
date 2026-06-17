import React, { useEffect, useRef, useState } from "react";
import { Search, Coffee, Heart } from "lucide-react";
import {
  getTrendingTags,
  searchActors,
  type ActorSearchItem,
  type Tag,
} from "../../api/client";
import { Avatar } from "../ui";
import { useTranslation } from "react-i18next";

function looksLikeUrl(query: string): boolean {
  const q = query.trim().toLowerCase();
  return (
    q.startsWith("http://") ||
    q.startsWith("https://") ||
    /\.(com|org|net|io|dev|me|co|app|xyz|edu|gov)\b/.test(q)
  );
}

interface RightSidebarProps {
  onNavigate?: (path: string) => void;
}

export default function RightSidebar({ onNavigate }: RightSidebarProps) {
  const { t } = useTranslation();
  const navigate = (path: string) => {
    if (onNavigate) onNavigate(path);
    else window.location.href = path;
  };
  const [tags, setTags] = useState<Tag[]>([]);
  const [browser] = useState<
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
  const [searchQuery, setSearchQuery] = useState("");
  const [suggestions, setSuggestions] = useState<ActorSearchItem[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);

  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionsRef = useRef<HTMLDivElement>(null);
  const isSelectionRef = useRef(false);
  const latestQueryRef = useRef(searchQuery);

  useEffect(() => {
    latestQueryRef.current = searchQuery;

    if (searchQuery.length < 3 || looksLikeUrl(searchQuery)) {
      return;
    }

    if (isSelectionRef.current) {
      isSelectionRef.current = false;
      return;
    }

    const capturedQuery = searchQuery;
    const timer = setTimeout(async () => {
      try {
        const data = await searchActors(capturedQuery);
        if (capturedQuery !== latestQueryRef.current) return;
        setSuggestions(data.actors || []);
        setShowSuggestions((data.actors || []).length > 0);
        setSelectedIndex(-1);
      } catch (e) {
        console.error("Search failed:", e);
      }
    }, 300);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        suggestionsRef.current &&
        !suggestionsRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        setShowSuggestions(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const selectSuggestion = (actor: ActorSearchItem) => {
    isSelectionRef.current = true;
    setSearchQuery("");
    setSuggestions([]);
    setShowSuggestions(false);
    navigate(`/profile/${encodeURIComponent(actor.handle)}`);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (showSuggestions && suggestions.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((prev) => Math.min(prev + 1, suggestions.length - 1));
        return;
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((prev) => Math.max(prev - 1, -1));
        return;
      } else if (e.key === "Enter" && selectedIndex >= 0) {
        e.preventDefault();
        selectSuggestion(suggestions[selectedIndex]);
        return;
      } else if (e.key === "Escape") {
        setShowSuggestions(false);
        return;
      }
    }

    if (e.key === "Enter" && searchQuery.trim()) {
      const q = searchQuery.trim();
      if (looksLikeUrl(q)) {
        navigate(`/url/${encodeURIComponent(q)}`);
      } else if (q.includes(".")) {
        navigate(`/profile/${encodeURIComponent(q)}`);
      } else {
        navigate(`/search?q=${encodeURIComponent(q)}`);
      }
      setSearchQuery("");
      setSuggestions([]);
      setShowSuggestions(false);
    }
  };

  useEffect(() => {
    getTrendingTags(10).then(setTags);
  }, []);

  const extensionLink =
    browser === "firefox"
      ? "https://addons.mozilla.org/en-US/firefox/addon/margin/"
      : browser === "edge"
        ? "https://microsoftedge.microsoft.com/addons/detail/margin/nfjnmllpdgcdnhmmggjihjbidmeadddn"
        : browser === "safari"
          ? "https://apps.apple.com/us/app/margin-for-safari/id6773549512"
          : "https://chromewebstore.google.com/detail/margin/cgpmbiiagnehkikhcbnhiagfomajncpa";

  return (
    <aside className="hidden xl:block w-[320px] shrink-0 sticky top-0 h-screen overflow-y-auto px-6 py-6">
      <div className="space-y-5">
        <div className="relative">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Search
              className="text-surface-400 dark:text-surface-500"
              size={15}
            />
          </div>
          <input
            ref={inputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              if (e.target.value.length < 3) {
                setSuggestions([]);
                setShowSuggestions(false);
              }
            }}
            onKeyDown={handleKeyDown}
            onFocus={() =>
              searchQuery.length >= 3 &&
              suggestions.length > 0 &&
              setShowSuggestions(true)
            }
            placeholder={t("sidebar.searchPlaceholder")}
            className="w-full bg-surface-100 dark:bg-surface-800/80 rounded-lg pl-9 pr-4 py-2 text-sm text-surface-900 dark:text-white placeholder:text-surface-400 dark:placeholder:text-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:bg-white dark:focus:bg-surface-800 transition-all border border-transparent focus:border-surface-200 dark:focus:border-surface-700"
          />

          {showSuggestions && suggestions.length > 0 && (
            <div
              ref={suggestionsRef}
              className="absolute top-[calc(100%+6px)] left-0 right-0 bg-white dark:bg-surface-900 border border-surface-200 dark:border-surface-700 rounded-xl shadow-xl overflow-hidden z-50 animate-fade-in max-h-[280px] overflow-y-auto"
            >
              {suggestions.map((actor, index) => (
                <button
                  key={actor.did}
                  type="button"
                  className={`w-full flex items-center gap-3 px-3.5 py-2.5 border-b border-surface-100 dark:border-surface-800 last:border-0 hover:bg-surface-50 dark:hover:bg-surface-800 transition-colors text-left ${index === selectedIndex ? "bg-surface-50 dark:bg-surface-800" : ""}`}
                  onClick={() => selectSuggestion(actor)}
                >
                  <Avatar src={actor.avatar} size="sm" />
                  <div className="min-w-0 flex-1">
                    <div className="font-semibold text-surface-900 dark:text-white truncate text-sm leading-tight">
                      {actor.displayName || actor.handle}
                    </div>
                    <div className="text-surface-500 dark:text-surface-400 text-xs truncate">
                      @{actor.handle}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="rounded-xl p-4 bg-gradient-to-br from-primary-50 to-primary-100/50 dark:from-primary-950/30 dark:to-primary-900/10 border border-primary-200/40 dark:border-primary-800/30">
          <h3 className="font-semibold text-sm mb-1 text-surface-900 dark:text-white">
            {t("sidebar.getExtension")}
          </h3>
          <p className="text-surface-500 dark:text-surface-400 text-xs mb-3 leading-relaxed">
            {t("sidebar.extensionTagline")}
          </p>
          <a
            href={extensionLink}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center justify-center w-full px-4 py-2 bg-primary-600 hover:bg-primary-700 dark:bg-primary-500 dark:hover:bg-primary-400 text-white dark:text-white rounded-lg transition-colors text-sm font-medium"
          >
            {browser === "firefox"
              ? t("sidebar.downloadForFirefox")
              : browser === "edge"
                ? t("sidebar.downloadForEdge")
                : browser === "safari"
                  ? t("sidebar.downloadForSafari")
                  : t("sidebar.downloadForChrome")}
          </a>
        </div>

        <div>
          <h3 className="font-semibold text-sm px-1 mb-3 text-surface-900 dark:text-white tracking-tight">
            {t("sidebar.trending")}
          </h3>
          {tags.length > 0 ? (
            <div className="flex flex-col">
              {tags.map((tag) => (
                <a
                  key={tag.tag}
                  href={`/home?tag=${encodeURIComponent(tag.tag)}`}
                  className="px-2 py-2.5 hover:bg-surface-100 dark:hover:bg-surface-800 rounded-lg transition-colors group"
                >
                  <div className="font-semibold text-sm text-surface-900 dark:text-white group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
                    #{tag.tag}
                  </div>
                  <div className="text-xs text-surface-400 dark:text-surface-500 mt-0.5">
                    {t("sidebar.postCount", { count: tag.count })}
                  </div>
                </a>
              ))}
            </div>
          ) : (
            <div className="px-2">
              <p className="text-sm text-surface-400 dark:text-surface-500">
                {t("sidebar.nothingTrending")}
              </p>
            </div>
          )}
        </div>

        <div className="px-1 pt-2">
          <div className="flex flex-wrap gap-x-3 gap-y-1 text-[12px] text-surface-400 dark:text-surface-500 leading-relaxed">
            <a
              href="/about"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              About
            </a>
            <a
              href="/privacy"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              Privacy
            </a>
            <a
              href="/terms"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              Terms
            </a>
            <a
              href="https://github.com/margin-at/margin"
              target="_blank"
              rel="noreferrer"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              GitHub
            </a>
            <a
              href="https://tangled.org/margin.at/margin"
              target="_blank"
              rel="noreferrer"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              Tangled
            </a>
            <a
              href="https://discord.gg/ZQbkGqwzBH"
              target="_blank"
              rel="noreferrer"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              Discord
            </a>
            <a
              href="https://matrix.to/#/#margin:blep.cat"
              target="_blank"
              rel="noreferrer"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              Matrix
            </a>
            <a
              href="https://stt.gg/wHnM6e3h"
              target="_blank"
              rel="noreferrer"
              className="hover:underline hover:text-surface-600 dark:hover:text-surface-300"
            >
              Stoat
            </a>
            <a
              href="https://opencollective.com/margin"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-[12px] text-surface-400 dark:text-surface-500 hover:text-[#7FADF2] dark:hover:text-[#7FADF2] transition-colors"
            >
              <Heart size={12} className="shrink-0" />
              Open Collective
            </a>
            <a
              href="https://ko-fi.com/scan"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-[12px] text-surface-400 dark:text-surface-500 hover:text-[#FF5E5B] dark:hover:text-[#FF5E5B] transition-colors"
            >
              <Coffee size={12} className="shrink-0" />
              Ko-fi
            </a>
            <span>{t("sidebar.copyright")}</span>
          </div>
        </div>
      </div>
    </aside>
  );
}
