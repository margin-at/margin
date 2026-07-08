import React, { useState, useEffect } from "react";
import { X, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import CollectionIcon from "../common/CollectionIcon";
import { ICON_MAP } from "../common/iconMap";
import { Theme } from "emoji-picker-react";
const EmojiPicker = React.lazy(() => import("emoji-picker-react"));
import { updateCollection, type Collection } from "../../api/client";
import { useStore } from "@nanostores/react";
import { $theme } from "../../store/theme";

interface EditCollectionModalProps {
  isOpen: boolean;
  onClose: () => void;
  collection: Collection;
  onUpdate: (updatedCollection: Collection) => void;
}

export default function EditCollectionModal({
  isOpen,
  onClose,
  collection,
  onUpdate,
}: EditCollectionModalProps) {
  const [name, setName] = useState(collection.name);
  const [description, setDescription] = useState(collection.description || "");
  const initialIsIcon = collection.icon?.startsWith("icon:") ?? false;
  const initialIconValue = collection.icon?.replace("icon:", "") || "";

  const [activeTab, setActiveTab] = useState<"icon" | "emoji">(
    initialIsIcon || !collection.icon ? "icon" : "emoji",
  );
  const [icon, setIcon] = useState(initialIconValue);
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const theme = useStore($theme);

  const [prevIsOpen, setPrevIsOpen] = useState(isOpen);
  const [prevCollection, setPrevCollection] = useState(collection);
  if (isOpen && (!prevIsOpen || prevCollection !== collection)) {
    setPrevIsOpen(isOpen);
    setPrevCollection(collection);
    setName(collection.name);
    setDescription(collection.description || "");
    const isIcon = collection.icon?.startsWith("icon:") ?? false;
    setActiveTab(isIcon || !collection.icon ? "icon" : "emoji");
    setIcon(collection.icon?.replace("icon:", "") || "");
    setError(null);
  }

  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    }
    return () => {
      document.body.style.overflow = "unset";
    };
  }, [isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    try {
      setLoading(true);
      setError(null);
      const iconValue = icon
        ? ICON_MAP[icon]
          ? `icon:${icon}`
          : icon
        : undefined;
      const updated = await updateCollection(
        collection.uri,
        name.trim(),
        description.trim() || undefined,
        iconValue,
      );

      if (updated) {
        onUpdate(updated);
        onClose();
      } else {
        setError(t("editCollection.failedUpdate"));
      }
    } catch (err) {
      console.error(err);
      setError(t("editCollection.errorUpdating"));
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md bg-white dark:bg-surface-900 rounded-3xl shadow-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 flex justify-between items-center border-b border-surface-100 dark:border-surface-800">
          <h2 className="text-xl font-display font-bold text-surface-900 dark:text-white">
            {t("editCollection.title")}
          </h2>
          <button
            onClick={onClose}
            className="p-2 text-surface-400 hover:text-surface-900 dark:hover:text-white hover:bg-surface-50 dark:hover:bg-surface-800 rounded-full transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        <div className="p-6">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1">
                {t("editCollection.nameLabel")}
              </label>
              <input
                type="text"
                className="w-full px-4 py-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl focus:border-primary-500 dark:focus:border-primary-400 focus:ring-4 focus:ring-primary-500/10 outline-none transition-all text-surface-900 dark:text-white placeholder-surface-400 dark:placeholder-surface-500"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("editCollection.namePlaceholder")}
                autoFocus
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-1">
                {t("editCollection.descriptionLabel")}
              </label>
              <textarea
                className="w-full px-4 py-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl focus:border-primary-500 dark:focus:border-primary-400 focus:ring-4 focus:ring-primary-500/10 outline-none transition-all text-surface-900 dark:text-white placeholder-surface-400 dark:placeholder-surface-500 resize-none"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t("editCollection.descriptionPlaceholder")}
                rows={3}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-700 dark:text-surface-300 mb-2">
                {t("editCollection.iconLabel")}
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
                  {t("editCollection.iconsTab")}
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
                  {t("editCollection.emojisTab")}
                </button>
              </div>

              {activeTab === "icon" ? (
                <div className="grid grid-cols-8 gap-1.5 max-h-60 overflow-y-auto p-2 bg-surface-50 dark:bg-surface-800 rounded-xl border border-surface-200 dark:border-surface-700 custom-scrollbar">
                  {Object.keys(ICON_MAP).map((iconName) => {
                    const isSelected = icon === iconName;
                    return (
                      <button
                        key={iconName}
                        type="button"
                        onClick={() => setIcon(isSelected ? "" : iconName)}
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
                      onEmojiClick={(emojiData) => setIcon(emojiData.emoji)}
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

              {icon && (
                <p className="mt-2 text-sm text-surface-600 dark:text-surface-300 flex items-center gap-2">
                  {t("editCollection.selected")}
                  <span className="inline-flex items-center justify-center w-8 h-8 bg-surface-100 dark:bg-surface-800 rounded-lg border border-surface-200 dark:border-surface-700">
                    <CollectionIcon
                      icon={ICON_MAP[icon] ? `icon:${icon}` : icon}
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

            <div className="flex gap-3 pt-2">
              <button
                type="button"
                className="flex-1 py-3 bg-white dark:bg-surface-800 border border-surface-200 dark:border-surface-700 text-surface-700 dark:text-surface-200 font-semibold rounded-xl hover:bg-surface-50 dark:hover:bg-surface-700 transition-colors"
                onClick={onClose}
              >
                {t("editCollection.cancel")}
              </button>
              <button
                type="submit"
                className="flex-1 py-3 bg-primary-600 text-white font-semibold rounded-xl hover:bg-primary-700 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                disabled={!name.trim() || loading}
              >
                {loading && <Loader2 size={16} className="animate-spin" />}
                {loading
                  ? t("editCollection.saving")
                  : t("editCollection.save")}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
