import type { APIContext } from "astro";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { clearSessionCacheForCookie, getSession } from "./lib/api";

const API_PORT = process.env.API_PORT || 8081;
const API_URL = process.env.API_URL || `http://localhost:${API_PORT}`;

const PROXY_PATHS = [
  "/api/",
  "/auth/",
  "/oauth-client-metadata.json",
  "/jwks.json",
  "/.well-known/site.standard.publication",
];

const CORS_HEADERS = {
  "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
  "Access-Control-Allow-Headers":
    "Accept, Authorization, Content-Type, X-CSRF-Token, X-Session-Token",
  "Access-Control-Expose-Headers": "Link",
  "Access-Control-Allow-Credentials": "true",
  "Access-Control-Max-Age": "300",
};

function isExtensionOrigin(origin: string | null): origin is string {
  if (!origin) return false;
  return (
    origin.startsWith("chrome-extension://") ||
    origin.startsWith("moz-extension://") ||
    origin.startsWith("safari-web-extension://")
  );
}

export async function onRequest(
  context: APIContext,
  next: () => Promise<Response>,
): Promise<Response> {
  const { request, url, locals } = context;

  if (url.pathname === "/favicon.ico") {
    try {
      const file = await readFile(
        join(process.cwd(), "dist", "client", "favicon.ico"),
      );
      return new Response(file, {
        headers: {
          "Content-Type": "image/x-icon",
          "Cache-Control": "public, max-age=86400",
        },
      });
    } catch {
      /* ignore */
    }
  }

  const shouldProxy = PROXY_PATHS.some(
    (p) => url.pathname.startsWith(p) || url.pathname === p.replace(/\/$/, ""),
  );

  if (shouldProxy) {
    const response = await proxyToBackend(request, url);
    if (url.pathname === "/auth/logout") {
      clearSessionCacheForCookie(request.headers.get("cookie") || "");
    }
    return response;
  }

  const cookie = request.headers.get("cookie") || "";

  if (cookie.includes("margin_session")) {
    locals.user = await getSession(cookie);
  } else {
    locals.user = null;
  }

  return next();
}

async function proxyToBackend(request: Request, url: URL): Promise<Response> {
  const origin = request.headers.get("origin");

  if (request.method === "OPTIONS" && isExtensionOrigin(origin)) {
    return new Response(null, {
      status: 204,
      headers: {
        "Access-Control-Allow-Origin": origin,
        ...CORS_HEADERS,
      },
    });
  }

  const target = new URL(url.pathname + url.search, API_URL);

  const headers = new Headers(request.headers);
  headers.delete("host");
  headers.delete("origin");
  headers.delete("referer");

  const init: RequestInit = {
    method: request.method,
    headers,
    redirect: "manual",
  };

  if (request.method !== "GET" && request.method !== "HEAD" && request.body) {
    init.body = request.body;
    // @ts-expect-error duplex is generic on RequestInit
    init.duplex = "half";
  }

  try {
    const res = await fetch(target.toString(), init);
    const responseHeaders = new Headers(res.headers);

    if (isExtensionOrigin(origin)) {
      responseHeaders.set("Access-Control-Allow-Origin", origin);
      for (const [key, value] of Object.entries(CORS_HEADERS)) {
        responseHeaders.set(key, value);
      }
    }

    return new Response(res.body, {
      status: res.status,
      statusText: res.statusText,
      headers: responseHeaders,
    });
  } catch {
    return new Response("Backend unavailable", { status: 502 });
  }
}
