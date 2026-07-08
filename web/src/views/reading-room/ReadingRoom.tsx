import { useEffect, useMemo, useState, useCallback } from "react";
import { useSearchParams, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Rss, X, Loader2 } from "lucide-react";
import {
  getPublicReadingRoom,
  getReadingRoomNotes,
  type ReadingRoomPublic,
} from "../../api/client";
import { displayHandle } from "../../lib/handle";
import { ensureFontLoaded, fontStack } from "../../lib/fonts";
import { buildPalette, noteKind, type NoteKind, type RRNote } from "./theme";
import ReadingRoomCard from "./ReadingRoomCard";

const ROOT_DOMAIN = "margin.at";
const onCustomDomain =
  typeof window !== "undefined" &&
  !!(window as unknown as { __READING_ROOM_HANDLE__?: string })
    .__READING_ROOM_HANDLE__;
const externalHref = (path: string) =>
  onCustomDomain ? `https://${ROOT_DOMAIN}${path}` : path;

const TYPE_ORDER: (NoteKind | "all")[] = [
  "all",
  "highlight",
  "note",
  "bookmark",
];
const FILTER_KEY: Record<string, string> = {
  all: "all",
  highlight: "highlights",
  note: "notes",
  bookmark: "bookmarks",
};

export default function ReadingRoom({
  handle: propHandle,
}: { handle?: string } = {}) {
  const { handle: paramHandle } = useParams<{ handle: string }>();
  const handle = propHandle || paramHandle;
  return <ReadingRoomView key={handle} handle={handle} />;
}

function ReadingRoomView({ handle }: { handle?: string }) {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const [data, setData] = useState<ReadingRoomPublic | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [typeFilter, setTypeFilter] = useState<NoteKind | "all">("all");
  const [activeTag, setActiveTag] = useState<string | null>(
    () => searchParams.get("tag") || null,
  );
  const [extraNotes, setExtraNotes] = useState<RRNote[]>([]);
  const [loadingMore, setLoadingMore] = useState(false);
  const [dbOffset, setDbOffset] = useState(20);

  useEffect(() => {
    if (!handle) return;
    let cancelled = false;
    getPublicReadingRoom(handle).then((result) => {
      if (cancelled) return;
      setData(result);
      setError(!result);
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [handle]);

  const avatar =
    data?.avatar ||
    (data?.did ? `https://margin.at/api/avatar/${data.did}` : "");

  useEffect(() => {
    if (!data?.did) return;
    const avatarUrl = avatar;
    let link: HTMLLinkElement | null = null;
    const oldLink = document.querySelector("link[rel='icon']");

    fetch(avatarUrl)
      .then((res) => res.blob())
      .then((blob) => {
        const reader = new FileReader();
        reader.onload = () => {
          const dataUri = reader.result as string;
          const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><defs><clipPath id="c"><circle cx="32" cy="32" r="30"/></clipPath></defs><image href="${dataUri}" width="64" height="64" clip-path="url(#c)"/></svg>`;
          link = document.createElement("link");
          link.rel = "icon";
          link.type = "image/svg+xml";
          link.href = `data:image/svg+xml,${encodeURIComponent(svg)}`;
          if (oldLink) oldLink.remove();
          document.head.appendChild(link);
        };
        reader.readAsDataURL(blob);
      })
      .catch(() => {
        link = document.createElement("link");
        link.rel = "icon";
        link.href = avatarUrl;
        if (oldLink) oldLink.remove();
        document.head.appendChild(link);
      });

    return () => {
      if (link) link.remove();
      if (oldLink) document.head.appendChild(oldLink);
    };
  }, [data?.did, avatar]);

  const fontFam = data?.theme?.fontFamily || "sans-serif";
  useEffect(() => {
    ensureFontLoaded(fontFam);
  }, [fontFam]);

  const pal = useMemo(
    () =>
      buildPalette(
        data?.theme?.backgroundColor || "#fcfcfc",
        data?.theme?.accentColor || "#3b82f6",
      ),
    [data],
  );

  const loadMore = useCallback(() => {
    if (!handle || loadingMore) return;
    setLoadingMore(true);
    getReadingRoomNotes(handle, dbOffset).then((result) => {
      if (result) {
        setExtraNotes((prev) => [
          ...prev,
          ...((result.notes as unknown as RRNote[]) || []),
        ]);
        setDbOffset((prev) => prev + 20);
      }
      setLoadingMore(false);
    });
  }, [handle, dbOffset, loadingMore]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-surface-100 dark:bg-surface-900 text-surface-400">
        <Loader2 size={24} className="animate-spin" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen bg-surface-100 dark:bg-surface-900 gap-3 px-6 text-center">
        <div className="text-2xl font-bold text-surface-900 dark:text-white">
          {t("readingRoom.notFound")}
        </div>
        <p className="text-surface-500 dark:text-surface-400">
          {t("readingRoom.notFoundDesc")}
        </p>
        <a
          href={externalHref("/home")}
          className="text-primary-500 hover:underline text-sm mt-2"
        >
          ← Back to Margin
        </a>
      </div>
    );
  }

  const layout = data.theme?.layout || "masonry";
  const title = data.title || `${data.displayName || handle}'s Reading Room`;

  const featured = (data.featured as unknown as RRNote[]) || [];
  const recentRaw = (data.recent as unknown as RRNote[]) || [];
  const featuredIds = new Set(featured.map((n) => n.id));
  const existingIds = new Set([...featuredIds, ...recentRaw.map((n) => n.id)]);
  const notes = [
    ...featured,
    ...recentRaw.filter((n) => !featuredIds.has(n.id)),
    ...extraNotes.filter((n) => !existingIds.has(n.id)),
  ];

  const hasMore = notes.length < data.totalCount;

  const typeCounts: Record<string, number> = {
    all: notes.length,
    highlight: 0,
    note: 0,
    bookmark: 0,
  };
  notes.forEach((n) => {
    typeCounts[noteKind(n)]++;
  });
  const typeOptions = TYPE_ORDER.filter(
    (k) => k === "all" || typeCounts[k] > 0,
  );
  const distinctKinds = typeOptions.length - 1;

  const filtered = notes.filter((n) => {
    if (typeFilter !== "all" && noteKind(n) !== typeFilter) return false;
    if (activeTag && !(n.tags || []).includes(activeTag)) return false;
    return true;
  });

  const hasFilters = distinctKinds >= 2 || activeTag !== null;

  const toggleTag = (tag: string) =>
    setActiveTag((cur) => (cur === tag ? null : tag));

  const containerWidth =
    layout === "list"
      ? "max-w-2xl"
      : layout === "grid"
        ? "max-w-5xl"
        : "max-w-6xl";

  return (
    <div
      className="min-h-screen flex flex-col animate-fade-in"
      style={{
        backgroundColor: pal.bg,
        color: pal.ink,
        fontFamily: fontStack(fontFam),
      }}
    >
      <header style={{ borderBottom: `1px solid ${pal.border}` }}>
        <div className="max-w-2xl mx-auto px-6 pt-16 pb-12 text-center">
          {avatar && (
            <a
              href={externalHref(`/profile/${data.did}`)}
              className="inline-block mb-5"
            >
              <img
                src={avatar}
                alt={data.displayName || data.handle}
                className="w-16 h-16 rounded-full object-cover"
                style={{ border: `2px solid ${pal.border}` }}
              />
            </a>
          )}
          <h1
            className="text-3xl md:text-4xl font-semibold leading-tight [text-wrap:balance]"
            style={{ fontFamily: "inherit", color: pal.ink }}
          >
            {title}
          </h1>
          {data.subtitle && (
            <p className="text-lg mt-3" style={{ color: pal.muted }}>
              {data.subtitle}
            </p>
          )}
          <div className="flex items-center justify-center gap-4 mt-5 text-sm">
            <a
              href={externalHref(`/profile/${data.did}`)}
              className="hover:underline underline-offset-4 font-medium"
              style={{ color: pal.accentText }}
            >
              @{displayHandle(data.handle)}
            </a>
            <a
              href={`/api/reading-room/rss/${encodeURIComponent(data.handle)}`}
              className="flex items-center gap-1.5 hover:underline underline-offset-4"
              style={{ color: pal.muted }}
            >
              <Rss size={13} />
              RSS
            </a>
          </div>
          {data.description && (
            <p
              className="mt-6 leading-relaxed max-w-prose mx-auto [text-wrap:pretty]"
              style={{ color: pal.muted }}
            >
              {data.description}
            </p>
          )}
        </div>
      </header>

      <main className={`w-full ${containerWidth} mx-auto px-6 py-12 flex-1`}>
        {hasFilters && (
          <div className="flex flex-wrap items-center gap-2 mb-8">
            {typeOptions.map((k) => {
              const active = typeFilter === k;
              return (
                <button
                  key={k}
                  onClick={() => setTypeFilter(k)}
                  className="text-sm font-medium px-3.5 py-1.5 rounded-full transition-colors"
                  style={
                    active
                      ? { backgroundColor: pal.accent, color: pal.accentOn }
                      : { color: pal.muted, border: `1px solid ${pal.border}` }
                  }
                >
                  {t(`readingRoom.filter.${FILTER_KEY[k]}`)}
                  <span className="ml-1.5 opacity-70">{typeCounts[k]}</span>
                </button>
              );
            })}
            {activeTag && (
              <button
                onClick={() => setActiveTag(null)}
                className="text-sm font-medium px-3.5 py-1.5 rounded-full inline-flex items-center gap-1.5 transition-colors"
                style={{
                  backgroundColor: pal.accentTint,
                  color: pal.accentText,
                }}
              >
                #{activeTag}
                <X size={13} />
              </button>
            )}
          </div>
        )}

        {notes.length === 0 ? (
          <p className="text-center py-16" style={{ color: pal.muted }}>
            {t("readingRoom.empty")}
          </p>
        ) : filtered.length === 0 ? (
          <div className="text-center py-16">
            <p style={{ color: pal.muted }}>{t("readingRoom.noMatches")}</p>
            <button
              onClick={() => {
                setTypeFilter("all");
                setActiveTag(null);
              }}
              className="mt-3 text-sm font-medium hover:underline underline-offset-4"
              style={{ color: pal.accentText }}
            >
              {t("readingRoom.clearFilters")}
            </button>
          </div>
        ) : layout === "grid" ? (
          <div className="grid gap-5 sm:grid-cols-2">
            {filtered.map((note, i) => (
              <ReadingRoomCard
                key={note.id || i}
                note={note}
                pal={pal}
                featured={featuredIds.has(note.id)}
                activeTag={activeTag}
                onTagClick={toggleTag}
                roomHandle={data.handle}
                variant="card"
              />
            ))}
          </div>
        ) : layout === "list" ? (
          <div>
            {filtered.map((note, i) => (
              <div
                key={note.id || i}
                style={
                  i > 0 ? { borderTop: `1px solid ${pal.border}` } : undefined
                }
              >
                <ReadingRoomCard
                  note={note}
                  pal={pal}
                  featured={featuredIds.has(note.id)}
                  activeTag={activeTag}
                  onTagClick={toggleTag}
                  roomHandle={data.handle}
                  variant="row"
                />
              </div>
            ))}
          </div>
        ) : (
          <div className="columns-1 sm:columns-2 lg:columns-3 gap-5">
            {filtered.map((note, i) => (
              <div key={note.id || i} className="mb-5 break-inside-avoid">
                <ReadingRoomCard
                  note={note}
                  pal={pal}
                  featured={featuredIds.has(note.id)}
                  activeTag={activeTag}
                  onTagClick={toggleTag}
                  roomHandle={data.handle}
                  variant="card"
                />
              </div>
            ))}
          </div>
        )}

        {hasMore && !activeTag && typeFilter === "all" && (
          <div className="flex justify-center mt-10">
            <button
              onClick={loadMore}
              disabled={loadingMore}
              className="inline-flex items-center gap-2 px-6 py-2.5 rounded-full text-sm font-medium transition-colors disabled:opacity-60"
              style={{
                border: `1px solid ${pal.border}`,
                color: pal.ink,
              }}
            >
              {loadingMore ? (
                <>
                  <Loader2 size={15} className="animate-spin" />
                  {t("readingRoom.loading")}
                </>
              ) : (
                t("readingRoom.loadMore")
              )}
            </button>
          </div>
        )}
      </main>

      <footer style={{ borderTop: `1px solid ${pal.border}` }}>
        <div
          className={`w-full ${containerWidth} mx-auto px-6 py-6 flex items-center justify-between text-sm`}
          style={{ color: pal.muted }}
        >
          <a
            href={externalHref("/home")}
            className="hover:underline underline-offset-4"
            style={{ color: pal.muted }}
          >
            Powered by Margin
          </a>
          <a
            href={externalHref(`/profile/${data.did}`)}
            className="hover:underline underline-offset-4"
            style={{ color: pal.muted }}
          >
            @{displayHandle(data.handle)}
          </a>
        </div>
      </footer>
    </div>
  );
}
