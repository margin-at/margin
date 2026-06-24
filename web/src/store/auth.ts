import { atom } from "nanostores";
import { loadPreferences } from "./preferences";
import { sessionAtom } from "../api/client";
import type { UserProfile } from "../types";
import { analytics } from "../lib/analytics";

export const $user = atom<UserProfile | null>(null);

$user.subscribe((user) => {
  if (user) {
    loadPreferences();
    analytics.identify(user.did, {
      handle: user.handle,
      displayName: user.displayName,
    });
  }
});

let syncing = false;
function keepInSync(from: typeof $user, to: typeof $user) {
  from.subscribe((value) => {
    if (syncing || to.get() === value) return;
    syncing = true;
    to.set(value);
    syncing = false;
  });
}
keepInSync(sessionAtom, $user);
keepInSync($user, sessionAtom);

export function logout() {
  analytics.capture("user_logged_out");
  analytics.reset();
  $user.set(null);
  fetch("/auth/logout", { method: "POST" }).finally(() => {
    window.location.href = "/";
  });
}
