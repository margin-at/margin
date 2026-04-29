import React, { useState, useEffect, useRef } from "react";
import { AtSign, ShieldOff } from "lucide-react";
import { useTranslation } from "react-i18next";
import "../../i18n";
import SignUpModal from "../../components/modals/SignUpModal";
import {
  searchActors,
  startLogin,
  type ActorSearchItem,
} from "../../api/client";
import { Avatar } from "../../components/ui";
import { useStore } from "@nanostores/react";
import { $theme } from "../../store/theme";
import { analytics } from "../../lib/analytics";

interface LoginProps {
  initialError?: string;
}

export default function Login({ initialError }: LoginProps) {
  const { t } = useTranslation();
  useStore($theme); // ensure theme is applied on this page
  const [handle, setHandle] = useState("");
  const [suggestions, setSuggestions] = useState<ActorSearchItem[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(initialError || null);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [showSignUp, setShowSignUp] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionsRef = useRef<HTMLDivElement>(null);
  const isSelectionRef = useRef(false);

  const [providerIndex, setProviderIndex] = useState(0);
  const [morphClass, setMorphClass] = useState(
    "opacity-100 translate-y-0 blur-0",
  );
  const providers = [
    "AT Protocol",
    "Margin",
    "Bluesky",
    "Eurosky",
    "Blacksky",
    "Tangled",
    "Northsky",
    "selfhosted.social",
    "witchcraft.systems",
    "tophhie.social",
    "altq.net",
  ];

  const [selectedAvatar, setSelectedAvatar] = useState<string | null>(null);

  useEffect(() => {
    const cycleText = () => {
      setMorphClass("opacity-0 translate-y-2 blur-sm");
      setTimeout(() => {
        setProviderIndex((prev) => (prev + 1) % providers.length);
        setMorphClass("opacity-100 translate-y-0 blur-0");
      }, 400);
    };
    const interval = setInterval(cycleText, 3000);
    return () => clearInterval(interval);
  }, [providers.length]);

  useEffect(() => {
    if (handle.length >= 3) {
      if (isSelectionRef.current) {
        isSelectionRef.current = false;
        return;
      }
      const timer = setTimeout(async () => {
        try {
          if (!handle.includes(".")) {
            const data = await searchActors(handle);
            setSuggestions(data.actors || []);

            const exactMatch = data.actors?.find((s) => s.handle === handle);
            if (exactMatch) {
              setSelectedAvatar(exactMatch.avatar || null);
            }

            setShowSuggestions(true);
            setSelectedIndex(-1);
          }
        } catch (e) {
          console.error("Search failed:", e);
        }
      }, 300);
      return () => clearTimeout(timer);
    }
  }, [handle]);

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

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showSuggestions || suggestions.length === 0) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prev) => Math.min(prev + 1, suggestions.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prev) => Math.max(prev - 1, -1));
    } else if (e.key === "Enter" && selectedIndex >= 0) {
      e.preventDefault();
      selectSuggestion(suggestions[selectedIndex]);
    } else if (e.key === "Escape") {
      setShowSuggestions(false);
    }
  };

  const selectSuggestion = (actor: ActorSearchItem) => {
    isSelectionRef.current = true;
    setHandle(actor.handle);
    setSelectedAvatar(actor.avatar || null);
    setSuggestions([]);
    setShowSuggestions(false);
    inputRef.current?.blur();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!handle.trim()) return;

    setLoading(true);
    setError(null);

    try {
      analytics.capture("login_initiated", { handle: handle.trim() });
      const result = await startLogin(handle.trim());
      if (result.authorizationUrl) {
        const url = new URL(result.authorizationUrl);
        if (url.protocol !== "https:")
          throw new Error("Invalid authorization URL");
        window.location.href = result.authorizationUrl;
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      analytics.captureException(err);
      setError(message || "Failed to initiate login. Please try again.");
      setLoading(false);
    }
  };

  if (initialError === "banned") {
    return (
      <div className="relative min-h-screen flex items-center justify-center bg-surface-100 dark:bg-surface-800 p-4 overflow-hidden">
        <div className="pointer-events-none absolute inset-0 -z-0">
          <div className="absolute top-1/4 left-1/2 -translate-x-1/2 h-96 w-96 rounded-full bg-red-200/30 dark:bg-red-900/20 blur-3xl" />
        </div>
        <div className="relative w-full max-w-[440px] bg-white dark:bg-surface-900 rounded-2xl border border-surface-200/60 dark:border-surface-800 p-8 shadow-sm dark:shadow-none text-center">
          <div className="flex justify-center mb-5">
            <div className="w-14 h-14 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
              <ShieldOff size={28} className="text-red-500" />
            </div>
          </div>
          <h1 className="text-xl font-bold font-display text-surface-900 dark:text-white mb-2">
            {t("login.bannedTitle")}
          </h1>
          <p className="text-sm text-surface-500 dark:text-surface-400 mb-1 leading-relaxed">
            {t("login.bannedMessage")}
          </p>
          <p className="text-sm text-surface-500 dark:text-surface-400 mb-6 leading-relaxed">
            {t("login.bannedAppeal")}{" "}
            <a
              href="mailto:hello@margin.at"
              className="text-[#027bff] hover:underline font-medium"
            >
              hello@margin.at
            </a>
            .
          </p>
          <button
            onClick={async () => {
              await fetch("/auth/logout", { method: "POST" }).catch(() => {});
              window.location.href = "/login";
            }}
            className="w-full py-3 bg-surface-100 dark:bg-surface-800 hover:bg-surface-200 dark:hover:bg-surface-700 text-surface-700 dark:text-surface-300 rounded-xl font-semibold transition-all text-sm"
          >
            {t("login.bannedSignOut")}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="relative min-h-screen flex items-center justify-center bg-surface-100 dark:bg-surface-800 p-4 overflow-hidden">
      <div className="pointer-events-none absolute inset-0 -z-0">
        <div className="absolute top-1/4 left-1/2 -translate-x-1/2 h-96 w-96 rounded-full bg-primary-200/30 dark:bg-primary-900/20 blur-3xl" />
      </div>
      <div className="relative w-full max-w-[440px] bg-white dark:bg-surface-900 rounded-2xl border border-surface-200/60 dark:border-surface-800 p-8 shadow-sm dark:shadow-none">
        <div className="flex flex-col items-center mb-8">
          <h1 className="text-2xl font-bold font-display text-surface-900 dark:text-white text-center leading-snug">
            {t("login.signInWith")} <br />
            <span
              className={`inline-block transition-all duration-400 ease-out text-transparent bg-clip-text bg-gradient-to-r from-[#027bff] to-[#0285FF] ${morphClass}`}
            >
              {providers[providerIndex]}
            </span>{" "}
            {t("login.handleSuffix")}
          </h1>
        </div>

        <form onSubmit={handleSubmit} className="w-full flex flex-col gap-4">
          <div className="relative group">
            <div className="absolute left-4 top-1/2 -translate-y-1/2 text-surface-400 dark:text-surface-500 transition-colors pointer-events-none">
              {selectedAvatar ? (
                <Avatar
                  src={selectedAvatar}
                  size="xs"
                  className="ring-2 ring-white dark:ring-surface-900 shadow-sm"
                />
              ) : (
                <AtSign
                  size={20}
                  className="stroke-[2.5] group-focus-within:text-[#027bff]"
                />
              )}
            </div>
            <input
              ref={inputRef}
              type="text"
              value={handle}
              onChange={(e) => {
                const val = e.target.value;
                setHandle(val);
                if (selectedAvatar) setSelectedAvatar(null);
                if (val.length < 3) {
                  setSuggestions([]);
                  setShowSuggestions(false);
                }
              }}
              onKeyDown={handleKeyDown}
              onFocus={() =>
                handle.length >= 3 &&
                suggestions.length > 0 &&
                !handle.includes(".") &&
                setShowSuggestions(true)
              }
              placeholder={t("login.handlePlaceholder")}
              className="w-full pl-12 pr-4 py-3.5 bg-surface-50 dark:bg-surface-950 border border-surface-200 dark:border-surface-700 rounded-xl focus:border-[#027bff] dark:focus:border-[#027bff] outline-none focus:ring-4 focus:ring-[#027bff]/10 transition-all font-medium text-lg text-surface-900 dark:text-white placeholder:text-surface-400 dark:placeholder:text-surface-500"
              autoCapitalize="none"
              autoCorrect="off"
              autoComplete="off"
              spellCheck={false}
              disabled={loading}
            />

            {showSuggestions && suggestions.length > 0 && (
              <div
                ref={suggestionsRef}
                className="absolute top-[calc(100%+8px)] left-0 right-0 bg-white dark:bg-surface-900 border border-surface-200 dark:border-surface-700 rounded-xl shadow-xl overflow-hidden z-50 animate-fade-in max-h-[300px] overflow-y-auto"
              >
                {suggestions.map((actor, index) => (
                  <button
                    key={actor.did}
                    type="button"
                    className={`w-full flex items-center gap-3 px-4 py-3 border-b border-surface-100 dark:border-surface-800 last:border-0 hover:bg-surface-50 dark:hover:bg-surface-800 transition-colors text-left ${index === selectedIndex ? "bg-surface-50 dark:bg-surface-800" : ""}`}
                    onClick={() => selectSuggestion(actor)}
                  >
                    <Avatar src={actor.avatar} size="sm" />
                    <div className="min-w-0">
                      <div className="font-semibold text-surface-900 dark:text-white truncate text-sm">
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

          {error && (
            <div className="p-3 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm rounded-lg border border-red-100 dark:border-red-800 text-center font-medium animate-fade-in">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading || !handle}
            className="w-full py-3.5 bg-[#027bff] hover:bg-[#0269d9] text-white rounded-xl font-semibold text-base tracking-wide disabled:opacity-40 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2 mt-2"
          >
            {loading ? t("login.connecting") : t("login.continue")}
          </button>

          <p className="text-center text-sm text-surface-400 dark:text-surface-500 mt-2 leading-relaxed">
            {t("login.termsPrefix")}{" "}
            <a
              href="/terms"
              className="text-surface-900 dark:text-white hover:underline font-medium hover:text-[#027bff] dark:hover:text-[#027bff] transition-colors"
            >
              {t("login.termsLink")}
            </a>{" "}
            {t("login.termsAnd")}{" "}
            <a
              href="/privacy"
              className="text-surface-900 dark:text-white hover:underline font-medium hover:text-[#027bff] dark:hover:text-[#027bff] transition-colors"
            >
              {t("login.privacyLink")}
            </a>
          </p>

          <div className="flex items-center justify-center py-1">
            <span className="text-xs text-surface-300 dark:text-surface-600">
              {t("login.or")}
            </span>
          </div>

          <button
            type="button"
            onClick={() => setShowSignUp(true)}
            className="w-full py-2.5 text-sm font-medium text-surface-500 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white bg-surface-50 dark:bg-surface-800/60 hover:bg-surface-100 dark:hover:bg-surface-800 rounded-xl transition-colors"
          >
            {t("login.createAccount")}
          </button>
        </form>
      </div>

      {showSignUp && <SignUpModal onClose={() => setShowSignUp(false)} />}
    </div>
  );
}
