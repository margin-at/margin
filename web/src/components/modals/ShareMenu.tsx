import React, { useState, useRef, useEffect, useCallback } from "react";
import {
  Copy,
  ExternalLink,
  Check,
  Share2,
  MoreHorizontal,
  X,
} from "lucide-react";
import {
  AturiIcon,
  BlueskyIcon,
  MuIcon,
  BlackskyIcon,
  WitchskyIcon,
  CatskyIcon,
  DeerIcon,
} from "../common/Icons";
import { analytics } from "../../lib/analytics";
import { useTranslation } from "react-i18next";

const SembleLogo = () => (
  <img src="/semble-logo.svg" alt="Semble" className="w-4 h-4 opacity-90" />
);

interface ShareMenuProps {
  uri: string;
  text?: string;
  customUrl?: string;
  handle?: string;
  type?: string;
  url?: string;
}

export default function ShareMenu({
  uri,
  text,
  customUrl,
  handle,
  type,
  url,
}: ShareMenuProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const [isMobile, setIsMobile] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const sheetRef = useRef<HTMLDivElement>(null);
  const dragStartY = useRef(0);
  const dragCurrentY = useRef(0);

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    dragStartY.current = e.touches[0].clientY;
    if (sheetRef.current) sheetRef.current.style.transition = "none";
  }, []);

  const handleTouchMove = useCallback((e: React.TouchEvent) => {
    const delta = e.touches[0].clientY - dragStartY.current;
    dragCurrentY.current = delta;
    if (delta > 0 && sheetRef.current) {
      sheetRef.current.style.transform = `translateY(${delta}px)`;
    }
  }, []);

  const handleTouchEnd = useCallback(() => {
    if (sheetRef.current) {
      sheetRef.current.style.transition = "transform 0.3s ease";
      if (dragCurrentY.current > 100) {
        sheetRef.current.style.transform = "translateY(100%)";
        setTimeout(() => setIsOpen(false), 300);
      } else {
        sheetRef.current.style.transform = "translateY(0)";
      }
    }
    dragCurrentY.current = 0;
  }, []);
  const [menuPosition, setMenuPosition] = useState({
    top: 0,
    left: 0,
    alignRight: false,
  });

  useEffect(() => {
    const check = () => setIsMobile(window.innerWidth < 640);
    check();
    window.addEventListener("resize", check);
    return () => window.removeEventListener("resize", check);
  }, []);

  const getShareUrl = () => {
    if (customUrl) return customUrl;
    if (!uri) return "";

    const origin = typeof window !== "undefined" ? window.location.origin : "";
    const uriParts = uri.split("/");
    const rkey = uriParts[uriParts.length - 1];
    const did = uriParts[2];
    const collection = uriParts[3] ?? "";

    const marginSegment = collection.startsWith("at.margin.note")
      ? "note"
      : collection.startsWith("at.margin.highlight")
        ? "highlight"
        : collection.startsWith("at.margin.bookmark")
          ? "bookmark"
          : collection.startsWith("at.margin.annotation")
            ? "annotation"
            : null;

    if (marginSegment && handle) {
      return `${origin}/${handle}/${marginSegment}/${rkey}`;
    }

    if (did && collection && rkey) {
      return `${origin}/at/${did}/${collection}/${rkey}`;
    }

    return `${origin}/at/${did}/${rkey}`;
  };

  const shareUrl = getShareUrl();
  const isSemble = uri && uri.includes("network.cosmik");

  const sembleUrl = (() => {
    if (!isSemble) return "";
    const parts = (uri || "").split("/");
    const rkey = parts[parts.length - 1];
    const userHandle = handle || (parts.length > 2 ? parts[2] : "");

    if (uri.includes("network.cosmik.collection"))
      return `https://semble.so/profile/${userHandle}/collections/${rkey}`;
    if (uri.includes("network.cosmik.card") && url)
      return `https://semble.so/url?id=${encodeURIComponent(url)}`;
    return `https://semble.so/profile/${userHandle}`;
  })();

  const handleCopy = async (textToCopy: string, key: string) => {
    try {
      await navigator.clipboard.writeText(textToCopy);
      setCopied(key);
      analytics.capture("item_shared", {
        method: "copy_link",
        destination: key,
        item_type: type,
      });
      setTimeout(() => {
        setCopied(null);
        setIsOpen(false);
      }, 1000);
    } catch {
      prompt("Copy this link:", textToCopy);
    }
  };

  const handleShareToFork = (domain: string) => {
    const composeText = text
      ? `${text.substring(0, 200)}...\n\n${shareUrl}`
      : shareUrl;
    const composeUrl = `https://${domain}/intent/compose?text=${encodeURIComponent(composeText)}`;
    analytics.capture("item_shared", {
      method: "social_app",
      destination: domain,
      item_type: type,
    });
    window.open(composeUrl, "_blank");
    setIsOpen(false);
  };

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        menuRef.current &&
        !menuRef.current.contains(e.target as Node) &&
        !buttonRef.current?.contains(e.target as Node)
      ) {
        setIsOpen(false);
      }
    };
    if (isOpen && !isMobile) {
      document.addEventListener("mousedown", handleClickOutside);
      window.addEventListener("scroll", () => setIsOpen(false), true);
      window.addEventListener("resize", () => setIsOpen(false));
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      window.removeEventListener("scroll", () => setIsOpen(false), true);
      window.removeEventListener("resize", () => setIsOpen(false));
    };
  }, [isOpen, isMobile]);

  const calculatePosition = () => {
    if (!buttonRef.current) return;
    const rect = buttonRef.current.getBoundingClientRect();
    const menuWidth = 260;
    const padding = 8;

    let top = rect.bottom + 8;
    let left = rect.left;
    let alignRight = false;

    if (left + menuWidth > window.innerWidth - padding) {
      left = rect.right - menuWidth;
      alignRight = true;
    }

    left = Math.max(
      padding,
      Math.min(left, window.innerWidth - menuWidth - padding),
    );

    if (top + 300 > window.innerHeight) {
      top = rect.top - 8;
    }

    setMenuPosition({ top, left, alignRight });
  };

  const toggleMenu = () => {
    if (!isOpen && !isMobile) calculatePosition();
    setIsOpen(!isOpen);
  };

  const renderMenuItem = (
    label: string,
    icon: React.ReactNode,
    onClick: () => void,
    isCopied: boolean = false,
    highlight: boolean = false,
  ) => (
    <button
      onClick={onClick}
      className={`w-full flex items-center gap-3 px-3 py-2.5 text-[14px] font-medium transition-colors rounded-lg group
                ${
                  highlight
                    ? "text-primary-700 dark:text-primary-400 bg-primary-50/50 dark:bg-primary-900/20 hover:bg-primary-50 dark:hover:bg-primary-900/30"
                    : "text-surface-700 dark:text-surface-200 hover:bg-surface-50 dark:hover:bg-surface-800 hover:text-surface-900 dark:hover:text-white"
                }`}
    >
      <span
        className={`flex items-center justify-center w-5 h-5 ${highlight ? "text-primary-600 dark:text-primary-400" : "text-surface-400 dark:text-surface-500 group-hover:text-surface-600 dark:group-hover:text-surface-300"}`}
      >
        {isCopied ? (
          <Check size={16} className="text-green-600 dark:text-green-400" />
        ) : (
          icon
        )}
      </span>
      <span className="flex-1 text-left">
        {isCopied ? t("shareMenu.copied") : label}
      </span>
    </button>
  );

  const shareForks = [
    {
      name: "Bluesky",
      domain: "bsky.app",
      icon: <BlueskyIcon size={18} />,
      hoverColor: "#1185fe",
    },
    {
      name: "Mu",
      domain: "mu.social",
      icon: <MuIcon size={21} />,
      iconClass:
        "group-hover:[--mu-outline:#481a3e] group-hover:[--mu-letters:#ffeefc]",
    },
    {
      name: "Blacksky",
      domain: "blacksky.community",
      icon: <BlackskyIcon size={18} />,
    },
    {
      name: "Witchsky",
      domain: "witchsky.app",
      icon: <WitchskyIcon size={18} />,
      hoverColor: "#ee5346",
    },
    {
      name: "Catsky",
      domain: "catsky.social",
      icon: <CatskyIcon size={18} />,
      hoverColor: "#cba7f7",
    },
    {
      name: "Deer",
      domain: "deer.social",
      icon: <DeerIcon size={18} />,
      hoverColor: "#739f7c",
    },
  ] as {
    name: string;
    domain: string;
    icon: React.ReactNode;
    hoverColor?: string;
    iconClass?: string;
  }[];

  const menuContent = (
    <div className="flex flex-col gap-0.5">
      {isSemble ? (
        <>
          <div className="px-3 py-2 text-[11px] font-bold text-surface-400 dark:text-surface-500 uppercase tracking-wider flex items-center gap-1.5 select-none">
            <SembleLogo />
            {t("shareMenu.sembleIntegration")}
          </div>
          {renderMenuItem(
            t("shareMenu.openOnSemble"),
            <ExternalLink size={16} />,
            () => window.open(sembleUrl, "_blank"),
            false,
            true,
          )}
          {renderMenuItem(
            t("shareMenu.copySembleLink"),
            <Copy size={16} />,
            () => handleCopy(sembleUrl, "semble"),
            copied === "semble",
          )}
          <div className="h-px bg-surface-100 dark:bg-surface-800 my-1 mx-2" />
        </>
      ) : null}

      {renderMenuItem(
        t("shareMenu.copyLink"),
        <Copy size={16} />,
        () => handleCopy(shareUrl, "link"),
        copied === "link",
      )}

      <div className="px-3 pt-3 pb-1 text-[11px] font-bold text-surface-400 dark:text-surface-500 uppercase tracking-wider select-none">
        {t("shareMenu.shareViaApp")}
      </div>

      <div className="grid grid-cols-6 gap-1 px-1 mb-1">
        {shareForks.map((fork) => (
          <button
            key={fork.domain}
            onClick={() => handleShareToFork(fork.domain)}
            style={
              {
                "--fork-color": fork.hoverColor ?? "currentColor",
              } as React.CSSProperties
            }
            className="group flex items-center justify-center p-2 rounded-lg hover:bg-surface-50 dark:hover:bg-surface-800 transition-colors duration-150 text-surface-400 dark:text-surface-500 hover:text-surface-900 dark:hover:text-white"
            title={`Share to ${fork.name}`}
          >
            <span
              className={`flex transition-[color,transform] duration-150 group-hover:scale-105 group-hover:text-[color:var(--fork-color)] ${fork.iconClass ?? ""}`}
            >
              {fork.icon}
            </span>
          </button>
        ))}
      </div>

      <div className="h-px bg-surface-100 dark:bg-surface-800 my-1 mx-2" />

      {renderMenuItem(
        t("shareMenu.copyUniversalLink"),
        <AturiIcon size={16} />,
        () => handleCopy(uri.replace("at://", "https://aturi.to/"), "aturi"),
        copied === "aturi",
      )}

      {typeof navigator !== "undefined" &&
        navigator.share &&
        renderMenuItem(
          t("shareMenu.moreOptions"),
          <MoreHorizontal size={16} />,
          () => {
            navigator
              .share({ title: "Margin", text, url: shareUrl })
              .catch(() => {});
            setIsOpen(false);
          },
        )}
    </div>
  );

  return (
    <div className="relative inline-block">
      <button
        ref={buttonRef}
        onClick={toggleMenu}
        className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg transition-all ${isOpen ? "text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-900/20" : "text-surface-400 dark:text-surface-500 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20"}`}
        title="Share"
      >
        <Share2 size={16} />
      </button>

      {isOpen && isMobile && (
        <>
          <div
            className="fixed inset-0 bg-black/40 z-[999]"
            onClick={() => setIsOpen(false)}
          />
          <div className="fixed bottom-0 left-0 right-0 z-[1000] animate-slide-up">
            <div
              ref={sheetRef}
              className="mx-2 mb-2 bg-white dark:bg-surface-900 rounded-2xl shadow-xl border border-surface-200 dark:border-surface-700 overflow-hidden"
              style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
            >
              <div
                className="flex justify-center pt-3 pb-1 cursor-grab active:cursor-grabbing touch-none"
                onTouchStart={handleTouchStart}
                onTouchMove={handleTouchMove}
                onTouchEnd={handleTouchEnd}
              >
                <div className="w-8 h-1 bg-surface-200 dark:bg-surface-700 rounded-full" />
              </div>
              <div className="flex items-center justify-between px-4 pt-1 pb-2">
                <span className="text-sm font-semibold text-surface-900 dark:text-white">
                  {t("shareMenu.share", { defaultValue: "Share" })}
                </span>
                <button
                  onClick={() => setIsOpen(false)}
                  className="p-1 rounded-lg text-surface-400 hover:text-surface-600 dark:hover:text-surface-200 hover:bg-surface-100 dark:hover:bg-surface-700 transition-colors"
                >
                  <X size={16} />
                </button>
              </div>
              <div className="px-2 pb-2">{menuContent}</div>
            </div>
          </div>
        </>
      )}
      {isOpen && !isMobile && (
        <div
          ref={menuRef}
          className="fixed z-[1000] w-[260px] bg-white dark:bg-surface-900 rounded-xl shadow-xl ring-1 ring-black/5 dark:ring-white/5 p-1.5 animate-in fade-in zoom-in-95 duration-150"
          style={{
            top: menuPosition.top,
            left: menuPosition.left,
            transformOrigin: menuPosition.alignRight ? "top right" : "top left",
          }}
        >
          {menuContent}
        </div>
      )}
    </div>
  );
}
