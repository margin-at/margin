import React from "react";
import { formatDistanceToNow } from "date-fns";
import { useTranslation } from "react-i18next";
import { MessageSquare, Trash2, Reply } from "lucide-react";
import type { AnnotationItem, UserProfile } from "../../types";
import { getAvatarUrl } from "../../api/client";
import { displayHandle } from "../../lib/handle";
import { clsx } from "clsx";

interface ReplyListProps {
  replies: AnnotationItem[];
  rootUri: string;
  user: UserProfile | null;
  onReply: (reply: AnnotationItem) => void;
  onDelete: (reply: AnnotationItem) => void;
  isInline?: boolean;
}

interface ReplyItemProps {
  reply: AnnotationItem & { children?: AnnotationItem[] };
  depth: number;
  user: UserProfile | null;
  onReply: (reply: AnnotationItem) => void;
  onDelete: (reply: AnnotationItem) => void;
  isInline: boolean;
}

const ReplyItem: React.FC<ReplyItemProps> = ({
  reply,
  depth = 0,
  user,
  onReply,
  onDelete,
  isInline,
}) => {
  const author = reply.author || reply.creator || {};
  const isReplyOwner = user?.did && author.did === user.did;

  if (!author.handle && !author.did) return null;

  return (
    <div key={reply.uri || reply.id}>
      <div
        className={clsx(
          "relative mb-2 transition-colors",
          isInline ? "flex gap-3" : "rounded-lg",
          depth > 0 &&
            "ml-4 pl-3 border-l-2 border-surface-200 dark:border-surface-700",
        )}
      >
        {isInline ? (
          <>
            <a
              href={`/profile/${author.did || author.handle}`}
              className="shrink-0"
            >
              {getAvatarUrl(author.did, author.avatar) ? (
                <img
                  src={getAvatarUrl(author.did, author.avatar)}
                  alt=""
                  className={clsx(
                    "rounded-full object-cover bg-surface-200 dark:bg-surface-700",
                    depth > 0 ? "w-6 h-6" : "w-7 h-7",
                  )}
                />
              ) : (
                <div
                  className={clsx(
                    "rounded-full bg-surface-200 dark:bg-surface-700 flex items-center justify-center text-surface-500 dark:text-surface-400 font-bold",
                    depth > 0 ? "w-6 h-6 text-[10px]" : "w-7 h-7 text-xs",
                  )}
                >
                  {(author.displayName ||
                    author.handle ||
                    "?")[0]?.toUpperCase()}
                </div>
              )}
            </a>
            <div className="flex-1 min-w-0">
              <div className="flex items-baseline gap-2 mb-0.5 flex-wrap">
                <span
                  className={clsx(
                    "font-medium text-surface-900 dark:text-white",
                    depth > 0 ? "text-xs" : "text-sm",
                  )}
                >
                  {author.displayName ||
                    displayHandle(author.handle, author.did)}
                </span>
                <span className="text-surface-400 dark:text-surface-500 text-xs">
                  {reply.createdAt
                    ? formatDistanceToNow(new Date(reply.createdAt), {
                        addSuffix: false,
                      })
                    : ""}
                </span>

                <div className="ml-auto flex gap-2">
                  <button
                    onClick={() => onReply(reply)}
                    className="text-surface-400 dark:text-surface-500 hover:text-surface-600 dark:hover:text-surface-300 transition-colors flex items-center gap-1 text-[10px] uppercase font-medium"
                  >
                    <MessageSquare size={12} />
                  </button>
                  {isReplyOwner && (
                    <button
                      onClick={() => onDelete(reply)}
                      className="text-surface-400 dark:text-surface-500 hover:text-red-500 dark:hover:text-red-400 transition-colors"
                    >
                      <Trash2 size={12} />
                    </button>
                  )}
                </div>
              </div>
              <p
                className={clsx(
                  "text-surface-800 dark:text-surface-200 whitespace-pre-wrap break-words leading-relaxed",
                  depth > 0 ? "text-sm" : "text-sm",
                )}
              >
                {reply.text || reply.body?.value}
              </p>
            </div>
          </>
        ) : (
          <div className="p-3 bg-white dark:bg-surface-900 rounded-lg ring-1 ring-black/5 dark:ring-white/5">
            <div className="flex items-center gap-2 mb-2">
              <a
                href={`/profile/${author.did || author.handle}`}
                className="shrink-0"
              >
                {getAvatarUrl(author.did, author.avatar) ? (
                  <img
                    src={getAvatarUrl(author.did, author.avatar)}
                    alt=""
                    className="w-7 h-7 rounded-full object-cover bg-surface-200 dark:bg-surface-700"
                  />
                ) : (
                  <div className="w-7 h-7 rounded-full bg-surface-200 dark:bg-surface-700 flex items-center justify-center text-surface-500 dark:text-surface-400 font-bold text-xs">
                    {(author.displayName ||
                      author.handle ||
                      "?")[0]?.toUpperCase()}
                  </div>
                )}
              </a>
              <div className="flex flex-col">
                <span className="font-medium text-surface-900 dark:text-white text-sm">
                  {author.displayName ||
                    displayHandle(author.handle, author.did)}
                </span>
              </div>
              <span className="text-surface-400 dark:text-surface-500 text-xs ml-auto">
                {reply.createdAt
                  ? formatDistanceToNow(new Date(reply.createdAt), {
                      addSuffix: false,
                    })
                  : ""}
              </span>
            </div>
            <p className="text-surface-800 dark:text-surface-200 text-sm pl-9 mb-2 whitespace-pre-wrap break-words">
              {reply.text || reply.body?.value}
            </p>
            <div className="flex items-center justify-end gap-2 pl-9">
              <button
                onClick={() => onReply(reply)}
                className="text-surface-400 dark:text-surface-500 hover:text-primary-600 dark:hover:text-primary-400 transition-colors p-1"
              >
                <Reply size={14} />
              </button>
              {isReplyOwner && (
                <button
                  onClick={() => onDelete(reply)}
                  className="text-surface-400 dark:text-surface-500 hover:text-red-500 dark:hover:text-red-400 transition-colors p-1"
                >
                  <Trash2 size={14} />
                </button>
              )}
            </div>
          </div>
        )}
      </div>
      {reply.children && reply.children.length > 0 && (
        <div className="flex flex-col">
          {reply.children.map((child) => (
            <ReplyItem
              key={child.uri || child.id}
              reply={child}
              depth={depth + 1}
              user={user}
              onReply={onReply}
              onDelete={onDelete}
              isInline={isInline}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default function ReplyList({
  replies,
  rootUri,
  user,
  onReply,
  onDelete,
  isInline = false,
}: ReplyListProps) {
  const { t } = useTranslation();
  if (!replies || replies.length === 0) {
    return (
      <div className="py-8 text-center">
        <p className="text-surface-500 dark:text-surface-400 text-sm">
          {t("replyList.noReplies")}
        </p>
      </div>
    );
  }

  const buildReplyTree = () => {
    const replyMap: Record<
      string,
      AnnotationItem & { children: AnnotationItem[] }
    > = {};
    const rootReplies: (AnnotationItem & { children: AnnotationItem[] })[] = [];

    replies.forEach((r) => {
      replyMap[r.uri || r.id || ""] = { ...r, children: [] };
    });

    replies.forEach((r) => {
      const parentUri = r.reply?.parent?.uri || r.parentUri;
      if (parentUri === rootUri || !parentUri || !replyMap[parentUri]) {
        rootReplies.push(replyMap[r.uri || r.id || ""]);
      } else {
        replyMap[parentUri].children.push(replyMap[r.uri || r.id || ""]);
      }
    });

    return rootReplies;
  };

  const replyTree = buildReplyTree();

  return (
    <div className="flex flex-col gap-1">
      {replyTree.map((reply) => (
        <ReplyItem
          key={reply.uri || reply.id}
          reply={reply}
          depth={0}
          user={user}
          onReply={onReply}
          onDelete={onDelete}
          isInline={isInline}
        />
      ))}
    </div>
  );
}
