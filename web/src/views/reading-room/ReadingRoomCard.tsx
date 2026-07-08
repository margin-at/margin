import { useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  Quote,
  MessageSquare,
  Bookmark,
  ExternalLink,
  Link2,
  Check,
} from "lucide-react";
import {
  type RRNote,
  type RRPalette,
  textFragmentUrl,
  highlightBg,
  highlightInk,
  readingRoomNoteUrl,
} from "./theme";

function ExternalBadges({
  noteId,
  pal,
  t,
}: {
  noteId: string;
  pal: RRPalette;
  t: (key: string) => string;
}) {
  const isSemble =
    noteId.includes("network.cosmik") || noteId.includes("semble");
  const isLichen = noteId.includes("wiki.lichen.bookmark");
  const isExternalBookmark = noteId.includes(
    "community.lexicon.bookmarks.bookmark",
  );

  const tooltipBg = pal.muted;
  const tooltipStyle: React.CSSProperties = {
    position: "absolute",
    bottom: "100%",
    left: "50%",
    transform: "translateX(-50%)",
    marginBottom: 8,
    padding: "4px 10px",
    borderRadius: 8,
    backgroundColor: tooltipBg,
    color: pal.card,
    fontSize: 11,
    fontWeight: 500,
    whiteSpace: "nowrap",
    pointerEvents: "none",
    boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
  };

  const arrowStyle: React.CSSProperties = {
    position: "absolute",
    top: "100%",
    left: "50%",
    transform: "translateX(-50%)",
    marginTop: -1,
    border: "4px solid transparent",
    borderTopColor: tooltipBg,
  };

  let sembleUrl = "";
  if (isSemble) {
    const parts = noteId.replace("at://", "").split("/");
    const userHandle = parts[0] || "";
    const rkey = parts[2] || "";
    sembleUrl = `https://semble.so/profile/${userHandle}`;
    if (noteId.includes("network.cosmik.collection"))
      sembleUrl = `https://semble.so/profile/${userHandle}/collections/${rkey}`;
  }

  return (
    <>
      {isSemble && (
        <span className="relative inline-flex items-center group/semble">
          <span style={{ color: pal.muted }}>·</span>
          <a
            href={sembleUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center ml-1"
          >
            <img src="/semble-logo.svg" alt="Semble" className="h-3.5" />
          </a>
          <span
            style={tooltipStyle}
            className="opacity-0 group-hover/semble:opacity-100 transition-opacity"
          >
            {t("card.openInSemble")}
            <span style={arrowStyle} />
          </span>
        </span>
      )}
      {isLichen && (
        <span className="relative inline-flex items-center group/lichen">
          <span style={{ color: pal.muted }}>·</span>
          <span className="inline-flex items-center ml-1">
            <img src="/lichen-logo.svg" alt="Lichen" className="h-3.5" />
          </span>
          <span
            style={tooltipStyle}
            className="opacity-0 group-hover/lichen:opacity-100 transition-opacity"
          >
            {t("card.lichenBookmark")}
            <span style={arrowStyle} />
          </span>
        </span>
      )}
      {isExternalBookmark && (
        <span className="relative inline-flex items-center group/cb">
          <span style={{ color: pal.muted }}>·</span>
          <span className="inline-flex items-center ml-1">
            <Bookmark
              size={12}
              style={{ color: pal.muted }}
              className="fill-current"
            />
          </span>
          <span
            style={tooltipStyle}
            className="opacity-0 group-hover/cb:opacity-100 transition-opacity"
          >
            {t("card.externalBookmark")}
            <span style={arrowStyle} />
          </span>
        </span>
      )}
    </>
  );
}

export default function ReadingRoomCard({
  note,
  pal,
  featured,
  variant,
  activeTag,
  onTagClick,
  roomHandle,
  linkable = true,
}: {
  note: RRNote;
  pal: RRPalette;
  featured: boolean;
  variant: "card" | "row";
  activeTag: string | null;
  onTagClick: (tag: string) => void;
  roomHandle: string;
  /** Set false on the note's own permalink page, where linking to itself is noise. */
  linkable?: boolean;
}) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const isHighlight = note.motivation === "highlighting";
  const isBookmark = note.motivation === "bookmarking";
  const kind = isHighlight ? "highlight" : isBookmark ? "bookmark" : "note";
  const targetTitle = note.target?.title || "";
  const targetSource = note.target?.source || "";
  const bodyValue = note.body?.value || "";
  const quoteText = note.target?.selector?.exact || "";
  const tags = note.tags || [];

  let domain = "";
  if (targetSource) {
    try {
      domain = new URL(targetSource).hostname.replace(/^www\./, "");
    } catch {
      /* ignore unparseable source */
    }
  }

  const cleanUrl = targetSource
    .replace(/^https?:\/\//, "")
    .replace(/^www\./, "")
    .replace(/\/$/, "");
  const displayTitle = targetTitle || cleanUrl || domain;

  const onCustomDomain =
    typeof window !== "undefined" &&
    !!(window as unknown as { __READING_ROOM_HANDLE__?: string })
      .__READING_ROOM_HANDLE__;
  const notePath = onCustomDomain
    ? `/note?uri=${encodeURIComponent(note.id)}`
    : readingRoomNoteUrl(roomHandle, note.id);
  const noteUrl = linkable && note.id ? notePath : "";

  const handleCopyLink = async () => {
    if (!note.id) return;
    const url = `${window.location.origin}${notePath}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable, e.g. insecure context */
    }
  };

  const fragUrl = textFragmentUrl(targetSource, note.target?.selector);
  const passageHref = fragUrl || targetSource || undefined;
  const hlBg = highlightBg(note.color, pal.bg);
  const hlInk = highlightInk(pal.bg);

  const Icon = isHighlight ? Quote : isBookmark ? Bookmark : MessageSquare;

  return (
    <article
      className={variant === "card" ? "rounded-2xl p-6" : "py-7"}
      style={
        variant === "card"
          ? { backgroundColor: pal.card, border: `1px solid ${pal.border}` }
          : undefined
      }
    >
      <div className="flex items-center justify-between gap-3 mb-3">
        <span
          className="flex items-center gap-2 text-xs font-medium min-w-0"
          style={{ color: pal.muted }}
        >
          <span
            className="flex items-center justify-center w-6 h-6 rounded-full shrink-0"
            style={{ backgroundColor: pal.accentTint }}
          >
            <Icon size={12} style={{ color: pal.accentText }} />
          </span>
          {t(`readingRoom.type.${kind}`)}
          {note.id && <ExternalBadges noteId={note.id} pal={pal} t={t} />}
        </span>
        {featured && (
          <span
            className="text-[11px] font-medium px-2.5 py-1 rounded-full shrink-0"
            style={{ backgroundColor: pal.accentTint, color: pal.accentText }}
          >
            {t("readingRoom.featured")}
          </span>
        )}
      </div>

      {displayTitle && (
        <div className="mb-3 min-w-0">
          {targetSource ? (
            <a
              href={targetSource}
              target="_blank"
              rel="noopener noreferrer"
              className={`block font-semibold truncate hover:underline underline-offset-4 ${
                isBookmark ? "text-lg" : "text-sm"
              }`}
              style={{ color: pal.ink }}
            >
              {displayTitle}
            </a>
          ) : (
            <span
              className={`block font-semibold truncate ${
                isBookmark ? "text-lg" : "text-sm"
              }`}
              style={{ color: pal.ink }}
            >
              {displayTitle}
            </span>
          )}
          {domain && targetTitle && (
            <span
              className="flex items-center gap-1 text-xs mt-0.5"
              style={{ color: pal.muted }}
            >
              <ExternalLink size={10} />
              {domain}
            </span>
          )}
        </div>
      )}

      {isHighlight ? (
        <>
          {quoteText &&
            (passageHref ? (
              <a
                href={passageHref}
                target="_blank"
                rel="noopener noreferrer"
                title={t("readingRoom.jumpToText")}
                className="group block my-4 no-underline"
              >
                <mark
                  className="text-lg leading-loose rounded px-1.5 py-0.5 [text-wrap:pretty] transition-opacity group-hover:opacity-90"
                  style={{
                    backgroundColor: hlBg,
                    color: hlInk,
                    boxDecorationBreak: "clone",
                    WebkitBoxDecorationBreak: "clone",
                  }}
                >
                  {quoteText}
                </mark>
              </a>
            ) : (
              <p className="my-4">
                <mark
                  className="text-lg leading-loose rounded px-1.5 py-0.5 [text-wrap:pretty]"
                  style={{
                    backgroundColor: hlBg,
                    color: hlInk,
                    boxDecorationBreak: "clone",
                    WebkitBoxDecorationBreak: "clone",
                  }}
                >
                  {quoteText}
                </mark>
              </p>
            ))}
          {bodyValue && (
            <p
              className="text-sm leading-relaxed mb-4 mt-2 [text-wrap:pretty]"
              style={{ color: pal.muted }}
            >
              {bodyValue}
            </p>
          )}
        </>
      ) : (
        <>
          {quoteText &&
            (passageHref ? (
              <a
                href={passageHref}
                target="_blank"
                rel="noopener noreferrer"
                title={t("readingRoom.jumpToText")}
                className="group block my-3 no-underline"
              >
                <mark
                  className="text-sm leading-relaxed rounded px-1 py-0.5 [text-wrap:pretty] transition-opacity group-hover:opacity-90"
                  style={{
                    backgroundColor: hlBg,
                    color: hlInk,
                    boxDecorationBreak: "clone",
                    WebkitBoxDecorationBreak: "clone",
                  }}
                >
                  {quoteText}
                </mark>
              </a>
            ) : null)}
          {bodyValue && (
            <p
              className="text-base leading-relaxed mb-4 [text-wrap:pretty]"
              style={{ color: pal.ink }}
            >
              {bodyValue}
            </p>
          )}
        </>
      )}

      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-3">
          {tags.slice(0, 6).map((tag, i) => {
            const active = activeTag === tag;
            return (
              <button
                key={i}
                onClick={() => onTagClick(tag)}
                aria-pressed={active}
                className="text-xs px-2 py-0.5 rounded-full transition-colors"
                style={
                  active
                    ? { backgroundColor: pal.accent, color: pal.accentOn }
                    : { backgroundColor: pal.accentTint, color: pal.accentText }
                }
              >
                #{tag}
              </button>
            );
          })}
        </div>
      )}

      <div className="flex items-center justify-between gap-3">
        {note.created &&
          (noteUrl ? (
            <Link
              to={noteUrl}
              className="text-xs hover:underline underline-offset-4"
              style={{ color: pal.muted }}
            >
              {new Date(note.created).toLocaleDateString(undefined, {
                year: "numeric",
                month: "short",
                day: "numeric",
              })}
            </Link>
          ) : (
            <span className="text-xs" style={{ color: pal.muted }}>
              {new Date(note.created).toLocaleDateString(undefined, {
                year: "numeric",
                month: "short",
                day: "numeric",
              })}
            </span>
          ))}

        {note.id && (
          <button
            onClick={handleCopyLink}
            title={t("readingRoom.copyLink")}
            className="flex items-center gap-1 text-xs shrink-0 hover:underline underline-offset-4 transition-colors"
            style={{ color: copied ? pal.accentText : pal.muted }}
          >
            {copied ? <Check size={11} /> : <Link2 size={11} />}
            {copied ? t("readingRoom.copied") : t("readingRoom.copyLink")}
          </button>
        )}
      </div>
    </article>
  );
}
