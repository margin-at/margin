import React, { useState, useEffect, useMemo } from "react";
import { X, ChevronRight, Loader2, AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  BlackskyIcon,
  NorthskyIcon,
  BlueskyIcon,
  TophhieIcon,
  MarginIcon,
  EuroskuIcon,
} from "../common/Icons";
import { startSignup } from "../../api/client";
import { analytics } from "../../lib/analytics";

interface Provider {
  id: string;
  name: string;
  service: string;
  Icon: React.ComponentType<{ size?: number }> | null;
  description: string;
  custom?: boolean;
  wide?: boolean;
}

type ProviderBase = {
  id: string;
  service: string;
  Icon: React.ComponentType<{ size?: number }> | null;
  custom?: boolean;
};

const MARGIN_PROVIDER_BASE: ProviderBase = {
  id: "margin",
  service: "https://margin.cafe",
  Icon: MarginIcon,
};

const OTHER_PROVIDERS_BASE: ProviderBase[] = [
  { id: "bluesky", service: "https://bsky.social", Icon: BlueskyIcon },
  { id: "blacksky", service: "https://blacksky.app", Icon: BlackskyIcon },
  { id: "eurosky", service: "https://eurosky.social", Icon: EuroskuIcon },
  { id: "selfhosted.social", service: "https://selfhosted.social", Icon: null },
  { id: "northsky", service: "https://northsky.social", Icon: NorthskyIcon },
  { id: "tophhie", service: "https://tophhie.social", Icon: TophhieIcon },
  { id: "custom", service: "", custom: true, Icon: null },
];

function shuffleArray<T>(arr: T[]): T[] {
  const shuffled = [...arr];
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
  }
  return shuffled;
}

const inviteStatusPromise: Promise<Record<string, boolean>> = (async () => {
  const results: Record<string, boolean> = {};
  await Promise.allSettled(
    [MARGIN_PROVIDER_BASE, ...OTHER_PROVIDERS_BASE]
      .filter((p) => p.service && !p.custom)
      .map(async (p) => {
        try {
          const res = await fetch(
            `${p.service}/xrpc/com.atproto.server.describeServer`,
          );
          if (res.ok) {
            const data = await res.json();
            results[p.id] = !!data.inviteCodeRequired;
          }
        } catch {
          // ignore unreachable providers
        }
      }),
  );
  return results;
})();

interface SignUpModalProps {
  onClose: () => void;
}

export default function SignUpModal({ onClose }: SignUpModalProps) {
  const { t } = useTranslation();
  const [showCustomInput, setShowCustomInput] = useState(false);
  const [showMore, setShowMore] = useState(false);
  const [customService, setCustomService] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [inviteStatus, setInviteStatus] = useState<Record<string, boolean>>({});
  const [statusLoaded, setStatusLoaded] = useState(false);

  const MARGIN_PROVIDER: Provider = {
    ...MARGIN_PROVIDER_BASE,
    name: t("signUp.providers.margin.name"),
    description: t("signUp.providers.margin.description"),
  };

  const providerI18nKey: Record<string, string> = {
    bluesky: "bluesky",
    blacksky: "blacksky",
    eurosky: "eurosky",
    "selfhosted.social": "selfhostedSocial",
    northsky: "northsky",
    tophhie: "tophhie",
    custom: "customPds",
  };
  const OTHER_PROVIDERS: Provider[] = OTHER_PROVIDERS_BASE.map((p) => {
    const k = providerI18nKey[p.id] || p.id;
    return {
      ...p,
      name: t(`signUp.providers.${k}.name`),
      description: t(`signUp.providers.${k}.description`),
    };
  });

  useEffect(() => {
    inviteStatusPromise.then((status) => {
      setInviteStatus(status);
      setStatusLoaded(true);
    });
  }, []);

  const providers = useMemo(() => {
    const nonCustom = OTHER_PROVIDERS.filter((p) => !p.custom);
    const custom = OTHER_PROVIDERS.find((p) => p.custom);

    if (!statusLoaded) {
      return [
        MARGIN_PROVIDER,
        ...shuffleArray(nonCustom),
        ...(custom ? [custom] : []),
      ];
    }

    const open = nonCustom.filter((p) => !inviteStatus[p.id]);
    const inviteOnly = nonCustom.filter((p) => inviteStatus[p.id]);
    return [
      MARGIN_PROVIDER,
      ...shuffleArray(open),
      ...shuffleArray(inviteOnly),
      ...(custom ? [custom] : []),
    ];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusLoaded, inviteStatus]);

  useEffect(() => {
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = "unset";
    };
  }, []);

  const handleProviderSelect = async (provider: Provider) => {
    if (provider.custom) {
      setShowCustomInput(true);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      analytics.capture("signup_initiated", { provider: provider.id });
      const result = await startSignup(provider.service);
      if (result.authorizationUrl) {
        window.location.assign(result.authorizationUrl);
      }
    } catch (err) {
      console.error(err);
      analytics.captureException(err);
      setError(t("signUp.providerError"));
      setLoading(false);
    }
  };

  const handleCustomSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!customService.trim()) return;

    setLoading(true);
    setError(null);

    let serviceUrl = customService.trim();
    if (!serviceUrl.startsWith("http")) {
      serviceUrl = `https://${serviceUrl}`;
    }

    try {
      analytics.capture("signup_initiated", { provider: "custom" });
      const result = await startSignup(serviceUrl);
      if (result.authorizationUrl) {
        const url = new URL(result.authorizationUrl);
        if (url.protocol !== "https:")
          throw new Error("Invalid authorization URL");
        window.location.href = result.authorizationUrl;
      }
    } catch (err) {
      console.error(err);
      analytics.captureException(err);
      setError(t("signUp.customPdsError"));
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[100] flex items-end sm:items-center justify-center sm:p-4 bg-black/60 backdrop-blur-sm animate-fade-in">
      <div className="w-full sm:max-w-md bg-white dark:bg-surface-900 rounded-t-3xl sm:rounded-3xl shadow-2xl overflow-hidden animate-slide-up flex flex-col">
        <div className="px-5 sm:px-8 pt-5 sm:pt-6 pb-2 flex items-center justify-between flex-shrink-0">
          <h2 className="text-xl font-display font-bold text-surface-900 dark:text-white">
            {loading
              ? t("signUp.connecting")
              : showCustomInput
                ? t("signUp.customPdsTitle")
                : t("signUp.title")}
          </h2>
          <button
            onClick={onClose}
            className="p-2 text-surface-400 dark:text-surface-500 hover:text-surface-900 dark:hover:text-white hover:bg-surface-50 dark:hover:bg-surface-800 rounded-full transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        <div className="px-5 sm:px-8 pb-8 sm:pb-10 overflow-y-auto custom-scrollbar">
          {loading ? (
            <div className="text-center py-10">
              <Loader2
                size={40}
                className="animate-spin text-primary-600 dark:text-primary-400 mx-auto mb-4"
              />
            </div>
          ) : showCustomInput ? (
            <div>
              <h2 className="sr-only">{t("signUp.customPdsTitle")}</h2>

              <p className="text-sm text-surface-500 dark:text-surface-400 mb-6">
                {t("signUp.customPdsSubtitle")}
              </p>
              <form onSubmit={handleCustomSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1">
                    {t("signUp.pdsAddressLabel")}
                  </label>
                  <input
                    type="text"
                    className="w-full px-4 py-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl text-surface-900 dark:text-white placeholder:text-surface-400 dark:placeholder:text-surface-500 focus:border-primary-500 dark:focus:border-primary-400 focus:ring-4 focus:ring-primary-500/10 dark:focus:ring-primary-400/10 outline-none transition-all"
                    value={customService}
                    onChange={(e) => setCustomService(e.target.value)}
                    placeholder={t("signUp.pdsAddressPlaceholder")}
                    autoFocus
                  />
                </div>

                {error && (
                  <div className="p-3 bg-red-50 dark:bg-red-950/40 text-red-600 dark:text-red-400 text-sm rounded-lg flex items-center gap-2 border border-red-100 dark:border-red-900/40">
                    <AlertCircle size={16} />
                    {error}
                  </div>
                )}

                <div className="flex gap-3 pt-4">
                  <button
                    type="button"
                    className="flex-1 py-3 bg-white dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-surface-700 dark:text-surface-300 font-semibold rounded-xl hover:bg-surface-50 dark:hover:bg-surface-700 transition-colors"
                    onClick={() => {
                      setShowCustomInput(false);
                      setError(null);
                    }}
                  >
                    {t("signUp.back")}
                  </button>
                  <button
                    type="submit"
                    className="flex-1 py-3 bg-primary-600 dark:bg-primary-500 text-white font-semibold rounded-xl hover:bg-primary-700 dark:hover:bg-primary-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    disabled={!customService.trim()}
                  >
                    {t("signUp.continue")}
                  </button>
                </div>
              </form>
            </div>
          ) : (
            <div>
              <p className="text-surface-500 dark:text-surface-400 mb-6">
                {t("signUp.subtitle")}{" "}
                <a
                  href="https://atproto.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary-600 dark:text-primary-400 hover:underline"
                >
                  {t("signUp.atProtocol")}
                </a>
                {t("signUp.subtitleSuffix")}
              </p>

              {error && (
                <div className="mb-4 p-3 bg-red-50 dark:bg-red-950/40 text-red-600 dark:text-red-400 text-sm rounded-lg flex items-center gap-2 border border-red-100 dark:border-red-900/40">
                  <AlertCircle size={16} />
                  {error}
                </div>
              )}

              <div className="space-y-2">
                {(showMore ? providers : providers.slice(0, 1)).map((p) => (
                  <button
                    key={p.id}
                    className={`w-full flex items-center gap-3 p-3 rounded-xl transition-all text-left group ${
                      p.id === "margin"
                        ? "bg-primary-50/60 dark:bg-primary-900/15 border border-transparent hover:bg-primary-100/60 dark:hover:bg-primary-900/25"
                        : "bg-surface-50 dark:bg-surface-800/60 hover:bg-surface-100 dark:hover:bg-surface-800 border border-transparent"
                    }`}
                    onClick={() => handleProviderSelect(p)}
                  >
                    <div
                      className={`w-9 h-9 flex items-center justify-center rounded-full flex-shrink-0 ${
                        p.id === "margin"
                          ? "bg-primary-100 dark:bg-primary-900/40 text-primary-600 dark:text-primary-400"
                          : "bg-white dark:bg-surface-700 shadow-sm dark:shadow-none text-surface-600 dark:text-surface-300"
                      }`}
                    >
                      {p.Icon ? (
                        <p.Icon size={18} />
                      ) : (
                        <span className="font-bold text-xs">{p.name[0]}</span>
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <h3 className="text-sm font-bold text-surface-900 dark:text-white">
                        {p.name}
                      </h3>
                      <p className="text-xs text-surface-500 dark:text-surface-400 line-clamp-1">
                        {p.description}
                      </p>
                    </div>
                    {inviteStatus[p.id] && (
                      <span className="text-[10px] font-medium text-surface-400 dark:text-surface-500 bg-surface-100 dark:bg-surface-800 px-1.5 py-0.5 rounded-md flex-shrink-0">
                        {t("signUp.invite")}
                      </span>
                    )}
                    <ChevronRight
                      size={16}
                      className="text-surface-300 dark:text-surface-600 group-hover:text-surface-600 dark:group-hover:text-surface-400"
                    />
                  </button>
                ))}
              </div>

              {!showMore && (
                <div className="mt-3 space-y-3">
                  <button
                    onClick={() => setShowMore(true)}
                    className="w-full py-2.5 text-sm font-medium text-surface-500 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white bg-surface-50 dark:bg-surface-800/60 hover:bg-surface-100 dark:hover:bg-surface-800 rounded-xl border border-transparent transition-colors"
                  >
                    {t("signUp.moreOptions")}
                  </button>
                  <p className="text-center text-xs text-surface-400 dark:text-surface-500">
                    {t("signUp.atmosphereNote")}
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
