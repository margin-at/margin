import React, { useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { clsx } from "clsx";
import {
  MessageSquare,
  Heart,
  Repeat2,
  ExternalLink,
  Play,
} from "lucide-react";
import RichText from "./RichText";
import { Avatar } from "../ui";
import { segmentByFacets } from "../../lib/socialPost";
import type { SocialPost, SocialPostFacet } from "../../lib/socialPost";
import { displayHandle } from "../../lib/handle";

interface SocialPostEmbedProps {
  post: SocialPost;
  postUrl: string;
  host: string;
  className?: string;
  onOpen: (e: React.MouseEvent, url: string) => void;
}

function shortTimestamp(createdAt?: string): string {
  if (!createdAt) return "";
  try {
    return formatDistanceToNow(new Date(createdAt), { addSuffix: false })
      .replace("less than a minute", "now")
      .replace("about ", "")
      .replace(/ hours?/, "h")
      .replace(/ minutes?/, "m")
      .replace(/ months?/, "mo")
      .replace(/ days?/, "d")
      .replace(/ years?/, "y");
  } catch {
    return "";
  }
}

function formatCount(count?: number): string | null {
  if (!count) return null;
  if (count >= 10000) return `${Math.round(count / 1000)}K`;
  if (count >= 1000) return `${(count / 1000).toFixed(1).replace(/\.0$/, "")}K`;
  return String(count);
}

const FACET_LINK_CLASS =
  "text-primary-600 dark:text-primary-400 hover:underline";

function PostText({
  text,
  facets,
  host,
  onOpen,
}: {
  text: string;
  facets?: SocialPostFacet[];
  host: string;
  onOpen: (e: React.MouseEvent, url: string) => void;
}) {
  if (!facets?.length) return <RichText text={text} />;
  return (
    <>
      {segmentByFacets(text, facets).map((segment, i) => {
        const facet = segment.facet;
        if (facet?.link) {
          const link = facet.link;
          return (
            <a
              key={i}
              href={link}
              target="_blank"
              rel="noopener noreferrer"
              className={FACET_LINK_CLASS}
              onClick={(e) => onOpen(e, link)}
            >
              {segment.text}
            </a>
          );
        }
        if (facet?.mention) {
          return (
            <a
              key={i}
              href={`/profile/${facet.mention}`}
              className={FACET_LINK_CLASS}
              onClick={(e) => e.stopPropagation()}
            >
              {segment.text}
            </a>
          );
        }
        if (facet?.tag) {
          const tagUrl = `https://${host}/hashtag/${encodeURIComponent(facet.tag)}`;
          return (
            <a
              key={i}
              href={tagUrl}
              target="_blank"
              rel="noopener noreferrer"
              className={FACET_LINK_CLASS}
              onClick={(e) => onOpen(e, tagUrl)}
            >
              {segment.text}
            </a>
          );
        }
        return <React.Fragment key={i}>{segment.text}</React.Fragment>;
      })}
    </>
  );
}

export default function SocialPostEmbed({
  post,
  postUrl,
  host,
  className,
  onOpen,
}: SocialPostEmbedProps) {
  const [videoThumbError, setVideoThumbError] = useState(false);
  const timestamp = shortTimestamp(post.createdAt);
  const counts = [
    { icon: MessageSquare, value: formatCount(post.replyCount) },
    { icon: Repeat2, value: formatCount(post.repostCount) },
    { icon: Heart, value: formatCount(post.likeCount) },
  ].filter((c) => c.value);

  return (
    <div
      onClick={(e) => {
        e.preventDefault();
        onOpen(e, postUrl);
      }}
      role="button"
      tabIndex={0}
      className={clsx(
        "block bg-surface-50 dark:bg-surface-800 rounded-xl border border-surface-200 dark:border-surface-700 hover:border-surface-300 dark:hover:border-surface-600 transition-colors group overflow-hidden cursor-pointer p-3 font-sans",
        className,
      )}
    >
      <div className="flex items-center gap-2 min-w-0">
        <Avatar
          src={post.author.avatar}
          did={post.author.did}
          alt={post.author.displayName || post.author.handle}
          size="xs"
        />
        <div className="flex items-baseline gap-1.5 min-w-0 flex-1">
          <span className="font-semibold text-surface-900 dark:text-white text-sm truncate min-w-0">
            {post.author.displayName ||
              displayHandle(post.author.handle, post.author.did)}
          </span>
          <span className="text-surface-400 dark:text-surface-500 text-xs truncate min-w-0">
            @{displayHandle(post.author.handle, post.author.did)}
          </span>
          {timestamp && (
            <span className="text-surface-400 dark:text-surface-500 text-xs shrink-0">
              · {timestamp}
            </span>
          )}
        </div>
        <span className="inline-flex items-center gap-1 text-[11px] font-medium text-surface-400 dark:text-surface-500 shrink-0 group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
          <ExternalLink size={11} />
          {host}
        </span>
      </div>

      {post.text && (
        <p className="mt-1.5 text-surface-900 dark:text-surface-50 whitespace-pre-wrap break-words leading-normal text-[15px]">
          <PostText
            text={post.text}
            facets={post.facets}
            host={host}
            onOpen={onOpen}
          />
        </p>
      )}

      {post.images.length > 0 && (
        <div
          className={clsx(
            "mt-2.5 gap-0.5 rounded-lg overflow-hidden border border-surface-200/60 dark:border-surface-700/60",
            post.images.length === 1 ? "block" : "grid grid-cols-2",
          )}
        >
          {post.images.slice(0, 4).map((img, i) => (
            <img
              key={i}
              src={img.thumb}
              alt={img.alt || ""}
              loading="lazy"
              className={clsx(
                "w-full object-cover bg-surface-200 dark:bg-surface-700",
                post.images.length === 1 ? "max-h-80" : "aspect-video h-full",
              )}
            />
          ))}
        </div>
      )}

      {post.video && (
        <div className="mt-2.5 relative rounded-lg overflow-hidden bg-surface-900 border border-surface-200/60 dark:border-surface-700/60">
          {post.video.thumbnail && !videoThumbError ? (
            <img
              src={post.video.thumbnail}
              alt=""
              loading="lazy"
              className="w-full max-h-72 object-cover opacity-90"
              onError={() => setVideoThumbError(true)}
            />
          ) : (
            <div className="w-full aspect-video" />
          )}
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="w-10 h-10 rounded-full bg-black/60 flex items-center justify-center">
              <Play size={18} className="text-white fill-current ml-0.5" />
            </div>
          </div>
        </div>
      )}

      {post.external && (
        <div className="mt-2.5 flex items-center gap-2.5 rounded-lg border border-surface-200 dark:border-surface-700 bg-white/50 dark:bg-surface-900/30 p-2.5 min-w-0">
          {post.external.thumb && (
            <img
              src={post.external.thumb}
              alt=""
              loading="lazy"
              className="w-10 h-10 rounded object-cover shrink-0 bg-surface-200 dark:bg-surface-700"
            />
          )}
          <div className="min-w-0">
            <p className="text-xs font-medium text-surface-900 dark:text-white truncate">
              {post.external.title || post.external.uri}
            </p>
            <p className="text-[11px] text-surface-400 dark:text-surface-500 truncate">
              {(() => {
                try {
                  return new URL(post.external.uri).hostname.replace(
                    /^www\./,
                    "",
                  );
                } catch {
                  return post.external.uri;
                }
              })()}
            </p>
          </div>
        </div>
      )}

      {post.quote && (
        <div className="mt-2.5 rounded-lg border border-surface-200 dark:border-surface-700 bg-white/50 dark:bg-surface-900/30 p-2.5">
          <div className="flex items-center gap-1.5 min-w-0">
            <Avatar
              src={post.quote.author.avatar}
              did={post.quote.author.did}
              alt={post.quote.author.displayName || post.quote.author.handle}
              size="xs"
            />
            <span className="font-semibold text-surface-900 dark:text-white text-xs truncate">
              {post.quote.author.displayName ||
                displayHandle(post.quote.author.handle, post.quote.author.did)}
            </span>
            <span className="text-surface-400 dark:text-surface-500 text-[11px] truncate">
              @{displayHandle(post.quote.author.handle, post.quote.author.did)}
            </span>
          </div>
          {post.quote.text && (
            <p className="mt-1.5 text-surface-600 dark:text-surface-300 text-[13px] leading-normal whitespace-pre-wrap break-words line-clamp-4">
              <PostText
                text={post.quote.text}
                facets={post.quote.facets}
                host={host}
                onOpen={onOpen}
              />
            </p>
          )}
        </div>
      )}

      {counts.length > 0 && (
        <div className="mt-2.5 flex items-center gap-5 text-surface-400 dark:text-surface-500">
          {counts.map(({ icon: Icon, value }, i) => (
            <span
              key={i}
              className="inline-flex items-center gap-1.5 text-xs tabular-nums"
            >
              <Icon size={13} />
              {value}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
