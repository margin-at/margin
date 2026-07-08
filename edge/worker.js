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
    let isNote = false;

    if (url.pathname === "/" || url.pathname === "") {
      originUrl.pathname = pathPrefix;
    } else if (url.pathname === "/feed" || url.pathname === "/feed.xml" || url.pathname === "/rss") {
      originUrl.pathname = "/api/reading-room/rss/" + handle;
    } else if (url.pathname === "/.well-known/site.standard.publication" || url.pathname.startsWith("/.well-known/site.standard.publication/")) {
      originUrl.pathname = "/.well-known/site.standard.publication/reading-room/" + handle;
    } else if (
      url.pathname.startsWith("/api/") ||
      url.pathname.startsWith("/_astro/") ||
      url.pathname.startsWith("/dist/") ||
      url.pathname.startsWith("/fonts/") ||
      url.pathname.startsWith("/.well-known/") ||
      /\.(js|css|woff2?|ttf|svg|png|jpg|jpeg|gif|ico|webp|webmanifest|map)$/.test(url.pathname)
    ) {
      originUrl.pathname = url.pathname;
    } else if (url.pathname === "/note" || url.pathname.startsWith("/note/")) {
      isNote = true;
      originUrl.pathname = pathPrefix + "/note";
    } else if (url.pathname.startsWith("/reading-room/")) {
      originUrl.pathname = url.pathname;
    } else {
      const redirectUrl = new URL(url.pathname + url.search, "https://" + ROOT_DOMAIN);
      return Response.redirect(redirectUrl.toString(), 302);
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

    const contentType = response.headers.get("content-type") || "";
    if (contentType.includes("text/html")) {
      let html = await response.text();
      const inject = `<script>window.__READING_ROOM_HANDLE__=${JSON.stringify(handle)};window.__READING_ROOM_IS_NOTE__=${isNote};</script>`;
      html = html.replace("</head>", inject + "</head>");
      const newResponse = new Response(html, {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers,
      });
      newResponse.headers.set("X-Reading-Room-Domain", hostname);
      return newResponse;
    }

    const newResponse = new Response(response.body, response);
    newResponse.headers.set("X-Reading-Room-Domain", hostname);
    return newResponse;
  },
};
