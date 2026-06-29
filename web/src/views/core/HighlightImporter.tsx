import {
  AlertCircle,
  CheckCircle2,
  Download,
  Loader2,
  Upload,
} from "lucide-react";
import type React from "react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createHighlight } from "../../api/client";
import type { TextQuoteSelector } from "../../types";
import { analytics } from "../../lib/analytics";

interface Highlight {
  url: string;
  text: string;
  title?: string;
  tags?: string[];
  color?: string;
  created_at?: string;
  note?: string;
}

interface ImportProgress {
  total: number;
  completed: number;
  failed: number;
  errors: { row: number; error: string }[];
}

export function HighlightImporter() {
  const { t } = useTranslation();
  const [progress, setProgress] = useState<ImportProgress | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const parseCSV = (csv: string): Highlight[] => {
    const lines = csv.split("\n");
    if (lines.length === 0) return [];

    // Parse header (case-insensitive)
    const header = lines[0].split(",").map((h) => h.trim().toLowerCase());

    // Find required columns (flexible matching)
    const urlIdx = header.findIndex((h) => h === "url" || h === "source");
    const textIdx = header.findIndex(
      (h) => h === "text" || h === "highlight" || h === "excerpt",
    );

    // Find optional columns
    const titleIdx = header.findIndex(
      (h) => h === "title" || h === "article_title",
    );
    const tagsIdx = header.findIndex((h) => h === "tags" || h === "tag");
    const colorIdx = header.findIndex(
      (h) => h === "color" || h === "highlight_color",
    );
    const createdAtIdx = header.findIndex(
      (h) => h === "created_at" || h === "date" || h === "date_highlighted",
    );
    const noteIdx = header.findIndex(
      (h) => h === "note" || h === "notes" || h === "comment",
    );

    // Validate required columns
    if (urlIdx === -1) {
      throw new Error("CSV must have a 'url' column");
    }
    if (textIdx === -1) {
      throw new Error(
        "CSV must have a 'text' column (also matches: highlight, excerpt)",
      );
    }

    const highlights: Highlight[] = [];

    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim();
      if (!line) continue;

      const cells = parseCSVLine(line);

      const url = cells[urlIdx]?.trim() || "";
      const text = cells[textIdx]?.trim() || "";

      if (url && text) {
        const highlight: Highlight = {
          url,
          text,
          title: titleIdx >= 0 ? cells[titleIdx]?.trim() : undefined,
          tags: tagsIdx >= 0 ? parseTags(cells[tagsIdx]) : undefined,
          color:
            colorIdx >= 0 ? validateColor(cells[colorIdx]?.trim()) : "yellow",
          created_at:
            createdAtIdx >= 0 ? cells[createdAtIdx]?.trim() : undefined,
          note: noteIdx >= 0 ? cells[noteIdx]?.trim() : undefined,
        };
        highlights.push(highlight);
      }
    }

    return highlights;
  };

  const validateColor = (color?: string): string => {
    if (!color) return "yellow";
    const valid = ["yellow", "blue", "green", "red", "orange", "purple"];
    return valid.includes(color.toLowerCase()) ? color.toLowerCase() : "yellow";
  };

  const parseCSVLine = (line: string): string[] => {
    const result: string[] = [];
    let current = "";
    let inQuotes = false;

    for (let i = 0; i < line.length; i++) {
      const char = line[i];
      const nextChar = line[i + 1];

      if (char === '"') {
        if (inQuotes && nextChar === '"') {
          current += '"';
          i++;
        } else {
          inQuotes = !inQuotes;
        }
      } else if (char === "," && !inQuotes) {
        result.push(current);
        current = "";
      } else {
        current += char;
      }
    }

    result.push(current);
    return result;
  };

  const parseTags = (tagString: string): string[] => {
    if (!tagString) return [];
    return tagString
      .split(/[,;]/)
      .map((t) => t.trim())
      .filter((t) => t.length > 0)
      .slice(0, 10); // Max 10 tags per highlight
  };

  const downloadTemplate = () => {
    const template = `url,text,title,tags,color,created_at
https://example.com,"Highlight text here","Page Title","tag1;tag2",yellow,2024-01-15T10:30:00Z
https://blog.example.com,"Another highlight","Article Title","reading",blue,2024-01-16T14:20:00Z`;

    const blob = new Blob([template], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "highlights-template.csv";
    a.click();
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      setIsImporting(true);
      const csv = await file.text();
      const highlights = parseCSV(csv);

      if (highlights.length === 0) {
        alert(t("highlightImporter.noHighlights"));
        setIsImporting(false);
        return;
      }

      // Start import
      const importState: ImportProgress = {
        total: highlights.length,
        completed: 0,
        failed: 0,
        errors: [],
      };

      setProgress(importState);

      // Import with rate limiting (1 per 500ms to avoid overload)
      for (let i = 0; i < highlights.length; i++) {
        const h = highlights[i];

        try {
          const selector: TextQuoteSelector = {
            type: "TextQuoteSelector",
            exact: h.text.substring(0, 5000), // Max 5000 chars
          };

          await createHighlight({
            url: h.url,
            selector,
            color: h.color || "yellow",
            tags: h.tags,
            title: h.title,
          });

          importState.completed++;
        } catch (error) {
          importState.failed++;
          importState.errors.push({
            row: i + 2, // +2 for header row + 0-indexing
            error: error instanceof Error ? error.message : "Unknown error",
          });
        }

        setProgress({ ...importState });

        // Rate limiting
        await new Promise((resolve) => setTimeout(resolve, 500));
      }

      analytics.capture("highlights_imported", {
        total: importState.total,
        completed: importState.completed,
        failed: importState.failed,
      });
      setIsImporting(false);
    } catch (error) {
      analytics.captureException(error);
      alert(
        t("highlightImporter.errorParsing", {
          message: error instanceof Error ? error.message : "Unknown error",
        }),
      );
      setIsImporting(false);
    }

    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  if (!progress) {
    return (
      <div className="w-full space-y-3">
        <label className="flex items-center justify-center w-full px-4 py-8 border-2 border-dashed border-surface-300 dark:border-surface-600 rounded-lg cursor-pointer hover:border-surface-400 dark:hover:border-surface-500 transition">
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv"
            onChange={handleFileSelect}
            disabled={isImporting}
            className="hidden"
          />
          <div className="flex flex-col items-center gap-2">
            <Upload className="w-6 h-6 text-surface-500 dark:text-surface-400" />
            <span className="text-sm font-medium text-surface-700 dark:text-surface-300">
              {isImporting
                ? t("highlightImporter.processing")
                : t("highlightImporter.clickToUpload")}
            </span>
            <span className="text-xs text-surface-500 dark:text-surface-400">
              {t("highlightImporter.requiredColumns")}
            </span>
          </div>
        </label>

        <button
          type="button"
          onClick={downloadTemplate}
          className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium text-surface-700 dark:text-surface-300 bg-surface-100 dark:bg-surface-700 hover:bg-surface-200 dark:hover:bg-surface-600 rounded-lg transition"
        >
          <Download size={16} />
          {t("highlightImporter.downloadTemplate")}
        </button>
      </div>
    );
  }

  const successRate =
    progress.total > 0
      ? ((progress.completed / progress.total) * 100).toFixed(1)
      : "0";

  return (
    <div className="w-full space-y-4">
      <div className="p-4 bg-surface-50 dark:bg-surface-800 rounded-lg">
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-surface-700 dark:text-surface-300">
              {t("highlightImporter.importProgress")}
            </span>
            <span className="text-sm text-surface-500 dark:text-surface-400">
              {progress.completed} / {progress.total}
            </span>
          </div>

          <div className="w-full bg-surface-200 dark:bg-surface-700 rounded-full h-2">
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style={{
                width: `${(progress.completed / progress.total) * 100}%`,
              }}
            />
          </div>

          <div className="flex items-center justify-between text-xs text-surface-600 dark:text-surface-400">
            <span>
              {t("highlightImporter.complete", { rate: successRate })}
            </span>
            {progress.failed > 0 && (
              <span className="text-red-500">
                {t("highlightImporter.failed", { count: progress.failed })}
              </span>
            )}
          </div>
        </div>
      </div>

      {isImporting && (
        <div className="flex items-center justify-center gap-2 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
          <span className="text-sm text-blue-700 dark:text-blue-300">
            {t("highlightImporter.importing")}
          </span>
        </div>
      )}

      {!isImporting &&
        progress.failed === 0 &&
        progress.completed === progress.total && (
          <div className="flex items-center justify-center gap-2 p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
            <CheckCircle2 className="w-4 h-4 text-green-600 dark:text-green-400" />
            <span className="text-sm text-green-700 dark:text-green-300">
              {t("highlightImporter.success", { count: progress.completed })}
            </span>
          </div>
        )}

      {progress.errors.length > 0 && (
        <div className="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
          <div className="flex items-start gap-2 mb-2">
            <AlertCircle className="w-4 h-4 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-700 dark:text-red-300">
                {t("highlightImporter.errorsTitle", {
                  count: progress.errors.length,
                })}
              </p>
              <ul className="mt-2 space-y-1">
                {progress.errors.slice(0, 5).map((err, idx) => (
                  <li
                    key={idx}
                    className="text-xs text-red-600 dark:text-red-400"
                  >
                    {t("highlightImporter.row", {
                      row: err.row,
                      error: err.error,
                    })}
                  </li>
                ))}
                {progress.errors.length > 5 && (
                  <li className="text-xs text-red-600 dark:text-red-400">
                    {t("highlightImporter.moreErrors", {
                      count: progress.errors.length - 5,
                    })}
                  </li>
                )}
              </ul>
            </div>
          </div>
        </div>
      )}

      {!isImporting && (
        <button
          onClick={() => setProgress(null)}
          className="w-full px-4 py-2 text-sm font-medium bg-surface-200 dark:bg-surface-700 hover:bg-surface-300 dark:hover:bg-surface-600 rounded-lg transition"
        >
          {t("highlightImporter.importAnother")}
        </button>
      )}
    </div>
  );
}
