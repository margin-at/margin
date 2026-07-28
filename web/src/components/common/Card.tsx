import React, { useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { useTranslation } from "react-i18next";
import RichText from "./RichText";
import MoreMenu from "./MoreMenu";
import type { MoreMenuItem } from "./MoreMenu";
import {
  MessageSquare,
  Heart,
  ExternalLink,
  FolderPlus,
  Trash2,
  Edit3,
  Globe,
  ShieldBan,
  VolumeX,
  Flag,
  EyeOff,
  Eye,
  Tag,
  Send,
  X,
  Bookmark,
  BookOpen,
} from "lucide-react";
import ShareMenu from "../modals/ShareMenu";
import AddToCollectionModal from "../modals/AddToCollectionModal";
import ExternalLinkModal from "../modals/ExternalLinkModal";
import ReportModal from "../modals/ReportModal";
import EditItemModal from "../modals/EditItemModal";
import EditHistoryModal from "../modals/EditHistoryModal";
import { clsx } from "clsx";
import {
  likeItem,
  unlikeItem,
  deleteItem,
  blockUser,
  muteUser,
  convertHighlightToAnnotation,
} from "../../api/client";
import { $user } from "../../store/auth";
import { $preferences } from "../../store/preferences";
import { displayHandle } from "../../lib/handle";
import { useStore } from "@nanostores/react";
import type {
  AnnotationItem,
  ContentLabel,
  LabelVisibility,
  Selector,
} from "../../types";
import { asTextQuote } from "../../types";

import { Avatar } from "../ui";
import CollectionIcon from "./CollectionIcon";
import SocialPostEmbed from "./SocialPostEmbed";
import {
  parseSocialPostUrl,
  postRefFromAtTags,
  fetchSocialPost,
  socialPostCacheKey,
} from "../../lib/socialPost";
import type { SocialPost } from "../../lib/socialPost";
import ProfileHoverCard from "./ProfileHoverCard";
import { analytics } from "../../lib/analytics";

const LABEL_DESCRIPTIONS: Record<string, string> = {
  sexual: "Sexual Content",
  nudity: "Nudity",
  violence: "Violence",
  gore: "Graphic Content",
  spam: "Spam",
  misleading: "Misleading",
};

function getContentWarning(
  labels?: ContentLabel[],
  prefs?: {
    labelPreferences: {
      labelerDid: string;
      label: string;
      visibility: LabelVisibility;
    }[];
  },
): {
  label: string;
  description: string;
  visibility: LabelVisibility;
  isAccountWide: boolean;
} | null {
  if (!labels || labels.length === 0) return null;
  const priority = [
    "gore",
    "violence",
    "nudity",
    "sexual",
    "misleading",
    "spam",
  ];
  for (const p of priority) {
    const match = labels.find((l) => l.val === p);
    if (match) {
      const pref = prefs?.labelPreferences.find(
        (lp) => lp.label === p && lp.labelerDid === match.src,
      );
      const visibility: LabelVisibility = pref?.visibility || "warn";
      if (visibility === "ignore") continue;
      return {
        label: p,
        description: LABEL_DESCRIPTIONS[p] || p,
        visibility,
        isAccountWide: match.scope === "account",
      };
    }
  }
  return null;
}

const PHYSICAL_BOOKS_SPEC = "https://margin.at/docs/specs/physical-books";

// Decode one application/x-www-form-urlencoded component. "+" maps to a space;
// the replace runs before decodeURIComponent so an escaped plus (%2B) survives.
const formDecode = (component: string): string =>
  decodeURIComponent(component.replace(/\+/g, " "));

// Printed page label from a selector's physical-books FragmentSelector, e.g.
// "255-256", or "A-1 - A-2" when a label itself contains an (encoded) hyphen.
function bookPageLabel(selector?: Selector | null): string | null {
  for (
    let sel: Selector | null | undefined = selector;
    sel;
    sel = sel.refinedBy
  ) {
    if (
      sel.type !== "FragmentSelector" ||
      sel.conformsTo !== PHYSICAL_BOOKS_SPEC ||
      !sel.value
    ) {
      continue;
    }
    // Keep the value raw (%2D still encoded) so the range hyphen stays unambiguous.
    const raw = sel.value
      .split("&")
      .map((pair) => pair.split("="))
      .find(([key]) => key === "page")?.[1];
    if (!raw) return null;
    try {
      const pages = raw.split("-").map(formDecode);
      const separator = /%2d/i.test(raw) ? " - " : "-";
      return pages.join(separator);
    } catch {
      return null;
    }
  }
  return null;
}

interface CardProps {
  item: AnnotationItem;
  onDelete?: (uri: string) => void;
  onUpdate?: (item: AnnotationItem) => void;
  hideShare?: boolean;
  hideCollection?: boolean;
  layout?: "list" | "mosaic";
}

export default function Card({
  item: initialItem,
  onDelete,
  onUpdate,
  hideShare,
  hideCollection = false,
  layout = "list",
}: CardProps) {
  const { t } = useTranslation();
  const [item, setItem] = useState(initialItem);
  const [prevInitial, setPrevInitial] = useState(initialItem);
  if (initialItem !== prevInitial) {
    setPrevInitial(initialItem);
    setItem(initialItem);
  }
  const user = useStore($user);
  const preferences = useStore($preferences);
  const isAuthor = user && item.author?.did === user.did;

  const [liked, setLiked] = useState(!!item.viewer?.like);
  const [likes, setLikes] = useState(item.likeCount || 0);
  const [prevLike, setPrevLike] = useState(item.viewer?.like);
  const [prevLikeCount, setPrevLikeCount] = useState(item.likeCount);
  if (item.viewer?.like !== prevLike || item.likeCount !== prevLikeCount) {
    setPrevLike(item.viewer?.like);
    setPrevLikeCount(item.likeCount);
    setLiked(!!item.viewer?.like);
    setLikes(item.likeCount || 0);
  }
  const [showCollectionModal, setShowCollectionModal] = useState(false);
  const [showExternalLinkModal, setShowExternalLinkModal] = useState(false);
  const [externalLinkUrl, setExternalLinkUrl] = useState<string | null>(null);
  const [showReportModal, setShowReportModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showEditHistory, setShowEditHistory] = useState(false);
  const [contentRevealed, setContentRevealed] = useState(false);
  const [showConvertInput, setShowConvertInput] = useState(false);
  const [convertText, setConvertText] = useState("");
  const [converting, setConverting] = useState(false);
  const [ogData, setOgData] = useState<{
    title?: string;
    description?: string;
    image?: string;
    icon?: string;
    "at:canonical"?: string;
  } | null>(() => {
    const sembleNote =
      (initialItem.uri?.includes("network.cosmik") ||
        initialItem.uri?.includes("semble")) &&
      initialItem.motivation === "commenting" &&
      initialItem.body?.value;
    if (initialItem.motivation !== "bookmarking" && !sembleNote) return null;
    const url = initialItem.target?.source || initialItem.source;
    if (!url) return null;
    try {
      const cached = sessionStorage.getItem(`og:${url}`);
      return cached ? JSON.parse(cached) : null;
    } catch {
      return null;
    }
  });
  const [imgError, setImgError] = useState(false);
  const [iconError, setIconError] = useState(false);

  const contentWarning = getContentWarning(item.labels, preferences);

  const type =
    item.motivation === "highlighting"
      ? "highlight"
      : item.motivation === "bookmarking"
        ? "bookmark"
        : "annotation";

  const isSemble =
    item.uri?.includes("network.cosmik") || item.uri?.includes("semble");

  const isLichen = item.uri?.includes("wiki.lichen.bookmark");

  const isCommunityBookmark = item.uri?.includes(
    "community.lexicon.bookmarks.bookmark",
  );

  const safeUrlHostname = (url: string | null | undefined) => {
    if (!url) return null;
    try {
      return new URL(url).hostname;
    } catch {
      return null;
    }
  };

  const pageUrl = item.target?.source || item.source;
  const isBookmark = type === "bookmark" && !item.body?.value;
  const isSembleNote =
    isSemble && type === "annotation" && !!item.body?.value && !!pageUrl;
  const showLinkPreview = isBookmark || isSembleNote;

  const socialPostRef = React.useMemo(
    () => parseSocialPostUrl(pageUrl),
    [pageUrl],
  );
  const atTagPostRef = React.useMemo(() => {
    if (socialPostRef) return null;
    return postRefFromAtTags(
      ogData?.["at:canonical"],
      safeUrlHostname(pageUrl) || "",
    );
  }, [socialPostRef, ogData, pageUrl]);
  const postRef = socialPostRef ?? atTagPostRef;
  const showSocialPost =
    !!postRef && (isBookmark || (type === "annotation" && !!item.body?.value));

  const [socialPost, setSocialPost] = useState<SocialPost | null>(() => {
    if (!pageUrl) return null;
    try {
      const cached = sessionStorage.getItem(socialPostCacheKey(pageUrl));
      return cached && cached !== "null" ? JSON.parse(cached) : null;
    } catch {
      return null;
    }
  });
  const [socialPostFailed, setSocialPostFailed] = useState(false);

  React.useEffect(() => {
    if (
      showSocialPost &&
      postRef &&
      pageUrl &&
      !socialPost &&
      !socialPostFailed
    ) {
      let cancelled = false;
      fetchSocialPost(pageUrl, postRef).then((data) => {
        if (cancelled) return;
        if (data) setSocialPost(data);
        else setSocialPostFailed(true);
      });
      return () => {
        cancelled = true;
      };
    }
  }, [showSocialPost, postRef, pageUrl, socialPost, socialPostFailed]);

  React.useEffect(() => {
    if (
      showLinkPreview &&
      (!showSocialPost || socialPostFailed) &&
      item.uri &&
      !ogData &&
      pageUrl
    ) {
      let cancelled = false;
      import("../../lib/metadataQueue").then(({ fetchMetadata }) => {
        fetchMetadata(pageUrl).then((data) => {
          if (!cancelled && data) setOgData(data);
        });
      });
      return () => {
        cancelled = true;
      };
    }
  }, [
    showLinkPreview,
    showSocialPost,
    socialPostFailed,
    item.uri,
    pageUrl,
    ogData,
  ]);

  if (contentWarning?.visibility === "hide") return null;

  const handleLike = async () => {
    const prev = { liked, likes };
    setLiked(!liked);
    setLikes((l) => (liked ? l - 1 : l + 1));

    const success = liked
      ? await unlikeItem(item.uri)
      : await likeItem(item.uri, item.cid);

    if (!success) {
      setLiked(prev.liked);
      setLikes(prev.likes);
    } else {
      analytics.capture("item_liked", {
        type,
        action: liked ? "unlike" : "like",
      });
    }
  };

  const handleDelete = async () => {
    if (window.confirm(t("card.deleteConfirm"))) {
      const success = await deleteItem(item.uri, type);
      if (success && onDelete) {
        analytics.capture("item_deleted", { type });
        onDelete(item.uri);
      }
    }
  };

  const handleConvert = async () => {
    if (!convertText.trim() || converting) return;
    setConverting(true);
    const pageUrl = item.target?.source || item.source || "";
    const res = await convertHighlightToAnnotation(
      item.uri,
      pageUrl,
      convertText.trim(),
      asTextQuote(item.target?.selector),
      item.target?.title,
    );
    setConverting(false);
    if (res.success) {
      setShowConvertInput(false);
      setConvertText("");
      if (onDelete) onDelete(item.uri);
    }
  };

  const handleExternalClick = (e: React.MouseEvent, url: string) => {
    e.preventDefault();
    e.stopPropagation();

    try {
      const hostname = safeUrlHostname(url);
      if (hostname) {
        if (
          hostname === "margin.at" ||
          hostname.endsWith(".margin.at") ||
          hostname === "semble.so" ||
          hostname.endsWith(".semble.so")
        ) {
          window.open(url, "_blank", "noopener,noreferrer");
          return;
        }

        if ($preferences.get().disableExternalLinkWarning) {
          window.open(url, "_blank", "noopener,noreferrer");
          return;
        }

        const skipped = $preferences.get().externalLinkSkippedHostnames;
        if (skipped.includes(hostname)) {
          window.open(url, "_blank", "noopener,noreferrer");
          return;
        }
      }
    } catch (err) {
      if (err instanceof Error && err.name !== "TypeError") {
        console.debug("Failed to check skipped hostname:", err);
      }
    }

    setExternalLinkUrl(url);
    setShowExternalLinkModal(true);
  };

  const timestamp = item.createdAt
    ? formatDistanceToNow(new Date(item.createdAt), { addSuffix: false })
        .replace("less than a minute", t("card.justNow"))
        .replace("about ", "")
        .replace(" hours", "h")
        .replace(" hour", "h")
        .replace(" minutes", "m")
        .replace(" minute", "m")
        .replace(" days", "d")
        .replace(" day", "d")
    : "";

  const uriCollection = item.uri?.split("/")[3] ?? "";
  const urlSegment = uriCollection.includes("at.margin.note")
    ? "note"
    : uriCollection.includes("at.margin.highlight")
      ? "highlight"
      : uriCollection.includes("at.margin.bookmark")
        ? "bookmark"
        : uriCollection.includes("at.margin.annotation")
          ? "annotation"
          : type;
  const detailUrl = `/${item.author?.handle || item.author?.did}/${urlSegment}/${(item.uri || "").split("/").pop()}`;

  const pageTitle =
    item.target?.title ||
    item.title ||
    (pageUrl ? safeUrlHostname(pageUrl) : null);
  const displayUrl = pageUrl
    ? (() => {
        const clean = pageUrl
          .replace(/^https?:\/\//, "")
          .replace(/^www\./, "")
          .replace(/\/$/, "");
        return clean.length > 60 ? clean.slice(0, 57) + "..." : clean;
      })()
    : null;

  const isbnMatch = pageUrl?.match(/^urn:isbn:(.+)$/i);
  const isbn = isbnMatch ? isbnMatch[1].replace(/-/g, "") : null;
  const bookUrl = isbn ? `https://openlibrary.org/isbn/${isbn}` : null;
  const bookTitle =
    item.target?.title || item.title || (isbn ? `ISBN ${isbn}` : null);
  const bookPage = bookPageLabel(item.target?.selector);

  const quoteLinkUrl = (() => {
    const sel = asTextQuote(item.target?.selector);
    if (!sel?.exact) return null;
    if (bookUrl) return bookUrl;
    if (!pageUrl) return null;
    const prefix = sel.prefix ? encodeURIComponent(sel.prefix) + "-," : "";
    const suffix = sel.suffix ? ",-" + encodeURIComponent(sel.suffix) : "";
    return `${pageUrl}#:~:text=${prefix}${encodeURIComponent(sel.exact)}${suffix}`;
  })();

  const decodeHTMLEntities = (text: string) => {
    if (!text.includes("&")) return text;
    try {
      const doc = new DOMParser().parseFromString(
        `<!doctype html><body>${text}`,
        "text/html",
      );
      return doc.body.textContent ?? text;
    } catch {
      return text;
    }
  };

  const displayTitle = decodeHTMLEntities(
    item.title || ogData?.title || pageTitle || t("card.untitledBookmark"),
  );
  const displayDescription =
    item.description || ogData?.description
      ? decodeHTMLEntities(item.description || ogData?.description || "")
      : undefined;
  const displayImage = ogData?.image;

  const linkPreview = (
    <div
      onClick={(e) => {
        e.preventDefault();
        if (pageUrl) handleExternalClick(e, pageUrl);
      }}
      role="button"
      tabIndex={0}
      className={clsx(
        "flex bg-surface-50 dark:bg-surface-800 rounded-xl border border-surface-200 dark:border-surface-700 hover:border-primary-300 dark:hover:border-primary-600 hover:bg-surface-100 dark:hover:bg-surface-700 transition-all group overflow-hidden cursor-pointer",
        layout === "mosaic"
          ? "flex-col items-stretch"
          : "flex-row items-stretch",
        isSembleNote && "mt-2.5",
      )}
    >
      {displayImage && !imgError && (
        <div
          className={clsx(
            "shrink-0 bg-surface-200 dark:bg-surface-700 relative",
            layout === "mosaic"
              ? "w-full aspect-video border-b border-surface-200 dark:border-surface-700"
              : "w-[90px] sm:w-[140px] border-r border-surface-200 dark:border-surface-700",
          )}
        >
          <div className="absolute inset-0 flex items-center justify-center overflow-hidden">
            <img
              src={displayImage}
              alt={displayTitle || "Link preview"}
              className="h-full w-full object-cover"
              onError={() => setImgError(true)}
            />
          </div>
        </div>
      )}
      <div
        className={clsx(
          "p-3 min-w-0 flex flex-col font-sans",
          layout === "mosaic" ? "w-full" : "flex-1 justify-center",
        )}
      >
        <h3 className="font-semibold text-surface-900 dark:text-white text-sm leading-snug group-hover:text-primary-600 dark:group-hover:text-primary-400 mb-1.5 transition-colors line-clamp-2">
          {displayTitle}
        </h3>

        {displayDescription && (
          <p className="text-surface-600 dark:text-surface-400 text-xs leading-relaxed mb-2 line-clamp-2">
            {displayDescription}
          </p>
        )}

        <div className="flex items-center gap-2 text-[11px] text-surface-500 dark:text-surface-500 mt-auto">
          <div className="w-4 h-4 rounded-full bg-surface-200 dark:bg-surface-700 flex items-center justify-center shrink-0 overflow-hidden">
            {ogData?.icon && !iconError ? (
              <img
                src={ogData.icon}
                alt=""
                onError={() => setIconError(true)}
                className="w-3 h-3 object-contain"
              />
            ) : (
              <Globe size={9} />
            )}
          </div>
          <span className="truncate min-w-0 flex-1">
            {displayUrl || pageUrl}
          </span>
        </div>
      </div>
    </div>
  );

  const socialEmbed =
    socialPost && postRef && pageUrl ? (
      <SocialPostEmbed
        post={socialPost}
        postUrl={pageUrl}
        host={postRef.host}
        onOpen={handleExternalClick}
        className={!isBookmark ? "mt-2.5" : undefined}
      />
    ) : null;

  const resolvedPreview =
    socialEmbed ??
    (socialPostRef && showSocialPost && !socialPostFailed ? null : linkPreview);

  return (
    <article className="card p-4 hover:ring-black/10 dark:hover:ring-white/10 transition-all relative overflow-visible min-w-0 w-full">
      {!hideCollection &&
        (item.collection || (item.context && item.context.length > 0)) && (
          <div className="flex items-center gap-1.5 text-xs text-surface-400 dark:text-surface-500 mb-2 flex-wrap">
            {item.addedBy && item.addedBy.did !== item.author?.did ? (
              <>
                <ProfileHoverCard did={item.addedBy.did}>
                  <a
                    href={`/profile/${item.addedBy.did}`}
                    className="flex items-center gap-1.5 font-medium hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                  >
                    <Avatar
                      did={item.addedBy.did}
                      avatar={item.addedBy.avatar}
                      size="xs"
                    />
                    <span>
                      {item.addedBy.displayName ||
                        `@${displayHandle(item.addedBy.handle, item.addedBy.did)}`}
                    </span>
                  </a>
                </ProfileHoverCard>
                <span>{t("card.addedToLower")}</span>
              </>
            ) : (
              <span>{t("card.addedTo")}</span>
            )}

            {item.context && item.context.length > 0 ? (
              item.context.map((col, index) => (
                <React.Fragment key={col.uri}>
                  {index > 0 && index < item.context!.length - 1 && (
                    <span className="text-surface-300 dark:text-surface-600">
                      ,
                    </span>
                  )}
                  {index > 0 && index === item.context!.length - 1 && (
                    <span>{t("card.and")}</span>
                  )}
                  <a
                    href={`/${item.addedBy?.handle || ""}/collection/${(col.uri || "").split("/").pop()}`}
                    className="inline-flex items-center gap-1 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                  >
                    <CollectionIcon icon={col.icon} size={14} />
                    <span className="font-medium">{col.name}</span>
                  </a>
                </React.Fragment>
              ))
            ) : (
              <a
                href={`/${item.addedBy?.handle || ""}/collection/${(item.collection!.uri || "").split("/").pop()}`}
                className="inline-flex items-center gap-1 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
              >
                <CollectionIcon icon={item.collection!.icon} size={14} />
                <span className="font-medium">{item.collection!.name}</span>
              </a>
            )}
          </div>
        )}

      <div className="flex items-start gap-3 min-w-0">
        <ProfileHoverCard did={item.author?.did}>
          <a href={`/profile/${item.author?.did}`} className="shrink-0">
            <div className="rounded-full overflow-hidden">
              <div
                className={clsx(
                  "transition-all",
                  contentWarning?.isAccountWide &&
                    !contentRevealed &&
                    "blur-md",
                )}
              >
                <Avatar
                  did={item.author?.did}
                  avatar={item.author?.avatar}
                  size="md"
                />
              </div>
            </div>
          </a>
        </ProfileHoverCard>

        <div
          className={clsx(
            "flex-1 min-w-0",
            isSembleNote && "min-h-10 flex flex-col justify-center",
          )}
        >
          <div className="flex items-center gap-1.5 flex-wrap min-w-0">
            <ProfileHoverCard
              did={item.author?.did}
              className="min-w-0 max-w-[180px] sm:max-w-none"
            >
              <a
                href={`/profile/${item.author?.did}`}
                className="font-semibold text-surface-900 dark:text-white text-[15px] hover:underline block truncate sm:whitespace-normal sm:overflow-visible"
              >
                {item.author?.displayName ||
                  displayHandle(item.author?.handle, item.author?.did)}
              </a>
            </ProfileHoverCard>
            <span className="text-surface-400 dark:text-surface-500 text-sm truncate max-w-[120px] sm:max-w-none sm:whitespace-normal sm:overflow-visible break-all">
              @{displayHandle(item.author?.handle, item.author?.did)}
            </span>
            <span className="text-surface-300 dark:text-surface-600">·</span>
            <span className="text-surface-400 dark:text-surface-500 text-sm">
              {timestamp}
              {item.editedAt && (
                <button
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setShowEditHistory(true);
                  }}
                  className="ml-1 text-surface-400 dark:text-surface-500 hover:text-surface-600 dark:hover:text-surface-400 hover:underline cursor-pointer"
                  title={`Edited ${new Date(item.editedAt).toLocaleString()}`}
                >
                  {t("card.edited")}
                </button>
              )}
            </span>

            {isSemble &&
              (() => {
                const uri = item.uri || "";
                const parts = uri.replace("at://", "").split("/");
                const userHandle = item.author?.handle || parts[0] || "";
                const rkey = parts[2] || "";
                const targetUrl = item.target?.source || item.source || "";
                let sembleUrl = `https://semble.so/profile/${userHandle}`;
                if (uri.includes("network.cosmik.collection"))
                  sembleUrl = `https://semble.so/profile/${userHandle}/collections/${rkey}`;
                else if (uri.includes("network.cosmik.card") && targetUrl)
                  sembleUrl = `https://semble.so/url?id=${encodeURIComponent(targetUrl)}`;
                return (
                  <span className="relative inline-flex items-center">
                    <span className="text-surface-300 dark:text-surface-600">
                      ·
                    </span>
                    <button
                      onClick={(e) => handleExternalClick(e, sembleUrl)}
                      className="group/semble relative inline-flex items-center ml-1 cursor-pointer"
                    >
                      <img
                        src="/semble-logo.svg"
                        alt="Semble"
                        className="h-3.5"
                      />
                      <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2.5 py-1 rounded-lg bg-surface-800 dark:bg-surface-700 text-white text-[11px] font-medium whitespace-nowrap opacity-0 group-hover/semble:opacity-100 transition-opacity shadow-lg">
                        {t("card.openInSemble")}
                        <span className="absolute top-full left-1/2 -translate-x-1/2 -mt-px border-4 border-transparent border-t-surface-800 dark:border-t-surface-700" />
                      </span>
                    </button>
                  </span>
                );
              })()}

            {isLichen && (
              <span className="relative inline-flex items-center">
                <span className="text-surface-300 dark:text-surface-600">
                  ·
                </span>
                <span className="group/lichen relative inline-flex items-center ml-1">
                  <img src="/lichen-logo.svg" alt="Lichen" className="h-3.5" />
                  <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2.5 py-1 rounded-lg bg-surface-800 dark:bg-surface-700 text-white text-[11px] font-medium whitespace-nowrap opacity-0 group-hover/lichen:opacity-100 transition-opacity shadow-lg">
                    {t("card.lichenBookmark")}
                    <span className="absolute top-full left-1/2 -translate-x-1/2 -mt-px border-4 border-transparent border-t-surface-800 dark:border-t-surface-700" />
                  </span>
                </span>
              </span>
            )}

            {isCommunityBookmark && (
              <span className="relative inline-flex items-center">
                <span className="text-surface-300 dark:text-surface-600">
                  ·
                </span>
                <span className="group/cb relative inline-flex items-center ml-1">
                  <Bookmark
                    size={12}
                    className="text-surface-400 dark:text-surface-500 fill-current"
                  />
                  <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2.5 py-1 rounded-lg bg-surface-800 dark:bg-surface-700 text-white text-[11px] font-medium whitespace-nowrap opacity-0 group-hover/cb:opacity-100 transition-opacity shadow-lg">
                    {t("card.externalBookmark")}
                    <span className="absolute top-full left-1/2 -translate-x-1/2 -mt-px border-4 border-transparent border-t-surface-800 dark:border-t-surface-700" />
                  </span>
                </span>
              </span>
            )}
          </div>

          {pageUrl &&
            !isBookmark &&
            !isSembleNote &&
            !socialPost &&
            !(contentWarning && !contentRevealed) && (
              <div className="flex items-center gap-1.5 mt-0.5 max-w-full">
                <a
                  href={bookUrl || pageUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={(e) => handleExternalClick(e, bookUrl || pageUrl)}
                  className="inline-flex items-center gap-1 text-xs text-primary-600 dark:text-primary-400 hover:underline min-w-0"
                >
                  {isbn ? (
                    <BookOpen size={10} className="flex-shrink-0" />
                  ) : (
                    <ExternalLink size={10} className="flex-shrink-0" />
                  )}
                  <span className="truncate">
                    {isbn ? bookTitle : displayUrl}
                  </span>
                </a>
                {isbn && bookPage && (
                  <span className="flex-shrink-0 text-xs text-surface-500 dark:text-surface-400">
                    (p.{bookPage})
                  </span>
                )}
              </div>
            )}
        </div>
      </div>

      <div
        className={clsx(
          "relative",
          isSembleNote ? "mt-0.5" : "mt-3",
          layout === "mosaic" ? "" : "ml-[52px]",
        )}
      >
        {contentWarning && !contentRevealed && (
          <div className="z-10 rounded-lg bg-surface-100 dark:bg-surface-800 flex flex-col items-center justify-center gap-2 py-6 min-h-[120px]">
            <div className="flex items-center gap-2 text-surface-500 dark:text-surface-400">
              <EyeOff size={16} />
              <span className="text-sm font-medium">
                {t(`card.labelDescriptions.${contentWarning.label}`, {
                  defaultValue: contentWarning.description,
                })}
              </span>
            </div>
            <button
              onClick={() => setContentRevealed(true)}
              className="flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-lg bg-surface-200 dark:bg-surface-700 text-surface-600 dark:text-surface-300 hover:bg-surface-300 dark:hover:bg-surface-600 transition-colors"
            >
              <Eye size={12} />
              {t("card.show")}
            </button>
          </div>
        )}
        {contentWarning && contentRevealed && (
          <button
            onClick={() => setContentRevealed(false)}
            className="flex items-center gap-1.5 mb-2 px-2.5 py-1 text-xs font-medium rounded-lg bg-surface-100 dark:bg-surface-800 text-surface-500 dark:text-surface-400 hover:bg-surface-200 dark:hover:bg-surface-700 transition-colors"
          >
            <EyeOff size={12} />
            {t("card.hideContent")}
          </button>
        )}
        {!(contentWarning && !contentRevealed) && isBookmark && resolvedPreview}

        {!(contentWarning && !contentRevealed) &&
          asTextQuote(item.target?.selector)?.exact && (
            <blockquote
              className={clsx(
                "pl-4 py-2 border-l-[3px] mb-3 text-[15px] italic text-surface-600 dark:text-surface-300 rounded-r-lg hover:bg-surface-50 dark:hover:bg-surface-800/50 transition-colors",
                !item.color &&
                  type === "highlight" &&
                  "border-yellow-400 bg-yellow-50/50 dark:bg-yellow-900/20",
                item.color === "yellow" &&
                  "border-yellow-400 bg-yellow-50/50 dark:bg-yellow-900/20",
                item.color === "green" &&
                  "border-green-400 bg-green-50/50 dark:bg-green-900/20",
                item.color === "red" &&
                  "border-red-400 bg-red-50/50 dark:bg-red-900/20",
                item.color === "blue" &&
                  "border-blue-400 bg-blue-50/50 dark:bg-blue-900/20",
                !item.color &&
                  type !== "highlight" &&
                  "border-surface-300 dark:border-surface-600",
              )}
              style={
                item.color?.startsWith("#")
                  ? {
                      borderColor: item.color,
                      backgroundColor: `${item.color}15`,
                    }
                  : undefined
              }
            >
              <a
                href={quoteLinkUrl || undefined}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => {
                  if (quoteLinkUrl) handleExternalClick(e, quoteLinkUrl);
                }}
                className="block break-words"
              >
                "{asTextQuote(item.target?.selector)?.exact}"
              </a>
            </blockquote>
          )}

        {!(contentWarning && !contentRevealed) && item.body?.value && (
          <p className="text-surface-900 dark:text-surface-100 whitespace-pre-wrap break-words leading-relaxed text-[15px]">
            <RichText text={item.body.value} />
          </p>
        )}

        {!(contentWarning && !contentRevealed) &&
          isSembleNote &&
          resolvedPreview}

        {!(contentWarning && !contentRevealed) &&
          !isBookmark &&
          !isSembleNote &&
          socialEmbed}

        {!(contentWarning && !contentRevealed) &&
          item.tags &&
          item.tags.length > 0 && (
            <div className="flex flex-wrap gap-2 mt-3">
              {item.tags.map((tag) => (
                <a
                  key={tag}
                  href={`/home?tag=${encodeURIComponent(tag)}`}
                  className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-surface-100 dark:bg-surface-800 text-xs font-medium text-surface-600 dark:text-surface-400 hover:bg-surface-200 dark:hover:bg-surface-700 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                  onClick={(e) => e.stopPropagation()}
                >
                  <Tag size={10} />
                  <span>{tag}</span>
                </a>
              ))}
            </div>
          )}
      </div>

      <div className="flex items-center gap-1 mt-3 ml-[52px] md:ml-0 md:gap-0">
        <button
          onClick={handleLike}
          className={clsx(
            "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm transition-all",
            liked
              ? "text-red-500 bg-red-50 dark:bg-red-900/20"
              : "text-surface-400 dark:text-surface-500 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20",
          )}
        >
          <Heart size={16} className={clsx(liked && "fill-current")} />
          {likes > 0 && <span className="text-xs font-medium">{likes}</span>}
        </button>

        {type === "annotation" && (
          <a
            href={detailUrl}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm text-surface-400 dark:text-surface-500 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
          >
            <MessageSquare size={16} />
            {(item.replyCount || 0) > 0 && (
              <span className="text-xs font-medium">{item.replyCount}</span>
            )}
          </a>
        )}

        {user && (
          <button
            onClick={() => setShowCollectionModal(true)}
            className="flex items-center px-2.5 py-1.5 rounded-lg text-surface-400 dark:text-surface-500 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
            title={t("card.addToCollectionTitle")}
          >
            <FolderPlus size={16} />
          </button>
        )}

        {!hideShare && (
          <ShareMenu
            uri={item.uri}
            text={item.body?.value || ""}
            handle={item.author?.handle}
            type={type}
            url={pageUrl}
          />
        )}

        {isAuthor && (
          <>
            <div className="flex-1" />
            {type === "highlight" && !showConvertInput && (
              <button
                onClick={() => setShowConvertInput(true)}
                className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-surface-400 dark:text-surface-500 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all text-xs font-medium"
                title={t("card.annotateTitle")}
              >
                <MessageSquare size={14} />
                <span className="hidden sm:inline">{t("card.annotate")}</span>
              </button>
            )}
            <button
              onClick={() => setShowEditModal(true)}
              className="flex items-center px-2.5 py-1.5 rounded-lg text-surface-400 dark:text-surface-500 hover:text-surface-600 dark:hover:text-surface-300 hover:bg-surface-50 dark:hover:bg-surface-800 transition-all"
              title={t("card.editTitle")}
            >
              <Edit3 size={14} />
            </button>
            <button
              onClick={handleDelete}
              className="flex items-center px-2.5 py-1.5 rounded-lg text-surface-400 dark:text-surface-500 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-all"
              title={t("card.deleteTitle")}
            >
              <Trash2 size={14} />
            </button>
          </>
        )}

        {!isAuthor && user && (
          <>
            <div className="flex-1" />
            <MoreMenu
              items={(() => {
                const menuItems: MoreMenuItem[] = [
                  {
                    label: t("card.report"),
                    icon: <Flag size={14} />,
                    onClick: () => setShowReportModal(true),
                    variant: "danger",
                  },
                  {
                    label: t("card.muteUser", {
                      handle: item.author?.handle || "user",
                    }),
                    icon: <VolumeX size={14} />,
                    onClick: async () => {
                      if (item.author?.did) {
                        await muteUser(item.author.did);
                        onDelete?.(item.uri);
                      }
                    },
                  },
                  {
                    label: t("card.blockUser", {
                      handle: item.author?.handle || "user",
                    }),
                    icon: <ShieldBan size={14} />,
                    onClick: async () => {
                      if (item.author?.did) {
                        await blockUser(item.author.did);
                        onDelete?.(item.uri);
                      }
                    },
                    variant: "danger",
                  },
                ];
                return menuItems;
              })()}
            />
          </>
        )}
      </div>

      {showConvertInput && (
        <div
          className={clsx(
            "mt-3 animate-fade-in",
            layout === "mosaic" ? "" : "ml-[52px]",
          )}
        >
          <div className="flex gap-2 items-end">
            <textarea
              value={convertText}
              onChange={(e) => setConvertText(e.target.value)}
              placeholder={t("card.addNotePlaceholder")}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  handleConvert();
                }
                if (e.key === "Escape") {
                  setShowConvertInput(false);
                  setConvertText("");
                }
              }}
              className="flex-1 p-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-xl text-sm resize-none focus:outline-none focus:border-primary-400 focus:ring-2 focus:ring-primary-400/20 min-h-[80px] placeholder:text-surface-400"
            />
            <div className="flex flex-col gap-1.5">
              <button
                onClick={handleConvert}
                disabled={converting || !convertText.trim()}
                className="p-2.5 bg-primary-600 text-white rounded-xl hover:bg-primary-700 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                title={t("card.convertToAnnotation")}
              >
                {converting ? (
                  <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent" />
                ) : (
                  <Send size={16} />
                )}
              </button>
              <button
                onClick={() => {
                  setShowConvertInput(false);
                  setConvertText("");
                }}
                className="p-2.5 text-surface-400 hover:text-surface-600 dark:hover:text-surface-200 hover:bg-surface-100 dark:hover:bg-surface-800 rounded-xl transition-all"
                title={t("common.cancel")}
              >
                <X size={16} />
              </button>
            </div>
          </div>
        </div>
      )}

      <AddToCollectionModal
        isOpen={showCollectionModal}
        onClose={() => setShowCollectionModal(false)}
        annotationUri={item.uri}
      />

      <ExternalLinkModal
        isOpen={showExternalLinkModal}
        onClose={() => setShowExternalLinkModal(false)}
        url={externalLinkUrl}
      />

      <ReportModal
        isOpen={showReportModal}
        onClose={() => setShowReportModal(false)}
        subjectDid={item.author?.did || ""}
        subjectUri={item.uri}
        subjectHandle={item.author?.handle}
      />

      <EditItemModal
        isOpen={showEditModal}
        onClose={() => setShowEditModal(false)}
        item={item}
        type={type}
        onSaved={(updated) => {
          setItem(updated);
          onUpdate?.(updated);
        }}
      />
      <EditHistoryModal
        isOpen={showEditHistory}
        onClose={() => setShowEditHistory(false)}
        item={item}
      />
    </article>
  );
}
