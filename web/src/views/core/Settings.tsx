import React, { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { useTranslation } from "react-i18next";
import { languages } from "virtual:i18n-languages";
import { $user, logout } from "../../store/auth";
import { $theme, setTheme, type Theme } from "../../store/theme";
import {
  $preferences,
  loadPreferences,
  addLabeler,
  removeLabeler,
  setLabelVisibility,
  getLabelVisibility,
  setDisableExternalLinkWarning,
  setEnableCommunityBookmarks,
} from "../../store/preferences";
import {
  getAPIKeys,
  createAPIKey,
  deleteAPIKey,
  getBlocks,
  getMutes,
  unblockUser,
  unmuteUser,
  getLabelerInfo,
  getBillingStatus,
  createPortal,
  type APIKey,
  type BillingStatus,
} from "../../api/client";
import type {
  BlockedUser,
  MutedUser,
  LabelerInfo,
  LabelVisibility as LabelVisibilityType,
  ContentLabelValue,
} from "../../types";
import {
  Copy,
  Trash2,
  Key,
  Plus,
  Check,
  Sun,
  Moon,
  Monitor,
  LogOut,
  ChevronRight,
  ShieldBan,
  VolumeX,
  ShieldOff,
  Volume2,
  Shield,
  Eye,
  EyeOff,
  XCircle,
  Upload,
  Sparkles,
  CreditCard,
  ArrowRight,
} from "lucide-react";
import {
  Avatar,
  Button,
  Input,
  Skeleton,
  EmptyState,
  Switch,
} from "../../components/ui";
import { AppleIcon } from "../../components/common/Icons";
import { HighlightImporter } from "./HighlightImporter";
import IOSShortcutModal from "../../components/modals/IOSShortcutModal";
import { analytics } from "../../lib/analytics";
import { displayHandle } from "../../lib/handle";

export default function Settings() {
  const { t } = useTranslation();
  const user = useStore($user);
  const theme = useStore($theme);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [newKeyName, setNewKeyName] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [justCopied, setJustCopied] = useState(false);
  const [creating, setCreating] = useState(false);
  const [blocks, setBlocks] = useState<BlockedUser[]>([]);
  const [mutes, setMutes] = useState<MutedUser[]>([]);
  const [modLoading, setModLoading] = useState(true);
  const [labelerInfo, setLabelerInfo] = useState<LabelerInfo | null>(null);
  const [newLabelerDid, setNewLabelerDid] = useState("");
  const [addingLabeler, setAddingLabeler] = useState(false);
  const [isShortcutModalOpen, setIsShortcutModalOpen] = useState(false);
  const [billing, setBilling] = useState<BillingStatus | null>(null);
  const [portalLoading, setPortalLoading] = useState(false);
  const preferences = useStore($preferences);
  const { i18n: i18nInstance } = useTranslation();
  const [currentLanguage, setCurrentLanguage] = useState(
    () => i18nInstance.resolvedLanguage ?? i18nInstance.language ?? "en",
  );

  useEffect(() => {
    const handler = (lng: string) => setCurrentLanguage(lng);
    i18nInstance.on("languageChanged", handler);
    return () => {
      i18nInstance.off("languageChanged", handler);
    };
  }, [i18nInstance]);

  useEffect(() => {
    const loadKeys = async () => {
      setLoading(true);
      const data = await getAPIKeys();
      setKeys(data);
      setLoading(false);
    };
    loadKeys();

    const loadModeration = async () => {
      setModLoading(true);
      const [blocksData, mutesData] = await Promise.all([
        getBlocks(),
        getMutes(),
      ]);
      setBlocks(blocksData);
      setMutes(mutesData);
      setModLoading(false);
    };
    loadModeration();

    loadPreferences();
    getLabelerInfo().then(setLabelerInfo);
    getBillingStatus().then(setBilling);
  }, []);

  const handlePortal = async () => {
    setPortalLoading(true);
    const url = await createPortal();
    if (url) window.location.href = url;
    else setPortalLoading(false);
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newKeyName.trim()) return;

    setCreating(true);
    const res = await createAPIKey(newKeyName);
    if (res) {
      setKeys([res, ...keys]);
      setCreatedKey(res.key || null);
      setNewKeyName("");
      analytics.capture("api_key_created");
    }
    setCreating(false);
  };

  const handleDelete = async (id: string) => {
    if (window.confirm(t("settings.apiKeys.revokeConfirm"))) {
      const success = await deleteAPIKey(id);
      if (success) {
        setKeys((prev) => prev.filter((k) => k.id !== id));
      }
    }
  };

  const copyToClipboard = async (text: string) => {
    await navigator.clipboard.writeText(text);
    setJustCopied(true);
    setTimeout(() => setJustCopied(false), 2000);
  };

  if (!user) return null;

  const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
    { value: "light", label: t("nav.themeLight"), icon: Sun },
    { value: "dark", label: t("nav.themeDark"), icon: Moon },
    { value: "system", label: t("nav.themeSystem"), icon: Monitor },
  ];

  return (
    <div className="max-w-2xl mx-auto animate-slide-up">
      <h1 className="text-3xl font-display font-bold text-surface-900 dark:text-white mb-8">
        {t("settings.title")}
      </h1>

      <div className="space-y-6">
        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-4">
            {t("settings.sections.profile")}
          </h2>
          <div className="flex gap-4 items-center">
            <Avatar did={user.did} avatar={user.avatar} size="lg" />
            <div className="flex-1">
              <p className="font-semibold text-surface-900 dark:text-white text-lg">
                {user.displayName || user.handle}
              </p>
              <p className="text-surface-500 dark:text-surface-400">
                @{displayHandle(user.handle, user.did)}
              </p>
            </div>
            <ChevronRight
              className="text-surface-300 dark:text-surface-600"
              size={20}
            />
          </div>
        </section>

        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-4">
            {t("settings.sections.appearance")}
          </h2>
          <div className="flex gap-2">
            {themeOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => {
                  setTheme(opt.value);
                  analytics.capture("theme_changed", {
                    theme: opt.value,
                  });
                }}
                className={`flex-1 flex flex-col items-center gap-2 p-4 rounded-xl border-2 transition-all ${
                  theme === opt.value
                    ? "border-primary-500 bg-primary-50 dark:bg-primary-900/20"
                    : "border-surface-200 dark:border-surface-700 hover:border-surface-300 dark:hover:border-surface-600"
                }`}
              >
                <opt.icon
                  size={24}
                  className={
                    theme === opt.value
                      ? "text-primary-600 dark:text-primary-400"
                      : "text-surface-400 dark:text-surface-500"
                  }
                />
                <span
                  className={`text-sm font-medium ${theme === opt.value ? "text-primary-600 dark:text-primary-400" : "text-surface-600 dark:text-surface-400"}`}
                >
                  {opt.label}
                </span>
              </button>
            ))}
          </div>

          <div className="mt-6 flex items-center justify-between gap-4">
            <div className="min-w-0 flex-1">
              <h3 className="text-sm font-medium text-surface-900 dark:text-white">
                {t("settings.appearance.disableExternalLinkWarning")}
              </h3>
              <p className="text-sm text-surface-500 dark:text-surface-400">
                {t("settings.appearance.disableExternalLinkWarningDesc")}
              </p>
            </div>
            <Switch
              checked={preferences.disableExternalLinkWarning}
              onCheckedChange={setDisableExternalLinkWarning}
              className="shrink-0"
            />
          </div>

          <div className="mt-6 flex items-center justify-between gap-4">
            <div className="min-w-0 flex-1">
              <h3 className="text-sm font-medium text-surface-900 dark:text-white">
                {t("settings.appearance.communityBookmarks")}
              </h3>
              <p className="text-sm text-surface-500 dark:text-surface-400">
                {t("settings.appearance.communityBookmarksDesc")}
              </p>
            </div>
            <Switch
              checked={preferences.enableCommunityBookmarks}
              onCheckedChange={setEnableCommunityBookmarks}
              className="shrink-0"
            />
          </div>
        </section>

        {languages.length > 1 && (
          <section className="card p-5">
            <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-4">
              {t("settings.sections.language")}
            </h2>
            <div>
              <p className="text-sm text-surface-500 dark:text-surface-400 mb-3">
                {t("settings.language.description")}
              </p>
              <div className="flex flex-wrap gap-2">
                {languages.map((lang) => (
                  <button
                    key={lang.code}
                    onClick={() => i18nInstance.changeLanguage(lang.code)}
                    className={`px-4 py-2 rounded-xl text-sm font-medium border-2 transition-all ${
                      currentLanguage.startsWith(lang.code)
                        ? "border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-300"
                        : "border-surface-200 dark:border-surface-700 text-surface-600 dark:text-surface-400 hover:border-surface-300 dark:hover:border-surface-600"
                    }`}
                  >
                    {lang.nativeName}
                    {lang.nativeName !== lang.name && (
                      <span className="ml-1.5 text-xs opacity-60">
                        ({lang.name})
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          </section>
        )}

        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <Upload size={16} />
            {t("settings.sections.batchImport")}
          </h2>
          <p className="text-sm text-surface-500 dark:text-surface-400 mb-4">
            {t("settings.batchImport.description")}
          </p>
          <HighlightImporter />
        </section>

        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-1">
            {t("settings.sections.apiKeys")}
          </h2>
          <p className="text-sm text-surface-400 dark:text-surface-500 mb-5">
            {t("settings.apiKeys.description")}
          </p>

          <form onSubmit={handleCreate} className="flex gap-2 mb-5">
            <div className="flex-1">
              <Input
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                placeholder={t("settings.apiKeys.keyNamePlaceholder")}
              />
            </div>
            <Button
              type="submit"
              disabled={!newKeyName.trim()}
              loading={creating}
              icon={<Plus size={16} />}
            >
              {t("settings.apiKeys.generate")}
            </Button>
          </form>

          {createdKey && (
            <div className="mb-5 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-xl animate-scale-in">
              <div className="flex items-start gap-3">
                <div className="p-2 bg-green-100 dark:bg-green-900/40 rounded-lg">
                  <Key
                    size={16}
                    className="text-green-600 dark:text-green-400"
                  />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-green-800 dark:text-green-200 text-sm font-medium mb-2">
                    {t("settings.apiKeys.copyNow")}
                  </p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 bg-white dark:bg-surface-900 border border-green-200 dark:border-green-800 px-3 py-2 rounded-lg text-xs font-mono text-green-900 dark:text-green-100 break-all">
                      {createdKey}
                    </code>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => copyToClipboard(createdKey)}
                      icon={
                        justCopied ? <Check size={16} /> : <Copy size={16} />
                      }
                    />
                  </div>
                </div>
              </div>
            </div>
          )}

          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-16 rounded-xl" />
              <Skeleton className="h-16 rounded-xl" />
            </div>
          ) : keys.length === 0 ? (
            <EmptyState
              icon={<Key size={40} />}
              message={t("settings.apiKeys.empty")}
            />
          ) : (
            <div className="space-y-2">
              {keys.map((key) => (
                <div
                  key={key.id}
                  className="flex items-center justify-between p-4 bg-surface-50 dark:bg-surface-800 rounded-xl group transition-all hover:bg-surface-100 dark:hover:bg-surface-700"
                >
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-surface-200 dark:bg-surface-700 rounded-lg">
                      <Key
                        size={16}
                        className="text-surface-500 dark:text-surface-400"
                      />
                    </div>
                    <div>
                      <p className="font-medium text-surface-900 dark:text-white">
                        {key.name}
                      </p>
                      <p className="text-xs text-surface-500 dark:text-surface-400">
                        {t("settings.apiKeys.created", {
                          date: new Date(key.createdAt).toLocaleDateString(),
                        })}
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDelete(key.id)}
                    className="p-2 text-surface-400 dark:text-surface-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all opacity-0 group-hover:opacity-100"
                  >
                    <Trash2 size={18} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-1">
            {t("settings.sections.moderation")}
          </h2>
          <p className="text-sm text-surface-400 dark:text-surface-500 mb-5">
            {t("settings.moderation.description")}
          </p>

          {modLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-14 rounded-xl" />
              <Skeleton className="h-14 rounded-xl" />
            </div>
          ) : (
            <div className="space-y-4">
              <div>
                <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-2 flex items-center gap-2">
                  <ShieldBan size={14} />
                  {t("settings.moderation.blockedAccounts", {
                    count: blocks.length,
                  })}
                </h3>
                {blocks.length === 0 ? (
                  <p className="text-sm text-surface-400 dark:text-surface-500 pl-6">
                    {t("settings.moderation.noBlocked")}
                  </p>
                ) : (
                  <div className="space-y-1.5">
                    {blocks.map((b) => (
                      <div
                        key={b.did}
                        className="flex items-center justify-between p-3 bg-surface-50 dark:bg-surface-800 rounded-xl group hover:bg-surface-100 dark:hover:bg-surface-700 transition-all"
                      >
                        <a
                          href={`/profile/${b.did}`}
                          className="flex items-center gap-3 min-w-0 flex-1"
                        >
                          <Avatar
                            did={b.did}
                            avatar={b.author?.avatar}
                            size="sm"
                          />
                          <div className="min-w-0">
                            <p className="font-medium text-surface-900 dark:text-white text-sm truncate">
                              {b.author?.displayName ||
                                b.author?.handle ||
                                b.did}
                            </p>
                            {b.author?.handle && (
                              <p className="text-xs text-surface-400 dark:text-surface-500 truncate">
                                @{displayHandle(b.author.handle, b.author.did)}
                              </p>
                            )}
                          </div>
                        </a>
                        <button
                          onClick={async () => {
                            await unblockUser(b.did);
                            setBlocks((prev) =>
                              prev.filter((x) => x.did !== b.did),
                            );
                          }}
                          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-surface-500 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all opacity-0 group-hover:opacity-100"
                        >
                          <ShieldOff size={12} />
                          {t("settings.moderation.unblock")}
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div>
                <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-2 flex items-center gap-2">
                  <VolumeX size={14} />
                  {t("settings.moderation.mutedAccounts", {
                    count: mutes.length,
                  })}
                </h3>
                {mutes.length === 0 ? (
                  <p className="text-sm text-surface-400 dark:text-surface-500 pl-6">
                    {t("settings.moderation.noMuted")}
                  </p>
                ) : (
                  <div className="space-y-1.5">
                    {mutes.map((m) => (
                      <div
                        key={m.did}
                        className="flex items-center justify-between p-3 bg-surface-50 dark:bg-surface-800 rounded-xl group hover:bg-surface-100 dark:hover:bg-surface-700 transition-all"
                      >
                        <a
                          href={`/profile/${m.did}`}
                          className="flex items-center gap-3 min-w-0 flex-1"
                        >
                          <Avatar
                            did={m.did}
                            avatar={m.author?.avatar}
                            size="sm"
                          />
                          <div className="min-w-0">
                            <p className="font-medium text-surface-900 dark:text-white text-sm truncate">
                              {m.author?.displayName ||
                                m.author?.handle ||
                                m.did}
                            </p>
                            {m.author?.handle && (
                              <p className="text-xs text-surface-400 dark:text-surface-500 truncate">
                                @{displayHandle(m.author.handle, m.author.did)}
                              </p>
                            )}
                          </div>
                        </a>
                        <button
                          onClick={async () => {
                            await unmuteUser(m.did);
                            setMutes((prev) =>
                              prev.filter((x) => x.did !== m.did),
                            );
                          }}
                          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-surface-500 hover:text-amber-600 dark:hover:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-900/20 rounded-lg transition-all opacity-0 group-hover:opacity-100"
                        >
                          <Volume2 size={12} />
                          {t("settings.moderation.unmute")}
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </section>

        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-1">
            {t("settings.sections.contentFiltering")}
          </h2>
          <p className="text-sm text-surface-400 dark:text-surface-500 mb-5">
            {t("settings.contentFiltering.description")}
          </p>

          <div className="space-y-5">
            <div>
              <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3 flex items-center gap-2">
                <Shield size={14} />
                {t("settings.contentFiltering.subscribedLabelers")}
              </h3>

              {preferences.subscribedLabelers.length === 0 ? (
                <p className="text-sm text-surface-400 dark:text-surface-500 pl-6 mb-3">
                  {t("settings.contentFiltering.noLabelers")}
                </p>
              ) : (
                <div className="space-y-1.5 mb-3">
                  {preferences.subscribedLabelers.map((labeler) => (
                    <div
                      key={labeler.did}
                      className="flex items-center justify-between p-3 bg-surface-50 dark:bg-surface-800 rounded-xl group hover:bg-surface-100 dark:hover:bg-surface-700 transition-all"
                    >
                      <div className="flex items-center gap-3 min-w-0 flex-1">
                        <div className="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
                          <Shield
                            size={14}
                            className="text-primary-600 dark:text-primary-400"
                          />
                        </div>
                        <div className="min-w-0">
                          <p className="font-medium text-surface-900 dark:text-white text-sm truncate">
                            {labelerInfo?.did === labeler.did
                              ? labelerInfo.name
                              : labeler.did}
                          </p>
                          <p className="text-xs text-surface-400 dark:text-surface-500 truncate font-mono">
                            {labeler.did}
                          </p>
                        </div>
                      </div>
                      <button
                        onClick={() => removeLabeler(labeler.did)}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-surface-500 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all opacity-0 group-hover:opacity-100"
                      >
                        <XCircle size={12} />
                        {t("settings.contentFiltering.remove")}
                      </button>
                    </div>
                  ))}
                </div>
              )}

              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  if (!newLabelerDid.trim()) return;
                  setAddingLabeler(true);
                  await addLabeler(newLabelerDid.trim());
                  setNewLabelerDid("");
                  setAddingLabeler(false);
                }}
                className="flex gap-2"
              >
                <div className="flex-1">
                  <Input
                    value={newLabelerDid}
                    onChange={(e) => setNewLabelerDid(e.target.value)}
                    placeholder={t(
                      "settings.contentFiltering.labelerDidPlaceholder",
                    )}
                  />
                </div>
                <Button
                  type="submit"
                  disabled={!newLabelerDid.trim()}
                  loading={addingLabeler}
                  icon={<Plus size={16} />}
                >
                  {t("settings.contentFiltering.add")}
                </Button>
              </form>
            </div>

            {preferences.subscribedLabelers.length > 0 && (
              <div>
                <h3 className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3 flex items-center gap-2">
                  <Eye size={14} />
                  {t("settings.contentFiltering.labelVisibility")}
                </h3>
                <p className="text-xs text-surface-400 dark:text-surface-500 mb-3 pl-6">
                  {t("settings.contentFiltering.labelVisibilityDesc")}
                </p>

                <div className="space-y-4">
                  {preferences.subscribedLabelers.map((labeler) => {
                    const labels: ContentLabelValue[] = [
                      "sexual",
                      "nudity",
                      "violence",
                      "gore",
                      "spam",
                      "misleading",
                    ];
                    return (
                      <div
                        key={labeler.did}
                        className="bg-surface-50 dark:bg-surface-800 rounded-xl p-4"
                      >
                        <p className="text-sm font-medium text-surface-700 dark:text-surface-300 mb-3 truncate">
                          {labelerInfo?.did === labeler.did
                            ? labelerInfo.name
                            : labeler.did}
                        </p>
                        <div className="space-y-2">
                          {labels.map((label) => {
                            const current = getLabelVisibility(
                              labeler.did,
                              label,
                            );
                            const options: {
                              value: LabelVisibilityType;
                              label: string;
                              icon: typeof Eye;
                            }[] = [
                              {
                                value: "warn",
                                label: t("settings.contentFiltering.warn"),
                                icon: EyeOff,
                              },
                              {
                                value: "hide",
                                label: t("settings.contentFiltering.hide"),
                                icon: XCircle,
                              },
                              {
                                value: "ignore",
                                label: t("settings.contentFiltering.ignore"),
                                icon: Eye,
                              },
                            ];
                            return (
                              <div
                                key={label}
                                className="flex items-center justify-between gap-2 py-1.5"
                              >
                                <span className="text-sm text-surface-600 dark:text-surface-400 min-w-0 flex-1">
                                  {t(`card.labelDescriptions.${label}`)}
                                </span>
                                <div className="flex gap-1">
                                  {options.map((opt) => (
                                    <button
                                      key={opt.value}
                                      onClick={() =>
                                        setLabelVisibility(
                                          labeler.did,
                                          label,
                                          opt.value,
                                        )
                                      }
                                      className={`px-2.5 py-1 text-xs font-medium rounded-lg transition-all flex items-center gap-1 ${
                                        current === opt.value
                                          ? opt.value === "hide"
                                            ? "bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400"
                                            : opt.value === "warn"
                                              ? "bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400"
                                              : "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400"
                                          : "text-surface-400 dark:text-surface-500 hover:bg-surface-200 dark:hover:bg-surface-700"
                                      }`}
                                    >
                                      <opt.icon size={12} />
                                      {opt.label}
                                    </button>
                                  ))}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </section>

        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-1">
            {t("settings.sections.iosShortcut")}
          </h2>
          <p className="text-sm text-surface-400 dark:text-surface-500 mb-4">
            {t("settings.iosShortcut.description")}
          </p>
          <button
            onClick={() => setIsShortcutModalOpen(true)}
            className="inline-flex items-center gap-2.5 px-4 py-2.5 bg-surface-900 dark:bg-white text-white dark:text-surface-900 rounded-xl font-medium text-sm transition-all hover:opacity-90"
          >
            <AppleIcon size={16} />
            {t("settings.iosShortcut.setupButton")}
          </button>
        </section>

        <section className="card p-5">
          <h2 className="text-xs font-semibold text-surface-500 dark:text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <Sparkles size={16} />
            {t("settings.sections.billing")}
          </h2>
          {billing === null ? (
            <div className="flex items-center justify-between gap-4">
              <div className="min-w-0 space-y-2">
                <div className="h-4 w-28 rounded bg-surface-100 dark:bg-surface-700/50 animate-pulse" />
                <div className="h-3 w-44 rounded bg-surface-100 dark:bg-surface-700/50 animate-pulse" />
              </div>
              <div className="h-9 w-24 rounded-lg bg-surface-100 dark:bg-surface-700/50 animate-pulse shrink-0" />
            </div>
          ) : billing.hasSubscription ? (
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="min-w-0">
                <p className="text-sm font-medium text-surface-900 dark:text-white flex items-center gap-2">
                  <Check
                    size={15}
                    className="text-emerald-600 dark:text-emerald-400"
                  />
                  {t("settings.billing.proActive")}
                </p>
                <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">
                  {billing.plan === "yearly"
                    ? t("settings.billing.yearlyPlan")
                    : t("settings.billing.monthlyPlan")}
                  {billing.currentPeriodEnd &&
                    ` · ${t("settings.billing.renewsOn", {
                      date: new Date(
                        billing.currentPeriodEnd,
                      ).toLocaleDateString(),
                    })}`}
                </p>
              </div>
              <Button
                onClick={handlePortal}
                loading={portalLoading}
                variant="secondary"
                size="sm"
                icon={<CreditCard size={14} />}
              >
                {t("settings.billing.manage")}
              </Button>
            </div>
          ) : (
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="min-w-0">
                <p className="text-sm font-medium text-surface-900 dark:text-white">
                  {t("settings.billing.freeTitle")}
                </p>
                <p className="text-sm text-surface-500 dark:text-surface-400 mt-1 max-w-md">
                  {t("settings.billing.freeDesc")}
                </p>
              </div>
              <a
                href="/pro"
                className="inline-flex items-center gap-1.5 px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-500 transition-colors shrink-0"
              >
                <Sparkles size={14} />
                {t("settings.billing.upgrade")}
                <ArrowRight size={14} />
              </a>
            </div>
          )}
        </section>

        <section className="card p-5">
          <a
            href="/settings/reading-room"
            className="flex items-center justify-between gap-3 w-full text-left p-3 -m-3 rounded-xl transition-colors hover:bg-surface-50 dark:hover:bg-surface-800"
          >
            <div className="flex flex-col">
              <span className="font-medium text-surface-900 dark:text-white">
                {t("settings.sections.readingRoom")}
              </span>
              <span className="text-sm text-surface-500">
                {t("settings.readingRoom.description")}
              </span>
            </div>
            <ChevronRight size={18} className="text-surface-400" />
          </a>
        </section>

        <section className="card p-5">
          <button
            onClick={logout}
            className="flex items-center gap-3 w-full text-left text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 p-3 -m-3 rounded-xl transition-colors"
          >
            <LogOut size={20} />
            <span className="font-medium">{t("settings.logout")}</span>
          </button>
        </section>
      </div>

      <IOSShortcutModal
        isOpen={isShortcutModalOpen}
        onClose={() => setIsShortcutModalOpen(false)}
      />
    </div>
  );
}
