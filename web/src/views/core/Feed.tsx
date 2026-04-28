import { useStore } from "@nanostores/react";
import { clsx } from "clsx";
import {
  Bookmark,
  Highlighter,
  MessageSquareText,
  User,
  Users,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import FeedItems from "../../components/feed/FeedItems";
import { Button, Tabs } from "../../components/ui";
import LayoutToggle from "../../components/ui/LayoutToggle";
import { $user } from "../../store/auth";
import { $feedLayout } from "../../store/feedLayout";
import type { AnnotationItem, UserProfile } from "../../types";

interface FeedProps {
  initialType?: string;
  initialTag?: string;
  initialUser?: UserProfile | null;
  motivation?: string;
  showTabs?: boolean;
  emptyMessage?: string;
  initialItems?: AnnotationItem[];
  initialHasMore?: boolean;
}

export default function Feed({
  initialType = "all",
  initialTag,
  initialUser,
  motivation,
  showTabs = true,
  emptyMessage,
  initialItems,
  initialHasMore,
}: FeedProps) {
  const { t } = useTranslation();
  const resolvedEmptyMessage = emptyMessage ?? t("feed.defaultEmptyMessage");
  const [tag, setTag] = useState<string | undefined>(
    initialTag ||
      (typeof window !== "undefined"
        ? new URLSearchParams(window.location.search).get("tag") || undefined
        : undefined),
  );
  const storeUser = useStore($user);
  const user = storeUser || initialUser || null;
  const layout = useStore($feedLayout);
  const [activeTab, setActiveTab] = useState(initialType);
  const [activeFilter, setActiveFilter] = useState<string | undefined>(
    motivation,
  );
  const [mineOnly, setMineOnly] = useState(false);

  const clearTag = () => {
    setTag(undefined);
    const url = new URL(window.location.href);
    url.searchParams.delete("tag");
    window.history.replaceState({}, "", url.toString());
  };

  const handleTabChange = (id: string) => {
    if (id === activeTab) return;
    setActiveTab(id);
    clearTag();
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const handleFilterChange = (id: string) => {
    const next = id === "all" ? undefined : id;
    if (next === activeFilter) return;
    setActiveFilter(next);
    clearTag();
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const tabs = [
    { id: "all", label: t("feed.tabs.recent") },
    { id: "popular", label: t("feed.tabs.popular") },
    { id: "shelved", label: t("feed.tabs.shelved") },
    { id: "margin", label: t("feed.tabs.margin") },
    { id: "semble", label: t("feed.tabs.semble") },
  ];

  const filters = [
    { id: "all", label: t("feed.filters.all"), icon: null },
    {
      id: "commenting",
      label: t("feed.filters.annotations"),
      icon: MessageSquareText,
    },
    {
      id: "highlighting",
      label: t("feed.filters.highlights"),
      icon: Highlighter,
    },
    { id: "bookmarking", label: t("feed.filters.bookmarks"), icon: Bookmark },
  ];

  return (
    <div className="mx-auto max-w-2xl xl:max-w-none">
      {!user && (
        <div className="relative text-center py-12 px-6 mb-4 animate-fade-in overflow-hidden">
          <div className="absolute inset-0 -z-10 flex items-center justify-center">
            <div className="h-48 w-48 rounded-full bg-primary-200/40 dark:bg-primary-900/20 blur-3xl" />
          </div>
          <h1 className="text-3xl sm:text-4xl font-display font-bold mb-3 tracking-tight text-surface-900 dark:text-white">
            {t("feed.welcome")}
          </h1>
          <p className="text-surface-500 dark:text-surface-400 mb-5 max-w-md mx-auto leading-relaxed">
            {t("feed.welcomeTagline")}
          </p>
          <div className="flex gap-3 justify-center">
            <Button onClick={() => (window.location.href = "/login")}>
              {t("feed.getStarted")}
            </Button>
            <Button
              variant="secondary"
              onClick={() => window.open("/about", "_blank")}
            >
              {t("feed.learnMore")}
            </Button>
          </div>
        </div>
      )}

      <div className="sticky top-0 z-10 bg-white/90 dark:bg-surface-800/90 backdrop-blur-md pb-3 mb-2 -mx-1 px-1 pt-2 space-y-2">
        {showTabs && !tag && (
          <Tabs tabs={tabs} activeTab={activeTab} onChange={handleTabChange} />
        )}
        {tag && (
          <div className="flex items-center justify-between mb-2">
            <h2 className="text-xl font-bold flex items-center gap-2">
              <span className="text-surface-500 font-normal">
                {t("feed.itemsWithTag")}
              </span>
              <span className="bg-primary-50 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 px-2 py-0.5 rounded-lg">
                #{tag}
              </span>
            </h2>
            <button
              onClick={clearTag}
              className="text-sm text-surface-500 hover:text-surface-900 dark:hover:text-white"
            >
              {t("feed.clearFilter")}
            </button>
          </div>
        )}
        {showTabs && (
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar flex-1 pb-0.5">
              {filters.map((f) => {
                const isActive =
                  f.id === "all" ? !activeFilter : activeFilter === f.id;
                return (
                  <button
                    key={f.id}
                    onClick={() => handleFilterChange(f.id)}
                    className={clsx(
                      "inline-flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-full border transition-all shrink-0",
                      isActive
                        ? "bg-primary-600 dark:bg-primary-500 text-white border-transparent shadow-sm"
                        : "bg-white dark:bg-surface-900 text-surface-500 dark:text-surface-400 border-surface-200 dark:border-surface-700 hover:border-primary-300 dark:hover:border-primary-700 hover:text-primary-600 dark:hover:text-primary-400",
                    )}
                  >
                    {f.icon && <f.icon size={12} />}
                    {f.label}
                  </button>
                );
              })}
            </div>
            <LayoutToggle className="hidden sm:inline-flex shrink-0" />
          </div>
        )}
        {!showTabs && user && (
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar flex-1 pb-0.5">
              {[
                { id: "everyone", label: t("feed.everyone"), icon: Users },
                { id: "mine", label: t("feed.mine"), icon: User },
              ].map((f) => {
                const isActive = f.id === "mine" ? mineOnly : !mineOnly;
                return (
                  <button
                    key={f.id}
                    onClick={() => setMineOnly(f.id === "mine")}
                    className={clsx(
                      "inline-flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-full border transition-all shrink-0",
                      isActive
                        ? "bg-primary-600 dark:bg-primary-500 text-white border-transparent shadow-sm"
                        : "bg-white dark:bg-surface-900 text-surface-500 dark:text-surface-400 border-surface-200 dark:border-surface-700 hover:border-primary-300 dark:hover:border-primary-700 hover:text-primary-600 dark:hover:text-primary-400",
                    )}
                  >
                    <f.icon size={12} />
                    {f.label}
                  </button>
                );
              })}
            </div>
            <LayoutToggle className="hidden sm:inline-flex shrink-0" />
          </div>
        )}
      </div>

      <FeedItems
        key={`${activeTab}-${activeFilter || "all"}-${tag || ""}-${mineOnly ? "mine" : "all"}`}
        type={activeTab}
        motivation={activeFilter}
        creator={mineOnly && user ? user.did : undefined}
        emptyMessage={resolvedEmptyMessage}
        layout={layout}
        tag={tag?.toLowerCase()}
        initialItems={
          activeTab === initialType && activeFilter === motivation && !mineOnly
            ? initialItems
            : undefined
        }
        initialHasMore={
          activeTab === initialType && activeFilter === motivation && !mineOnly
            ? initialHasMore
            : undefined
        }
      />
    </div>
  );
}
