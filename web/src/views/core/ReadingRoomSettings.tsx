import { useEffect, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import {
  ExternalLink,
  Globe,
  Check,
  RefreshCw,
  Trash2,
  Palette,
  Loader2,
  ArrowLeft,
  ArrowRight,
  ImageOff,
  BookOpen,
  Sparkles,
  Type,
  Columns3,
  LayoutGrid,
  List,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Button, Input, FontPicker } from "../../components/ui";
import {
  getReadingRoomConfig,
  updateReadingRoomConfig,
  getBillingStatus,
  getCustomDomainStatus,
  addCustomDomain,
  pollCustomDomain,
  removeCustomDomain,
  type BillingStatus,
  type CustomDomainInfo,
} from "../../api/client";
import { useStore } from "@nanostores/react";
import { $user } from "../../store/auth";

const ACCENT_PRESETS = [
  "#3b82f6",
  "#8b5cf6",
  "#ec4899",
  "#ef4444",
  "#f59e0b",
  "#10b981",
  "#14b8a6",
  "#6366f1",
  "#000000",
];

const BG_PRESETS = [
  "#ffffff",
  "#fcfcfc",
  "#fdfbf7",
  "#f3f4f6",
  "#1a1a1a",
  "#0f172a",
];

const LAYOUTS: { id: string; icon: LucideIcon }[] = [
  { id: "masonry", icon: Columns3 },
  { id: "grid", icon: LayoutGrid },
  { id: "list", icon: List },
];

export default function ReadingRoomSettings() {
  const { t } = useTranslation();
  const user = useStore($user);
  const [billing, setBilling] = useState<BillingStatus | null>(null);
  const [domain, setDomain] = useState<CustomDomainInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<"success" | "error" | null>(
    null,
  );
  const [ogVersion, setOgVersion] = useState(() => Date.now());
  const [ogImageFailed, setOgImageFailed] = useState(false);
  const [domainInput, setDomainInput] = useState("");
  const [domainLoading, setDomainLoading] = useState(false);
  const [polling, setPolling] = useState(false);

  const [title, setTitle] = useState("");
  const [subtitle, setSubtitle] = useState("");
  const [description, setDescription] = useState("");
  const [backgroundColor, setBackgroundColor] = useState("#fcfcfc");
  const [accentColor, setAccentColor] = useState("#3b82f6");
  const [fontFamily, setFontFamily] = useState("sans-serif");
  const [layout, setLayout] = useState("masonry");
  const [showExternalBookmarks, setShowExternalBookmarks] = useState(true);

  useEffect(() => {
    Promise.all([
      getReadingRoomConfig(),
      getBillingStatus(),
      getCustomDomainStatus(),
    ]).then(([cfg, bill, dom]) => {
      setBilling(bill);
      setDomain(dom);
      if (cfg) {
        setTitle(cfg.title);
        setSubtitle(cfg.subtitle);
        setDescription(cfg.description);
        setBackgroundColor(cfg.theme.backgroundColor || "#fcfcfc");
        setAccentColor(cfg.theme.accentColor || "#3b82f6");
        setFontFamily(cfg.theme.fontFamily || "sans-serif");
        setLayout(cfg.theme.layout || "masonry");
        setShowExternalBookmarks(cfg.showExternalBookmarks ?? true);
      }
      setLoading(false);
    });
  }, []);

  const hasSub = billing?.hasSubscription || false;

  useEffect(() => {
    if (!saveResult) return;
    const timer = window.setTimeout(() => setSaveResult(null), 3000);
    return () => window.clearTimeout(timer);
  }, [saveResult]);

  const handleSave = async () => {
    setSaving(true);
    setSaveResult(null);
    const ok = await updateReadingRoomConfig({
      title,
      subtitle,
      description,
      theme: { backgroundColor, accentColor, fontFamily, layout },
      showExternalBookmarks,
    });
    setSaving(false);
    setSaveResult(ok ? "success" : "error");
    if (ok) {
      setOgImageFailed(false);
      setOgVersion(Date.now());
    }
  };

  const handleAddDomain = async () => {
    if (!domainInput.trim()) return;
    setDomainLoading(true);
    const result = await addCustomDomain(domainInput.trim());
    if (result) setDomain(result);
    setDomainInput("");
    setDomainLoading(false);
  };

  const handlePollDomain = async () => {
    setPolling(true);
    const result = await pollCustomDomain();
    if (result) setDomain(result);
    setPolling(false);
  };

  const handleRemoveDomain = async () => {
    await removeCustomDomain();
    setDomain({ domain: "", status: "", verificationRecords: [] });
  };

  const header = (
    <div className="flex items-center gap-3 mb-8">
      <Link
        to="/settings"
        className="p-2 -ml-2 rounded-full hover:bg-surface-100 dark:hover:bg-surface-800 transition-colors text-surface-500 hover:text-surface-900 dark:hover:text-white"
        aria-label="Back to settings"
      >
        <ArrowLeft size={20} />
      </Link>
      <h1 className="text-3xl font-display font-bold text-surface-900 dark:text-white">
        {t("settings.sections.readingRoom")}
      </h1>
    </div>
  );

  if (loading || billing === null) {
    return (
      <div className="max-w-2xl mx-auto animate-slide-up">
        {header}
        <div className="card p-5">
          <div className="flex items-center justify-center p-12 text-surface-400">
            <Loader2 size={24} className="animate-spin" />
          </div>
        </div>
      </div>
    );
  }

  if (!hasSub) {
    return (
      <div className="max-w-2xl mx-auto animate-slide-up">
        {header}
        <div className="card p-8 sm:p-10 text-center">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-50 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 ring-1 ring-primary-100 dark:ring-primary-800/50">
            <BookOpen size={26} />
          </div>
          <h2 className="mt-5 text-xl font-display font-semibold text-surface-900 dark:text-white">
            {t("settings.readingRoom.lockedTitle")}
          </h2>
          <p className="mt-2.5 mx-auto max-w-md text-sm leading-relaxed text-surface-500 dark:text-surface-400">
            {t("settings.readingRoom.lockedDesc")}
          </p>
          <div className="mt-7 flex flex-wrap items-center justify-center gap-3">
            <a
              href="/pro"
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-primary-600 text-white rounded-lg text-sm font-semibold hover:bg-primary-500 transition-colors shadow-sm shadow-primary-600/25"
            >
              <Sparkles size={15} />
              {t("settings.readingRoom.lockedCta")}
            </a>
            <a
              href="/pro"
              className="inline-flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium text-surface-600 dark:text-surface-300 hover:text-surface-900 dark:hover:text-white transition-colors"
            >
              {t("settings.readingRoom.lockedLearnMore")}
              <ArrowRight size={14} />
            </a>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto animate-slide-up pb-28">
      {header}

      <div className="space-y-6">
        {/* Profile */}
        <section className="card p-5 sm:p-6">
          <SectionHeading
            title={t("settings.readingRoom.profile")}
            desc={t("settings.readingRoom.profileDesc")}
          />
          <div className="space-y-4">
            <Input
              label={t("settings.readingRoom.titleLabel")}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("settings.readingRoom.titlePlaceholder")}
            />
            <Input
              label={t("settings.readingRoom.subtitleLabel")}
              value={subtitle}
              onChange={(e) => setSubtitle(e.target.value)}
              placeholder={t("settings.readingRoom.subtitlePlaceholder")}
            />
            <div>
              <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1.5">
                {t("settings.readingRoom.descriptionLabel")}
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t("settings.readingRoom.descriptionPlaceholder")}
                rows={3}
                className="w-full px-3 py-2 rounded-lg bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-sm text-surface-900 dark:text-white placeholder:text-surface-400 dark:placeholder:text-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 dark:focus:border-primary-400 transition-colors resize-none"
              />
            </div>
            <label className="flex items-center justify-between gap-3 cursor-pointer pt-1">
              <div>
                <span className="block text-sm font-medium text-surface-700 dark:text-surface-300">
                  {t("settings.readingRoom.showExternalBookmarks")}
                </span>
                <span className="block text-xs text-surface-500 dark:text-surface-400 mt-0.5">
                  {t("settings.readingRoom.showExternalBookmarksDesc")}
                </span>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={showExternalBookmarks}
                onClick={() => setShowExternalBookmarks(!showExternalBookmarks)}
                className={`relative inline-flex h-5 w-9 shrink-0 rounded-full transition-colors ${showExternalBookmarks ? "bg-primary-600" : "bg-surface-300 dark:bg-surface-600"}`}
              >
                <span
                  className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${showExternalBookmarks ? "translate-x-[18px]" : "translate-x-0.5"} mt-0.5`}
                />
              </button>
            </label>
          </div>
        </section>

        {/* Theme */}
        <section className="card p-5 sm:p-6">
          <SectionHeading
            icon={Palette}
            title={t("settings.readingRoom.theme")}
            desc={t("settings.readingRoom.themeDesc")}
          />
          <div className="space-y-6">
            <SwatchRow
              label={t("settings.readingRoom.backgroundColor")}
              presets={BG_PRESETS}
              value={backgroundColor}
              onChange={setBackgroundColor}
            />
            <SwatchRow
              label={t("settings.readingRoom.accentColor")}
              presets={ACCENT_PRESETS}
              value={accentColor}
              onChange={setAccentColor}
            />

            <div className="grid gap-6 sm:grid-cols-2">
              <div>
                <FieldLabel icon={Type}>
                  {t("settings.readingRoom.font")}
                </FieldLabel>
                <FontPicker value={fontFamily} onChange={setFontFamily} />
              </div>
              <div>
                <FieldLabel>{t("settings.readingRoom.layout")}</FieldLabel>
                <div className="inline-flex w-full rounded-lg border border-surface-200 dark:border-surface-700 p-0.5 bg-surface-50 dark:bg-surface-800/60">
                  {LAYOUTS.map((l) => {
                    const Icon = l.icon;
                    const active = layout === l.id;
                    return (
                      <button
                        key={l.id}
                        type="button"
                        onClick={() => setLayout(l.id)}
                        className={`flex-1 flex items-center justify-center gap-1.5 h-9 rounded-md text-sm font-medium capitalize transition-all ${
                          active
                            ? "bg-white dark:bg-surface-700 text-surface-900 dark:text-white shadow-sm"
                            : "text-surface-500 dark:text-surface-400 hover:text-surface-800 dark:hover:text-surface-200"
                        }`}
                      >
                        <Icon size={14} />
                        <span className="hidden xs:inline sm:inline">
                          {l.id}
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Social preview */}
        {user?.handle && (
          <section className="card p-5 sm:p-6">
            <div className="flex items-start justify-between gap-4">
              <SectionHeading
                title={t("settings.readingRoom.socialPreview")}
                desc={t("settings.readingRoom.socialPreviewDesc")}
                noMargin
              />
              <button
                onClick={() => {
                  setOgImageFailed(false);
                  setOgVersion(Date.now());
                }}
                className="shrink-0 flex items-center gap-1.5 text-xs font-medium text-surface-500 dark:text-surface-400 hover:text-surface-900 dark:hover:text-white transition-colors"
              >
                <RefreshCw size={12} />
                {t("settings.readingRoom.refreshPreview")}
              </button>
            </div>
            <div className="mt-4 rounded-xl overflow-hidden border border-surface-200 dark:border-surface-700 bg-surface-50 dark:bg-surface-800">
              {ogImageFailed ? (
                <div className="aspect-[1200/630] flex flex-col items-center justify-center gap-2 text-surface-400 dark:text-surface-500">
                  <ImageOff size={20} />
                  <span className="text-xs">
                    {t("settings.readingRoom.previewUnavailable")}
                  </span>
                </div>
              ) : (
                <img
                  key={ogVersion}
                  src={`/og-image?readingRoom=${encodeURIComponent(user.handle)}&v=${ogVersion}`}
                  alt={t("settings.readingRoom.socialPreview")}
                  className="w-full aspect-[1200/630] object-cover"
                  onError={() => setOgImageFailed(true)}
                />
              )}
            </div>
          </section>
        )}

        {/* Custom domain */}
        <section className="card p-5 sm:p-6">
          <SectionHeading
            icon={Globe}
            title={t("settings.readingRoom.customDomain")}
          />

          {domain?.domain ? (
            <div className="space-y-3">
              <div className="flex items-center justify-between gap-3 p-3 bg-surface-50 dark:bg-surface-800 rounded-lg border border-surface-200 dark:border-surface-700">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-surface-900 dark:text-white truncate">
                    {domain.domain}
                  </p>
                  <p className="text-xs text-surface-500 dark:text-surface-400 mt-0.5">
                    {t(
                      `settings.readingRoom.domainStatus.${domain.status || "pending"}`,
                    )}
                  </p>
                </div>
                <div className="flex gap-1 shrink-0">
                  {domain.status !== "active" && (
                    <Button
                      onClick={handlePollDomain}
                      loading={polling}
                      variant="ghost"
                      size="sm"
                      icon={<RefreshCw size={14} />}
                    />
                  )}
                  <Button
                    onClick={handleRemoveDomain}
                    variant="ghost"
                    size="sm"
                    icon={<Trash2 size={14} />}
                  />
                </div>
              </div>

              {domain.status !== "active" &&
                domain.verificationRecords?.length > 0 && (
                  <div className="bg-surface-50 dark:bg-surface-800 rounded-lg p-4 text-sm space-y-3 border border-surface-200 dark:border-surface-700">
                    <p className="font-medium text-surface-700 dark:text-surface-300">
                      {t("settings.readingRoom.dnsInstructions")}
                    </p>
                    <div>
                      <p className="text-xs text-surface-500 mb-1">
                        {t("settings.readingRoom.dnsCname")}
                      </p>
                      <code className="block bg-white dark:bg-surface-900 p-2.5 rounded border border-surface-100 dark:border-surface-800 text-surface-800 dark:text-surface-200 font-mono text-xs">
                        {domain.domain} → reading.margin.at
                      </code>
                    </div>
                    {domain.verificationRecords.map((rec, i) => (
                      <div key={i} className="space-y-1">
                        {rec.name && rec.value && (
                          <>
                            <p className="text-xs text-surface-500 mb-1">
                              TXT: {rec.name}
                            </p>
                            <code className="block bg-white dark:bg-surface-900 p-2.5 rounded border border-surface-100 dark:border-surface-800 text-surface-800 dark:text-surface-200 font-mono text-xs">
                              {rec.value}
                            </code>
                          </>
                        )}
                      </div>
                    ))}
                  </div>
                )}
            </div>
          ) : (
            <div className="flex items-start gap-2">
              <div className="flex-1">
                <Input
                  value={domainInput}
                  onChange={(e) => setDomainInput(e.target.value)}
                  placeholder="notes.example.com"
                  onKeyDown={(e) => e.key === "Enter" && handleAddDomain()}
                />
              </div>
              <Button onClick={handleAddDomain} loading={domainLoading}>
                {t("settings.readingRoom.addDomain")}
              </Button>
            </div>
          )}
        </section>
      </div>

      {/* Sticky save bar */}
      <div className="sticky bottom-4 mt-6 z-10">
        <div className="bg-white dark:bg-surface-800 rounded-xl border border-surface-200 dark:border-surface-700 shadow-md px-4 py-3 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3 min-w-0">
            <Button
              onClick={handleSave}
              loading={saving}
              className="px-6 font-medium"
            >
              {t("settings.readingRoom.save")}
            </Button>
            {saveResult === "success" && (
              <span className="flex items-center gap-1.5 text-sm text-emerald-600 dark:text-emerald-400 animate-fade-in">
                <Check size={15} />
                {t("settings.readingRoom.saved")}
              </span>
            )}
            {saveResult === "error" && (
              <span className="text-sm text-red-600 dark:text-red-400 animate-fade-in">
                {t("settings.readingRoom.saveFailed")}
              </span>
            )}
          </div>

          {user?.handle && (
            <a
              href={
                domain?.status === "active" && domain?.domain
                  ? `https://${domain.domain}`
                  : `/reading-room/${user.handle}`
              }
              className="shrink-0 text-sm font-medium flex items-center gap-1.5 hover:opacity-80 transition-opacity text-primary-600 dark:text-primary-400"
            >
              {t("settings.readingRoom.view")}
              <ExternalLink size={14} />
            </a>
          )}
        </div>
      </div>
    </div>
  );
}

function SectionHeading({
  icon: Icon,
  title,
  desc,
  noMargin,
}: {
  icon?: LucideIcon;
  title: string;
  desc?: string;
  noMargin?: boolean;
}) {
  return (
    <div className={noMargin ? "" : "mb-5"}>
      <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider flex items-center gap-2">
        {Icon && <Icon size={14} />}
        {title}
      </h2>
      {desc && (
        <p className="mt-1.5 text-sm text-surface-500 dark:text-surface-400">
          {desc}
        </p>
      )}
    </div>
  );
}

function FieldLabel({
  icon: Icon,
  children,
}: {
  icon?: LucideIcon;
  children: ReactNode;
}) {
  return (
    <label className="flex items-center gap-1.5 text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">
      {Icon && <Icon size={14} className="text-surface-400" />}
      {children}
    </label>
  );
}

function SwatchRow({
  label,
  presets,
  value,
  onChange,
}: {
  label: string;
  presets: string[];
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <FieldLabel>{label}</FieldLabel>
      <div className="flex items-center gap-2.5 flex-wrap">
        {presets.map((color) => (
          <button
            key={color}
            type="button"
            onClick={() => onChange(color)}
            aria-label={color}
            className={`w-8 h-8 rounded-full transition-transform hover:scale-110 border ${
              value.toLowerCase() === color.toLowerCase()
                ? "ring-2 ring-offset-2 ring-primary-500 dark:ring-offset-surface-900 border-transparent"
                : "border-surface-200 dark:border-surface-700"
            }`}
            style={{ backgroundColor: color }}
          />
        ))}
        <label
          className="relative w-8 h-8 rounded-full overflow-hidden border border-dashed border-surface-300 dark:border-surface-600 cursor-pointer grid place-items-center text-surface-400"
          title="Custom color"
        >
          <input
            type="color"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className="absolute inset-0 opacity-0 cursor-pointer"
          />
          <Palette size={13} />
        </label>
      </div>
    </div>
  );
}
