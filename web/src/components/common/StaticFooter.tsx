import { useStore } from "@nanostores/react";
import { useTranslation } from "react-i18next";
import { Sun, Moon, Monitor } from "lucide-react";
import { $theme, cycleTheme } from "../../store/theme";

export default function StaticFooter() {
  const { t } = useTranslation();
  const theme = useStore($theme);

  const themeLabel =
    theme === "light"
      ? t("nav.themeLight")
      : theme === "dark"
        ? t("nav.themeDark")
        : t("nav.themeSystem");

  return (
    <footer className="border-t border-surface-200 dark:border-surface-800">
      <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-8 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div className="flex items-center gap-2.5 text-surface-500 dark:text-surface-400">
          <svg
            viewBox="0 0 180 180"
            fill="currentColor"
            aria-hidden="true"
            className="w-4 h-4"
          >
            <path d="M0,60 L0,0 L60,0 L60,20 L20,20 L20,60 Z" />
            <path d="M120,0 L180,0 L180,60 L160,60 L160,20 L120,20 Z" />
            <path d="M0,120 L20,120 L20,160 L60,160 L60,180 L0,180 Z" />
            <path d="M160,120 L180,120 L180,180 L120,180 L120,160 L160,160 Z" />
            <rect x="80" y="62" width="20" height="56" />
            <rect x="62" y="80" width="56" height="20" />
          </svg>
          <span className="text-[13px]">{t("sidebar.copyright")}</span>
        </div>
        <div className="flex items-center gap-5 text-[13px] text-surface-500 dark:text-surface-400 flex-wrap">
          <a
            href="/privacy"
            className="hover:text-surface-900 dark:hover:text-white transition-colors"
          >
            {t("about.footer.privacy")}
          </a>
          <a
            href="/terms"
            className="hover:text-surface-900 dark:hover:text-white transition-colors"
          >
            {t("about.footer.terms")}
          </a>
          <a
            href="/brand"
            className="hover:text-surface-900 dark:hover:text-white transition-colors"
          >
            Brand
          </a>
          <a
            href="/imprint"
            className="hover:text-surface-900 dark:hover:text-white transition-colors"
          >
            Imprint
          </a>
          <a
            href="mailto:hello@margin.at"
            className="hover:text-surface-900 dark:hover:text-white transition-colors"
          >
            {t("about.footer.contact")}
          </a>
          <button
            onClick={cycleTheme}
            title={themeLabel}
            aria-label={themeLabel}
            className="inline-flex items-center gap-1.5 hover:text-surface-900 dark:hover:text-white transition-colors"
          >
            {theme === "light" ? (
              <Sun size={14} />
            ) : theme === "dark" ? (
              <Moon size={14} />
            ) : (
              <Monitor size={14} />
            )}
          </button>
        </div>
      </div>
    </footer>
  );
}
