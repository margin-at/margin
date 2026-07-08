import React, { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import {
  X,
  Plus,
  Check,
  Loader2,
  ChevronRight,
  FolderPlus,
} from "lucide-react";
import CollectionIcon from "../common/CollectionIcon";
import { ICON_MAP } from "../common/iconMap";
import { Theme } from "emoji-picker-react";
const EmojiPicker = React.lazy(() => import("emoji-picker-react"));
import { useStore } from "@nanostores/react";
import { $user } from "../../store/auth";
import { $theme } from "../../store/theme";
import { analytics } from "../../lib/analytics";
import {
  getCollections,
  addCollectionItem,
  createCollection,
  getCollectionsContaining,
  type Collection,
} from "../../api/client";

interface AddToCollectionModalProps {
  isOpen: boolean;
  onClose: () => void;
  annotationUri: string;
}

export default function AddToCollectionModal({
  isOpen,
  onClose,
  annotationUri,
}: AddToCollectionModalProps) {
  const { t } = useTranslation();
  const user = useStore($user);
  const theme = useStore($theme);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);
  const [addingTo, setAddingTo] = useState<string | null>(null);
  const [addedTo, setAddedTo] = useState<Set<string>>(new Set());
  const [error, setError] = useState<string | null>(null);

  const sheetRef = useRef<HTMLDivElement>(null);
  const dragStartY = useRef(0);
  const dragCurrentY = useRef(0);

  const handleTouchStart = (e: React.TouchEvent) => {
    dragStartY.current = e.touches[0].clientY;
    if (sheetRef.current) sheetRef.current.style.transition = "none";
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    const delta = e.touches[0].clientY - dragStartY.current;
    dragCurrentY.current = delta;
    if (delta > 0 && sheetRef.current) {
      sheetRef.current.style.transform = `translateY(${delta}px)`;
    }
  };

  const handleTouchEnd = () => {
    if (sheetRef.current) {
      sheetRef.current.style.transition = "transform 0.3s ease";
      if (dragCurrentY.current > 100) {
        sheetRef.current.style.transform = "translateY(100%)";
        setTimeout(onClose, 300);
      } else {
        sheetRef.current.style.transform = "translateY(0)";
      }
    }
    dragCurrentY.current = 0;
  };

  const [showNewForm, setShowNewForm] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDescription, setNewDescription] = useState("");
  const [newIcon, setNewIcon] = useState("");
  const [activeTab, setActiveTab] = useState<"icon" | "emoji">("icon");
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    }
    return () => {
      document.body.style.overflow = "unset";
    };
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen || !user) return;
    let cancelled = false;

    getCollections(user.did)
      .then((data) => {
        if (!cancelled) setCollections(data);
      })
      .catch((err) => {
        console.error(err);
        if (!cancelled) setError(t("addToCollection.failedLoad"));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    getCollectionsContaining(annotationUri).then((uris) => {
      if (!cancelled) setAddedTo(new Set(uris));
    });

    return () => {
      cancelled = true;
    };
  }, [isOpen, user, annotationUri, t]);

  const handleAdd = async (collectionUri: string) => {
    if (addedTo.has(collectionUri)) return;

    try {
      setAddingTo(collectionUri);
      await addCollectionItem(collectionUri, annotationUri);
      setAddedTo((prev) => new Set([...prev, collectionUri]));
      analytics.capture("item_added_to_collection");
    } catch (err) {
      console.error(err);
      setError(t("addToCollection.failedAdd"));
    } finally {
      setAddingTo(null);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim()) return;
    try {
      setCreating(true);
      const iconValue = newIcon
        ? ICON_MAP[newIcon]
          ? `icon:${newIcon}`
          : newIcon
        : undefined;
      const newCollection = await createCollection(
        newName.trim(),
        newDescription.trim() || undefined,
        iconValue,
      );
      if (newCollection) {
        setCollections((prev) => [newCollection, ...prev]);
        setNewName("");
        setNewDescription("");
        setNewIcon("");
        setShowNewForm(false);
      }
    } catch (err) {
      console.error(err);
      setError(t("addToCollection.failedCreate"));
    } finally {
      setCreating(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-end sm:items-center justify-center sm:p-4 bg-black/40 backdrop-blur-sm animate-fade-in"
      onClick={onClose}
    >
      <div
        ref={sheetRef}
        className="w-full sm:max-w-md bg-white dark:bg-surface-900 rounded-t-3xl sm:rounded-3xl shadow-2xl flex flex-col animate-slide-up border border-surface-200 dark:border-surface-700 border-b-0 sm:border-b"
        style={{
          paddingBottom: "env(safe-area-inset-bottom)",
          maxHeight: "90dvh",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div
          className="flex justify-center pt-3 pb-1 sm:hidden shrink-0 cursor-grab active:cursor-grabbing touch-none"
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
        >
          <div className="w-8 h-1 bg-surface-200 dark:bg-surface-700 rounded-full" />
        </div>

        <div className="px-4 sm:px-4 py-3 flex justify-between items-center border-b border-surface-100 dark:border-surface-800 shrink-0">
          <h2 className="text-lg font-display font-bold text-surface-900 dark:text-white">
            {t("addToCollection.title")}
          </h2>
          <button
            onClick={onClose}
            className="p-2 text-surface-400 hover:text-surface-900 dark:hover:text-white hover:bg-surface-50 dark:hover:bg-surface-800 rounded-full transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        <div className="px-4 sm:px-6 pb-4 pt-4 overflow-y-auto flex-1">
          {loading ? (
            <div className="text-center py-10">
              <Loader2
                size={32}
                className="animate-spin text-primary-600 dark:text-primary-400 mx-auto mb-3"
              />
              <p className="text-surface-500 dark:text-surface-400 font-medium">
                {t("addToCollection.loading")}
              </p>
            </div>
          ) : showNewForm ? (
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1">
                  {t("addToCollection.collectionNameLabel")}
                </label>
                <input
                  type="text"
                  className="w-full px-4 py-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl focus:border-primary-500 dark:focus:border-primary-400 focus:ring-4 focus:ring-primary-500/10 outline-none transition-all text-surface-900 dark:text-white placeholder-surface-400 dark:placeholder-surface-500"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder={t("addToCollection.namePlaceholder")}
                  autoFocus
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1">
                  {t("addToCollection.descriptionLabel")}
                </label>
                <textarea
                  className="w-full px-4 py-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl focus:border-primary-500 dark:focus:border-primary-400 focus:ring-4 focus:ring-primary-500/10 outline-none transition-all text-surface-900 dark:text-white placeholder-surface-400 dark:placeholder-surface-500 resize-none"
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder={t("addToCollection.descriptionPlaceholder")}
                  rows={2}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">
                  {t("addToCollection.iconLabel")}
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
                    {t("addToCollection.iconsTab")}
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
                    {t("addToCollection.emojisTab")}
                  </button>
                </div>

                {activeTab === "icon" ? (
                  <div className="grid grid-cols-8 gap-1.5 max-h-60 overflow-y-auto p-2 bg-surface-50 dark:bg-surface-800 rounded-xl border border-surface-200 dark:border-surface-700 custom-scrollbar">
                    {Object.keys(ICON_MAP).map((iconName) => {
                      const isSelected = newIcon === iconName;
                      return (
                        <button
                          key={iconName}
                          type="button"
                          onClick={() => setNewIcon(isSelected ? "" : iconName)}
                          className={`w-8 h-8 flex items-center justify-center rounded-lg transition-all ${
                            isSelected
                              ? "bg-primary-600 text-white"
                              : "hover:bg-surface-200 dark:hover:bg-surface-700 text-surface-600 dark:text-surface-400"
                          }`}
                          title={iconName}
                        >
                          <CollectionIcon icon={`icon:${iconName}`} size={16} />
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
                          setNewIcon(emojiData.emoji)
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

                {newIcon && (
                  <p className="mt-2 text-sm text-surface-600 dark:text-surface-300 flex items-center gap-2">
                    {t("addToCollection.selected")}
                    <span className="inline-flex items-center justify-center w-8 h-8 bg-surface-100 dark:bg-surface-800 rounded-lg border border-surface-200 dark:border-surface-700">
                      <CollectionIcon
                        icon={ICON_MAP[newIcon] ? `icon:${newIcon}` : newIcon}
                        size={18}
                      />
                    </span>
                  </p>
                )}
              </div>

              {error && (
                <div className="p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded-lg">
                  {error}
                </div>
              )}

              <div className="flex gap-2 pt-2">
                <button
                  type="button"
                  className="flex-1 py-2.5 text-sm font-medium text-surface-600 dark:text-surface-300 hover:text-surface-900 dark:hover:text-white hover:bg-surface-50 dark:hover:bg-surface-800 rounded-xl transition-colors"
                  onClick={() => {
                    setShowNewForm(false);
                    setNewDescription("");
                    setNewIcon("");
                    setError(null);
                  }}
                >
                  {t("addToCollection.back")}
                </button>
                <button
                  type="submit"
                  className="flex-1 py-2.5 text-sm bg-primary-600 text-white font-medium rounded-xl hover:bg-primary-700 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                  disabled={!newName.trim() || creating}
                >
                  {creating && <Loader2 size={14} className="animate-spin" />}
                  {creating
                    ? t("addToCollection.creating")
                    : t("addToCollection.create")}
                </button>
              </div>
            </form>
          ) : (
            <div className="-mx-4 sm:mx-0">
              {error && (
                <div className="mx-4 sm:mx-0 mb-2 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded-lg">
                  {error}
                </div>
              )}

              <button
                className="w-full flex items-center gap-3 px-4 sm:px-3 py-2.5 text-[14px] font-medium transition-colors rounded-lg text-primary-600 dark:text-primary-400 hover:bg-surface-50 dark:hover:bg-surface-800"
                onClick={() => setShowNewForm(true)}
              >
                <span className="flex items-center justify-center w-5 h-5 text-primary-500 dark:text-primary-400">
                  <FolderPlus size={16} />
                </span>
                <span className="flex-1 text-left">
                  {t("addToCollection.newCollectionButton")}
                </span>
                <ChevronRight
                  size={14}
                  className="text-surface-300 dark:text-surface-600"
                />
              </button>

              <div className="h-px bg-surface-100 dark:bg-surface-800 my-1 mx-4 sm:mx-2" />

              {collections.length === 0 ? (
                <div className="text-center py-6 px-4">
                  <p className="text-sm text-surface-400 dark:text-surface-500">
                    {t("addToCollection.none")}
                  </p>
                </div>
              ) : (
                <div className="overflow-y-auto max-h-[50vh] sm:max-h-[300px]">
                  {collections.map((col) => {
                    const isAdded = addedTo.has(col.uri);
                    const isAdding = addingTo === col.uri;

                    return (
                      <button
                        key={col.uri}
                        onClick={() => handleAdd(col.uri)}
                        disabled={isAdding || isAdded}
                        className="w-full flex items-center gap-3 px-4 sm:px-3 py-2.5 text-[14px] font-medium transition-colors rounded-lg text-surface-700 dark:text-surface-200 hover:bg-surface-50 dark:hover:bg-surface-800 disabled:opacity-60"
                      >
                        <span className="flex items-center justify-center w-5 h-5 text-surface-400 dark:text-surface-500">
                          <CollectionIcon icon={col.icon} size={16} />
                        </span>
                        <span className="flex-1 text-left truncate">
                          {col.name}
                        </span>
                        {isAdding ? (
                          <Loader2
                            size={15}
                            className="animate-spin text-surface-400 shrink-0"
                          />
                        ) : isAdded ? (
                          <Check
                            size={15}
                            className="text-green-500 shrink-0"
                          />
                        ) : (
                          <Plus
                            size={15}
                            className="text-surface-300 dark:text-surface-600 shrink-0"
                          />
                        )}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
