import type { UserProfile } from "../types";

const API_URL =
  process.env.API_URL || `http://localhost:${process.env.API_PORT || 8081}`;

function serverFetch(path: string, cookie?: string): Promise<Response> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (cookie) headers["Cookie"] = cookie;
  return fetch(`${API_URL}${path}`, { headers });
}

export function clearSessionCacheForCookie(_cookie: string) {}

export async function getSession(cookie: string): Promise<UserProfile | null> {
  try {
    const res = await serverFetch("/auth/session", cookie);
    if (!res.ok) return null;
    const data = await res.json();
    if (!data.authenticated && !data.did) return null;

    const profile: UserProfile = {
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

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 3000);

    Promise.allSettled([
      fetch(
        `https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile?actor=${encodeURIComponent(data.did)}`,
        { signal: controller.signal },
      ),
      serverFetch(`/api/profile/${data.did}`, cookie),
    ])
      .then(([bskyRes, marginRes]) => {
        clearTimeout(timeout);

        if (bskyRes.status === "fulfilled" && bskyRes.value.ok) {
          bskyRes.value
            .json()
            .then((bsky) => {
              if (bsky.avatar) profile.avatar = bsky.avatar;
              if (bsky.displayName) profile.displayName = bsky.displayName;
            })
            .catch(() => {
              /* ignore */
            });
        }

        if (marginRes.status === "fulfilled" && marginRes.value.ok) {
          marginRes.value
            .json()
            .then((mp) => {
              if (mp?.description) profile.description = mp.description;
              if (mp?.followersCount)
                profile.followersCount = mp.followersCount;
              if (mp?.followsCount) profile.followsCount = mp.followsCount;
              if (mp?.postsCount) profile.postsCount = mp.postsCount;
              if (mp?.website) profile.website = mp.website;
              if (mp?.links) profile.links = mp.links;
            })
            .catch(() => {
              /* ignore */
            });
        }
      })
      .catch(() => {
        clearTimeout(timeout);
      });

    return profile;
  } catch {
    return null;
  }
}
