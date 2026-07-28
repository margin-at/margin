const PUBLIC_API = "https://public.api.bsky.app/xrpc";

export interface SocialPostRef {
  host: string;
  actor: string;
  rkey: string;
}

export interface SocialPostAuthor {
  did: string;
  handle: string;
  displayName?: string;
  avatar?: string;
}

export interface SocialPostImage {
  thumb: string;
  fullsize: string;
  alt?: string;
}

export interface SocialPostFacet {
  byteStart: number;
  byteEnd: number;
  link?: string;
  mention?: string;
  tag?: string;
}

export interface FacetSegment {
  text: string;
  facet?: SocialPostFacet;
}

export interface SocialPostGif {
  playerUri: string;
  sources?: { src: string; type: string }[];
  isVideo: boolean;
  width?: number;
  height?: number;
  alt?: string;
}

export interface SocialPost {
  uri: string;
  author: SocialPostAuthor;
  text: string;
  facets?: SocialPostFacet[];
  createdAt?: string;
  images: SocialPostImage[];
  video?: { thumbnail?: string };
  gif?: SocialPostGif;
  external?: {
    uri: string;
    title?: string;
    description?: string;
    thumb?: string;
  };
  quote?: {
    author: SocialPostAuthor;
    text: string;
    facets?: SocialPostFacet[];
    createdAt?: string;
  };
  replyCount?: number;
  repostCount?: number;
  likeCount?: number;
}

export function parseSocialPostUrl(
  url: string | null | undefined,
): SocialPostRef | null {
  if (!url) return null;
  try {
    const u = new URL(url);
    if (u.protocol !== "https:" && u.protocol !== "http:") return null;
    const match = u.pathname.match(/^\/profile\/([^/]+)\/post\/([^/]+)\/?$/);
    if (!match) return null;
    return {
      host: u.hostname.replace(/^www\./, ""),
      actor: decodeURIComponent(match[1]),
      rkey: decodeURIComponent(match[2]),
    };
  } catch {
    return null;
  }
}

export function parseSocialPostAtUri(
  uri: string,
  host: string,
): SocialPostRef | null {
  const match = uri.match(/^at:\/\/([^/]+)\/app\.bsky\.feed\.post\/([^/]+)$/);
  if (!match) return null;
  return { host, actor: match[1], rkey: match[2] };
}

export function postRefFromAtTags(
  canonical: string | null | undefined,
  host: string,
): SocialPostRef | null {
  if (!canonical) return null;
  for (const uri of canonical.trim().split(/\s+/)) {
    const ref = parseSocialPostAtUri(uri, host);
    if (ref) return ref;
  }
  return null;
}

const giphyMediaHostRegex = /^media(?:[0-4])?\.giphy\.com$/i;
const gifFilenameRegex = /^\S+\.(webp|gif|mp4)$/i;

function giphyGif(gifId: string): Omit<SocialPostGif, "alt"> {
  return {
    playerUri: `https://i.giphy.com/media/${gifId}/200.webp`,
    isVideo: false,
  };
}

export function parseGifEmbed(
  url: string | null | undefined,
): Omit<SocialPostGif, "alt"> | null {
  if (!url) return null;
  let u: URL;
  try {
    u = new URL(url);
  } catch {
    return null;
  }

  if (u.hostname === "media.tenor.com") {
    const [, id, filename] = u.pathname.split("/");
    if (!id || !filename || !id.includes("AAAAC")) return null;
    const width = Number(u.searchParams.get("ww"));
    const height = Number(u.searchParams.get("hh"));
    if (!(width > 0) || !(height > 0)) return null;
    const webm = `https://t.gifs.bsky.app/${id.replace("AAAAC", "AAAP3")}/${filename.replace(".gif", ".webm")}`;
    const mp4 = `https://t.gifs.bsky.app/${id.replace("AAAAC", "AAAP1")}/${filename.replace(".gif", ".mp4")}`;
    return {
      playerUri: mp4,
      sources: [
        { src: webm, type: "video/webm" },
        { src: mp4, type: "video/mp4" },
      ],
      isVideo: true,
      width,
      height,
    };
  }

  if (u.hostname === "static.klipy.com" && u.pathname.startsWith("/ii/")) {
    const width = Number(u.searchParams.get("ww"));
    const height = Number(u.searchParams.get("hh"));
    if (!(width > 0) || !(height > 0)) return null;
    const webmSlug = u.searchParams.get("webm");
    const mp4Slug = u.searchParams.get("mp4");
    if (!webmSlug && !mp4Slug) return null;
    const buildVideoUrl = (slug: string, ext: string) => {
      const v = new URL(u.href);
      v.hostname = "k.gifs.bsky.app";
      for (const param of ["hh", "ww", "mp4", "webm"]) {
        v.searchParams.delete(param);
      }
      const parts = v.pathname.split("/");
      parts[parts.length - 1] = `${slug}.${ext}`;
      v.pathname = parts.join("/");
      return v.href;
    };
    const sources: { src: string; type: string }[] = [];
    if (webmSlug) {
      sources.push({
        src: buildVideoUrl(webmSlug, "webm"),
        type: "video/webm",
      });
    }
    if (mp4Slug) {
      sources.push({ src: buildVideoUrl(mp4Slug, "mp4"), type: "video/mp4" });
    }
    return {
      playerUri: mp4Slug
        ? buildVideoUrl(mp4Slug, "mp4")
        : buildVideoUrl(webmSlug as string, "webm"),
      sources,
      isVideo: true,
      width,
      height,
    };
  }

  if (u.hostname === "giphy.com" || u.hostname === "www.giphy.com") {
    const [, gifs, nameAndId] = u.pathname.split("/");
    if (gifs === "gifs" && nameAndId) {
      const gifId = nameAndId.split("-").pop();
      if (gifId) return giphyGif(gifId);
    }
    return null;
  }

  if (giphyMediaHostRegex.test(u.hostname)) {
    const [, media, trackingOrId, idOrFilename, filename] =
      u.pathname.split("/");
    if (media === "media") {
      if (idOrFilename && gifFilenameRegex.test(idOrFilename)) {
        return giphyGif(trackingOrId);
      }
      if (filename && gifFilenameRegex.test(filename)) {
        return giphyGif(idOrFilename);
      }
    }
    return null;
  }

  if (u.hostname === "i.giphy.com" || u.hostname === "www.i.giphy.com") {
    const [, mediaOrFilename, filename] = u.pathname.split("/");
    if (mediaOrFilename === "media" && filename) {
      return giphyGif(filename.split(".")[0]);
    }
    if (mediaOrFilename) {
      return giphyGif(mediaOrFilename.split(".")[0]);
    }
    return null;
  }

  return null;
}

function gifAltText(description: unknown): string | undefined {
  if (typeof description !== "string") return undefined;
  const alt = description.replace(/^(Alt|ALT): /, "").trim();
  return alt || undefined;
}

function pickAuthor(author: Record<string, unknown> | undefined) {
  return {
    did: (author?.did as string) || "",
    handle: (author?.handle as string) || "",
    displayName: author?.displayName as string | undefined,
    avatar: author?.avatar as string | undefined,
  };
}

export function segmentByFacets(
  text: string,
  facets: SocialPostFacet[] | undefined,
): FacetSegment[] {
  if (!facets?.length) return [{ text }];
  const bytes = new TextEncoder().encode(text);
  const decoder = new TextDecoder();
  const segments: FacetSegment[] = [];
  let cursor = 0;
  for (const facet of facets) {
    if (facet.byteStart < cursor || facet.byteEnd > bytes.length) continue;
    if (facet.byteStart > cursor) {
      segments.push({
        text: decoder.decode(bytes.subarray(cursor, facet.byteStart)),
      });
    }
    segments.push({
      text: decoder.decode(bytes.subarray(facet.byteStart, facet.byteEnd)),
      facet,
    });
    cursor = facet.byteEnd;
  }
  if (cursor < bytes.length) {
    segments.push({ text: decoder.decode(bytes.subarray(cursor)) });
  }
  return segments;
}

/* eslint-disable @typescript-eslint/no-explicit-any */
function extractFacets(record: any): SocialPostFacet[] | undefined {
  if (!Array.isArray(record?.facets)) return undefined;
  const out: SocialPostFacet[] = [];
  for (const facet of record.facets) {
    const index = facet?.index;
    if (
      typeof index?.byteStart !== "number" ||
      typeof index?.byteEnd !== "number" ||
      index.byteEnd <= index.byteStart
    ) {
      continue;
    }
    for (const feature of facet.features || []) {
      const base = { byteStart: index.byteStart, byteEnd: index.byteEnd };
      if (feature?.$type === "app.bsky.richtext.facet#link" && feature.uri) {
        out.push({ ...base, link: feature.uri });
        break;
      }
      if (feature?.$type === "app.bsky.richtext.facet#mention" && feature.did) {
        out.push({ ...base, mention: feature.did });
        break;
      }
      if (feature?.$type === "app.bsky.richtext.facet#tag" && feature.tag) {
        out.push({ ...base, tag: feature.tag });
        break;
      }
    }
  }
  if (out.length === 0) return undefined;
  return out.sort((a, b) => a.byteStart - b.byteStart);
}

function extractQuote(record: any): SocialPost["quote"] {
  if (!record || record.$type !== "app.bsky.embed.record#viewRecord") {
    return undefined;
  }
  return {
    author: pickAuthor(record.author),
    text: record.value?.text || "",
    facets: extractFacets(record.value),
    createdAt: record.value?.createdAt,
  };
}

function extractEmbed(embed: any, out: SocialPost) {
  if (!embed || typeof embed.$type !== "string") return;
  if (embed.$type === "app.bsky.embed.images#view") {
    out.images = (embed.images || [])
      .filter((img: any) => img?.thumb)
      .map((img: any) => ({
        thumb: img.thumb,
        fullsize: img.fullsize || img.thumb,
        alt: img.alt,
      }));
  } else if (embed.$type === "app.bsky.embed.video#view") {
    out.video = { thumbnail: embed.thumbnail };
  } else if (embed.$type === "app.bsky.embed.external#view") {
    if (embed.external?.uri) {
      const gif = parseGifEmbed(embed.external.uri);
      if (gif) {
        out.gif = { ...gif, alt: gifAltText(embed.external.description) };
      } else {
        out.external = {
          uri: embed.external.uri,
          title: embed.external.title,
          description: embed.external.description,
          thumb: embed.external.thumb,
        };
      }
    }
  } else if (embed.$type === "app.bsky.embed.record#view") {
    out.quote = extractQuote(embed.record);
  } else if (embed.$type === "app.bsky.embed.recordWithMedia#view") {
    extractEmbed(embed.media, out);
    out.quote = extractQuote(embed.record?.record);
  }
}

function simplify(post: any): SocialPost | null {
  if (!post?.uri || !post.author?.did) return null;
  const simplified: SocialPost = {
    uri: post.uri,
    author: pickAuthor(post.author),
    text: post.record?.text || "",
    facets: extractFacets(post.record),
    createdAt: post.record?.createdAt || post.indexedAt,
    images: [],
    replyCount: post.replyCount,
    repostCount: post.repostCount,
    likeCount: post.likeCount,
  };
  extractEmbed(post.embed, simplified);
  return simplified;
}

async function doFetch(ref: SocialPostRef): Promise<SocialPost | null> {
  let did = ref.actor;
  if (!did.startsWith("did:")) {
    const res = await fetch(
      `${PUBLIC_API}/com.atproto.identity.resolveHandle?handle=${encodeURIComponent(ref.actor)}`,
    );
    if (!res.ok) return null;
    did = (await res.json()).did;
    if (!did) return null;
  }
  const uri = `at://${did}/app.bsky.feed.post/${ref.rkey}`;
  const res = await fetch(
    `${PUBLIC_API}/app.bsky.feed.getPosts?uris=${encodeURIComponent(uri)}`,
  );
  if (!res.ok) return null;
  const data = await res.json();
  return simplify(data.posts?.[0]);
}
/* eslint-enable @typescript-eslint/no-explicit-any */

const inflight = new Map<string, Promise<SocialPost | null>>();

export function socialPostCacheKey(url: string): string {
  return `socialpost:v3:${url}`;
}

export function fetchSocialPost(
  url: string,
  ref: SocialPostRef,
): Promise<SocialPost | null> {
  const cacheKey = socialPostCacheKey(url);
  try {
    const cached = sessionStorage.getItem(cacheKey);
    if (cached) {
      return Promise.resolve(cached === "null" ? null : JSON.parse(cached));
    }
  } catch {
    /* ignore */
  }

  const existing = inflight.get(cacheKey);
  if (existing) return existing;

  const promise = doFetch(ref)
    .catch(() => null)
    .then((post) => {
      inflight.delete(cacheKey);
      try {
        sessionStorage.setItem(cacheKey, post ? JSON.stringify(post) : "null");
      } catch {
        /* ignore */
      }
      return post;
    });

  inflight.set(cacheKey, promise);
  return promise;
}
