import { useState, useEffect } from 'react';
import { capture } from '@/utils/analytics';
import { sendMessage } from '@/utils/messaging';
import { themeItem, apiUrlItem, overlayEnabledItem } from '@/utils/storage';
import type { MarginSession, Annotation, Bookmark, Highlight, Collection } from '@/utils/types';
import CollectionIcon from '@/components/CollectionIcon';
import TagInput from '@/components/TagInput';
import {
  Settings,
  ExternalLink,
  Bookmark as BookmarkIcon,
  Highlighter,
  X,
  Sun,
  Moon,
  Monitor,
  Check,
  Globe,
  ChevronRight,
  Sparkles,
  FolderPlus,
  Folder,
  PenTool,
  Eye,
  Send,
  MessageSquare,
} from 'lucide-react';

type Tab = 'page' | 'bookmarks' | 'highlights' | 'collections';
type PageFilter = 'all' | 'annotations' | 'highlights';

export function App() {
  const [session, setSession] = useState<MarginSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<Tab>('page');
  const [pageFilter, setPageFilter] = useState<PageFilter>('all');
  const [annotations, setAnnotations] = useState<Annotation[]>([]);
  const [pageHighlights, setPageHighlights] = useState<Annotation[]>([]);
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
  const [highlights, setHighlights] = useState<Highlight[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loadingAnnotations, setLoadingAnnotations] = useState(false);
  const [loadingBookmarks, setLoadingBookmarks] = useState(false);
  const [loadingHighlights, setLoadingHighlights] = useState(false);
  const [loadingCollections, setLoadingCollections] = useState(false);
  const [collectionModalItem, setCollectionModalItem] = useState<string | null>(null);
  const [addingToCollection, setAddingToCollection] = useState<string | null>(null);
  const [containingCollections, setContainingCollections] = useState<Set<string>>(new Set());
  const [currentUrl, setCurrentUrl] = useState('');
  const [currentTitle, setCurrentTitle] = useState('');
  const [text, setText] = useState('');
  const [posting, setPosting] = useState(false);
  const [bookmarking, setBookmarking] = useState(false);
  const [bookmarked, setBookmarked] = useState(false);
  const [theme, setTheme] = useState<'light' | 'dark' | 'system'>('system');
  const [showSettings, setShowSettings] = useState(false);
  const [apiUrl, setApiUrl] = useState('https://margin.at');
  const [overlayEnabled, setOverlayEnabled] = useState(true);
  const [tags, setTags] = useState<string[]>([]);
  const [bookmarkTags, setBookmarkTags] = useState<string[]>([]);
  const [showBookmarkTags, setShowBookmarkTags] = useState(false);
  const [tagSuggestions, setTagSuggestions] = useState<string[]>([]);
  useEffect(() => {
    checkSession();
    loadCurrentTab();
    loadTheme();
    loadSettings();

    sendMessage('checkSession', undefined)
      .then((s) =>
        capture('popup_opened', { authenticated: s?.authenticated ?? false }, s?.did ?? undefined)
      )
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (session?.authenticated && session.did) {
      Promise.all([
        sendMessage('getUserTags', { did: session.did }).catch(() => [] as string[]),
        sendMessage('getTrendingTags', undefined).catch(() => [] as string[]),
      ]).then(([userTags, trendingTags]) => {
        const seen = new Set(userTags);
        const merged = [...userTags];
        for (const t of trendingTags) {
          if (!seen.has(t)) {
            merged.push(t);
            seen.add(t);
          }
        }
        setTagSuggestions(merged);
      });
    }
  }, [session]);

  useEffect(() => {
    if (session?.authenticated && currentUrl) {
      if (activeTab === 'page') loadAnnotations();
      else if (activeTab === 'bookmarks') loadBookmarks();
      else if (activeTab === 'highlights') loadHighlights();
      else if (activeTab === 'collections') loadCollections();
    }
  }, [activeTab, session, currentUrl]);

  async function loadSettings() {
    const url = await apiUrlItem.getValue();
    const overlay = await overlayEnabledItem.getValue();
    setApiUrl(url);
    setOverlayEnabled(overlay);
  }

  async function saveSettings() {
    const cleanUrl = apiUrl.replace(/\/$/, '');
    await apiUrlItem.setValue(cleanUrl);
    await overlayEnabledItem.setValue(overlayEnabled);

    const tabs = await browser.tabs.query({});
    for (const tab of tabs) {
      if (tab.id) {
        try {
          await browser.tabs.sendMessage(tab.id, {
            type: 'UPDATE_OVERLAY_VISIBILITY',
            show: overlayEnabled,
          });
        } catch {
          /* ignore */
        }
      }
    }

    setShowSettings(false);
    checkSession();
  }

  async function loadTheme() {
    const t = await themeItem.getValue();
    setTheme(t);
    applyTheme(t);

    themeItem.watch((newTheme) => {
      setTheme(newTheme);
      applyTheme(newTheme);
    });
  }

  function applyTheme(t: string) {
    document.body.classList.remove('light', 'dark');
    if (t === 'system') {
      if (window.matchMedia('(prefers-color-scheme: light)').matches) {
        document.body.classList.add('light');
      }
    } else {
      document.body.classList.add(t);
    }
  }

  async function handleThemeChange(newTheme: 'light' | 'dark' | 'system') {
    await themeItem.setValue(newTheme);
    setTheme(newTheme);
    applyTheme(newTheme);
  }

  async function checkSession() {
    try {
      const result = await sendMessage('checkSession', undefined);
      setSession(result);
    } catch (error) {
      console.error('Session check error:', error);
      setSession({ authenticated: false });
    } finally {
      setLoading(false);
    }
  }

  async function loadCurrentTab() {
    const [tab] = await browser.tabs.query({ active: true, currentWindow: true });
    if (tab?.url) {
      if (isPdfUrl(tab.url) && tab.id) {
        await sendMessage('activateOnPdf', { tabId: tab.id, url: tab.url });
        window.close();
        return;
      }

      const resolved = extractOriginalUrl(tab.url);
      setCurrentUrl(resolved);
      setCurrentTitle(tab.title || '');
    }
  }

  function extractOriginalUrl(url: string): string {
    if (url.includes('/pdfjs/web/viewer.html')) {
      try {
        const fileParam = new URL(url).searchParams.get('file');
        if (fileParam) return fileParam;
      } catch {
        /* ignore */
      }
    }
    return url;
  }

  function isPdfUrl(url: string): boolean {
    if (!url.startsWith('http://') && !url.startsWith('https://')) return false;
    try {
      const { pathname } = new URL(url);
      return /\.pdf$/i.test(pathname);
    } catch {
      return false;
    }
  }

  async function loadAnnotations() {
    if (!currentUrl) return;
    setLoadingAnnotations(true);
    try {
      let result = await sendMessage('getCachedAnnotations', { url: currentUrl });

      if (!result) {
        result = await sendMessage('getAnnotations', { url: currentUrl });
      }

      const all = result || [];
      const annots = all.filter(
        (item: any) => item.type !== 'Bookmark' && item.type !== 'Highlight'
      );
      const hlights = all.filter((item: any) => item.type === 'Highlight');
      setAnnotations(annots);
      setPageHighlights(hlights);

      const isBookmarked = all.some(
        (item: any) => item.type === 'Bookmark' && item.creator?.did === session?.did
      );
      setBookmarked(isBookmarked);
    } catch (error) {
      console.error('Load annotations error:', error);
    } finally {
      setLoadingAnnotations(false);
    }
  }

  async function loadBookmarks() {
    if (!session?.did) return;
    setLoadingBookmarks(true);
    try {
      const result = await sendMessage('getUserBookmarks', { did: session.did });
      setBookmarks(result || []);
    } catch (error) {
      console.error('Load bookmarks error:', error);
    } finally {
      setLoadingBookmarks(false);
    }
  }

  async function loadHighlights() {
    if (!session?.did) return;
    setLoadingHighlights(true);
    try {
      const result = await sendMessage('getUserHighlights', { did: session.did });
      setHighlights(result || []);
    } catch (error) {
      console.error('Load highlights error:', error);
    } finally {
      setLoadingHighlights(false);
    }
  }

  async function loadCollections() {
    if (!session?.did) return;
    setLoadingCollections(true);
    try {
      const result = await sendMessage('getUserCollections', { did: session.did });
      setCollections(result || []);
    } catch (error) {
      console.error('Load collections error:', error);
    } finally {
      setLoadingCollections(false);
    }
  }

  async function openCollectionModal(itemUri: string) {
    setCollectionModalItem(itemUri);
    setContainingCollections(new Set());

    if (collections.length === 0) {
      await loadCollections();
    }

    try {
      const itemCollectionUris = await sendMessage('getItemCollections', {
        annotationUri: itemUri,
      });
      setContainingCollections(new Set(itemCollectionUris));
    } catch (error) {
      console.error('Failed to get item collections:', error);
    }
  }

  async function handleAddToCollection(collectionUri: string) {
    if (!collectionModalItem) return;

    if (containingCollections.has(collectionUri)) {
      setCollectionModalItem(null);
      return;
    }

    setAddingToCollection(collectionUri);
    try {
      const result = await sendMessage('addToCollection', {
        collectionUri,
        annotationUri: collectionModalItem,
      });
      if (result.success) {
        setContainingCollections((prev) => new Set([...prev, collectionUri]));
      } else {
        alert('Failed to add to collection');
      }
    } catch (error) {
      console.error('Add to collection error:', error);
      alert('Error adding to collection');
    } finally {
      setAddingToCollection(null);
    }
  }

  async function handlePost() {
    if (!text.trim()) return;
    setPosting(true);
    try {
      const result = await sendMessage('createAnnotation', {
        url: currentUrl,
        text: text.trim(),
        title: currentTitle,
        tags: tags.length > 0 ? tags : undefined,
      });
      if (result.success) {
        setText('');
        setTags([]);
        loadAnnotations();
        capture(
          'annotation_created',
          { url: currentUrl, tag_count: tags.length, source: 'extension' },
          session?.did ?? undefined
        );
      } else {
        alert('Failed to post annotation');
      }
    } catch (error) {
      console.error('Post error:', error);
      alert('Error posting annotation');
    } finally {
      setPosting(false);
    }
  }

  async function handleBookmark() {
    setBookmarking(true);
    try {
      const result = await sendMessage('createBookmark', {
        url: currentUrl,
        title: currentTitle,
        tags: bookmarkTags.length > 0 ? bookmarkTags : undefined,
      });
      if (result.success) {
        setBookmarked(true);
        setBookmarkTags([]);
        setShowBookmarkTags(false);
        capture(
          'bookmark_created',
          { url: currentUrl, tag_count: bookmarkTags.length, source: 'extension' },
          session?.did ?? undefined
        );
      } else {
        alert('Failed to bookmark page');
      }
    } catch (error) {
      console.error('Bookmark error:', error);
      alert('Error bookmarking page');
    } finally {
      setBookmarking(false);
    }
  }

  function formatDate(dateString?: string) {
    if (!dateString) return '';
    try {
      return new Date(dateString).toLocaleDateString();
    } catch {
      return dateString;
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="w-5 h-5 border-2 border-[var(--accent)] border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (!session?.authenticated) {
    return (
      <div className="flex flex-col h-screen">
        {showSettings && (
          <div className="absolute inset-0 bg-[var(--bg-primary)] z-10 flex flex-col">
            <header className="flex items-center justify-between px-4 py-3 border-b border-[var(--border)]">
              <span className="text-sm font-semibold">Settings</span>
              <button
                onClick={() => setShowSettings(false)}
                className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)] rounded-lg transition-colors"
              >
                <X size={16} />
              </button>
            </header>

            <div className="flex-1 overflow-y-auto p-4 space-y-5">
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
                  API URL
                </label>
                <input
                  type="text"
                  value={apiUrl}
                  onChange={(e) => setApiUrl(e.target.value)}
                  className="w-full px-3 py-2 bg-[var(--bg-card)] border border-[var(--border)] rounded-lg text-sm focus:outline-none focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent-subtle)] transition-all"
                  placeholder="https://margin.at"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
                  Theme
                </label>
                <div className="flex gap-1.5">
                  {(['light', 'dark', 'system'] as const).map((t) => (
                    <button
                      key={t}
                      onClick={() => handleThemeChange(t)}
                      className={`flex-1 py-2 px-3 text-xs font-medium rounded-lg transition-colors flex items-center justify-center gap-1.5 ${
                        theme === t
                          ? 'bg-[var(--accent)] text-white'
                          : 'bg-[var(--bg-card)] border border-[var(--border)] hover:bg-[var(--bg-hover)]'
                      }`}
                    >
                      {t === 'light' ? (
                        <Sun size={12} />
                      ) : t === 'dark' ? (
                        <Moon size={12} />
                      ) : (
                        <Monitor size={12} />
                      )}
                      {t.charAt(0).toUpperCase() + t.slice(1)}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <div className="p-4 border-t border-[var(--border)]">
              <button
                onClick={saveSettings}
                className="w-full py-2.5 bg-[var(--accent)] text-white rounded-lg text-sm font-medium hover:bg-[var(--accent-hover)] transition-colors"
              >
                Save Settings
              </button>
            </div>
          </div>
        )}

        <div className="flex flex-col items-center justify-center flex-1 p-8 text-center">
          <div className="w-14 h-14 rounded-2xl bg-[var(--accent-subtle)] flex items-center justify-center mb-5">
            <img src="/icons/logo.svg" alt="Margin" className="w-8 h-8" />
          </div>
          <h2 className="font-display text-xl font-bold tracking-tight mb-2">Welcome to Margin</h2>
          <p className="text-[var(--text-secondary)] text-sm leading-relaxed mb-5 max-w-[280px]">
            Annotate, highlight, and bookmark the web with your AT Protocol identity.
          </p>
          <button
            onClick={() => browser.tabs.create({ url: `${apiUrl}/login` })}
            className="w-full max-w-[280px] px-6 py-2.5 bg-[var(--accent)] text-white rounded-xl text-sm font-semibold hover:bg-[var(--accent-hover)] transition-colors"
          >
            Sign In
          </button>

          <button
            onClick={() => setShowSettings(true)}
            className="mt-4 text-xs text-[var(--text-tertiary)] hover:text-[var(--text-primary)] flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-[var(--bg-hover)] transition-colors"
          >
            <Settings size={12} /> Settings
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen">
      {showSettings && (
        <div className="absolute inset-0 bg-[var(--bg-primary)] z-10 flex flex-col">
          <header className="flex items-center justify-between px-4 py-3 border-b border-[var(--border)]">
            <span className="text-sm font-semibold">Settings</span>
            <button
              onClick={() => setShowSettings(false)}
              className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)] rounded-lg transition-colors"
            >
              <X size={16} />
            </button>
          </header>

          <div className="flex-1 overflow-y-auto p-4 space-y-5">
            <div>
              <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
                API URL
              </label>
              <input
                type="text"
                value={apiUrl}
                onChange={(e) => setApiUrl(e.target.value)}
                className="w-full px-3 py-2 bg-[var(--bg-card)] border border-[var(--border)] rounded-lg text-sm focus:outline-none focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent-subtle)] transition-all"
                placeholder="https://margin.at"
              />
              <p className="text-[11px] text-[var(--text-tertiary)] mt-1">
                For development or self-hosted instances
              </p>
            </div>

            <div className="flex items-center justify-between p-3 bg-[var(--bg-card)] border border-[var(--border)] rounded-lg">
              <div>
                <div className="text-sm font-medium">Page Overlay</div>
                <p className="text-[11px] text-[var(--text-tertiary)] mt-0.5">
                  Show highlights and annotations on pages
                </p>
              </div>
              <button
                onClick={() => setOverlayEnabled(!overlayEnabled)}
                className={`relative w-10 h-[22px] rounded-full transition-colors ${
                  overlayEnabled
                    ? 'bg-[var(--accent)]'
                    : 'bg-[var(--bg-hover)] border border-[var(--border)]'
                }`}
              >
                <div
                  className={`absolute top-[3px] w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${
                    overlayEnabled ? 'left-[22px]' : 'left-[3px]'
                  }`}
                />
              </button>
            </div>

            <div>
              <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
                Theme
              </label>
              <div className="flex gap-1.5">
                {(['light', 'dark', 'system'] as const).map((t) => (
                  <button
                    key={t}
                    onClick={() => handleThemeChange(t)}
                    className={`flex-1 py-2 px-3 text-xs font-medium rounded-lg transition-colors flex items-center justify-center gap-1.5 ${
                      theme === t
                        ? 'bg-[var(--accent)] text-white'
                        : 'bg-[var(--bg-card)] border border-[var(--border)] hover:bg-[var(--bg-hover)]'
                    }`}
                  >
                    {t === 'light' ? (
                      <Sun size={12} />
                    ) : t === 'dark' ? (
                      <Moon size={12} />
                    ) : (
                      <Monitor size={12} />
                    )}
                    {t.charAt(0).toUpperCase() + t.slice(1)}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="p-4 border-t border-[var(--border)]">
            <button
              onClick={saveSettings}
              className="w-full py-2.5 bg-[var(--accent)] text-white rounded-lg text-sm font-medium hover:bg-[var(--accent-hover)] transition-colors"
            >
              Save
            </button>
          </div>
        </div>
      )}

      <header className="flex items-center justify-between px-4 py-2.5 border-b border-[var(--border)]">
        <div className="flex items-center gap-2">
          <img src="/icons/logo.svg" alt="Margin" className="w-5 h-5" />
          <span className="font-display font-bold text-sm tracking-tight">Margin</span>
        </div>
        <div className="flex items-center gap-1.5">
          <button
            onClick={() => browser.tabs.create({ url: apiUrl })}
            className="text-[11px] text-[var(--text-tertiary)] hover:text-[var(--accent)] px-2 py-1 rounded-md hover:bg-[var(--bg-hover)] transition-colors"
            style={{ display: session.handle ? undefined : 'none' }}
          >
            @{session.handle}
          </button>
          <button
            onClick={() => setShowSettings(true)}
            className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)] rounded-lg transition-colors"
            title="Settings"
          >
            <Settings size={15} />
          </button>
        </div>
      </header>

      <div className="flex border-b border-[var(--border)] px-2 gap-0.5">
        {(['page', 'bookmarks', 'highlights', 'collections'] as Tab[])
          .filter((tab) => tab === 'page' || !!session.did)
          .map((tab) => {
            const icons: Record<Tab, JSX.Element> = {
              page: <Globe size={13} />,
              bookmarks: <BookmarkIcon size={13} />,
              highlights: <Highlighter size={13} />,
              collections: <Folder size={13} />,
            };
            const labels: Record<Tab, string> = {
              page: 'Page',
              bookmarks: 'Bookmarks',
              highlights: 'Highlights',
              collections: 'Collections',
            };
            return (
              <button
                key={tab}
                onClick={() => {
                  setActiveTab(tab);
                  capture('extension_tab_switched', { tab }, session?.did ?? undefined);
                }}
                className={`flex-1 py-2.5 text-[11px] font-medium flex items-center justify-center gap-1 border-b-2 transition-all ${
                  activeTab === tab
                    ? 'border-[var(--accent)] text-[var(--accent)]'
                    : 'border-transparent text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
                }`}
              >
                {icons[tab]}
                {labels[tab]}
              </button>
            );
          })}
      </div>

      <div className="flex-1 overflow-y-auto">
        {activeTab === 'page' && (
          <div>
            <div className="p-4 border-b border-[var(--border)]">
              <div className="flex items-center gap-3 p-3 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl">
                <div className="w-9 h-9 rounded-lg bg-[var(--bg-hover)] flex items-center justify-center flex-shrink-0 overflow-hidden">
                  {currentUrl ? (
                    <img
                      src={`https://www.google.com/s2/favicons?domain=${new URL(currentUrl).hostname}&sz=64`}
                      alt=""
                      className="w-5 h-5"
                      onError={(e) => {
                        (e.target as HTMLImageElement).style.display = 'none';
                        (e.target as HTMLImageElement).nextElementSibling?.classList.remove(
                          'hidden'
                        );
                      }}
                    />
                  ) : null}
                  <Globe
                    size={16}
                    className={`text-[var(--text-tertiary)] ${currentUrl ? 'hidden' : ''}`}
                  />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">{currentTitle || 'Untitled'}</div>
                  <div className="text-[11px] text-[var(--text-tertiary)] truncate">
                    {currentUrl ? new URL(currentUrl).hostname : ''}
                  </div>
                </div>
                <div className="flex items-center gap-1 flex-shrink-0">
                  <button
                    onClick={() => {
                      if (currentUrl) {
                        const shareUrl = `${apiUrl}/url/${encodeURIComponent(currentUrl)}`;
                        browser.tabs.create({ url: shareUrl });
                      }
                    }}
                    className="p-1.5 rounded-md text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--bg-hover)] transition-colors"
                    title="View on Margin"
                  >
                    <Eye size={15} />
                  </button>
                  <button
                    onClick={() => {
                      if (!bookmarked) setShowBookmarkTags(!showBookmarkTags);
                    }}
                    disabled={bookmarking || bookmarked}
                    className={`p-1.5 rounded-md transition-colors ${
                      bookmarked
                        ? 'text-emerald-400'
                        : showBookmarkTags
                          ? 'text-[var(--accent)] bg-[var(--accent-subtle)]'
                          : 'text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--bg-hover)]'
                    }`}
                    title={bookmarked ? 'Bookmarked' : 'Bookmark page'}
                  >
                    {bookmarked ? <Check size={15} /> : <BookmarkIcon size={15} />}
                  </button>
                </div>
              </div>
              {showBookmarkTags && !bookmarked && (
                <div className="mt-2 pt-2 border-t border-[var(--border)]">
                  <div className="mb-1.5">
                    <TagInput
                      tags={bookmarkTags}
                      onChange={setBookmarkTags}
                      suggestions={tagSuggestions}
                      placeholder="Add bookmark tags..."
                    />
                  </div>
                  <button
                    onClick={handleBookmark}
                    disabled={bookmarking}
                    className="w-full py-1.5 bg-[var(--accent)] text-white text-xs rounded-lg font-medium hover:bg-[var(--accent-hover)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                  >
                    {bookmarking ? 'Bookmarking...' : 'Bookmark page'}
                  </button>
                </div>
              )}
            </div>

            <div className="p-4 border-b border-[var(--border)]">
              <textarea
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder="Share your thoughts on this page..."
                className="w-full px-3 py-2.5 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl text-sm resize-none focus:outline-none focus:border-[var(--accent)] focus:ring-2 focus:ring-[var(--accent-subtle)] min-h-[80px] transition-all"
              />
              <div className="mt-2">
                <TagInput tags={tags} onChange={setTags} suggestions={tagSuggestions} />
              </div>
              <div className="flex items-center justify-between mt-2">
                <span className="text-[11px] text-[var(--text-tertiary)]">
                  {text.length > 0 ? `${text.length} chars` : ''}
                </span>
                <button
                  onClick={handlePost}
                  disabled={posting || !text.trim()}
                  className="px-4 py-1.5 bg-[var(--accent)] text-white text-xs rounded-lg font-medium hover:bg-[var(--accent-hover)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  {posting ? 'Posting...' : 'Post'}
                </button>
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border)]">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-semibold text-[var(--text-secondary)]">
                    Activity
                  </span>
                  <span className="text-xs font-semibold bg-[var(--accent-subtle)] text-[var(--accent)] px-2 py-0.5 rounded-full">
                    {annotations.length + pageHighlights.length}
                  </span>
                </div>
                <div className="flex items-center bg-[var(--bg-card)] border border-[var(--border)] rounded-lg p-0.5">
                  {(
                    [
                      { key: 'all', label: 'All', icon: undefined },
                      {
                        key: 'annotations',
                        label: `${annotations.length}`,
                        icon: <PenTool size={10} />,
                      },
                      {
                        key: 'highlights',
                        label: `${pageHighlights.length}`,
                        icon: <Highlighter size={10} />,
                      },
                    ] as const
                  ).map(({ key, label, icon }) => (
                    <button
                      key={key}
                      onClick={() => setPageFilter(key as PageFilter)}
                      className={`px-2 py-1 text-[10px] font-medium rounded-md transition-all flex items-center gap-1 ${
                        pageFilter === key
                          ? 'bg-[var(--accent)] text-white shadow-sm'
                          : 'text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
                      }`}
                    >
                      {icon}
                      {label}
                    </button>
                  ))}
                </div>
              </div>

              {loadingAnnotations ? (
                <div className="flex items-center justify-center py-12">
                  <div className="animate-spin rounded-full h-6 w-6 border-2 border-[var(--accent)] border-t-transparent" />
                </div>
              ) : annotations.length + pageHighlights.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-[var(--text-tertiary)]">
                  <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mb-4">
                    <Sparkles size={22} className="text-[var(--accent)] opacity-60" />
                  </div>
                  <p className="text-sm font-medium mb-1">No activity yet</p>
                  <p className="text-xs text-[var(--text-tertiary)]">
                    Be the first to annotate or highlight this page
                  </p>
                </div>
              ) : (
                <div className="divide-y divide-[var(--border)]">
                  {(pageFilter === 'all' || pageFilter === 'annotations') &&
                    annotations.map((item) => (
                      <AnnotationCard
                        key={item.uri || item.id}
                        item={item}
                        sessionDid={session?.did}
                        formatDate={formatDate}
                        onAddToCollection={() => openCollectionModal(item.uri || item.id || '')}
                        onConverted={loadAnnotations}
                      />
                    ))}
                  {(pageFilter === 'all' || pageFilter === 'highlights') &&
                    pageHighlights.map((item) => (
                      <AnnotationCard
                        key={item.uri || item.id}
                        item={item}
                        sessionDid={session?.did}
                        formatDate={formatDate}
                        onAddToCollection={() => openCollectionModal(item.uri || item.id || '')}
                        onConverted={loadAnnotations}
                      />
                    ))}
                  {((pageFilter === 'annotations' && annotations.length === 0) ||
                    (pageFilter === 'highlights' && pageHighlights.length === 0)) && (
                    <div className="flex flex-col items-center justify-center py-10 text-[var(--text-tertiary)]">
                      <p className="text-xs">
                        No {pageFilter === 'annotations' ? 'annotations' : 'highlights'} on this
                        page
                      </p>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 'bookmarks' && (
          <div className="p-4">
            {loadingBookmarks ? (
              <div className="flex items-center justify-center py-16">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-[var(--accent)] border-t-transparent" />
              </div>
            ) : bookmarks.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16 text-[var(--text-tertiary)]">
                <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mb-4">
                  <BookmarkIcon size={22} className="text-[var(--accent)] opacity-60" />
                </div>
                <p className="text-sm font-medium mb-1">No bookmarks yet</p>
                <p className="text-xs text-[var(--text-tertiary)]">Save pages to read later</p>
              </div>
            ) : (
              <div className="space-y-2">
                {bookmarks.map((item) => (
                  <div
                    key={item.uri || item.id}
                    className="flex items-center gap-3 p-3 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl hover:bg-[var(--bg-hover)] hover:border-[var(--border-strong)] transition-all group"
                  >
                    <div className="w-9 h-9 rounded-lg bg-[var(--accent-subtle)] flex items-center justify-center flex-shrink-0">
                      <BookmarkIcon size={16} className="text-[var(--accent)]" />
                    </div>
                    <a
                      href={item.target?.source || item.source}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex-1 min-w-0"
                    >
                      <div className="text-sm font-medium truncate group-hover:text-[var(--accent)] transition-colors">
                        {item.target?.title || item.title || 'Untitled'}
                      </div>
                      <div className="text-xs text-[var(--text-tertiary)] truncate">
                        {item.target?.source || item.source
                          ? new URL(item.target?.source || item.source!).hostname
                          : ''}
                      </div>
                    </a>
                    <button
                      onClick={() => openCollectionModal(item.uri || item.id || '')}
                      className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent-subtle)] rounded-lg transition-all"
                      title="Add to collection"
                    >
                      <FolderPlus size={14} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'highlights' && (
          <div className="p-4">
            {loadingHighlights ? (
              <div className="flex items-center justify-center py-16">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-[var(--accent)] border-t-transparent" />
              </div>
            ) : highlights.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16 text-[var(--text-tertiary)]">
                <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mb-4">
                  <Highlighter size={22} className="text-[var(--accent)] opacity-60" />
                </div>
                <p className="text-sm font-medium mb-1">No highlights yet</p>
                <p className="text-xs text-[var(--text-tertiary)]">
                  Select text on any page to highlight
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {highlights.map((item) => (
                  <HighlightCard
                    key={item.uri || item.id}
                    item={item}
                    onAddToCollection={() => openCollectionModal(item.uri || item.id || '')}
                    onConverted={loadHighlights}
                  />
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'collections' && (
          <div className="p-4">
            {loadingCollections ? (
              <div className="flex items-center justify-center py-16">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-[var(--accent)] border-t-transparent" />
              </div>
            ) : collections.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16 text-[var(--text-tertiary)]">
                <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mb-4">
                  <Folder size={22} className="text-[var(--accent)] opacity-60" />
                </div>
                <p className="text-sm font-medium mb-1">No collections yet</p>
                <p className="text-xs text-[var(--text-tertiary)]">
                  Organize your annotations into collections
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {collections.map((item) => (
                  <button
                    key={item.uri || item.id}
                    onClick={() =>
                      browser.tabs.create({
                        url: `${apiUrl}/collection/${encodeURIComponent(item.uri || item.id || '')}`,
                      })
                    }
                    className="w-full text-left p-4 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl hover:bg-[var(--bg-hover)] hover:border-[var(--border-strong)] transition-all group flex items-center gap-3"
                  >
                    <div className="w-10 h-10 rounded-lg bg-[var(--accent)]/15 flex items-center justify-center flex-shrink-0 text-[var(--accent)] text-lg">
                      <CollectionIcon icon={item.icon} size={18} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium group-hover:text-[var(--accent)] transition-colors">
                        {item.name}
                      </div>
                      {item.description && (
                        <div className="text-xs text-[var(--text-tertiary)] truncate">
                          {item.description}
                        </div>
                      )}
                    </div>
                    <ChevronRight
                      size={16}
                      className="text-[var(--text-tertiary)] group-hover:text-[var(--accent)] transition-colors"
                    />
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {collectionModalItem && (
        <div
          className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 animate-fadeIn"
          onClick={() => setCollectionModalItem(null)}
        >
          <div
            className="bg-[var(--bg-primary)] rounded-2xl w-[90%] max-w-[340px] max-h-[80vh] overflow-hidden shadow-2xl animate-scaleIn"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-4 border-b border-[var(--border)]">
              <h3 className="text-sm font-bold">Add to Collection</h3>
              <button
                onClick={() => setCollectionModalItem(null)}
                className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)] rounded-lg transition-all"
              >
                <X size={16} />
              </button>
            </div>
            <div className="p-4 max-h-[300px] overflow-y-auto">
              {loadingCollections ? (
                <div className="flex items-center justify-center py-8">
                  <div className="animate-spin rounded-full h-6 w-6 border-2 border-[var(--accent)] border-t-transparent" />
                </div>
              ) : collections.length === 0 ? (
                <div className="text-center py-8 text-[var(--text-tertiary)]">
                  <Folder size={32} className="mx-auto mb-2 opacity-40" />
                  <p className="text-sm">No collections yet</p>
                  <p className="text-xs mt-1">Create collections on margin.at</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {collections.map((col) => {
                    const colUri = col.uri || col.id || '';
                    const isInCollection = containingCollections.has(colUri);
                    const isAdding = addingToCollection === colUri;
                    return (
                      <button
                        key={colUri}
                        onClick={() => !isInCollection && handleAddToCollection(colUri)}
                        disabled={isAdding || isInCollection}
                        className={`w-full text-left p-3 border rounded-xl transition-all flex items-center gap-3 ${
                          isInCollection
                            ? 'bg-emerald-400/10 border-emerald-400/30 cursor-default'
                            : 'bg-[var(--bg-card)] border-[var(--border)] hover:bg-[var(--bg-hover)] hover:border-[var(--accent)]'
                        }`}
                      >
                        <div
                          className={`w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 text-base ${
                            isInCollection
                              ? 'bg-emerald-400/15 text-emerald-400'
                              : 'bg-[var(--accent)]/15 text-[var(--accent)]'
                          }`}
                        >
                          <CollectionIcon icon={col.icon} size={16} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium">{col.name}</div>
                        </div>
                        {isAdding ? (
                          <div className="animate-spin rounded-full h-4 w-4 border-2 border-[var(--accent)] border-t-transparent" />
                        ) : isInCollection ? (
                          <Check size={16} className="text-emerald-400" />
                        ) : (
                          <FolderPlus size={16} className="text-[var(--text-tertiary)]" />
                        )}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      <footer className="flex items-center justify-center px-4 py-2 border-t border-[var(--border)]">
        <button
          onClick={() => browser.tabs.create({ url: apiUrl })}
          className="text-xs text-[var(--text-tertiary)] hover:text-[var(--accent)] flex items-center gap-1.5 py-1.5 px-3 rounded-lg hover:bg-[var(--bg-hover)] transition-colors"
        >
          Open Margin <ExternalLink size={11} />
        </button>
      </footer>
    </div>
  );
}

function AnnotationCard({
  item,
  sessionDid,
  formatDate,
  onAddToCollection,
  onConverted,
}: {
  item: Annotation;
  sessionDid?: string;
  formatDate: (d?: string) => string;
  onAddToCollection?: () => void;
  onConverted?: () => void;
}) {
  const [noteInput, setNoteInput] = useState('');
  const [showNoteInput, setShowNoteInput] = useState(false);
  const [converting, setConverting] = useState(false);

  const author = item.author || item.creator || {};
  const handle = author.handle || 'User';
  const text = item.body?.value || item.text || '';
  const selector = item.target?.selector;
  const quote = selector?.exact || '';
  const isHighlight = (item as any).type === 'Highlight';
  const isOwned = sessionDid && author.did === sessionDid;
  const highlightColor = item.color || (isHighlight ? '#fbbf24' : 'var(--accent)');

  async function handleConvert() {
    if (!noteInput.trim()) return;
    setConverting(true);
    try {
      const result = await sendMessage('convertHighlightToAnnotation', {
        highlightUri: item.uri || item.id || '',
        url: item.target?.source || '',
        text: noteInput.trim(),
        selector: item.target?.selector,
      });
      if (result.success) {
        setShowNoteInput(false);
        setNoteInput('');
        onConverted?.();
      } else {
        console.error('Convert failed:', result.error);
      }
    } catch (error) {
      console.error('Convert error:', error);
    } finally {
      setConverting(false);
    }
  }

  return (
    <div className="px-4 py-4 hover:bg-[var(--bg-hover)] transition-colors">
      <div className="flex items-start gap-3">
        <div className="w-9 h-9 rounded-full bg-gradient-to-br from-[var(--accent)] to-[var(--accent-hover)] flex items-center justify-center text-white text-xs font-bold flex-shrink-0 overflow-hidden shadow-sm">
          {author.avatar ? (
            <img src={author.avatar} alt={handle} className="w-full h-full object-cover" />
          ) : (
            handle[0]?.toUpperCase() || 'U'
          )}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1.5">
            <a
              href={`https://margin.at/profile/${author.did || ''}`}
              target="_blank"
              rel="noopener"
              className="text-sm font-semibold hover:text-[var(--accent)] cursor-pointer transition-colors no-underline text-inherit"
            >
              @{handle}
            </a>
            <span className="text-[11px] text-[var(--text-tertiary)]">
              {formatDate(item.created || item.createdAt)}
            </span>
            {isHighlight && (
              <span
                className="text-[10px] font-semibold px-2 py-0.5 rounded-full flex items-center gap-1"
                style={{
                  backgroundColor: `${highlightColor}20`,
                  color: highlightColor,
                }}
              >
                <Highlighter size={10} /> Highlight
              </span>
            )}
            <div className="ml-auto flex items-center gap-0.5">
              {isHighlight && isOwned && !showNoteInput && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowNoteInput(true);
                  }}
                  className="p-1 text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent-subtle)] rounded transition-all"
                  title="Add note (convert to annotation)"
                >
                  <PenTool size={13} />
                </button>
              )}
              {onAddToCollection && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onAddToCollection();
                  }}
                  className="p-1 text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent-subtle)] rounded transition-all"
                  title="Add to collection"
                >
                  <FolderPlus size={13} />
                </button>
              )}
              {!isHighlight && (
                <a
                  href={`https://margin.at/annotation/${encodeURIComponent(item.uri || item.id || '')}`}
                  target="_blank"
                  rel="noopener"
                  className="p-1 text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent-subtle)] rounded transition-all"
                  title="Reply"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MessageSquare size={13} />
                </a>
              )}
            </div>
          </div>

          {quote && (
            <div
              className="text-sm text-[var(--text-secondary)] border-l-2 pl-3 mb-2.5 py-1.5 rounded-r italic cursor-pointer hover:opacity-80 transition-all"
              style={{
                borderColor: highlightColor,
                backgroundColor: `${highlightColor}12`,
              }}
              onClick={async (e) => {
                e.stopPropagation();
                const [tab] = await browser.tabs.query({ active: true, currentWindow: true });
                if (tab?.id) {
                  browser.tabs.sendMessage(tab.id, { type: 'SCROLL_TO_TEXT', text: quote });
                  window.close();
                }
              }}
              title="Jump to text on page"
            >
              "{quote.length > 200 ? quote.slice(0, 200) + '...' : quote}"
            </div>
          )}

          {text && (
            <div className="text-[13px] leading-relaxed text-[var(--text-primary)]">{text}</div>
          )}

          {showNoteInput && (
            <div className="mt-2.5 flex gap-2 items-end animate-fadeIn">
              <textarea
                value={noteInput}
                onChange={(e) => setNoteInput(e.target.value)}
                placeholder="Add your note..."
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    handleConvert();
                  }
                  if (e.key === 'Escape') {
                    setShowNoteInput(false);
                    setNoteInput('');
                  }
                }}
                className="flex-1 p-2.5 bg-[var(--bg-card)] border border-[var(--border)] rounded-lg text-xs resize-none focus:outline-none focus:border-[var(--accent)] focus:ring-1 focus:ring-[var(--accent-subtle)] min-h-[60px]"
              />
              <div className="flex flex-col gap-1.5">
                <button
                  onClick={handleConvert}
                  disabled={converting || !noteInput.trim()}
                  className="p-2 bg-[var(--accent)] text-white rounded-lg hover:bg-[var(--accent-hover)] disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                  title="Convert to annotation"
                >
                  {converting ? (
                    <div className="animate-spin rounded-full h-3.5 w-3.5 border-2 border-white border-t-transparent" />
                  ) : (
                    <Send size={14} />
                  )}
                </button>
                <button
                  onClick={() => {
                    setShowNoteInput(false);
                    setNoteInput('');
                  }}
                  className="p-2 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)] rounded-lg transition-all"
                  title="Cancel"
                >
                  <X size={14} />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function HighlightCard({
  item,
  onAddToCollection,
  onConverted,
}: {
  item: Highlight;
  onAddToCollection?: () => void;
  onConverted?: () => void;
}) {
  const [noteInput, setNoteInput] = useState('');
  const [showNoteInput, setShowNoteInput] = useState(false);
  const [converting, setConverting] = useState(false);

  const hlItem = item as any;
  const selector = hlItem.target?.selector || hlItem.selector;
  const source = hlItem.target?.source || hlItem.source || hlItem.url || '';
  const quote = selector?.exact || '';
  const color = hlItem.color || '#fbbf24';

  async function handleConvert() {
    if (!noteInput.trim()) return;
    setConverting(true);
    try {
      const result = await sendMessage('convertHighlightToAnnotation', {
        highlightUri: hlItem.uri || hlItem.id || '',
        url: source,
        text: noteInput.trim(),
        selector: selector,
      });
      if (result.success) {
        setShowNoteInput(false);
        setNoteInput('');
        onConverted?.();
      } else {
        console.error('Convert failed:', result.error);
      }
    } catch (error) {
      console.error('Convert error:', error);
    } finally {
      setConverting(false);
    }
  }

  return (
    <div className="p-4 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl hover:bg-[var(--bg-hover)] hover:border-[var(--border-strong)] transition-all group">
      {quote && (
        <div
          className="text-sm leading-relaxed border-l-3 pl-3 mb-3 py-1"
          style={{
            borderColor: color,
            background: `linear-gradient(90deg, ${color}15, transparent)`,
          }}
        >
          "{quote.length > 120 ? quote.slice(0, 120) + '...' : quote}"
        </div>
      )}

      <div className="flex items-center justify-between">
        <div
          className="flex items-center gap-2 text-xs text-[var(--text-tertiary)] flex-1 cursor-pointer hover:text-[var(--accent)]"
          onClick={() => {
            if (source) browser.tabs.create({ url: source });
          }}
        >
          <Globe size={12} />
          {source ? new URL(source).hostname : ''}
          <ChevronRight
            size={14}
            className="ml-auto text-[var(--text-tertiary)] group-hover:text-[var(--accent)] transition-colors"
          />
        </div>
        <div className="flex items-center gap-0.5 ml-2">
          {!showNoteInput && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowNoteInput(true);
              }}
              className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent-subtle)] rounded-lg transition-all"
              title="Add note (convert to annotation)"
            >
              <PenTool size={13} />
            </button>
          )}
          {onAddToCollection && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onAddToCollection();
              }}
              className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent-subtle)] rounded-lg transition-all"
              title="Add to collection"
            >
              <FolderPlus size={14} />
            </button>
          )}
        </div>
      </div>

      {showNoteInput && (
        <div className="mt-3 flex gap-2 items-end animate-fadeIn">
          <textarea
            value={noteInput}
            onChange={(e) => setNoteInput(e.target.value)}
            placeholder="Add your note..."
            autoFocus
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleConvert();
              }
              if (e.key === 'Escape') {
                setShowNoteInput(false);
                setNoteInput('');
              }
            }}
            className="flex-1 p-2.5 bg-[var(--bg-primary)] border border-[var(--border)] rounded-lg text-xs resize-none focus:outline-none focus:border-[var(--accent)] focus:ring-1 focus:ring-[var(--accent-subtle)] min-h-[60px]"
          />
          <div className="flex flex-col gap-1.5">
            <button
              onClick={handleConvert}
              disabled={converting || !noteInput.trim()}
              className="p-2 bg-[var(--accent)] text-white rounded-lg hover:bg-[var(--accent-hover)] disabled:opacity-40 disabled:cursor-not-allowed transition-all"
              title="Convert to annotation"
            >
              {converting ? (
                <div className="animate-spin rounded-full h-3.5 w-3.5 border-2 border-white border-t-transparent" />
              ) : (
                <Send size={14} />
              )}
            </button>
            <button
              onClick={() => {
                setShowNoteInput(false);
                setNoteInput('');
              }}
              className="p-2 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)] rounded-lg transition-all"
              title="Cancel"
            >
              <X size={14} />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
