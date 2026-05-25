import type { MarginSession, TextSelector, Annotation } from './types';
import { apiUrlItem } from './storage';

async function safariFetch(url: string, options: RequestInit = {}): Promise<Response> {
  try {
    const tabs = await browser.tabs.query({ url: 'https://margin.at/*' });
    const tab = tabs[0];
    if (!tab?.id) return new Response('{}', { status: 401 });

    const result = (await browser.tabs.sendMessage(tab.id, {
      type: 'SAFARI_FETCH',
      url,
      options: {
        method: options.method || 'GET',
        headers: options.headers || {},
        body: options.body,
      },
    })) as { ok: boolean; status: number; body: string };

    return new Response(result.body, { status: result.status });
  } catch {
    return new Response('{}', { status: 500 });
  }
}

async function getApiUrl(): Promise<string> {
  return await apiUrlItem.getValue();
}

async function getSessionCookie(): Promise<string | null> {
  try {
    const apiUrl = await getApiUrl();

    const readFrom = async (storeId?: string) => {
      const cookie = await browser.cookies.get({
        url: apiUrl,
        name: 'margin_session',
        ...(storeId ? { storeId } : {}),
      });
      return cookie?.value || null;
    };

    const defaultValue = await readFrom();
    if (defaultValue) return defaultValue;

    if (import.meta.env.BROWSER !== 'safari') {
      let activeStoreId: string | undefined;
      try {
        const [activeTab] = await browser.tabs.query({ active: true, currentWindow: true });
        activeStoreId = (activeTab as { cookieStoreId?: string } | undefined)?.cookieStoreId;
      } catch {
        // ignore: tabs.query can fail in some contexts
      }

      if (activeStoreId) {
        const value = await readFrom(activeStoreId);
        if (value) return value;
      }

      try {
        const stores = await browser.cookies.getAllCookieStores();
        for (const store of stores) {
          if (store.id === activeStoreId) continue;
          const value = await readFrom(store.id);
          if (value) return value;
        }
      } catch {
        // ignore: getAllCookieStores may not be supported everywhere
      }
    }

    return null;
  } catch (error) {
    console.error('Get cookie error:', error);
    return null;
  }
}

export async function checkSession(): Promise<MarginSession> {
  try {
    const apiUrl = await getApiUrl();

    if (import.meta.env.BROWSER === 'safari') {
      const res = await safariFetch(`${apiUrl}/auth/session`);
      if (!res.ok) return { authenticated: false };
      const sessionData = await res.json();
      if (!sessionData.did || !sessionData.handle) return { authenticated: false };
      return {
        authenticated: true,
        did: sessionData.did,
        handle: sessionData.handle,
        accessJwt: sessionData.accessJwt,
        refreshJwt: sessionData.refreshJwt,
      };
    }

    const cookie = await getSessionCookie();

    if (!cookie) return { authenticated: false };

    const res = await fetch(`${apiUrl}/auth/session`, {
      headers: { 'X-Session-Token': cookie },
    });

    if (!res.ok) return { authenticated: false };

    const sessionData = await res.json();
    if (!sessionData.did || !sessionData.handle) return { authenticated: false };

    return {
      authenticated: true,
      did: sessionData.did,
      handle: sessionData.handle,
      accessJwt: sessionData.accessJwt,
      refreshJwt: sessionData.refreshJwt,
    };
  } catch (error) {
    console.error('Session check error:', error);
    return { authenticated: false };
  }
}

async function apiRequest(path: string, options: RequestInit = {}): Promise<Response> {
  const apiUrl = await getApiUrl();
  const apiPath = path.startsWith('/api') ? path : `/api${path}`;

  if (import.meta.env.BROWSER === 'safari') {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };
    return safariFetch(`${apiUrl}${apiPath}`, { ...options, headers });
  }

  const cookie = await getSessionCookie();

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (cookie) {
    headers['X-Session-Token'] = cookie;
  }

  return fetch(`${apiUrl}${apiPath}`, {
    ...options,
    headers,
    credentials: 'include',
  });
}

async function hashUrl(rawUrl: string): Promise<string> {
  let toHash: string;
  try {
    const parsed = new URL(rawUrl);
    let host = parsed.host.toLowerCase();
    if (host.startsWith('www.')) host = host.slice(4);
    let normalized = host + parsed.pathname;
    if (parsed.search) normalized += parsed.search;
    normalized = normalized.replace(/\/$/, '');
    toHash = normalized;
  } catch {
    toHash = rawUrl;
  }
  const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(toHash));
  return Array.from(new Uint8Array(buf))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

export async function getAnnotations(
  url: string,
  citedUrls: string[] = [],
  cacheBust: boolean = false
) {
  try {
    const apiUrl = await getApiUrl();
    const uniqueUrls = [...new Set([url, ...citedUrls])];

    const hashes = await Promise.all(uniqueUrls.map(hashUrl));

    const params = new URLSearchParams();
    hashes.forEach((h) => params.append('h', h));
    if (cacheBust) params.append('t', Date.now().toString());

    const res = await fetch(`${apiUrl}/api/targets/hash?${params}`);
    const data = res.ok ? await res.json() : { annotations: [], highlights: [], bookmarks: [] };

    const allItems: any[] = [];
    const seenIds = new Set<string>();
    const items = [
      ...(data.annotations || []),
      ...(data.highlights || []),
      ...(data.bookmarks || []),
    ];
    items.forEach((item: any) => {
      const id = item.uri || item.id;
      if (id && !seenIds.has(id)) {
        seenIds.add(id);
        allItems.push(item);
      }
    });

    return allItems;
  } catch (error) {
    console.error('Get annotations error:', error);
    return [];
  }
}

export async function createAnnotation(data: {
  url: string;
  text: string;
  title?: string;
  selector?: TextSelector;
  tags?: string[];
}): Promise<{ success: boolean; data?: Annotation; error?: string }> {
  try {
    const res = await apiRequest('/annotations', {
      method: 'POST',
      body: JSON.stringify({
        url: data.url,
        text: data.text,
        title: data.title,
        selector: data.selector,
        tags: data.tags,
      }),
    });

    if (!res.ok) {
      const error = await res.text();
      return { success: false, error };
    }

    return { success: true, data: await res.json() };
  } catch (error) {
    return { success: false, error: String(error) };
  }
}

export async function createBookmark(data: { url: string; title?: string; tags?: string[] }) {
  try {
    const res = await apiRequest('/bookmarks', {
      method: 'POST',
      body: JSON.stringify({ url: data.url, title: data.title, tags: data.tags }),
    });

    if (!res.ok) {
      const error = await res.text();
      return { success: false, error };
    }

    return { success: true, data: await res.json() };
  } catch (error) {
    return { success: false, error: String(error) };
  }
}

export async function createHighlight(data: {
  url: string;
  title?: string;
  selector: TextSelector;
  color?: string;
  tags?: string[];
}) {
  try {
    const res = await apiRequest('/highlights', {
      method: 'POST',
      body: JSON.stringify({
        url: data.url,
        title: data.title,
        selector: data.selector,
        color: data.color,
        tags: data.tags,
      }),
    });

    if (!res.ok) {
      const error = await res.text();
      return { success: false, error };
    }

    return { success: true, data: await res.json() };
  } catch (error) {
    return { success: false, error: String(error) };
  }
}

export async function updateHighlightTags(uri: string, tags: string[]) {
  try {
    const res = await apiRequest(`/highlights?uri=${encodeURIComponent(uri)}`, {
      method: 'PUT',
      body: JSON.stringify({ tags }),
    });
    if (!res.ok) {
      const error = await res.text();
      return { success: false, error };
    }
    return { success: true };
  } catch (error) {
    return { success: false, error: String(error) };
  }
}

export async function getUserBookmarks(did: string) {
  try {
    const res = await apiRequest(`/users/${did}/bookmarks`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.items || data || [];
  } catch (error) {
    console.error('Get bookmarks error:', error);
    return [];
  }
}

export async function getUserHighlights(did: string) {
  try {
    const res = await apiRequest(`/users/${did}/highlights`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.items || data || [];
  } catch (error) {
    console.error('Get highlights error:', error);
    return [];
  }
}

export async function getUserCollections(did: string) {
  try {
    const res = await apiRequest(`/collections?author=${encodeURIComponent(did)}`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.items || data || [];
  } catch (error) {
    console.error('Get collections error:', error);
    return [];
  }
}

export async function addToCollection(collectionUri: string, annotationUri: string) {
  try {
    const res = await apiRequest(`/collections/${encodeURIComponent(collectionUri)}/items`, {
      method: 'POST',
      body: JSON.stringify({ annotationUri, position: 0 }),
    });

    if (!res.ok) {
      const error = await res.text();
      return { success: false, error };
    }

    return { success: true };
  } catch (error) {
    return { success: false, error: String(error) };
  }
}

export async function getItemCollections(annotationUri: string): Promise<string[]> {
  try {
    const res = await apiRequest(
      `/collections/containing?uri=${encodeURIComponent(annotationUri)}`
    );
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  } catch (error) {
    console.error('Get item collections error:', error);
    return [];
  }
}

export async function deleteHighlight(uri: string) {
  try {
    const rkey = (uri || '').split('/').pop();
    if (!rkey) return { success: false, error: 'Invalid URI' };

    const res = await apiRequest(`/highlights?rkey=${rkey}`, {
      method: 'DELETE',
    });

    if (!res.ok) {
      const error = await res.text();
      return { success: false, error };
    }

    return { success: true };
  } catch (error) {
    return { success: false, error: String(error) };
  }
}

export async function getUserTags(did: string) {
  try {
    const res = await apiRequest(`/users/${did}/tags?limit=50`);
    if (!res.ok) return [];
    const data = await res.json();
    return (data || []).map((t: { tag: string }) => t.tag);
  } catch (error) {
    console.error('Get user tags error:', error);
    return [];
  }
}

export async function getTrendingTags() {
  try {
    const res = await apiRequest('/trending-tags?limit=50');
    if (!res.ok) return [];
    const data = await res.json();
    return (data || []).map((t: { tag: string }) => t.tag);
  } catch (error) {
    console.error('Get trending tags error:', error);
    return [];
  }
}

export async function getReplies(uri: string) {
  try {
    const res = await apiRequest(`/annotations/${encodeURIComponent(uri)}/replies`);
    if (!res.ok) return [];
    const data = await res.json();
    return data.items || data || [];
  } catch (error) {
    console.error('Get replies error:', error);
    return [];
  }
}

export async function createReply(data: {
  parentUri: string;
  parentCid: string;
  rootUri: string;
  rootCid: string;
  text: string;
}) {
  try {
    const res = await apiRequest('/replies', {
      method: 'POST',
      body: JSON.stringify(data),
    });

    if (!res.ok) {
      const error = await res.text();
      return { success: false, error };
    }

    return { success: true };
  } catch (error) {
    return { success: false, error: String(error) };
  }
}
