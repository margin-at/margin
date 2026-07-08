import { useEffect, useMemo, useState } from "react";
import { useSearchParams, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Loader2 } from "lucide-react";
import { getReadingRoomNote } from "../../api/client";
import { displayHandle } from "../../lib/handle";
import { ensureFontLoaded, fontStack } from "../../lib/fonts";
import { buildPalette, type RRNote } from "./theme";
import ReadingRoomCard from "./ReadingRoomCard";

const ROOT_DOMAIN = "margin.at";
const onCustomDomain =
  typeof window !== "undefined" &&
  !!(window as unknown as { __READING_ROOM_HANDLE__?: string })
    .__READING_ROOM_HANDLE__;
const externalHref = (path: string) =>
  onCustomDomain ? `https://${ROOT_DOMAIN}${path}` : path;
const roomHref = (path: string) => (onCustomDomain ? path : path);

export default function ReadingRoomNote({
  handle: propHandle,
}: { handle?: string } = {}) {
  const { handle: paramHandle } = useParams<{ handle: string }>();
  const handle = propHandle || paramHandle;
  const [searchParams] = useSearchParams();
  const uri = searchParams.get("uri") || "";
  return (
    <ReadingRoomNoteView key={`${handle}:${uri}`} handle={handle} uri={uri} />
  );
}

function ReadingRoomNoteView({
  handle,
  uri,
}: {
  handle?: string;
  uri: string;
}) {
  const { t } = useTranslation();
  const missingParams = !handle || !uri;
  const [data, setData] = useState<Awaited<
    ReturnType<typeof getReadingRoomNote>
  > | null>(null);
  const [loading, setLoading] = useState(!missingParams);
  const [error, setError] = useState(missingParams);

  useEffect(() => {
    if (!handle || !uri) return;
    let cancelled = false;
    getReadingRoomNote(handle, uri).then((result) => {
      if (cancelled) return;
      setData(result);
      setError(!result);
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [handle, uri]);

  useEffect(() => {
    if (!data?.did || data.avatar) return;
    setData((prev) => prev ? { ...prev, avatar: `https://margin.at/api/avatar/${data.did}` } : prev);
  }, [data?.did, data?.avatar]);

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
          {t("readingRoom.noteNotFound")}
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

  const note = data.note as unknown as RRNote;
  const roomTitle =
    data.roomTitle || `${data.displayName || handle}'s Reading Room`;

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
        <div className="max-w-2xl mx-auto px-6 pt-10 pb-6">
          <a
            href={roomHref("/")}
            className="inline-flex items-center gap-2 text-sm hover:underline underline-offset-4"
            style={{ color: pal.muted }}
          >
            <ArrowLeft size={14} />
            {roomTitle}
          </a>
          <div className="flex items-center gap-3 mt-4">
            {data.avatar && (
              <a href={externalHref(`/profile/${data.did}`)}>
                <img
                  src={data.avatar}
                  alt={data.displayName || data.handle}
                  className="w-9 h-9 rounded-full object-cover"
                  style={{ border: `2px solid ${pal.border}` }}
                />
              </a>
            )}
            <div className="min-w-0">
              <p
                className="font-semibold text-sm truncate"
                style={{ color: pal.ink }}
              >
                {data.displayName || data.handle}
              </p>
              <a
                href={externalHref(`/profile/${data.did}`)}
                className="text-xs hover:underline underline-offset-4"
                style={{ color: pal.accentText }}
              >
                @{displayHandle(data.handle)}
              </a>
            </div>
          </div>
        </div>
      </header>

      <main className="w-full max-w-2xl mx-auto px-6 py-12 flex-1">
        <ReadingRoomCard
          note={note}
          pal={pal}
          featured={false}
          activeTag={null}
          onTagClick={(tag) => {
            window.location.href = `/reading-room/${encodeURIComponent(data.handle)}?tag=${encodeURIComponent(tag)}`;
          }}
          roomHandle={data.handle}
          linkable={false}
          variant="card"
        />
      </main>

      <footer style={{ borderTop: `1px solid ${pal.border}` }}>
        <div
          className="w-full max-w-2xl mx-auto px-6 py-6 flex items-center justify-between text-sm"
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
            href={roomHref("/")}
            className="hover:underline underline-offset-4"
            style={{ color: pal.muted }}
          >
            {t("readingRoom.viewFullRoom")}
          </a>
        </div>
      </footer>
    </div>
  );
}
