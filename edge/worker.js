const ROOT_DOMAIN = "margin.at";

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const hostname = (request.headers.get("x-forwarded-host") || url.hostname).toLowerCase();

    if (hostname === ROOT_DOMAIN || hostname.endsWith("." + ROOT_DOMAIN)) {
      return fetch(request);
    }

    const handle = await env.ROUTING.get("host:" + hostname);
    if (!handle) {
      return new Response("Reading room not found", { status: 404 });
    }

    const originUrl = new URL(request.url);
    originUrl.hostname = ROOT_DOMAIN;

    let pathPrefix = "/reading-room/" + handle;

    if (url.pathname === "/" || url.pathname === "") {
      originUrl.pathname = pathPrefix;
    } else if (url.pathname === "/feed" || url.pathname === "/feed.xml" || url.pathname === "/rss") {
      originUrl.pathname = "/api/reading-room/rss/" + handle;
    } else if (url.pathname === "/.well-known/site.standard.publication" || url.pathname.startsWith("/.well-known/site.standard.publication/")) {
      originUrl.pathname = "/.well-known/site.standard.publication/reading-room/" + handle;
    } else if (url.pathname.startsWith("/api/")) {
      originUrl.pathname = url.pathname;
    } else {
      originUrl.pathname = pathPrefix + url.pathname;
    }

    const modifiedRequest = new Request(originUrl, {
      method: request.method,
      headers: new Headers(request.headers),
      body: request.body,
      redirect: "manual",
    });
    modifiedRequest.headers.set("X-Reading-Room-Handle", handle);
    modifiedRequest.headers.set("X-Reading-Room-Domain", hostname);
    modifiedRequest.headers.delete("host");
    modifiedRequest.headers.set("host", ROOT_DOMAIN);

    const response = await fetch(modifiedRequest);

    const newResponse = new Response(response.body, response);
    newResponse.headers.set("X-Reading-Room-Domain", hostname);
    return newResponse;
  },
};
