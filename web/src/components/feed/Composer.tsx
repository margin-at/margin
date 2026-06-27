import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  createAnnotation,
  createHighlight,
  sessionAtom,
  getUserTags,
  getTrendingTags,
} from "../../api/client";
import type { Selector, ContentLabelValue } from "../../types";
import { asTextQuote } from "../../types";
import { X, ShieldAlert, Highlighter, PenLine } from "lucide-react";
import TagInput from "../ui/TagInput";
import { analytics } from "../../lib/analytics";

const SELF_LABEL_VALUES: ContentLabelValue[] = [
  "sexual",
  "nudity",
  "violence",
  "gore",
  "spam",
  "misleading",
];

interface ComposerProps {
  url: string;
  selector?: Selector | null;
  onSuccess?: () => void;
  onCancel?: () => void;
}

export default function Composer({
  url,
  selector: initialSelector,
  onSuccess,
  onCancel,
}: ComposerProps) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [quoteText, setQuoteText] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [tagSuggestions, setTagSuggestions] = useState<string[]>([]);
  const [selector, setSelector] = useState(initialSelector);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showQuoteInput, setShowQuoteInput] = useState(false);
  const [selfLabels, setSelfLabels] = useState<ContentLabelValue[]>([]);
  const [showLabelPicker, setShowLabelPicker] = useState(false);

  useEffect(() => {
    const session = sessionAtom.get();
    if (session?.did) {
      Promise.all([
        getUserTags(session.did).catch(() => [] as string[]),
        getTrendingTags(50)
          .then((tags) => tags.map((t) => t.tag))
          .catch(() => [] as string[]),
      ]).then(([userTags, trendingTags]) => {
        const seen = new Set(userTags);
        const merged = [...userTags];
        for (const t of trendingTags) {
          if (!seen.has(t)) {
            merged.push(t);
            seen.add(t);
          }
        }
        setTagSuggestions(merged);
      });
    }
  }, []);

  const highlightedText =
    selector?.type === "TextQuoteSelector" ? selector.exact : null;

  const hasQuote = !!(highlightedText || quoteText.trim());
  const hasText = !!text.trim();
  const mode: "highlight" | "annotation" | "note" =
    hasQuote && !hasText ? "highlight" : hasQuote ? "annotation" : "note";

  const modeCopy = {
    highlight: {
      title: t("composer.newHighlight"),
      icon: Highlighter,
      submit: t("composer.saveHighlight"),
      hint: t("composer.highlightHint"),
    },
    annotation: {
      title: t("composer.newAnnotation"),
      icon: PenLine,
      submit: t("composer.postAnnotation"),
      hint: null,
    },
    note: {
      title: t("composer.newNote"),
      icon: PenLine,
      submit: t("composer.postNote"),
      hint: null,
    },
  }[mode];
  const ModeIcon = modeCopy.icon;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!text.trim() && !highlightedText && !quoteText.trim()) return;

    try {
      setLoading(true);
      setError(null);

      let finalSelector = selector;
      if (!finalSelector && quoteText.trim()) {
        finalSelector = {
          type: "TextQuoteSelector",
          exact: quoteText.trim(),
        };
      }

      const quoteSelector = asTextQuote(finalSelector);
      const tagList = tags.filter(Boolean);

      if (!text.trim()) {
        if (!quoteSelector) throw new Error("No text selected");
        await createHighlight({
          url,
          selector: quoteSelector,
          color: "yellow",
          tags: tagList,
          labels: selfLabels.length > 0 ? selfLabels : undefined,
        });
        analytics.capture("highlight_created", {
          url,
          tag_count: tagList.length,
          has_color: true,
          has_labels: selfLabels.length > 0,
        });
      } else {
        await createAnnotation({
          url,
          text: text.trim(),
          selector: quoteSelector,
          tags: tagList,
          labels: selfLabels.length > 0 ? selfLabels : undefined,
        });
        analytics.capture("annotation_created", {
          url,
          has_quote: !!quoteSelector,
          tag_count: tagList.length,
          has_labels: selfLabels.length > 0,
        });
      }

      setText("");
      setQuoteText("");
      setTags([]);
      setSelector(null);
      if (onSuccess) onSuccess();
    } catch (err) {
      analytics.captureException(err);
      setError(
        (err instanceof Error ? err.message : "Unknown error") ||
          "Failed to post",
      );
    } finally {
      setLoading(false);
    }
  };

  const handleRemoveSelector = () => {
    setSelector(null);
    setQuoteText("");
    setShowQuoteInput(false);
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-2">
        <h3 className="flex items-center gap-2 text-lg font-bold text-surface-900 dark:text-white shrink-0">
          <ModeIcon
            size={18}
            className={
              mode === "highlight"
                ? "text-amber-500"
                : "text-primary-500 dark:text-primary-400"
            }
          />
          {modeCopy.title}
        </h3>
        {url && (
          <div className="text-xs text-surface-400 dark:text-surface-500 truncate min-w-0 pt-1 text-right">
            {(() => {
              try {
                return new URL(url).hostname.replace(/^www\./, "");
              } catch {
                return url;
              }
            })()}
          </div>
        )}
      </div>

      {modeCopy.hint && (
        <div className="text-xs text-surface-500 dark:text-surface-400 bg-amber-50 dark:bg-amber-900/10 border border-amber-200 dark:border-amber-900/30 rounded-lg px-3 py-2">
          {modeCopy.hint}
        </div>
      )}

      {highlightedText && (
        <div className="relative p-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-lg">
          <button
            type="button"
            className="absolute top-2 right-2 text-surface-400 dark:text-surface-500 hover:text-surface-600 dark:hover:text-surface-300"
            onClick={handleRemoveSelector}
          >
            <X size={16} />
          </button>
          <blockquote className="italic text-surface-600 dark:text-surface-300 border-l-2 border-primary-400 dark:border-primary-500 pl-3 text-sm">
            "{highlightedText}"
          </blockquote>
        </div>
      )}

      {!highlightedText && (
        <>
          {!showQuoteInput ? (
            <button
              type="button"
              className="text-left text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 font-medium py-1"
              onClick={() => setShowQuoteInput(true)}
            >
              {t("composer.addQuote")}
            </button>
          ) : (
            <div className="flex flex-col gap-2">
              <textarea
                value={quoteText}
                onChange={(e) => setQuoteText(e.target.value)}
                placeholder={t("composer.quotePlaceholder")}
                className="w-full text-sm p-3 bg-surface-50 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-lg text-surface-900 dark:text-white placeholder:text-surface-400 dark:placeholder:text-surface-500 focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 dark:focus:border-primary-400 outline-none"
                rows={2}
              />
              <div className="flex justify-end">
                <button
                  type="button"
                  className="text-xs text-red-500 dark:text-red-400 font-medium"
                  onClick={handleRemoveSelector}
                >
                  {t("composer.removeQuote")}
                </button>
              </div>
            </div>
          )}
        </>
      )}

      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder={
          hasQuote
            ? t("composer.thoughtsPlaceholder")
            : t("composer.mindPlaceholder")
        }
        className="w-full p-3 bg-white dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-lg text-surface-900 dark:text-white placeholder:text-surface-400 dark:placeholder:text-surface-500 focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 dark:focus:border-primary-400 outline-none min-h-[100px] resize-none"
        maxLength={3000}
        disabled={loading}
      />

      <TagInput
        tags={tags}
        onChange={setTags}
        suggestions={tagSuggestions}
        placeholder={t("composer.tagsPlaceholder")}
        disabled={loading}
      />

      <div>
        <button
          type="button"
          onClick={() => setShowLabelPicker(!showLabelPicker)}
          className="flex items-center gap-1.5 text-sm text-surface-500 dark:text-surface-400 hover:text-surface-700 dark:hover:text-surface-200 transition-colors"
        >
          <ShieldAlert size={14} />
          <span>
            {selfLabels.length > 0
              ? t("composer.contentWarningCount", { count: selfLabels.length })
              : t("composer.contentWarning")}
          </span>
        </button>

        {showLabelPicker && (
          <div className="mt-2 flex flex-wrap gap-1.5">
            {SELF_LABEL_VALUES.map((value) => (
              <button
                key={value}
                type="button"
                onClick={() =>
                  setSelfLabels((prev) =>
                    prev.includes(value)
                      ? prev.filter((v) => v !== value)
                      : [...prev, value],
                  )
                }
                className={`px-2.5 py-1 text-xs font-medium rounded-lg transition-all ${
                  selfLabels.includes(value)
                    ? "bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 ring-1 ring-amber-300 dark:ring-amber-700"
                    : "bg-surface-100 dark:bg-surface-800 text-surface-500 dark:text-surface-400 hover:bg-surface-200 dark:hover:bg-surface-700"
                }`}
              >
                {t(`composer.labels.${value}`)}
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="flex items-center justify-between pt-2 gap-2">
        <span
          className={
            text.length > 2900
              ? "text-red-500 dark:text-red-400 text-xs font-medium"
              : "text-surface-400 dark:text-surface-500 text-xs"
          }
        >
          {text.length}/3000
        </span>
        <div className="flex items-center gap-2 shrink-0">
          {onCancel && (
            <button
              type="button"
              className="text-sm font-medium text-surface-500 dark:text-surface-400 hover:text-surface-800 dark:hover:text-surface-200 px-3 py-1.5"
              onClick={onCancel}
              disabled={loading}
            >
              {t("composer.cancel")}
            </button>
          )}
          <button
            type="submit"
            className="bg-primary-600 hover:bg-primary-700 text-white font-medium px-4 py-1.5 rounded-lg transition-colors disabled:opacity-50 text-sm"
            disabled={
              loading || (!text.trim() && !highlightedText && !quoteText.trim())
            }
          >
            {loading ? "…" : modeCopy.submit}
          </button>
        </div>
      </div>

      {error && (
        <div className="text-red-500 dark:text-red-400 text-sm text-center bg-red-50 dark:bg-red-900/20 py-2 rounded-lg">
          {error}
        </div>
      )}
    </form>
  );
}
