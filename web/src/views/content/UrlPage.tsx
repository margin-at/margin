import { useStore } from "@nanostores/react";
import { useTranslation } from "react-i18next";
import {
  AlertTriangle,
  Check,
  Copy,
  ExternalLink,
  Globe,
  Highlighter,
  Loader2,
  PenTool,
  Search,
  User,
  Users,
} from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { getByTarget } from "../../api/client";
import Card from "../../components/common/Card";
import { Button, EmptyState, Input, Tabs } from "../../components/ui";
import { $user } from "../../store/auth";
import type { AnnotationItem } from "../../types";

interface UrlPageProps {
  urlPath?: string;
}

export default function UrlPage({ urlPath }: UrlPageProps) {
  const targetUrl = urlPath ? decodeURIComponent(urlPath) : "";

  const [annotations, setAnnotations] = useState<AnnotationItem[]>([]);
  const [highlights, setHighlights] = useState<AnnotationItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreError, setLoadMoreError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [offset, setOffset] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<
    "all" | "annotations" | "highlights"
  >("all");
  const [copied, setCopied] = useState(false);
  const { t } = useTranslation();
  const user = useStore($user);

  const LIMIT = 50;
  const loadMoreTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (loadMoreTimerRef.current) clearTimeout(loadMoreTimerRef.current);
    };
  }, []);

  useEffect(() => {
    async function fetchData() {
      if (!targetUrl) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        setError(null);

        const data = await getByTarget(targetUrl, LIMIT, 0);
        const fetchedAnnotations = data.annotations || [];
        const fetchedHighlights = data.highlights || [];
        setAnnotations(fetchedAnnotations);
        setHighlights(fetchedHighlights);
        const totalFetched =
          fetchedAnnotations.length + fetchedHighlights.length;
        setHasMore(totalFetched >= LIMIT);
        setOffset(totalFetched);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load data");
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [targetUrl]);

  const loadMore = useCallback(async () => {
    setLoadingMore(true);
    setLoadMoreError(null);
    try {
      const data = await getByTarget(targetUrl, LIMIT, offset);
      const fetchedAnnotations = data.annotations || [];
      const fetchedHighlights = data.highlights || [];
      setAnnotations((prev) => [...prev, ...fetchedAnnotations]);
      setHighlights((prev) => [...prev, ...fetchedHighlights]);
      const totalFetched = fetchedAnnotations.length + fetchedHighlights.length;
      setHasMore(totalFetched >= LIMIT);
      setOffset((prev) => prev + totalFetched);
    } catch (err) {
      console.error("Failed to load more:", err);
      const msg = err instanceof Error ? err.message : "Something went wrong";
      setLoadMoreError(msg);
      if (loadMoreTimerRef.current) clearTimeout(loadMoreTimerRef.current);
      loadMoreTimerRef.current = setTimeout(() => setLoadMoreError(null), 5000);
    } finally {
      setLoadingMore(false);
    }
  }, [targetUrl, offset]);

  const handleCopyLink = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy link:", err);
    }
  }, []);

  const handleNavigateMyAnnotations = async () => {
    if (!user?.handle || !targetUrl) return;
    window.location.href = `/${user.handle}/url/${encodeURIComponent(targetUrl)}`;
  };

  const totalItems = annotations.length + highlights.length;

  const uniqueAuthors = new Map<
    string,
    { did: string; handle?: string; displayName?: string; avatar?: string }
  >();
  [...annotations, ...highlights].forEach((item) => {
    const author = item.author || item.creator;
    if (author?.did && !uniqueAuthors.has(author.did)) {
      uniqueAuthors.set(author.did, author);
    }
  });
  const authorCount = uniqueAuthors.size;

  const hostname = (() => {
    try {
      return new URL(targetUrl).hostname;
    } catch {
      return targetUrl;
    }
  })();

  const favicon = targetUrl
    ? `https://www.google.com/s2/favicons?domain=${hostname}&sz=32`
    : null;

  if (!targetUrl) {
    return (
      <div className="max-w-2xl mx-auto pb-20 animate-fade-in">
        <div className="text-center py-10">
          <div className="w-16 h-16 bg-primary-50 dark:bg-primary-900/20 rounded-2xl flex items-center justify-center mx-auto mb-6 rotate-3">
            <Globe
              size={32}
              className="text-primary-600 dark:text-primary-400"
            />
          </div>
          <h1 className="text-3xl font-display font-bold text-surface-900 dark:text-white mb-3">
            {t("urlPage.title")}
          </h1>
          <p className="text-surface-500 dark:text-surface-400 max-w-md mx-auto mb-8">
            {t("urlPage.description")}
          </p>

          <form
            onSubmit={(e) => {
              e.preventDefault();
              const formData = new FormData(e.currentTarget);
              const q = (formData.get("q") as string)?.trim();
              if (q) {
                const encoded = encodeURIComponent(q);
                window.location.href = `/url/${encoded}`;
              }
            }}
            className="max-w-md mx-auto flex gap-2"
          >
            <div className="flex-1">
              <Input
                name="q"
                placeholder={t("urlPage.urlPlaceholder")}
                className="w-full bg-surface-50 dark:bg-surface-800"
                autoFocus
              />
            </div>
            <Button type="submit">{t("urlPage.view")}</Button>
          </form>
        </div>
      </div>
    );
  }

  const items = [
    ...(activeTab === "all" || activeTab === "annotations" ? annotations : []),
    ...(activeTab === "all" || activeTab === "highlights" ? highlights : []),
  ];

  if (activeTab === "all") {
    items.sort((a, b) => {
      const dateA = new Date(a.createdAt).getTime();
      const dateB = new Date(b.createdAt).getTime();
      return dateB - dateA;
    });
  }

  return (
    <div className="max-w-3xl mx-auto pb-20 animate-fade-in">
      <header className="mb-8 p-6 bg-white dark:bg-surface-800 rounded-2xl border border-surface-200 dark:border-surface-700 shadow-sm">
        <div className="flex items-start gap-4">
          {favicon && (
            <img
              src={favicon}
              alt=""
              className="w-8 h-8 rounded-lg mt-1 shrink-0"
              onError={(e) => {
                (e.target as HTMLImageElement).style.display = "none";
              }}
            />
          )}
          <div className="flex-1 min-w-0">
            <h1 className="text-xl font-bold text-surface-900 dark:text-white mb-1 break-all">
              {hostname}
            </h1>
            <a
              href={targetUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm text-primary-600 dark:text-primary-400 hover:underline break-all flex items-center gap-1 leading-relaxed"
            >
              <span className="truncate">{targetUrl}</span>
              <ExternalLink size={12} className="shrink-0" />
            </a>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {user && (
              <button
                onClick={handleNavigateMyAnnotations}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-100 dark:bg-surface-700 hover:bg-surface-200 dark:hover:bg-surface-600 text-surface-700 dark:text-surface-200 text-sm font-medium rounded-lg transition-colors"
                title={t("urlPage.myAnnotations")}
              >
                <User size={14} /> {t("urlPage.myAnnotations")}
              </button>
            )}
            <button
              onClick={handleCopyLink}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-100 dark:bg-surface-700 hover:bg-surface-200 dark:hover:bg-surface-600 text-surface-700 dark:text-surface-200 text-sm font-medium rounded-lg transition-colors"
              title={t("urlPage.share")}
            >
              {copied ? <Check size={14} /> : <Copy size={14} />}
              {copied ? t("urlPage.copied") : t("urlPage.share")}
            </button>
          </div>
        </div>

        {!loading && totalItems > 0 && (
          <div className="mt-4 pt-4 border-t border-surface-100 dark:border-surface-700 flex items-center gap-4 text-sm text-surface-500 dark:text-surface-400">
            <span className="flex items-center gap-1.5">
              <Users size={14} />
              {t("urlPage.contributor", { count: authorCount })}
            </span>
          </div>
        )}
      </header>

      {loading && (
        <div className="flex flex-col items-center justify-center py-20">
          <Loader2
            className="animate-spin text-primary-600 dark:text-primary-400 mb-4"
            size={32}
          />
          <p className="text-surface-500 dark:text-surface-400">
            {t("urlPage.loadingAnnotations")}
          </p>
        </div>
      )}

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-4 rounded-xl flex items-start gap-3 border border-red-100 dark:border-red-900/30 mb-6">
          <AlertTriangle className="shrink-0 mt-0.5" size={18} />
          <p>{error}</p>
        </div>
      )}

      {!loading && !error && totalItems === 0 && (
        <EmptyState
          icon={<Search size={48} />}
          title={t("urlPage.blankCanvas")}
          message={t("urlPage.blankCanvasMessage")}
        />
      )}

      {!loading && !error && totalItems > 0 && (
        <div>
          <div className="mb-6">
            <Tabs
              tabs={[
                { id: "all", label: t("urlPage.tabs.all") },
                { id: "annotations", label: t("urlPage.tabs.annotations") },
                { id: "highlights", label: t("urlPage.tabs.highlights") },
              ]}
              activeTab={activeTab}
              onChange={(id: string) =>
                setActiveTab(id as "all" | "annotations" | "highlights")
              }
            />
          </div>

          <div className="space-y-4">
            {activeTab === "annotations" && annotations.length === 0 && (
              <EmptyState
                icon={<PenTool size={32} />}
                title={t("urlPage.noAnnotationsYet")}
                message={t("urlPage.noAnnotationsMessage")}
              />
            )}
            {activeTab === "highlights" && highlights.length === 0 && (
              <EmptyState
                icon={<Highlighter size={32} />}
                title={t("urlPage.noHighlightsYet")}
                message={t("urlPage.noHighlightsMessage")}
              />
            )}

            {items.map((item) => (
              <Card key={item.uri} item={item} />
            ))}
          </div>

          {hasMore && (
            <div className="flex flex-col items-center gap-2 py-6">
              {loadMoreError && (
                <p className="text-sm text-red-500 dark:text-red-400">
                  {t("urlPage.failedLoadMore", { message: loadMoreError })}
                </p>
              )}
              <button
                onClick={loadMore}
                disabled={loadingMore}
                className="inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium rounded-xl bg-surface-100 dark:bg-surface-800 text-surface-600 dark:text-surface-300 hover:bg-surface-200 dark:hover:bg-surface-700 transition-colors disabled:opacity-50"
              >
                {loadingMore ? (
                  <>
                    <Loader2 size={16} className="animate-spin" />
                    {t("urlPage.loading")}
                  </>
                ) : (
                  t("urlPage.loadMore")
                )}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
