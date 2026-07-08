import { atom } from "nanostores";
import type {
  AnnotationItem,
  Collection,
  FeedResponse,
  HydratedLabel,
  NotificationItem,
  Selector,
  Target,
  TextQuoteSelector,
  UserProfile,
} from "../types";

export type { Collection } from "../types";

export const sessionAtom = atom<UserProfile | null>(null);

export async function checkSession(): Promise<UserProfile | null | undefined> {
  try {
    const res = await fetch("/auth/session");
    if (!res.ok) {
      return undefined;
    }
    const data = await res.json();

    if (data.authenticated || data.did) {
      const baseProfile: UserProfile = {
        did: data.did,
        handle: data.handle,
        displayName: data.displayName,
        avatar: data.avatar,
        description: data.description,
        website: data.website,
        links: data.links,
        followersCount: data.followersCount,
        followsCount: data.followsCount,
        postsCount: data.postsCount,
      };

      const [bskyResult, marginResult] = await Promise.allSettled([
        fetch(
          `https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile?actor=${encodeURIComponent(data.did)}`,
        ),
        fetch(`/api/profile/${data.did}`),
      ]);

      if (bskyResult.status === "fulfilled" && bskyResult.value.ok) {
        try {
          const bskyData = await bskyResult.value.json();
          if (bskyData.avatar) baseProfile.avatar = bskyData.avatar;
          if (bskyData.displayName)
            baseProfile.displayName = bskyData.displayName;
        } catch {
          /* ignore */
        }
      }

      if (marginResult.status === "fulfilled" && marginResult.value.ok) {
        try {
          const marginProfile = await marginResult.value.json();
          if (marginProfile) {
            if (marginProfile.description)
              baseProfile.description = marginProfile.description;
            if (marginProfile.followersCount)
              baseProfile.followersCount = marginProfile.followersCount;
            if (marginProfile.followsCount)
              baseProfile.followsCount = marginProfile.followsCount;
            if (marginProfile.postsCount)
              baseProfile.postsCount = marginProfile.postsCount;
            if (marginProfile.website)
              baseProfile.website = marginProfile.website;
            if (marginProfile.links) baseProfile.links = marginProfile.links;
          }
        } catch {
          /* ignore */
        }
      }

      sessionAtom.set(baseProfile);
      return baseProfile;
    }

    sessionAtom.set(null);
    return null;
  } catch (e) {
    console.error("Session check failed:", e);
    return undefined;
  }
}

async function apiRequest(
  path: string,
  options: RequestInit & { skipAuthRedirect?: boolean } = {},
): Promise<Response> {
  const { skipAuthRedirect, ...fetchOptions } = options;
  const headers = {
    "Content-Type": "application/json",
    ...(fetchOptions.headers || {}),
  };

  const apiPath =
    path.startsWith("/api") || path.startsWith("/auth") ? path : `/api${path}`;

  const response = await fetch(apiPath, {
    ...fetchOptions,
    headers,
  });

  if (response.status === 401 && !skipAuthRedirect) {
    const verified = await checkSession();
    if (verified === null) {
      sessionAtom.set(null);
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    }
  }

  return response;
}

export interface GetFeedParams {
  source?: string;
  type?: string;
  limit?: number;
  offset?: number;
  motivation?: string;
  tag?: string;
  creator?: string;
}

interface RawItem {
  type?: string;
  collectionUri?: string;
  annotation?: RawItem;
  highlight?: RawItem;
  bookmark?: RawItem;
  uri?: string;
  id?: string;
  cid?: string;
  author?: UserProfile;
  creator?: UserProfile;
  collection?: {
    uri: string;
    name: string;
    icon?: string;
  };
  context?: {
    uri: string;
    name: string;
    icon?: string;
  }[];
  created?: string;
  createdAt?: string;
  target?: string | { source?: string; title?: string; selector?: Selector };
  url?: string;
  targetUrl?: string;
  title?: string;
  selector?: Selector;
  viewer?: { like?: string; [key: string]: unknown };
  viewerHasLiked?: boolean;
  motivation?: string;
  [key: string]: unknown;
}

function normalizeItem(raw: RawItem): AnnotationItem {
  if (raw.type === "CollectionItem" || raw.collectionUri) {
    const inner = raw.annotation || raw.highlight || raw.bookmark || {};
    const normalizedInner = normalizeItem(inner);

    return {
      ...normalizedInner,
      uri: normalizedInner.uri || raw.uri || "",
      cid: normalizedInner.cid || raw.cid || "",
      author: (normalizedInner.author ||
        raw.author ||
        raw.creator) as UserProfile,
      collection: raw.collection
        ? {
            uri: raw.collection.uri,
            name: raw.collection.name,
            icon: raw.collection.icon,
          }
        : undefined,
      context: raw.context
        ? raw.context.map((c) => ({
            uri: c.uri,
            name: c.name,
            icon: c.icon,
          }))
        : undefined,
      addedBy: raw.creator || raw.author,
      createdAt:
        normalizedInner.createdAt ||
        raw.created ||
        raw.createdAt ||
        new Date().toISOString(),
      collectionItemUri: raw.id || raw.uri,
    };
  }

  let target: Target | undefined;

  if (raw.target) {
    if (typeof raw.target === "string") {
      target = { source: raw.target, title: raw.title, selector: raw.selector };
    } else {
      target = {
        source: raw.target.source || "",
        title: raw.target.title || raw.title,
        selector: raw.target.selector || raw.selector,
      };
    }
  }

  if (!target || !target.source) {
    const url =
      raw.url ||
      raw.targetUrl ||
      (typeof raw.target === "string" ? raw.target : raw.target?.source);
    if (url) {
      target = {
        source: url,
        title:
          raw.title ||
          (typeof raw.target !== "string" ? raw.target?.title : undefined),
        selector:
          raw.selector ||
          (typeof raw.target !== "string" ? raw.target?.selector : undefined),
      };
    }
  }

  return {
    ...raw,
    uri: raw.id || raw.uri || "",
    cid: raw.cid || "",
    author: (raw.creator || raw.author) as UserProfile,
    createdAt: raw.created || raw.createdAt || new Date().toISOString(),
    target: target,
    viewer: raw.viewer || { like: raw.viewerHasLiked ? "true" : undefined },
    motivation: raw.motivation || "highlighting",
    parentUri: (raw as Record<string, unknown>).inReplyTo as string | undefined,
  };
}

export async function searchItems(
  query: string,
  options: { creator?: string; limit?: number; offset?: number } = {},
): Promise<FeedResponse> {
  const params = new URLSearchParams();
  params.append("q", query);
  if (options.creator) params.append("creator", options.creator);
  if (options.limit) params.append("limit", options.limit.toString());
  if (options.offset) params.append("offset", options.offset.toString());

  try {
    const res = await apiRequest(`/api/search?${params.toString()}`, {
      skipAuthRedirect: true,
    });
    if (!res.ok) throw new Error("Search failed");
    const data = await res.json();
    const items: AnnotationItem[] = (data.items || []).map(normalizeItem);
    return {
      items,
      hasMore: items.length >= (options.limit || 50),
      fetchedCount: items.length,
    };
  } catch (e) {
    console.error("Search error:", e);
    return { items: [], hasMore: false, fetchedCount: 0 };
  }
}

export async function getFeed({
  source,
  type = "all",
  limit = 50,
  offset = 0,
  motivation,
  tag,
  creator,
}: GetFeedParams): Promise<FeedResponse> {
  const params = new URLSearchParams();
  if (source) params.append("source", source);
  if (type) params.append("type", type);
  if (limit) params.append("limit", limit.toString());
  if (offset) params.append("offset", offset.toString());
  if (motivation) params.append("motivation", motivation);
  if (tag) params.append("tag", tag);
  if (creator) params.append("creator", creator);

  const endpoint = source ? "/api/targets" : "/api/notes/feed";

  try {
    const res = await apiRequest(`${endpoint}?${params.toString()}`, {
      skipAuthRedirect: true,
    });
    if (!res.ok) throw new Error("Failed to fetch feed");
    const data = await res.json();
    const normalizedItems: AnnotationItem[] = (data.items || []).map(
      normalizeItem,
    );

    const groupedItems: AnnotationItem[] = [];
    if (normalizedItems.length > 0) {
      groupedItems.push(normalizedItems[0]);

      for (let i = 1; i < normalizedItems.length; i++) {
        const prev = groupedItems[groupedItems.length - 1];
        const curr = normalizedItems[i];

        if (prev.collection && curr.collection) {
          if (
            prev.uri === curr.uri &&
            prev.addedBy?.did === curr.addedBy?.did
          ) {
            if (!prev.context) {
              prev.context = [prev.collection];
            }
            prev.context.push(curr.collection);
            groupedItems[groupedItems.length - 1] = prev;
            continue;
          }
        }
        groupedItems.push(curr);
      }
    }

    return {
      items: groupedItems,
      hasMore: normalizedItems.length >= limit,
      fetchedCount: normalizedItems.length,
    };
  } catch (e) {
    console.error(e);
    return { items: [], hasMore: false, fetchedCount: 0 };
  }
}

interface CreateAnnotationParams {
  url: string;
  text?: string;
  title?: string;
  selector?: TextQuoteSelector;
  tags?: string[];
  labels?: string[];
}

export async function createAnnotation({
  url,
  text,
  title,
  selector,
  tags,
  labels,
}: CreateAnnotationParams) {
  try {
    const res = await apiRequest("/api/notes", {
      method: "POST",
      body: JSON.stringify({ url, text, title, selector, tags, labels }),
    });
    if (!res.ok) throw new Error(await res.text());
    const raw = await res.json();
    return normalizeItem(raw);
  } catch (e) {
    console.error(e);
    return { error: e instanceof Error ? e.message : "Unknown error" };
  }
}

interface CreateHighlightParams {
  url: string;
  selector: TextQuoteSelector;
  color?: string;
  tags?: string[];
  title?: string;
  labels?: string[];
}

export async function createHighlight({
  url,
  selector,
  color,
  tags,
  title,
  labels,
}: CreateHighlightParams) {
  try {
    const res = await apiRequest("/api/highlights", {
      method: "POST",
      body: JSON.stringify({ url, selector, color, tags, title, labels }),
    });
    if (!res.ok) throw new Error(await res.text());
    const raw = await res.json();
    return normalizeItem(raw);
  } catch (e) {
    console.error(e);
    return { error: e instanceof Error ? e.message : "Unknown error" };
  }
}

export async function createBookmark({
  url,
  title,
  description,
  tags,
}: {
  url: string;
  title?: string;
  description?: string;
  tags?: string[];
}) {
  try {
    const res = await apiRequest("/api/bookmarks", {
      method: "POST",
      body: JSON.stringify({ url, title, description, tags }),
    });
    if (!res.ok) throw new Error(await res.text());
    const raw = await res.json();
    return normalizeItem(raw);
  } catch (e) {
    console.error(e);
    return { error: e instanceof Error ? e.message : "Unknown error" };
  }
}

export async function uploadAvatar(
  file: File,
): Promise<{ blob: Blob | string }> {
  const formData = new FormData();
  formData.append("file", file);
  const res = await fetch("/api/upload/avatar", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${(await checkSession())?.did}`,
    },
    body: formData,
  });
  if (!res.ok) throw new Error("Failed to upload avatar");
  return res.json();
}

export async function updateProfile(updates: {
  displayName?: string;
  description?: string;
  avatar?: Blob | string | null;
  website?: string;
  links?: string[];
}): Promise<boolean> {
  try {
    const { description, ...rest } = updates;
    const body = { ...rest, bio: description };
    const res = await apiRequest("/api/profile", {
      method: "PUT",
      body: JSON.stringify(body),
    });
    return res.ok;
  } catch (e) {
    console.error(e);
    return false;
  }
}

export async function likeItem(uri: string, cid: string): Promise<boolean> {
  try {
    const res = await apiRequest("/api/notes/like", {
      method: "POST",
      body: JSON.stringify({ subjectUri: uri, subjectCid: cid }),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to like item:", e);
    return false;
  }
}

export async function unlikeItem(uri: string): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/notes/like?uri=${encodeURIComponent(uri)}`,
      {
        method: "DELETE",
      },
    );
    return res.ok;
  } catch (e) {
    console.error("Failed to unlike item:", e);
    return false;
  }
}

export async function deleteItem(
  uri: string,
  type: string = "annotation",
): Promise<boolean> {
  const rkey = (uri || "").split("/").pop();

  let endpoint = "/api/notes";
  if (type === "highlight" || uri.includes("highlight")) {
    endpoint = "/api/highlights";
  } else if (type === "bookmark" || uri.includes("bookmark")) {
    endpoint = "/api/bookmarks";
  }

  try {
    const res = await apiRequest(`${endpoint}?rkey=${rkey}`, {
      method: "DELETE",
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to delete item:", e);
    return false;
  }
}

export async function convertHighlightToAnnotation(
  highlightUri: string,
  url: string,
  text: string,
  selector?: TextQuoteSelector,
  title?: string,
): Promise<{ success: boolean; item?: AnnotationItem; error?: string }> {
  try {
    const createRes = await apiRequest("/api/notes", {
      method: "POST",
      body: JSON.stringify({ url, text, title, selector }),
    });
    if (!createRes.ok) {
      const err = await createRes.text();
      return { success: false, error: err };
    }
    const created = normalizeItem(await createRes.json());

    const rkey = (highlightUri || "").split("/").pop();
    if (rkey) {
      await apiRequest(`/api/highlights?rkey=${rkey}`, { method: "DELETE" });
    }

    return { success: true, item: created };
  } catch (e) {
    console.error("Failed to convert highlight:", e);
    return {
      success: false,
      error: e instanceof Error ? e.message : "Unknown error",
    };
  }
}

export async function updateAnnotation(
  uri: string,
  text: string,
  tags?: string[],
  labels?: string[],
): Promise<boolean> {
  try {
    const res = await apiRequest(`/api/notes?uri=${encodeURIComponent(uri)}`, {
      method: "PUT",
      body: JSON.stringify({ text, tags, labels }),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to update annotation:", e);
    return false;
  }
}

export async function updateHighlight(
  uri: string,
  color: string,
  tags?: string[],
  labels?: string[],
): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/highlights?uri=${encodeURIComponent(uri)}`,
      {
        method: "PUT",
        body: JSON.stringify({ color, tags, labels }),
      },
    );
    return res.ok;
  } catch (e) {
    console.error("Failed to update highlight:", e);
    return false;
  }
}

export async function updateBookmark(
  uri: string,
  title?: string,
  description?: string,
  tags?: string[],
  labels?: string[],
): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/bookmarks?uri=${encodeURIComponent(uri)}`,
      {
        method: "PUT",
        body: JSON.stringify({ title, description, tags, labels }),
      },
    );
    return res.ok;
  } catch (e) {
    console.error("Failed to save bookmark:", e);
    return false;
  }
}

export async function getCollectionsContaining(
  annotationUri: string,
): Promise<string[]> {
  try {
    const res = await apiRequest(
      `/api/collections/containing?uri=${encodeURIComponent(annotationUri)}`,
    );
    if (!res.ok) return [];
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch containing collections:", e);
    return [];
  }
}

import type { EditHistoryItem } from "../types";

export async function getEditHistory(uri: string): Promise<EditHistoryItem[]> {
  try {
    const res = await apiRequest(
      `/api/notes/history?uri=${encodeURIComponent(uri)}`,
    );
    if (!res.ok) return [];
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch edit history:", e);
    return [];
  }
}

export async function getProfile(did: string): Promise<UserProfile | null> {
  try {
    const res = await apiRequest(`/api/profile/${did}`);
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch profile:", e);
    return null;
  }
}

export interface ActorSearchItem {
  did: string;
  handle: string;
  displayName?: string;
  avatar?: string;
}

export function getAvatarUrl(
  did?: string,
  avatar?: string,
): string | undefined {
  if (!avatar && !did) return undefined;
  if (avatar && !avatar.includes("cdn.bsky.app")) return avatar;
  if (!did) return avatar;

  return `/api/avatar/${encodeURIComponent(did)}`;
}

export async function searchActors(
  query: string,
): Promise<{ actors: ActorSearchItem[] }> {
  try {
    const res = await fetch(
      `https://public.api.bsky.app/xrpc/app.bsky.actor.searchActorsTypeahead?q=${encodeURIComponent(query)}&limit=5`,
    );
    if (!res.ok) throw new Error("Search failed");
    return await res.json();
  } catch (e) {
    console.error("Failed to search actors:", e);
    return { actors: [] };
  }
}

export async function resolveHandle(handle: string): Promise<string | null> {
  if (handle.startsWith("did:")) return handle;
  try {
    const res = await fetch(
      `https://public.api.bsky.app/xrpc/com.atproto.identity.resolveHandle?handle=${encodeURIComponent(handle)}`,
    );
    if (!res.ok) throw new Error("Failed to resolve handle");
    const data = await res.json();
    return data.did;
  } catch (e) {
    console.error("Failed to resolve handle:", e);
    return null;
  }
}

export async function startLogin(
  handle: string,
): Promise<{ authorizationUrl?: string }> {
  const res = await apiRequest("/auth/start", {
    method: "POST",
    body: JSON.stringify({ handle }),
  });
  if (!res.ok) throw new Error("Failed to start login");
  return await res.json();
}

export async function startSignup(
  pdsUrl: string,
): Promise<{ authorizationUrl?: string }> {
  const res = await apiRequest("/auth/signup", {
    method: "POST",
    body: JSON.stringify({ pds_url: pdsUrl }),
  });
  if (!res.ok) throw new Error("Failed to start signup");
  return await res.json();
}

export async function getNotifications(
  limit = 50,
  offset = 0,
): Promise<NotificationItem[]> {
  try {
    const res = await apiRequest(
      `/api/notifications?limit=${limit}&offset=${offset}`,
    );
    if (!res.ok) throw new Error("Failed to fetch notifications");
    const data = await res.json();
    return (data.items || []).map((n: NotificationItem) => ({
      ...n,
      subject: n.subject ? normalizeItem(n.subject as RawItem) : undefined,
    }));
  } catch (e) {
    console.error("Failed to fetch notifications:", e);
    return [];
  }
}

export async function getUnreadNotificationCount(): Promise<number> {
  try {
    const res = await apiRequest("/api/notifications/count", {
      skipAuthRedirect: true,
    });
    if (!res.ok) return 0;
    const data = await res.json();
    return data.count || 0;
  } catch (e) {
    console.error("Failed to fetch unread notification count:", e);
    return 0;
  }
}

export async function markNotificationsRead(): Promise<boolean> {
  try {
    const res = await apiRequest("/api/notifications/read", { method: "POST" });
    return res.ok;
  } catch (e) {
    console.error("Failed to mark notifications as read:", e);
    return false;
  }
}

export interface APIKey {
  id: string;
  name: string;
  key?: string;
  createdAt: string;
}

export async function getAPIKeys(): Promise<APIKey[]> {
  try {
    const res = await apiRequest("/api/keys");
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : data.keys || [];
  } catch (e) {
    console.error("Failed to fetch API keys:", e);
    return [];
  }
}

export async function createAPIKey(name: string): Promise<APIKey | null> {
  try {
    const res = await apiRequest("/api/keys", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to create API key:", e);
    return null;
  }
}

export async function deleteAPIKey(id: string): Promise<boolean> {
  try {
    const res = await apiRequest(`/api/keys/${id}`, { method: "DELETE" });
    return res.ok;
  } catch (e) {
    console.error("Failed to delete API key:", e);
    return false;
  }
}

export interface Tag {
  tag: string;
  count: number;
}

export async function getTrendingTags(limit = 50): Promise<Tag[]> {
  try {
    const res = await apiRequest(`/api/trending-tags?limit=${limit}`, {
      skipAuthRedirect: true,
    });
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : data.tags || [];
  } catch (e) {
    console.error("Failed to fetch trending tags:", e);
    return [];
  }
}

export async function getUserTags(did: string, limit = 50): Promise<string[]> {
  try {
    const res = await apiRequest(`/api/users/${did}/tags?limit=${limit}`, {
      skipAuthRedirect: true,
    });
    if (!res.ok) return [];
    const data = await res.json();
    return (data || []).map((t: Tag) => t.tag);
  } catch (e) {
    console.error("Failed to fetch user tags:", e);
    return [];
  }
}

export async function getCollections(creator?: string): Promise<Collection[]> {
  try {
    const query = creator ? `?author=${encodeURIComponent(creator)}` : "";
    const res = await apiRequest(`/api/collections${query}`);
    if (!res.ok) throw new Error("Failed to fetch collections");
    const data = await res.json();
    let items = Array.isArray(data)
      ? data
      : data.items || data.collections || [];

    items = items.map((item: Record<string, unknown>) => {
      if (!item.id && item.uri) {
        item.id = (item.uri as string).split("/").pop();
      }
      return item;
    });

    return items;
  } catch (e) {
    console.error(e);
    return [];
  }
}

export async function getCollection(uri: string): Promise<Collection | null> {
  try {
    const res = await apiRequest(
      `/api/collection?uri=${encodeURIComponent(uri)}`,
    );
    if (!res.ok) throw new Error("Failed to fetch collection");
    return await res.json();
  } catch (e) {
    console.error(e);
    return null;
  }
}

export async function createCollection(
  name: string,
  description?: string,
  icon?: string,
): Promise<Collection | null> {
  try {
    const res = await apiRequest("/api/collections", {
      method: "POST",
      body: JSON.stringify({ name, description, icon }),
    });
    if (!res.ok) throw new Error("Failed to create collection");
    return await res.json();
  } catch (e) {
    console.error(e);
    return null;
  }
}

export async function deleteCollection(id: string): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/collections?uri=${encodeURIComponent(id)}`,
      { method: "DELETE" },
    );
    return res.ok;
  } catch (e) {
    console.error(e);
    return false;
  }
}

export async function getCollectionItems(
  uri: string,
): Promise<AnnotationItem[]> {
  try {
    const res = await apiRequest(
      `/api/collections/${encodeURIComponent(uri)}/items`,
    );
    if (!res.ok) throw new Error("Failed to fetch collection items");
    const data = await res.json();
    return (data || []).map(normalizeItem);
  } catch (e) {
    console.error(e);
    return [];
  }
}

export async function updateCollection(
  uri: string,
  name: string,
  description?: string,
  icon?: string,
): Promise<Collection | null> {
  try {
    const res = await apiRequest(
      `/api/collections?uri=${encodeURIComponent(uri)}`,
      {
        method: "PUT",
        body: JSON.stringify({ name, description, icon }),
      },
    );
    if (!res.ok) throw new Error("Failed to update collection");
    return await res.json();
  } catch (e) {
    console.error(e);
    return null;
  }
}

export async function addCollectionItem(
  collectionUri: string,
  annotationUri: string,
  position: number = 0,
): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/collections/${encodeURIComponent(collectionUri)}/items`,
      {
        method: "POST",
        body: JSON.stringify({ annotationUri, position }),
      },
    );
    return res.ok;
  } catch (e) {
    console.error(e);
    return false;
  }
}

export async function removeCollectionItem(itemUri: string): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/collections/items?uri=${encodeURIComponent(itemUri)}`,
      {
        method: "DELETE",
      },
    );
    return res.ok;
  } catch (e) {
    console.error(e);
    return false;
  }
}

export async function createReply(
  parentUri: string,
  parentCid: string,
  rootUri: string,
  rootCid: string,
  text: string,
): Promise<string | null> {
  try {
    const res = await apiRequest("/api/notes/reply", {
      method: "POST",
      body: JSON.stringify({ parentUri, parentCid, rootUri, rootCid, text }),
    });
    if (!res.ok) throw new Error("Failed to create reply");
    const data = await res.json();
    return data.uri;
  } catch (e) {
    console.error(e);
    return null;
  }
}

export async function deleteReply(uri: string): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/notes/reply?uri=${encodeURIComponent(uri)}`,
      {
        method: "DELETE",
      },
    );
    return res.ok;
  } catch (e) {
    console.error(e);
    return false;
  }
}

export async function getAnnotation(
  uri: string,
): Promise<AnnotationItem | null> {
  try {
    const res = await apiRequest(`/api/note?uri=${encodeURIComponent(uri)}`);
    if (!res.ok) return null;
    return normalizeItem(await res.json());
  } catch {
    return null;
  }
}

export async function getReplies(
  uri: string,
): Promise<{ items: AnnotationItem[] }> {
  try {
    const res = await apiRequest(`/api/replies?uri=${encodeURIComponent(uri)}`);
    if (!res.ok) return { items: [] };
    const data = await res.json();
    return { items: (data.items || []).map(normalizeItem) };
  } catch {
    return { items: [] };
  }
}

export async function getByTarget(
  url: string,
  limit = 50,
  offset = 0,
): Promise<{ annotations: AnnotationItem[]; highlights: AnnotationItem[] }> {
  try {
    const res = await apiRequest(
      `/api/targets?source=${encodeURIComponent(url)}&limit=${limit}&offset=${offset}`,
    );
    if (!res.ok) return { annotations: [], highlights: [] };
    const data = await res.json();
    return {
      annotations: (data.annotations || []).map(normalizeItem),
      highlights: (data.highlights || []).map(normalizeItem),
    };
  } catch {
    return { annotations: [], highlights: [] };
  }
}

export async function getUserTargetItems(
  did: string,
  url: string,
  limit = 50,
  offset = 0,
): Promise<{ annotations: AnnotationItem[]; highlights: AnnotationItem[] }> {
  try {
    const res = await apiRequest(
      `/api/users/${encodeURIComponent(did)}/targets?source=${encodeURIComponent(url)}&limit=${limit}&offset=${offset}`,
    );
    if (!res.ok) return { annotations: [], highlights: [] };
    const data = await res.json();
    return {
      annotations: (data.annotations || []).map(normalizeItem),
      highlights: (data.highlights || []).map(normalizeItem),
    };
  } catch {
    return { annotations: [], highlights: [] };
  }
}

import type {
  LabelerInfo,
  LabelerSubscription,
  LabelPreference,
} from "../types";

export interface PreferencesResponse {
  externalLinkSkippedHostnames?: string[];
  subscribedLabelers?: LabelerSubscription[];
  labelPreferences?: LabelPreference[];
  disableExternalLinkWarning?: boolean;
  enableCommunityBookmarks?: boolean;
}

export async function getPreferences(): Promise<PreferencesResponse> {
  try {
    const res = await apiRequest("/api/preferences", {
      skipAuthRedirect: true,
    });
    if (!res.ok) return {};
    return await res.json();
  } catch (e) {
    console.error(e);
    return {};
  }
}

export async function updatePreferences(prefs: {
  externalLinkSkippedHostnames?: string[];
  subscribedLabelers?: LabelerSubscription[];
  labelPreferences?: LabelPreference[];
  disableExternalLinkWarning?: boolean;
  enableCommunityBookmarks?: boolean;
}): Promise<boolean> {
  try {
    const res = await apiRequest("/api/preferences", {
      method: "PUT",
      body: JSON.stringify(prefs),
    });
    return res.ok;
  } catch (e) {
    console.error(e);
    return false;
  }
}

export async function getLabelerInfo(): Promise<LabelerInfo | null> {
  try {
    const res = await apiRequest("/moderation/labeler", {
      skipAuthRedirect: true,
    });
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch labeler info:", e);
    return null;
  }
}

import type {
  BlockedUser,
  ModerationRelationship,
  ModerationReport,
  MutedUser,
  ReportReasonType,
} from "../types";

export async function blockUser(did: string): Promise<boolean> {
  try {
    const res = await apiRequest("/api/moderation/block", {
      method: "POST",
      body: JSON.stringify({ did }),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to block user:", e);
    return false;
  }
}

export async function unblockUser(did: string): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/moderation/block?did=${encodeURIComponent(did)}`,
      { method: "DELETE" },
    );
    return res.ok;
  } catch (e) {
    console.error("Failed to unblock user:", e);
    return false;
  }
}

export async function getBlocks(): Promise<BlockedUser[]> {
  try {
    const res = await apiRequest("/api/moderation/blocks");
    if (!res.ok) return [];
    const data = await res.json();
    return data.items || [];
  } catch (e) {
    console.error("Failed to fetch blocks:", e);
    return [];
  }
}

export async function muteUser(did: string): Promise<boolean> {
  try {
    const res = await apiRequest("/api/moderation/mute", {
      method: "POST",
      body: JSON.stringify({ did }),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to mute user:", e);
    return false;
  }
}

export async function unmuteUser(did: string): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/moderation/mute?did=${encodeURIComponent(did)}`,
      { method: "DELETE" },
    );
    return res.ok;
  } catch (e) {
    console.error("Failed to unmute user:", e);
    return false;
  }
}

export async function getMutes(): Promise<MutedUser[]> {
  try {
    const res = await apiRequest("/api/moderation/mutes");
    if (!res.ok) return [];
    const data = await res.json();
    return data.items || [];
  } catch (e) {
    console.error("Failed to fetch mutes:", e);
    return [];
  }
}

export async function getModerationRelationship(
  did: string,
): Promise<ModerationRelationship> {
  try {
    const res = await apiRequest(
      `/api/moderation/relationship?did=${encodeURIComponent(did)}`,
      { skipAuthRedirect: true },
    );
    if (!res.ok) return { blocking: false, muting: false, blockedBy: false };
    return await res.json();
  } catch (e) {
    console.error("Failed to get moderation relationship:", e);
    return { blocking: false, muting: false, blockedBy: false };
  }
}

export async function reportUser(params: {
  subjectDid: string;
  subjectUri?: string;
  reasonType: ReportReasonType;
  reasonText?: string;
}): Promise<boolean> {
  try {
    const res = await apiRequest("/api/moderation/report", {
      method: "POST",
      body: JSON.stringify(params),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to submit report:", e);
    return false;
  }
}

export async function checkAdminAccess(): Promise<boolean> {
  try {
    const res = await apiRequest("/api/moderation/admin/check", {
      skipAuthRedirect: true,
    });
    if (!res.ok) return false;
    const data = await res.json();
    return data.isAdmin || false;
  } catch {
    return false;
  }
}

export async function getAdminReports(
  status?: string,
  limit = 50,
  offset = 0,
): Promise<{
  items: ModerationReport[];
  totalItems: number;
  pendingCount: number;
}> {
  try {
    const params = new URLSearchParams();
    if (status) params.append("status", status);
    params.append("limit", limit.toString());
    params.append("offset", offset.toString());
    const res = await apiRequest(
      `/api/moderation/admin/reports?${params.toString()}`,
    );
    if (!res.ok) return { items: [], totalItems: 0, pendingCount: 0 };
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch admin reports:", e);
    return { items: [], totalItems: 0, pendingCount: 0 };
  }
}

export async function adminTakeAction(params: {
  reportId: number;
  action: string;
  comment?: string;
}): Promise<boolean> {
  try {
    const res = await apiRequest("/api/moderation/admin/action", {
      method: "POST",
      body: JSON.stringify(params),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to take moderation action:", e);
    return false;
  }
}

export async function adminCreateLabel(params: {
  src: string;
  uri?: string;
  val: string;
}): Promise<boolean> {
  try {
    const res = await apiRequest("/api/moderation/admin/label", {
      method: "POST",
      body: JSON.stringify(params),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to create label:", e);
    return false;
  }
}

export async function adminDeleteLabel(id: number): Promise<boolean> {
  try {
    const res = await apiRequest(`/api/moderation/admin/label?id=${id}`, {
      method: "DELETE",
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to delete label:", e);
    return false;
  }
}

export async function adminGetLabels(
  limit = 50,
  offset = 0,
): Promise<{ items: HydratedLabel[] }> {
  try {
    const res = await apiRequest(
      `/api/moderation/admin/labels?limit=${limit}&offset=${offset}`,
    );
    if (!res.ok) return { items: [] };
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch labels:", e);
    return { items: [] };
  }
}

export interface BannedAccount {
  did: string;
  reason?: string;
  bannedBy: string;
  bannedAt: string;
  profile?: {
    did: string;
    handle: string;
    displayName?: string;
    avatar?: string;
  };
}

export async function adminBanAccount(params: {
  did: string;
  reason?: string;
}): Promise<boolean> {
  try {
    const res = await apiRequest("/api/moderation/admin/ban", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(params),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to ban account:", e);
    return false;
  }
}

export async function adminUnbanAccount(did: string): Promise<boolean> {
  try {
    const res = await apiRequest(
      `/api/moderation/admin/ban?did=${encodeURIComponent(did)}`,
      { method: "DELETE" },
    );
    return res.ok;
  } catch (e) {
    console.error("Failed to unban account:", e);
    return false;
  }
}

export async function adminGetBannedAccounts(): Promise<{
  items: BannedAccount[];
  total: number;
}> {
  try {
    const res = await apiRequest("/api/moderation/admin/bans");
    if (!res.ok) return { items: [], total: 0 };
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch banned accounts:", e);
    return { items: [], total: 0 };
  }
}

export interface DocumentItem {
  uri: string;
  authorDid: string;
  site: string;
  path?: string;
  title: string;
  description?: string;
  tags?: string[];
  canonicalUrl: string;
  publishedAt: string;
}

export interface DocumentsResponse {
  items: DocumentItem[];
  totalItems: number;
}

export async function getDocuments({
  sort = "new",
  limit = 30,
  offset = 0,
}: {
  sort?: string;
  limit?: number;
  offset?: number;
}): Promise<DocumentsResponse> {
  try {
    const params = new URLSearchParams();
    if (sort) params.append("sort", sort);
    params.append("limit", limit.toString());
    params.append("offset", offset.toString());

    const res = await apiRequest(`/api/documents?${params.toString()}`, {
      skipAuthRedirect: true,
    });
    if (!res.ok) throw new Error("Failed to fetch documents");
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch documents:", e);
    return { items: [], totalItems: 0 };
  }
}

export async function getRecommendations(
  limit = 20,
): Promise<DocumentsResponse & { unavailable?: boolean }> {
  try {
    const res = await apiRequest(`/api/recommendations?limit=${limit}`);
    if (res.status === 503)
      return { items: [], totalItems: 0, unavailable: true };
    if (!res.ok) throw new Error("Failed to fetch recommendations");
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch recommendations:", e);
    return { items: [], totalItems: 0, unavailable: true };
  }
}

export interface ReadingRoomTheme {
  backgroundColor: string;
  accentColor: string;
  fontFamily: string;
  layout: string;
}

export interface ReadingRoomConfig {
  enabled: boolean;
  title: string;
  subtitle: string;
  description: string;
  theme: ReadingRoomTheme;
  featuredUris: string[];
  hasSubscription: boolean;
  showExternalBookmarks: boolean;
}

export interface ReadingRoomPublic {
  handle: string;
  did: string;
  title: string;
  subtitle: string;
  description: string;
  theme: ReadingRoomTheme;
  displayName: string;
  avatar: string;
  bio: string;
  featured: AnnotationItem[];
  recent: AnnotationItem[];
  totalCount: number;
  typeCounts?: { highlight: number; note: number; bookmark: number };
}

export interface BillingStatus {
  status: string;
  plan: string;
  currentPeriodEnd?: string;
  hasSubscription: boolean;
}

export interface CustomDomainInfo {
  domain: string;
  status: string;
  verificationRecords: Array<{
    type: string;
    name?: string;
    value?: string;
    httpUrl?: string;
    httpBody?: string;
  }>;
}

export async function getPublicReadingRoom(
  handle: string,
): Promise<ReadingRoomPublic | null> {
  try {
    const res = await apiRequest(`/api/reading-room/${handle}`, {
      skipAuthRedirect: true,
    });
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch reading room:", e);
    return null;
  }
}

export interface ReadingRoomNotePublic {
  handle: string;
  did: string;
  roomTitle: string;
  displayName: string;
  avatar: string;
  theme: ReadingRoomTheme;
  note: AnnotationItem;
}

export async function getReadingRoomNote(
  handle: string,
  uri: string,
): Promise<ReadingRoomNotePublic | null> {
  try {
    const res = await apiRequest(
      `/api/reading-room/${handle}/note?uri=${encodeURIComponent(uri)}`,
      { skipAuthRedirect: true },
    );
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch reading room note:", e);
    return null;
  }
}

export interface ReadingRoomNotesPage {
  notes: AnnotationItem[];
  totalCount: number;
}

export async function getReadingRoomNotes(
  handle: string,
  offset: number,
  limit = 20,
): Promise<ReadingRoomNotesPage | null> {
  try {
    const res = await apiRequest(
      `/api/reading-room/${handle}/notes?offset=${offset}&limit=${limit}`,
      { skipAuthRedirect: true },
    );
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch reading room notes:", e);
    return null;
  }
}

export async function getReadingRoomConfig(): Promise<ReadingRoomConfig | null> {
  try {
    const res = await apiRequest("/api/reading-room/config");
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch reading room config:", e);
    return null;
  }
}

export async function updateReadingRoomConfig(
  config: Partial<ReadingRoomConfig>,
): Promise<boolean> {
  try {
    const res = await apiRequest("/api/reading-room/config", {
      method: "PUT",
      body: JSON.stringify(config),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to update reading room config:", e);
    return false;
  }
}

export async function updateFeaturedItems(uris: string[]): Promise<boolean> {
  try {
    const res = await apiRequest("/api/reading-room/featured", {
      method: "PUT",
      body: JSON.stringify({ uris }),
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to update featured items:", e);
    return false;
  }
}

export async function getCustomDomainStatus(): Promise<CustomDomainInfo | null> {
  try {
    const res = await apiRequest("/api/reading-room/domain");
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch custom domain status:", e);
    return null;
  }
}

export async function addCustomDomain(
  domain: string,
): Promise<CustomDomainInfo | null> {
  try {
    const res = await apiRequest("/api/reading-room/domain", {
      method: "POST",
      body: JSON.stringify({ domain }),
    });
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to add custom domain:", e);
    return null;
  }
}

export async function pollCustomDomain(): Promise<CustomDomainInfo | null> {
  try {
    const res = await apiRequest("/api/reading-room/domain/poll", {
      method: "POST",
    });
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to poll custom domain:", e);
    return null;
  }
}

export async function removeCustomDomain(): Promise<boolean> {
  try {
    const res = await apiRequest("/api/reading-room/domain", {
      method: "DELETE",
    });
    return res.ok;
  } catch (e) {
    console.error("Failed to remove custom domain:", e);
    return false;
  }
}

export async function createCheckout(plan: string): Promise<string | null> {
  try {
    const res = await apiRequest("/api/billing/checkout", {
      method: "POST",
      body: JSON.stringify({ plan }),
    });
    if (!res.ok) return null;
    const data = await res.json();
    return data.url;
  } catch (e) {
    console.error("Failed to create checkout:", e);
    return null;
  }
}

export async function createPortal(): Promise<string | null> {
  try {
    const res = await apiRequest("/api/billing/portal", {
      method: "POST",
    });
    if (!res.ok) return null;
    const data = await res.json();
    return data.url;
  } catch (e) {
    console.error("Failed to create portal session:", e);
    return null;
  }
}

export async function getBillingStatus(): Promise<BillingStatus | null> {
  try {
    const res = await apiRequest("/api/billing/status");
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    console.error("Failed to fetch billing status:", e);
    return null;
  }
}
