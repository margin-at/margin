import { defineExtensionMessaging } from '@webext-core/messaging';
import type {
  MarginSession,
  Annotation,
  Bookmark,
  Highlight,
  Collection,
  TextSelector,
} from './types';

interface ProtocolMap {
  checkSession(): MarginSession;
  sessionPing(): void;

  getAnnotations(data: { url: string; citedUrls?: string[]; cacheBust?: boolean }): Annotation[];
  activateOnPdf(data: { tabId: number; url: string }): { redirected: boolean };
  createAnnotation(data: {
    url: string;
    text: string;
    title?: string;
    selector?: TextSelector;
    tags?: string[];
  }): {
    success: boolean;
    data?: Annotation;
    error?: string;
  };

  createBookmark(data: { url: string; title?: string; tags?: string[] }): {
    success: boolean;
    data?: Bookmark;
    error?: string;
  };
  getUserBookmarks(data: { did: string }): Bookmark[];

  createHighlight(data: {
    url: string;
    title?: string;
    selector: TextSelector;
    color?: string;
    tags?: string[];
  }): {
    success: boolean;
    data?: Highlight;
    error?: string;
  };
  getUserHighlights(data: { did: string }): Highlight[];

  getUserCollections(data: { did: string }): Collection[];
  addToCollection(data: { collectionUri: string; annotationUri: string }): {
    success: boolean;
    error?: string;
  };
  getItemCollections(data: { annotationUri: string }): string[];

  deleteHighlight(data: { uri: string }): { success: boolean; error?: string };
  updateHighlightTags(data: { uri: string; tags: string[] }): { success: boolean; error?: string };
  convertHighlightToAnnotation(data: {
    highlightUri: string;
    url: string;
    text: string;
    title?: string;
    selector?: TextSelector;
  }): { success: boolean; error?: string };

  getReplies(data: { uri: string }): Annotation[];
  createReply(data: {
    parentUri: string;
    parentCid: string;
    rootUri: string;
    rootCid: string;
    text: string;
  }): { success: boolean; error?: string };

  getOverlayEnabled(): boolean;

  getUserTags(data: { did: string }): string[];
  getTrendingTags(): string[];

  openAppUrl(data: { path: string }): void;

  updateBadge(data: { count: number; tabId?: number }): void;

  cacheAnnotations(data: { url: string; annotations: Annotation[] }): void;
  getCachedAnnotations(data: { url: string }): Annotation[] | null;
}

export const { sendMessage, onMessage } = defineExtensionMessaging<ProtocolMap>();
