import { useStore } from "@nanostores/react";
import {
  Bell,
  Bookmark,
  Folder,
  Highlighter,
  Home,
  LogOut,
  MessageSquareText,
  PenSquare,
  Search,
  Settings,
  User,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";
import { getUnreadNotificationCount } from "../../api/client";
import { $user, logout } from "../../store/auth";
import { AppleIcon } from "../common/Icons";
import { useTranslation } from "react-i18next";

interface MobileNavProps {
  currentPath?: string;
  onNavigate?: (path: string) => void;
}

export default function MobileNav({
  currentPath: initialPath,
  onNavigate,
}: MobileNavProps) {
  const { t } = useTranslation();
  const user = useStore($user);
  const [currentPath, setCurrentPath] = useState(initialPath || "/");
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);

  const isAuthenticated = !!user;

  const isActive = (path: string) => {
    if (path === "/") return currentPath === "/";
    return currentPath.startsWith(path);
  };

  useEffect(() => {
    if (isAuthenticated) {
      getUnreadNotificationCount()
        .then((count) => setUnreadCount(count || 0))
        .catch(() => {});
    }
  }, [isAuthenticated]);

  const closeMenu = () => setIsMenuOpen(false);

  return (
    <>
      {isMenuOpen && (
        <div
          className="fixed inset-0 bg-black/40 z-40 md:hidden"
          onClick={closeMenu}
        />
      )}

      {isMenuOpen && (
        <div
          className="fixed left-0 right-0 z-50 md:hidden animate-slide-up"
          style={{ bottom: "calc(3.5rem + env(safe-area-inset-bottom))" }}
        >
          <div className="mx-2 mb-2 bg-white dark:bg-surface-900 rounded-2xl shadow-xl border border-surface-200 dark:border-surface-700 overflow-hidden">
            <div className="flex justify-center pt-3 pb-1">
              <div className="w-8 h-1 bg-surface-200 dark:bg-surface-600 rounded-full" />
            </div>

            <div className="p-2">
              {isAuthenticated && user ? (
                <>
                  <a
                    href={`/profile/${user.did}`}
                    className="flex items-center gap-3 p-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors"
                    onClick={(e) => {
                      if (onNavigate) {
                        e.preventDefault();
                        onNavigate(`/profile/${user.did}`);
                      }
                      closeMenu();
                    }}
                  >
                    {user.avatar ? (
                      <img
                        src={user.avatar}
                        alt=""
                        className="w-9 h-9 rounded-full object-cover shrink-0"
                      />
                    ) : (
                      <div className="w-9 h-9 rounded-full bg-surface-100 dark:bg-surface-700 flex items-center justify-center shrink-0">
                        <User size={16} className="text-surface-500" />
                      </div>
                    )}
                    <div className="flex flex-col min-w-0">
                      <span className="font-semibold text-surface-900 dark:text-white text-sm truncate">
                        {user.displayName || user.handle}
                      </span>
                      <span className="text-xs text-surface-400 dark:text-surface-500 truncate">
                        @{user.handle}
                      </span>
                    </div>
                  </a>

                  <div className="h-px bg-surface-100 dark:bg-surface-700 my-1 mx-3" />

                  <div className="grid grid-cols-2 gap-1">
                    {[
                      {
                        href: "/annotations",
                        icon: MessageSquareText,
                        label: t("nav.annotations"),
                      },
                      {
                        href: "/highlights",
                        icon: Highlighter,
                        label: t("nav.highlights"),
                      },
                      {
                        href: "/bookmarks",
                        icon: Bookmark,
                        label: t("nav.bookmarks"),
                      },
                      {
                        href: "/collections",
                        icon: Folder,
                        label: t("nav.collections"),
                      },
                    ].map(({ href, icon: Icon, label }) => (
                      <a
                        key={href}
                        href={href}
                        className="flex items-center gap-2.5 p-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors text-surface-700 dark:text-surface-200"
                        onClick={(e) => {
                          if (onNavigate) {
                            e.preventDefault();
                            onNavigate(href);
                          }
                          closeMenu();
                        }}
                      >
                        <Icon size={16} className="shrink-0" />
                        <span className="text-sm font-medium truncate">
                          {label}
                        </span>
                      </a>
                    ))}
                  </div>

                  <div className="h-px bg-surface-100 dark:bg-surface-700 my-1 mx-3" />

                  <div className="flex gap-1">
                    <a
                      href="/settings"
                      className="flex-1 flex items-center gap-2.5 p-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors text-surface-700 dark:text-surface-200"
                      onClick={(e) => {
                        if (onNavigate) {
                          e.preventDefault();
                          onNavigate("/settings");
                        }
                        closeMenu();
                      }}
                    >
                      <Settings size={16} className="shrink-0" />
                      <span className="text-sm font-medium">
                        {t("nav.settings")}
                      </span>
                    </a>

                    <a
                      href="https://www.icloud.com/shortcuts/1e33ebf52f55431fae1e187cfe9738c3"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex-1 flex items-center gap-2.5 p-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors text-surface-700 dark:text-surface-200"
                      onClick={closeMenu}
                    >
                      <AppleIcon size={16} />
                      <span className="text-sm font-medium">
                        {t("mobileNav.iosShortcut")}
                      </span>
                    </a>
                  </div>

                  <div className="h-px bg-surface-100 dark:bg-surface-700 my-1 mx-3" />

                  <button
                    className="w-full flex items-center gap-2.5 p-3 rounded-xl hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors text-red-500 dark:text-red-400"
                    onClick={() => {
                      logout();
                      closeMenu();
                    }}
                  >
                    <LogOut size={16} className="shrink-0" />
                    <span className="text-sm font-medium">
                      {t("nav.logOut")}
                    </span>
                  </button>
                </>
              ) : (
                <>
                  <a
                    href="/login"
                    className="flex items-center gap-2.5 p-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors text-surface-700 dark:text-surface-200"
                    onClick={closeMenu}
                  >
                    <User size={16} className="shrink-0" />
                    <span className="text-sm font-medium">
                      {t("nav.signIn")}
                    </span>
                  </a>
                  {[
                    {
                      href: "/collections",
                      icon: Folder,
                      label: t("nav.collections"),
                    },
                    {
                      href: "/settings",
                      icon: Settings,
                      label: t("nav.settings"),
                    },
                  ].map(({ href, icon: Icon, label }) => (
                    <a
                      key={href}
                      href={href}
                      className="flex items-center gap-2.5 p-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors text-surface-700 dark:text-surface-200"
                      onClick={(e) => {
                        if (onNavigate) {
                          e.preventDefault();
                          onNavigate(href);
                        }
                        closeMenu();
                      }}
                    >
                      <Icon size={16} className="shrink-0" />
                      <span className="text-sm font-medium">{label}</span>
                    </a>
                  ))}

                  <div className="h-px bg-surface-100 dark:bg-surface-700 my-1 mx-3" />

                  <a
                    href="https://www.icloud.com/shortcuts/1e33ebf52f55431fae1e187cfe9738c3"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-2.5 p-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors text-surface-700 dark:text-surface-200"
                    onClick={closeMenu}
                  >
                    <AppleIcon size={16} />
                    <span className="text-sm font-medium">
                      {t("mobileNav.iosShortcut")}
                    </span>
                  </a>
                </>
              )}
            </div>
          </div>
        </div>
      )}

      <nav
        className="fixed bottom-0 left-0 right-0 bg-white/95 dark:bg-surface-900/95 backdrop-blur-md border-t border-surface-200 dark:border-surface-800 flex items-center justify-around z-50 md:hidden"
        style={{
          height: "calc(3.5rem + env(safe-area-inset-bottom))",
          paddingBottom: "env(safe-area-inset-bottom)",
        }}
      >
        <a
          href="/home"
          className="flex flex-col items-center justify-center w-14 h-14 gap-0.5 transition-colors"
          onClick={(e) => {
            if (onNavigate) {
              e.preventDefault();
              onNavigate("/home");
            }
            setCurrentPath("/home");
            closeMenu();
          }}
        >
          <div
            className={`p-2 rounded-xl transition-colors ${
              isActive("/home")
                ? "bg-primary-50 dark:bg-primary-950/50 text-primary-600 dark:text-primary-400"
                : "text-surface-400 dark:text-surface-500"
            }`}
          >
            <Home size={22} strokeWidth={isActive("/home") ? 2 : 1.5} />
          </div>
        </a>

        <a
          href="/search"
          className="flex flex-col items-center justify-center w-14 h-14 gap-0.5 transition-colors"
          onClick={(e) => {
            if (onNavigate) {
              e.preventDefault();
              onNavigate("/search");
            }
            setCurrentPath("/search");
            closeMenu();
          }}
        >
          <div
            className={`p-2 rounded-xl transition-colors ${
              isActive("/search")
                ? "bg-primary-50 dark:bg-primary-950/50 text-primary-600 dark:text-primary-400"
                : "text-surface-400 dark:text-surface-500"
            }`}
          >
            <Search size={22} strokeWidth={isActive("/search") ? 2 : 1.5} />
          </div>
        </a>

        {isAuthenticated ? (
          <a
            href="/new"
            className="flex items-center justify-center w-11 h-11 rounded-2xl bg-primary-600 dark:bg-primary-600 text-white shadow-md active:scale-95 transition-transform"
            onClick={(e) => {
              if (onNavigate) {
                e.preventDefault();
                onNavigate("/new");
              }
              setCurrentPath("/new");
              closeMenu();
            }}
          >
            <PenSquare size={18} strokeWidth={2} />
          </a>
        ) : (
          <a
            href="/login"
            className="flex items-center justify-center w-11 h-11 rounded-2xl bg-primary-600 text-white shadow-md active:scale-95 transition-transform"
            onClick={closeMenu}
          >
            <User size={18} strokeWidth={2} />
          </a>
        )}

        {isAuthenticated ? (
          <a
            href="/notifications"
            className="flex flex-col items-center justify-center w-14 h-14 gap-0.5 relative transition-colors"
            onClick={(e) => {
              if (onNavigate) {
                e.preventDefault();
                onNavigate("/notifications");
              }
              setCurrentPath("/notifications");
              closeMenu();
            }}
          >
            <div
              className={`p-2 rounded-xl transition-colors relative ${
                isActive("/notifications")
                  ? "bg-primary-50 dark:bg-primary-950/50 text-primary-600 dark:text-primary-400"
                  : "text-surface-400 dark:text-surface-500"
              }`}
            >
              <Bell
                size={22}
                strokeWidth={isActive("/notifications") ? 2 : 1.5}
              />
              {unreadCount > 0 && (
                <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-red-500 rounded-full ring-2 ring-white dark:ring-surface-900" />
              )}
            </div>
          </a>
        ) : (
          <div className="w-14" />
        )}

        <button
          className="flex flex-col items-center justify-center w-14 h-14 gap-0.5 transition-colors"
          onClick={() => setIsMenuOpen(!isMenuOpen)}
        >
          <div
            className={`p-2 rounded-xl transition-colors ${
              isMenuOpen
                ? "bg-surface-100 dark:bg-surface-700 text-surface-700 dark:text-surface-200"
                : "text-surface-400 dark:text-surface-500"
            }`}
          >
            {isMenuOpen ? (
              <X size={22} strokeWidth={1.5} />
            ) : (
              <svg
                width="22"
                height="22"
                viewBox="0 0 22 22"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <circle cx="4" cy="11" r="1.75" fill="currentColor" />
                <circle cx="11" cy="11" r="1.75" fill="currentColor" />
                <circle cx="18" cy="11" r="1.75" fill="currentColor" />
              </svg>
            )}
          </div>
        </button>
      </nav>
    </>
  );
}
