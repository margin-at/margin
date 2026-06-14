export type AnalyticsEvents = {
  login_initiated: { handle: string };
  login_success: { handle: string; pds?: string };
  signup_initiated: { provider: string };
  user_logged_out: Record<string, never>;

  annotation_created: {
    url: string;
    has_quote: boolean;
    tag_count: number;
    has_labels: boolean;
    source?: "web" | "extension";
  };
  highlight_created: {
    url: string;
    tag_count: number;
    has_color: boolean;
    has_labels: boolean;
    source?: "web" | "extension";
  };
  bookmark_created: {
    url: string;
    tag_count: number;
    source?: "web" | "extension";
  };
  reply_created: { parent_uri: string; root_uri: string };

  item_liked: {
    action: "like" | "unlike";
    type: "annotation" | "highlight" | "bookmark";
  };
  item_deleted: { type: "annotation" | "highlight" | "bookmark" };
  item_shared: {
    method: "copy_link" | "social_app";
    destination?: string;
    item_type?: string;
  };
  item_added_to_collection: Record<string, never>;

  collection_created: { has_description: boolean };
  collection_deleted: Record<string, never>;

  extension_installed: { version: string; browser: string };
  extension_connected: { did: string };
  popup_opened: { authenticated: boolean };
  extension_tab_switched: { tab: string };

  highlights_imported: { total: number; completed: number; failed: number };

  search_performed: { query: string };

  api_key_created: Record<string, never>;
  theme_changed: { theme: string };
};

function getPostHog() {
  if (typeof window === "undefined") return null;
  return window.posthog ?? null;
}

export const analytics = {
  capture<E extends keyof AnalyticsEvents>(
    event: E,
    properties?: AnalyticsEvents[E],
  ): void {
    try {
      getPostHog()?.capture(
        event as string,
        properties as Record<string, unknown>,
      );
    } catch {
      // ignore
    }
  },

  identify(
    did: string,
    properties: { handle: string; displayName?: string },
  ): void {
    try {
      getPostHog()?.identify(did, {
        handle: properties.handle,
        display_name: properties.displayName ?? undefined,
        $set_once: { first_seen_at: new Date().toISOString() },
      });
    } catch {
      // noop
    }
  },

  reset(): void {
    try {
      getPostHog()?.reset();
    } catch {
      // noop
    }
  },

  captureException(error: unknown, properties?: Record<string, unknown>): void {
    try {
      const ph = getPostHog();
      if (!ph) return;
      if (typeof ph.captureException === "function") {
        ph.captureException(error, properties);
      } else {
        ph.capture("$exception", {
          message: error instanceof Error ? error.message : String(error),
          ...properties,
        });
      }
    } catch {
      // noop
    }
  },
};
