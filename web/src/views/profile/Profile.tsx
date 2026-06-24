import { useStore } from "@nanostores/react";
import { clsx } from "clsx";
import { useTranslation } from "react-i18next";
import {
  Edit2,
  Eye,
  EyeOff,
  Flag,
  Folder,
  Github,
  Link2,
  Linkedin,
  Loader2,
  ShieldBan,
  ShieldOff,
  Volume2,
  VolumeX,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  blockUser,
  getCollections,
  getModerationRelationship,
  getProfile,
  muteUser,
  unblockUser,
  unmuteUser,
} from "../../api/client";
import CollectionIcon from "../../components/common/CollectionIcon";
import { BlueskyIcon, TangledIcon } from "../../components/common/Icons";
import type { MoreMenuItem } from "../../components/common/MoreMenu";
import MoreMenu from "../../components/common/MoreMenu";
import RichText from "../../components/common/RichText";
import FeedItems from "../../components/feed/FeedItems";
import EditProfileModal from "../../components/modals/EditProfileModal";
import ExternalLinkModal from "../../components/modals/ExternalLinkModal";
import ReportModal from "../../components/modals/ReportModal";
import {
  Avatar,
  Button,
  EmptyState,
  Skeleton,
  Tabs,
} from "../../components/ui";
import { $user } from "../../store/auth";
import { displayHandle } from "../../lib/handle";
import { $preferences, loadPreferences } from "../../store/preferences";
import type {
  Collection,
  ContentLabel,
  ModerationRelationship,
  UserProfile,
} from "../../types";

const profileCache = new Map<
  string,
  {
    profile: UserProfile;
    labels: ContentLabel[];
    relation: ModerationRelationship;
    timestamp: number;
  }
>();

const profileCollectionsCache = new Map<
  string,
  {
    collections: Collection[];
    timestamp: number;
  }
>();

interface ProfileProps {
  did: string;
  initialProfile?: UserProfile | null;
}

type Tab = "all" | "annotations" | "highlights" | "bookmarks" | "collections";

const motivationMap: Record<Tab, string | undefined> = {
  all: undefined,
  annotations: "commenting",
  highlights: "highlighting",
  bookmarks: "bookmarking",
  collections: undefined,
};

export default function Profile({ did, initialProfile }: ProfileProps) {
  const { t } = useTranslation();
  const [profile, setProfile] = useState<UserProfile | null>(
    initialProfile || null,
  );
  const [loading, setLoading] = useState(!initialProfile);
  const [activeTab, setActiveTab] = useState<Tab>("all");

  const [collections, setCollections] = useState<Collection[]>([]);
  const [dataLoading, setDataLoading] = useState(false);

  const user = useStore($user);
  const isOwner = user?.did === did;
  const [showEdit, setShowEdit] = useState(false);
  const [externalLink, setExternalLink] = useState<string | null>(null);
  const [showReportModal, setShowReportModal] = useState(false);
  const loadMoreTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [modRelation, setModRelation] = useState<ModerationRelationship>({
    blocking: false,
    muting: false,
    blockedBy: false,
  });
  const [accountLabels, setAccountLabels] = useState<ContentLabel[]>([]);
  const [profileRevealed, setProfileRevealed] = useState(false);
  const preferences = useStore($preferences);

  const formatLinkText = (url: string) => {
    try {
      const urlObj = new URL(url.startsWith("http") ? url : `https://${url}`);
      const domain = urlObj.hostname.replace(/^www\./, "");
      const path = urlObj.pathname.replace(/^\/|\/$/g, "");

      if (
        domain.includes("github.com") ||
        domain.includes("twitter.com") ||
        domain.includes("x.com")
      ) {
        return path ? `${domain}/${path}` : domain;
      }
      if (domain.includes("linkedin.com") && path.includes("in/")) {
        return `linkedin.com/${path.split("in/")[1]}`;
      }
      if (domain.includes("tangled")) {
        return path ? `${domain}/${path}` : domain;
      }

      return domain + (path && path.length < 20 ? `/${path}` : "");
    } catch {
      return url;
    }
  };

  const skipInitialProfileFetch = useRef(!!initialProfile);
  useEffect(() => {
    if (skipInitialProfileFetch.current) {
      skipInitialProfileFetch.current = false;
    } else {
      setProfile(null);
      setCollections([]);
      setActiveTab("all");
      setLoading(true);
    }

    const loadProfile = async () => {
      const cached = profileCache.get(did);
      if (cached && Date.now() - cached.timestamp < 5 * 60 * 1000) {
        setProfile(cached.profile);
        setAccountLabels(cached.labels);
        setModRelation(cached.relation);
        setLoading(false);
      } else if (!initialProfile) {
        setLoading(true);
      }

      try {
        const marginPromise = getProfile(did);
        const bskyPromise = fetch(
          `https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile?actor=${encodeURIComponent(did)}`,
        )
          .then((res) => (res.ok ? res.json() : null))
          .catch(() => null);

        const [marginData, bskyData] = await Promise.all([
          marginPromise,
          bskyPromise,
        ]);

        const merged: UserProfile = {
          did: marginData?.did || bskyData?.did || did,
          handle: marginData?.handle || bskyData?.handle || "",
          displayName: marginData?.displayName || bskyData?.displayName,
          avatar: marginData?.avatar || bskyData?.avatar,
          description: marginData?.description || bskyData?.description,
          banner: marginData?.banner || bskyData?.banner,
          website: marginData?.website,
          links: marginData?.links || [],
          followersCount:
            bskyData?.followersCount || marginData?.followersCount,
          followsCount: bskyData?.followsCount || marginData?.followsCount,
          postsCount: bskyData?.postsCount || marginData?.postsCount,
        };

        if (marginData?.labels && Array.isArray(marginData.labels)) {
          setAccountLabels(marginData.labels);
        }

        setProfile(merged);

        if (user && user.did !== did) {
          try {
            const rel = await getModerationRelationship(did);
            setModRelation(rel);
            profileCache.set(did, {
              profile: merged,
              labels: marginData?.labels || [],
              relation: rel,
              timestamp: Date.now(),
            });
          } catch {
            profileCache.set(did, {
              profile: merged,
              labels: marginData?.labels || [],
              relation: modRelation,
              timestamp: Date.now(),
            });
          }
        } else {
          profileCache.set(did, {
            profile: merged,
            labels: marginData?.labels || [],
            relation: modRelation,
            timestamp: Date.now(),
          });
        }
      } catch (e) {
        console.error("Profile load failed", e);
      } finally {
        setLoading(false);
      }
    };
    if (did) loadProfile();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [did, user, initialProfile]);

  useEffect(() => {
    loadPreferences();
  }, []);

  useEffect(() => {
    const timer = loadMoreTimerRef.current;
    return () => {
      if (timer) clearTimeout(timer);
    };
  }, []);

  const isHandle = !did.startsWith("did:");
  const resolvedDid = isHandle ? profile?.did : did;

  useEffect(() => {
    const loadTabContent = async () => {
      const isHandle = !did.startsWith("did:");
      const resolvedDid = isHandle ? profile?.did : did;

      if (!resolvedDid) return;

      setDataLoading(true);
      try {
        if (activeTab === "collections") {
          const cached = profileCollectionsCache.get(resolvedDid);
          if (cached && Date.now() - cached.timestamp < 5 * 60 * 1000) {
            setCollections(cached.collections);
            setDataLoading(false);
          }
          const res = await getCollections(resolvedDid);
          setCollections(res);
          profileCollectionsCache.set(resolvedDid, {
            collections: res,
            timestamp: Date.now(),
          });
        }
      } catch (e) {
        console.error(e);
      } finally {
        setDataLoading(false);
      }
    };
    loadTabContent();
  }, [profile?.did, did, activeTab]);

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto animate-fade-in">
        <div className="card p-5 mb-4">
          <div className="flex items-start gap-4">
            <Skeleton variant="circular" className="w-16 h-16" />
            <div className="flex-1 space-y-2">
              <Skeleton width="40%" className="h-6" />
              <Skeleton width="25%" className="h-4" />
              <Skeleton width="60%" className="h-4" />
            </div>
          </div>
        </div>
        <Skeleton className="h-10 mb-4" />
        <div className="space-y-3">
          <Skeleton className="h-32 rounded-lg" />
          <Skeleton className="h-32 rounded-lg" />
        </div>
      </div>
    );
  }

  if (!profile) {
    return (
      <EmptyState
        title={t("profile.notFound")}
        message={t("profile.notFoundMessage")}
      />
    );
  }

  const tabs = [
    { id: "all", label: t("urlPage.tabs.all") },
    { id: "annotations", label: t("urlPage.tabs.annotations") },
    { id: "highlights", label: t("urlPage.tabs.highlights") },
    { id: "bookmarks", label: t("urlPage.tabs.bookmarks") },
    { id: "collections", label: t("urlPage.tabs.collections") },
  ];

  const LABEL_DESCRIPTIONS: Record<string, string> = {
    sexual: t("card.labelDescriptions.sexual"),
    nudity: t("card.labelDescriptions.nudity"),
    violence: t("card.labelDescriptions.violence"),
    gore: t("card.labelDescriptions.gore"),
    spam: t("card.labelDescriptions.spam"),
    misleading: t("card.labelDescriptions.misleading"),
  };

  const accountWarning = (() => {
    if (!accountLabels.length) return null;
    const priority = [
      "gore",
      "violence",
      "nudity",
      "sexual",
      "misleading",
      "spam",
    ];
    for (const p of priority) {
      const match = accountLabels.find((l) => l.val === p);
      if (match) {
        const pref = preferences.labelPreferences.find(
          (lp) => lp.label === p && lp.labelerDid === match.src,
        );
        const visibility = pref?.visibility || "warn";
        if (visibility === "ignore") continue;
        return {
          label: p,
          description: LABEL_DESCRIPTIONS[p] || p,
          visibility,
        };
      }
    }
    return null;
  })();

  const shouldBlurAvatar = accountWarning && !profileRevealed;

  return (
    <div className="max-w-2xl mx-auto animate-slide-up">
      <div className="card p-5 mb-4">
        <div className="flex items-start gap-4">
          <div className="relative">
            <div className="rounded-full overflow-hidden">
              <div
                className={clsx(
                  "transition-all",
                  shouldBlurAvatar && "blur-lg",
                )}
              >
                <Avatar
                  did={profile.did}
                  avatar={profile.avatar}
                  size="xl"
                  className="ring-4 ring-surface-100 dark:ring-surface-800"
                />
              </div>
            </div>
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <h1 className="text-xl font-bold text-surface-900 dark:text-white truncate">
                  {profile.displayName ||
                    displayHandle(profile.handle, profile.did)}
                </h1>
                <p className="text-surface-500 dark:text-surface-400">
                  @{displayHandle(profile.handle, profile.did)}
                </p>
              </div>
              <div className="flex items-center gap-2">
                {isOwner && (
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setShowEdit(true)}
                    icon={<Edit2 size={14} />}
                  >
                    <span className="hidden sm:inline">
                      {t("profile.edit")}
                    </span>
                  </Button>
                )}
                {!isOwner && user && (
                  <MoreMenu
                    items={(() => {
                      const items: MoreMenuItem[] = [];
                      items.push({
                        label: t("profile.viewInBluesky"),
                        icon: <BlueskyIcon size={16} />,
                        onClick: () => {
                          const handle = profile.handle || did;
                          window.open(
                            `https://bsky.app/profile/${encodeURIComponent(handle)}`,
                            "_blank",
                          );
                        },
                      });
                      if (modRelation.blocking) {
                        items.push({
                          label: t("profile.unblock", {
                            handle: profile.handle || "user",
                          }),
                          icon: <ShieldOff size={14} />,
                          onClick: async () => {
                            await unblockUser(did);
                            setModRelation((prev) => ({
                              ...prev,
                              blocking: false,
                            }));
                          },
                        });
                      } else {
                        items.push({
                          label: t("profile.block", {
                            handle: profile.handle || "user",
                          }),
                          icon: <ShieldBan size={14} />,
                          onClick: async () => {
                            await blockUser(did);
                            setModRelation((prev) => ({
                              ...prev,
                              blocking: true,
                            }));
                          },
                          variant: "danger",
                        });
                      }
                      if (modRelation.muting) {
                        items.push({
                          label: t("profile.unmute", {
                            handle: profile.handle || "user",
                          }),
                          icon: <Volume2 size={14} />,
                          onClick: async () => {
                            await unmuteUser(did);
                            setModRelation((prev) => ({
                              ...prev,
                              muting: false,
                            }));
                          },
                        });
                      } else {
                        items.push({
                          label: t("profile.mute", {
                            handle: profile.handle || "user",
                          }),
                          icon: <VolumeX size={14} />,
                          onClick: async () => {
                            await muteUser(did);
                            setModRelation((prev) => ({
                              ...prev,
                              muting: true,
                            }));
                          },
                        });
                      }
                      items.push({
                        label: t("profile.report"),
                        icon: <Flag size={14} />,
                        onClick: () => setShowReportModal(true),
                        variant: "danger",
                      });
                      return items;
                    })()}
                  />
                )}
              </div>
            </div>

            {profile.description && (
              <p className="text-surface-600 dark:text-surface-300 text-sm mt-3 whitespace-pre-line break-words">
                <RichText text={profile.description} />
              </p>
            )}

            <div className="flex flex-wrap gap-3 mt-3">
              {[
                ...(profile.website ? [profile.website] : []),
                ...(profile.links || []),
              ]
                .filter((link, index, self) => self.indexOf(link) === index)
                .map((link) => {
                  let icon;
                  if (link.includes("github.com")) {
                    icon = <Github size={16} />;
                  } else if (link.includes("linkedin.com")) {
                    icon = <Linkedin size={16} />;
                  } else if (
                    link.includes("tangled.sh") ||
                    link.includes("tangled.org")
                  ) {
                    icon = <TangledIcon size={16} />;
                  } else {
                    icon = <Link2 size={16} />;
                  }

                  return (
                    <button
                      key={link}
                      onClick={() => {
                        const fullUrl = link.startsWith("http")
                          ? link
                          : `https://${link}`;
                        try {
                          const prefs = $preferences.get();
                          if (prefs.disableExternalLinkWarning) {
                            window.open(
                              fullUrl,
                              "_blank",
                              "noopener,noreferrer",
                            );
                            return;
                          }
                          const hostname = new URL(fullUrl).hostname;
                          const skipped = prefs.externalLinkSkippedHostnames;
                          if (skipped.includes(hostname)) {
                            window.open(
                              fullUrl,
                              "_blank",
                              "noopener,noreferrer",
                            );
                          } else {
                            setExternalLink(fullUrl);
                          }
                        } catch {
                          setExternalLink(fullUrl);
                        }
                      }}
                      className="flex items-center gap-1.5 text-sm text-surface-500 dark:text-surface-400 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                    >
                      {icon}
                      <span className="truncate max-w-[200px]">
                        {formatLinkText(link)}
                      </span>
                    </button>
                  );
                })}
            </div>
          </div>
        </div>
      </div>

      {accountWarning && (
        <div className="card p-4 mb-4 border-amber-200 dark:border-amber-800/50 bg-amber-50/50 dark:bg-amber-900/10">
          <div className="flex items-center gap-3">
            <EyeOff size={18} className="text-amber-500 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-sm font-medium text-amber-700 dark:text-amber-400">
                {t("profile.accountLabeled", {
                  description: accountWarning.description,
                })}
              </p>
              <p className="text-xs text-amber-600/70 dark:text-amber-400/60 mt-0.5">
                {t("profile.labelApplied")}
              </p>
            </div>
            {!profileRevealed ? (
              <button
                onClick={() => setProfileRevealed(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-amber-600 dark:text-amber-400 hover:bg-amber-100 dark:hover:bg-amber-900/30 rounded-lg transition-colors"
              >
                <Eye size={12} />
                {t("profile.show")}
              </button>
            ) : (
              <button
                onClick={() => setProfileRevealed(false)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-amber-600 dark:text-amber-400 hover:bg-amber-100 dark:hover:bg-amber-900/30 rounded-lg transition-colors"
              >
                <EyeOff size={12} />
                {t("profile.hide")}
              </button>
            )}
          </div>
        </div>
      )}

      {modRelation.blocking && (
        <div className="card p-4 mb-4 border-red-200 dark:border-red-800/50 bg-red-50/50 dark:bg-red-900/10">
          <div className="flex items-center gap-3">
            <ShieldBan size={18} className="text-red-500 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-sm font-medium text-red-700 dark:text-red-400">
                {t("profile.blockedBanner", { handle: profile.handle })}
              </p>
              <p className="text-xs text-red-600/70 dark:text-red-400/60 mt-0.5">
                {t("profile.blockedMessage")}
              </p>
            </div>
            <button
              onClick={async () => {
                await unblockUser(did);
                setModRelation((prev) => ({ ...prev, blocking: false }));
              }}
              className="px-3 py-1.5 text-xs font-medium text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-lg transition-colors"
            >
              {t("profile.unblock_action")}
            </button>
          </div>
        </div>
      )}

      {modRelation.muting && !modRelation.blocking && (
        <div className="card p-4 mb-4 border-amber-200 dark:border-amber-800/50 bg-amber-50/50 dark:bg-amber-900/10">
          <div className="flex items-center gap-3">
            <VolumeX size={18} className="text-amber-500 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-sm font-medium text-amber-700 dark:text-amber-400">
                {t("profile.mutedBanner", { handle: profile.handle })}
              </p>
              <p className="text-xs text-amber-600/70 dark:text-amber-400/60 mt-0.5">
                {t("profile.mutedMessage")}
              </p>
            </div>
            <button
              onClick={async () => {
                await unmuteUser(did);
                setModRelation((prev) => ({ ...prev, muting: false }));
              }}
              className="px-3 py-1.5 text-xs font-medium text-amber-600 dark:text-amber-400 hover:bg-amber-100 dark:hover:bg-amber-900/30 rounded-lg transition-colors"
            >
              {t("profile.unmute_action")}
            </button>
          </div>
        </div>
      )}

      {modRelation.blockedBy && !modRelation.blocking && (
        <div className="card p-4 mb-4 border-surface-200 dark:border-surface-700">
          <div className="flex items-center gap-3">
            <ShieldBan size={18} className="text-surface-400 flex-shrink-0" />
            <p className="text-sm text-surface-500 dark:text-surface-400">
              {t("profile.blockedByBanner", { handle: profile.handle })}
            </p>
          </div>
        </div>
      )}

      <Tabs
        tabs={tabs}
        activeTab={activeTab}
        onChange={(id) => setActiveTab(id as Tab)}
        className="mb-4"
      />

      <div className="min-h-[200px]">
        {dataLoading ? (
          <div className="flex flex-col items-center justify-center py-12 gap-3">
            <Loader2
              className="animate-spin text-primary-600 dark:text-primary-400"
              size={24}
            />
            <p className="text-sm text-surface-400 dark:text-surface-500">
              {t("common.loading")}
            </p>
          </div>
        ) : activeTab === "collections" ? (
          collections.length === 0 ? (
            <EmptyState
              icon={<Folder size={40} />}
              message={
                isOwner
                  ? t("profile.emptyCollectionsOwn")
                  : t("profile.emptyCollectionsOther")
              }
            />
          ) : (
            <div className="grid grid-cols-1 gap-2">
              {collections.map((collection) => (
                <a
                  key={collection.id}
                  href={`/${collection.creator?.handle || profile.handle}/collection/${(collection.uri || "").split("/").pop()}`}
                  className="group card p-4 hover:ring-primary-300 dark:hover:ring-primary-600 transition-all flex items-center gap-4"
                >
                  <div className="p-2.5 bg-primary-50 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 rounded-xl">
                    <CollectionIcon icon={collection.icon} size={20} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <h3 className="font-semibold text-surface-900 dark:text-white truncate group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
                      {collection.name}
                    </h3>
                    <p className="text-sm text-surface-500 dark:text-surface-400">
                      {t("profile.itemCount", { count: collection.itemCount })}
                    </p>
                  </div>
                </a>
              ))}
            </div>
          )
        ) : (
          <FeedItems
            key={activeTab}
            type="all"
            motivation={motivationMap[activeTab]}
            creator={resolvedDid}
            layout="list"
            emptyMessage={
              isOwner
                ? t("profile.emptyTabOwn", { tab: activeTab })
                : t("profile.emptyTabOther")
            }
          />
        )}
      </div>

      {showEdit && profile && (
        <EditProfileModal
          profile={profile}
          onClose={() => setShowEdit(false)}
          onUpdate={(updated) => setProfile(updated)}
        />
      )}

      <ExternalLinkModal
        isOpen={!!externalLink}
        onClose={() => setExternalLink(null)}
        url={externalLink}
      />

      <ReportModal
        isOpen={showReportModal}
        onClose={() => setShowReportModal(false)}
        subjectDid={did}
        subjectHandle={profile?.handle}
      />
    </div>
  );
}
