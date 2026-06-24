import React, { useState, useEffect, useRef } from "react";
import Avatar from "../ui/Avatar";
import RichText from "./RichText";
import { getProfile } from "../../api/client";
import { displayHandle } from "../../lib/handle";
import type { UserProfile } from "../../types";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";

interface ProfileHoverCardProps {
  did?: string;
  handle?: string;
  children: React.ReactNode;
  className?: string;
}

export default function ProfileHoverCard({
  did,
  handle,
  children,
  className,
}: ProfileHoverCardProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const cardRef = useRef<HTMLDivElement>(null);

  const handleMouseEnter = () => {
    timeoutRef.current = setTimeout(async () => {
      setIsOpen(true);
      if (!profile && (did || handle)) {
        setLoading(true);
        try {
          const identifier = did || handle || "";

          const [marginData, bskyData] = await Promise.all([
            getProfile(identifier).catch(() => null),
            fetch(
              `https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile?actor=${encodeURIComponent(identifier)}`,
            )
              .then((res) => (res.ok ? res.json() : null))
              .catch(() => null),
          ]);

          const merged: UserProfile = {
            did: marginData?.did || bskyData?.did || identifier,
            handle: marginData?.handle || bskyData?.handle || "",
            displayName: marginData?.displayName || bskyData?.displayName,
            avatar: marginData?.avatar || bskyData?.avatar,
            description: marginData?.description || bskyData?.description,
          };

          setProfile(merged);
        } catch (e) {
          console.error("Failed to load profile", e);
        } finally {
          setLoading(false);
        }
      }
    }, 400);
  };

  const handleMouseLeave = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    closeTimeoutRef.current = setTimeout(() => {
      setIsOpen(false);
    }, 300);
  };

  const handleCardMouseEnter = () => {
    if (closeTimeoutRef.current) {
      clearTimeout(closeTimeoutRef.current);
      closeTimeoutRef.current = null;
    }
  };

  const handleCardMouseLeave = () => {
    setIsOpen(false);
  };

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      if (closeTimeoutRef.current) {
        clearTimeout(closeTimeoutRef.current);
      }
    };
  }, []);

  return (
    <div
      className={`relative inline-block ${className || ""}`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      ref={cardRef}
    >
      {children}

      {isOpen && (
        <div
          className="absolute z-50 left-0 top-full mt-2 w-72 bg-white dark:bg-surface-800 rounded-xl shadow-xl border border-surface-200 dark:border-surface-700 p-4 animate-in fade-in slide-in-from-top-1 duration-150"
          onMouseEnter={handleCardMouseEnter}
          onMouseLeave={handleCardMouseLeave}
        >
          {loading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 size={20} className="animate-spin text-primary-600" />
            </div>
          ) : profile ? (
            <div className="space-y-3">
              <a
                href={`/profile/${profile.did}`}
                className="flex items-start gap-3 group"
              >
                <Avatar
                  did={profile.did}
                  avatar={profile.avatar}
                  size="lg"
                  className="shrink-0"
                />
                <div className="flex-1 min-w-0">
                  <p className="font-semibold text-surface-900 dark:text-white truncate group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
                    {profile.displayName ||
                      displayHandle(profile.handle, profile.did)}
                  </p>
                  <p className="text-sm text-surface-500 dark:text-surface-400 truncate">
                    @{displayHandle(profile.handle, profile.did)}
                  </p>
                </div>
              </a>

              {profile.description && (
                <p className="text-sm text-surface-600 dark:text-surface-300 whitespace-pre-line line-clamp-3">
                  <RichText text={profile.description} />
                </p>
              )}

              <a
                href={`/profile/${profile.did}`}
                className="block w-full text-center py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
              >
                {t("profileHoverCard.viewProfile")}
              </a>
            </div>
          ) : (
            <p className="text-sm text-surface-500 text-center py-2">
              {t("profileHoverCard.notFound")}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
