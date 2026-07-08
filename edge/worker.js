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

    const path = url.pathname;
    const originUrl = new URL(request.url);
    originUrl.hostname = ROOT_DOMAIN;
    const pathPrefix = "/reading-room/" + handle;
    let isNote = false;

    if (path === "/" || path === "") {
      originUrl.pathname = pathPrefix;
    } else if (path === "/note" || path.startsWith("/note/")) {
      isNote = true;
      originUrl.pathname = pathPrefix + "/note";
    } else if (path === "/feed" || path === "/feed.xml" || path === "/rss") {
      originUrl.pathname = "/api/reading-room/rss/" + handle;
    } else if (path === "/.well-known/site.standard.publication" || path.startsWith("/.well-known/site.standard.publication/")) {
      originUrl.pathname = "/.well-known/site.standard.publication/reading-room/" + handle;
    } else if (
      path.startsWith("/api/reading-room/") ||
      path.startsWith("/_astro/") ||
      path.startsWith("/dist/") ||
      path.startsWith("/fonts/") ||
      /\.(js|css|woff2?|ttf|svg|png|jpg|jpeg|gif|ico|webp|webmanifest|map)$/.test(path)
    ) {
      originUrl.pathname = path;
    } else {
      return Response.redirect("https://" + ROOT_DOMAIN + path + url.search, 302);
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
    modifiedRequest.headers.delete("origin");
    modifiedRequest.headers.delete("referer");

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
      newResponse.headers.set("Access-Control-Allow-Origin", "*");
      return newResponse;
    }

    const newResponse = new Response(response.body, response);
    newResponse.headers.set("X-Reading-Room-Domain", hostname);
    newResponse.headers.set("Access-Control-Allow-Origin", "*");
    return newResponse;
  },
};
