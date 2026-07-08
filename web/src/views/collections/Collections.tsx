import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  getCollections,
  createCollection,
  deleteCollection,
} from "../../api/client";
import { Plus, Folder, Trash2, X, Loader2 } from "lucide-react";
import CollectionIcon from "../../components/common/CollectionIcon";
import { ICON_MAP } from "../../components/common/iconMap";
import { useStore } from "@nanostores/react";
import { $user } from "../../store/auth";
import { Theme } from "emoji-picker-react";
const EmojiPicker = React.lazy(() => import("emoji-picker-react"));
import { $theme } from "../../store/theme";
import type { Collection } from "../../types";
import { formatDistanceToNow } from "date-fns";
import { clsx } from "clsx";
import { Button, Input, EmptyState, Skeleton } from "../../components/ui";
import { analytics } from "../../lib/analytics";

const collectionsCache = {
  data: null as Collection[] | null,
  timestamp: 0,
};

interface CollectionsProps {
  initialCollections?: Collection[];
}

export default function Collections({ initialCollections }: CollectionsProps) {
  const { t } = useTranslation();
  const user = useStore($user);
  const theme = useStore($theme);
  const [cacheState] = useState(() => {
    const c = collectionsCache.data;
    return {
      data: c,
      hasCache: !!c && Date.now() - collectionsCache.timestamp < 5 * 60 * 1000,
    };
  });
  const { data: cachedData, hasCache } = cacheState;

  const [collections, setCollections] = useState<Collection[]>(
    Array.isArray(initialCollections)
      ? initialCollections
      : hasCache
        ? cachedData!
        : [],
  );
  const [loading, setLoading] = useState(
    !Array.isArray(initialCollections) && !hasCache,
  );
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newItemName, setNewItemName] = useState("");
  const [newItemDesc, setNewItemDesc] = useState("");
  const [newItemIcon, setNewItemIcon] = useState("folder");
  const [activeTab, setActiveTab] = useState<"icon" | "emoji">("icon");
  const [creating, setCreating] = useState(false);

  const fetchCollections = async () => {
    try {
      const data = await getCollections();
      setCollections(data);
      collectionsCache.data = data;
      collectionsCache.timestamp = Date.now();
    } catch (error) {
      console.error("Failed to load collections:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (initialCollections) return;
    if (hasCache) {
      getCollections()
        .then((data) => {
          setCollections(data);
          collectionsCache.data = data;
          collectionsCache.timestamp = Date.now();
        })
        .catch(console.error);
      return;
    }
    let cancelled = false;
    getCollections()
      .then((data) => {
        if (cancelled) return;
        setCollections(data);
        collectionsCache.data = data;
        collectionsCache.timestamp = Date.now();
      })
      .catch((err) => {
        if (!cancelled) console.error("Failed to load collections:", err);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [initialCollections, hasCache]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newItemName.trim()) return;

    setCreating(true);
    const finalIcon = ICON_MAP[newItemIcon]
      ? `icon:${newItemIcon}`
      : newItemIcon;

    const res = await createCollection(newItemName, newItemDesc, finalIcon);
    if (res) {
      const newCollections = [res, ...collections];
      setCollections(newCollections);
      collectionsCache.data = newCollections;
      collectionsCache.timestamp = Date.now();
      setShowCreateModal(false);
      setNewItemName("");
      setNewItemDesc("");
      setNewItemIcon("folder");
      setActiveTab("icon");
      fetchCollections();
      analytics.capture("collection_created", {
        has_description: !!newItemDesc.trim(),
      });
    }
    setCreating(false);
  };

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.preventDefault();
    if (window.confirm(t("collections.deleteConfirm"))) {
      const success = await deleteCollection(id);
      if (success) {
        setCollections((prev) => {
          const updated = prev.filter((c) => c.id !== id);
          collectionsCache.data = updated;
          collectionsCache.timestamp = Date.now();
          return updated;
        });
        analytics.capture("collection_deleted");
      }
    }
  };

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto animate-fade-in">
        <div className="flex items-center justify-between mb-6">
          <div>
            <Skeleton width="180px" className="h-8 mb-2" />
            <Skeleton width="240px" className="h-4" />
          </div>
          <Skeleton width="90px" className="h-10 rounded-lg" />
        </div>
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="card p-4 flex gap-3 items-center">
              <Skeleton className="w-10 h-10 rounded-lg" />
              <div className="flex-1 space-y-2">
                <Skeleton width="50%" />
                <Skeleton width="30%" className="h-3" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto animate-slide-up">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-900 dark:text-white">
            {t("collections.title")}
          </h1>
          <p className="text-surface-500 dark:text-surface-400 mt-1">
            {t("collections.subtitle")}
          </p>
        </div>
        <Button
          onClick={() => setShowCreateModal(true)}
          icon={<Plus size={16} />}
        >
          {t("common.new")}
        </Button>
      </div>

      {collections.length === 0 ? (
        <EmptyState
          icon={<Folder size={48} />}
          title={t("collections.none")}
          message={t("collections.noneMessage")}
          action={{
            label: t("collections.createButton"),
            onClick: () => setShowCreateModal(true),
          }}
        />
      ) : (
        <div className="space-y-2">
          {collections
            .filter((c) => c && c.id && c.name)
            .map((collection) => (
              <a
                key={collection.id}
                href={`/${collection.creator?.handle || user?.handle}/collection/${(collection.uri || "").split("/").pop()}`}
                className="group card p-4 hover:ring-primary-300 dark:hover:ring-primary-600 transition-all flex items-center gap-4"
              >
                <div className="w-10 h-10 flex items-center justify-center shrink-0 bg-primary-50 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 rounded-xl">
                  <CollectionIcon icon={collection.icon} size={20} />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="font-semibold text-surface-900 dark:text-white truncate group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
                    {collection.name}
                  </h3>
                  <p className="text-sm text-surface-500 dark:text-surface-400">
                    {t("collections.itemCount", {
                      count: collection.itemCount,
                    })}
                    {collection.createdAt &&
                      ` · ${formatDistanceToNow(new Date(collection.createdAt), { addSuffix: true })}`}
                  </p>
                </div>
                {!collection.uri.includes("network.cosmik") && (
                  <button
                    onClick={(e) => handleDelete(collection.id, e)}
                    className="p-2 text-surface-400 dark:text-surface-500 hover:text-red-500 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all opacity-0 group-hover:opacity-100"
                  >
                    <Trash2 size={18} />
                  </button>
                )}
              </a>
            ))}
        </div>
      )}

      {showCreateModal && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in">
          <div className="bg-white dark:bg-surface-900 rounded-2xl shadow-2xl max-w-md w-full animate-scale-in ring-1 ring-black/5 dark:ring-white/10">
            <div className="flex items-center justify-between p-5 border-b border-surface-100 dark:border-surface-800">
              <h2 className="text-xl font-bold text-surface-900 dark:text-white">
                {t("collections.newTitle")}
              </h2>
              <button
                onClick={() => setShowCreateModal(false)}
                className="p-2 text-surface-400 dark:text-surface-500 hover:bg-surface-100 dark:hover:bg-surface-800 rounded-lg transition-colors"
              >
                <X size={18} />
              </button>
            </div>
            <form onSubmit={handleCreate} className="p-5">
              <div className="mb-4">
                <Input
                  label={t("collections.nameLabel")}
                  value={newItemName}
                  onChange={(e) => setNewItemName(e.target.value)}
                  placeholder="e.g. Design Inspiration"
                  autoFocus
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">
                  {t("collections.iconLabel")}
                </label>

                <div className="flex gap-2 mb-3 bg-surface-100 dark:bg-surface-800 p-1 rounded-xl">
                  <button
                    type="button"
                    onClick={() => setActiveTab("icon")}
                    className={`flex-1 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      activeTab === "icon"
                        ? "bg-white dark:bg-surface-700 text-surface-900 dark:text-white shadow-sm"
                        : "text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-surface-200"
                    }`}
                  >
                    {t("collections.iconsTab")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setActiveTab("emoji")}
                    className={`flex-1 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      activeTab === "emoji"
                        ? "bg-white dark:bg-surface-700 text-surface-900 dark:text-white shadow-sm"
                        : "text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-surface-200"
                    }`}
                  >
                    {t("collections.emojisTab")}
                  </button>
                </div>

                {activeTab === "icon" ? (
                  <div className="grid grid-cols-7 gap-1.5 p-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl max-h-48 overflow-y-auto custom-scrollbar">
                    {Object.keys(ICON_MAP).map((key) => {
                      const Icon = ICON_MAP[key];
                      return (
                        <button
                          key={key}
                          type="button"
                          onClick={() => setNewItemIcon(key)}
                          className={clsx(
                            "p-2 rounded-lg flex items-center justify-center transition-all",
                            newItemIcon === key
                              ? "bg-primary-100 dark:bg-primary-900/50 text-primary-600 dark:text-primary-400 ring-2 ring-primary-500"
                              : "hover:bg-surface-100 dark:hover:bg-surface-700 text-surface-500 dark:text-surface-400",
                          )}
                        >
                          <Icon size={18} />
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <div className="w-full bg-surface-50 dark:bg-surface-800 rounded-xl border border-surface-200 dark:border-surface-700 overflow-hidden">
                    <React.Suspense
                      fallback={
                        <div className="flex items-center justify-center h-[300px]">
                          <Loader2
                            className="animate-spin text-surface-400"
                            size={24}
                          />
                        </div>
                      }
                    >
                      <EmojiPicker
                        className="custom-emoji-picker"
                        onEmojiClick={(emojiData) =>
                          setNewItemIcon(emojiData.emoji)
                        }
                        autoFocusSearch={false}
                        width="100%"
                        height={300}
                        previewConfig={{ showPreview: false }}
                        skinTonesDisabled
                        lazyLoadEmojis
                        theme={
                          theme === "dark" ||
                          (theme === "system" &&
                            window.matchMedia("(prefers-color-scheme: dark)")
                              .matches)
                            ? (Theme.DARK as Theme)
                            : (Theme.LIGHT as Theme)
                        }
                      />
                    </React.Suspense>
                  </div>
                )}
              </div>
              <div className="mb-6">
                <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">
                  {t("collections.descriptionLabel")}
                </label>
                <textarea
                  value={newItemDesc}
                  onChange={(e) => setNewItemDesc(e.target.value)}
                  className="w-full px-3 py-2.5 bg-white dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl text-surface-900 dark:text-white placeholder:text-surface-400 dark:placeholder:text-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 dark:focus:border-primary-400 min-h-[80px] resize-none"
                  placeholder={t("collections.descriptionPlaceholder")}
                />
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setShowCreateModal(false)}
                >
                  {t("collections.cancel")}
                </Button>
                <Button type="submit" loading={creating}>
                  {t("collections.create")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
