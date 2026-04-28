import { useStore } from "@nanostores/react";
import { Suspense, useEffect, useState } from "react";
import { I18nextProvider } from "react-i18next";
import i18n from "../i18n";
import type { UserProfile } from "../types";

declare global {
  interface Window {
    __MARGIN_USER__?: UserProfile | null;
  }
}

import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useParams,
} from "react-router-dom";
import { checkSession } from "../api/client";

import MobileNav from "../components/navigation/MobileNav";
import RightSidebar from "../components/navigation/RightSidebar";
import Sidebar from "../components/navigation/Sidebar";
import { $user } from "../store/auth";
import { analytics } from "../lib/analytics";

import AdminModeration from "./core/AdminModeration";
import Discover from "./core/Discover";
import Feed from "./core/Feed";
import New from "./core/New";
import Notifications from "./core/Notifications";
import Search from "./core/Search";
import Settings from "./core/Settings";
import Collections from "./collections/Collections";
import CollectionDetail from "./collections/CollectionDetail";
import AnnotationDetail from "./content/AnnotationDetail";
import UrlPage from "./content/UrlPage";
import UserUrlPage from "./content/UserUrlPage";
import Profile from "./profile/Profile";

const PAGE_TITLES: Record<string, string> = {
  "/home": "Home — Margin",
  "/bookmarks": "Bookmarks — Margin",
  "/highlights": "Highlights — Margin",
  "/annotations": "Annotations — Margin",
  "/discover": "Discover — Margin",
  "/search": "Search — Margin",
  "/notifications": "Notifications — Margin",
  "/new": "New Annotation — Margin",
  "/settings": "Settings — Margin",
  "/collections": "Collections — Margin",
  "/admin/moderation": "Admin — Margin",
};

function AuthGuard({ children }: { children: React.ReactNode }) {
  const user = useStore($user);
  const [checked, setChecked] = useState(() => "__MARGIN_USER__" in window);

  useEffect(() => {
    if (!checked) {
      const unsub = $user.subscribe(() => setChecked(true));
      const t = setTimeout(() => setChecked(true), 3000);
      return () => {
        unsub();
        clearTimeout(t);
      };
    }
  }, [checked]);

  useEffect(() => {
    if (checked && !user) {
      window.location.href = "/login";
    }
  }, [checked, user]);

  if (!checked || !user) return null;
  return <>{children}</>;
}

function CollectionDetailRoute() {
  const { handle, rkey } = useParams<{ handle: string; rkey: string }>();
  return <CollectionDetail handle={handle} rkey={rkey} />;
}

function AnnotationDetailRoute() {
  const { handle, rkey, type } = useParams<{
    handle: string;
    rkey: string;
    type: string;
  }>();
  return <AnnotationDetail handle={handle} rkey={rkey} type={type} />;
}

function AtAnnotationRoute() {
  const { did, rkey } = useParams<{ did: string; rkey: string }>();
  return <AnnotationDetail did={did} rkey={rkey} />;
}

function UriAnnotationRoute() {
  const { uri } = useParams<{ uri: string }>();
  return <AnnotationDetail uri={uri ? decodeURIComponent(uri) : undefined} />;
}

function ProfileRoute() {
  const { did } = useParams<{ did: string }>();
  if (!did) return <Navigate to="/home" replace />;
  return <Profile did={did} />;
}

function UrlRoute() {
  const params = useParams();
  const urlPath = params["*"];
  return <UrlPage urlPath={urlPath} />;
}

function UserUrlRoute() {
  const params = useParams();
  return <UserUrlPage handle={params.handle} urlPath={params["*"]} />;
}

function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const searchParams = new URLSearchParams(location.search);

  useEffect(() => {
    document.title = PAGE_TITLES[location.pathname] ?? "Margin";
  }, [location.pathname]);

  useEffect(() => {
    if (searchParams.get("logged_in") !== "true") return;
    const user = $user.get();
    analytics.capture("login_success", {
      handle: user?.handle ?? "",
      pds: undefined,
    });
    const url = new URL(window.location.href);
    url.searchParams.delete("logged_in");
    window.history.replaceState({}, "", url.toString());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.search]);

  useEffect(() => {
    const SERVER_PATHS = [
      "/login",
      "/about",
      "/privacy",
      "/terms",
      "/brand",
      "/auth/",
      "/api/",
      "/og-image",
    ];
    const handleClick = (e: MouseEvent) => {
      const a = (e.target as Element).closest("a");
      if (!a) return;
      if (a.hasAttribute("target") || a.hasAttribute("download")) return;
      const href = a.getAttribute("href");
      if (!href || !href.startsWith("/")) return;
      if (href === "/" || SERVER_PATHS.some((p: string) => href.startsWith(p)))
        return;
      e.preventDefault();
      navigate(href);
    };
    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, [navigate]);

  return (
    <div className="min-h-screen bg-surface-100 dark:bg-surface-900 flex">
      <Sidebar currentPath={location.pathname} onNavigate={navigate} />

      <div className="flex-1 min-w-0 transition-all duration-200">
        <div className="flex w-full max-w-[1800px] mx-auto">
          <main className="flex-1 w-full min-w-0 p-2 md:py-3 md:px-0">
            <div className="bg-white dark:bg-surface-800 rounded-2xl min-h-[calc(100vh-16px)] md:min-h-[calc(100vh-24px)] py-6 px-4 md:px-6 lg:px-8 pb-28 md:pb-6">
              <Routes>
                <Route
                  path="/home"
                  element={
                    <Feed
                      key="home"
                      initialType="all"
                      initialTag={searchParams.get("tag") ?? undefined}
                    />
                  }
                />
                <Route
                  path="/bookmarks"
                  element={
                    <Feed
                      key="bookmarks"
                      initialType="all"
                      motivation="bookmarking"
                      showTabs={false}
                    />
                  }
                />
                <Route
                  path="/highlights"
                  element={
                    <Feed
                      key="highlights"
                      initialType="all"
                      motivation="highlighting"
                      showTabs={false}
                    />
                  }
                />
                <Route
                  path="/annotations"
                  element={
                    <Feed
                      key="annotations"
                      initialType="all"
                      motivation="commenting"
                      showTabs={false}
                    />
                  }
                />
                <Route path="/discover" element={<Discover />} />
                <Route
                  path="/search"
                  element={
                    <Search
                      key={searchParams.get("q") ?? ""}
                      initialQuery={searchParams.get("q") ?? undefined}
                    />
                  }
                />
                <Route
                  path="/notifications"
                  element={
                    <AuthGuard>
                      <Notifications />
                    </AuthGuard>
                  }
                />
                <Route
                  path="/new"
                  element={
                    <AuthGuard>
                      <New
                        initialUrl={searchParams.get("url") ?? undefined}
                        initialSelectorJson={
                          searchParams.get("selector") ?? undefined
                        }
                        initialQuote={searchParams.get("quote") ?? undefined}
                      />
                    </AuthGuard>
                  }
                />
                <Route path="/settings" element={<Settings />} />
                <Route
                  path="/admin/moderation"
                  element={
                    <AuthGuard>
                      <AdminModeration />
                    </AuthGuard>
                  }
                />
                <Route path="/collections" element={<Collections />} />
                <Route
                  path="/collections/:rkey"
                  element={<CollectionDetail />}
                />
                <Route
                  path="/:handle/collection/:rkey"
                  element={<CollectionDetailRoute />}
                />
                <Route
                  path="/:handle/note/:rkey"
                  element={<AnnotationDetailRoute />}
                />
                <Route
                  path="/:handle/annotation/:rkey"
                  element={<AnnotationDetailRoute />}
                />
                <Route
                  path="/:handle/highlight/:rkey"
                  element={<AnnotationDetailRoute />}
                />
                <Route
                  path="/:handle/bookmark/:rkey"
                  element={<AnnotationDetailRoute />}
                />
                <Route
                  path="/annotation/:uri"
                  element={<UriAnnotationRoute />}
                />
                <Route path="/at/:did/:rkey" element={<AtAnnotationRoute />} />
                <Route path="/url/*" element={<UrlRoute />} />
                <Route path="/:handle/url/*" element={<UserUrlRoute />} />
                <Route path="/profile/:did" element={<ProfileRoute />} />
                <Route
                  path="/profile"
                  element={
                    <AuthGuard>
                      <ProfileSelfRedirect />
                    </AuthGuard>
                  }
                />
                <Route path="*" element={<Navigate to="/home" replace />} />
              </Routes>
            </div>
          </main>

          <RightSidebar onNavigate={navigate} />
        </div>
      </div>

      <MobileNav currentPath={location.pathname} onNavigate={navigate} />
    </div>
  );
}

function ProfileSelfRedirect() {
  const user = useStore($user);
  if (!user) return null;
  return <Navigate to={`/profile/${user.did}`} replace />;
}

export default function AppShell() {
  useState(() => {
    const ssrUser = window.__MARGIN_USER__;
    if (ssrUser !== undefined) {
      $user.set(ssrUser);
    }
  });

  useEffect(() => {
    const ssrUser = window.__MARGIN_USER__;
    if ($user.get() === null && ssrUser === null) return;

    if (ssrUser) {
      checkSession().then((user) => {
        if (user) $user.set(user);
      });
    } else if (ssrUser === undefined) {
      checkSession().then((user) => {
        $user.set(user);
      });
    }
  }, []);

  return (
    <I18nextProvider i18n={i18n}>
      <Suspense fallback={null}>
        <BrowserRouter>
          <AppLayout />
        </BrowserRouter>
      </Suspense>
    </I18nextProvider>
  );
}
