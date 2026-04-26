import { useState, useEffect } from 'react';
import { sendMessage } from '@/utils/messaging';
import { themeItem, overlayEnabledItem } from '@/utils/storage';
import type { MarginSession, Annotation, Bookmark, Highlight, Collection } from '@/utils/types';
import { APP_URL } from '@/utils/types';
import CollectionIcon from '@/components/CollectionIcon';
import TagInput from '@/components/TagInput';

type Tab = 'page' | 'bookmarks' | 'highlights' | 'collections';
type PageFilter = 'all' | 'annotations' | 'highlights';

const Icons = {
  settings: (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="M12 15a3 3 0 100-6 3 3 0 000 6z" />
      <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z" />
    </svg>
  ),
  globe: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <circle cx="12" cy="12" r="10" />
      <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" />
      <path d="M2 12h20" />
    </svg>
  ),
  bookmark: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="m19 21-7-4-7 4V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v16z" />
    </svg>
  ),
  highlighter: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="m9 11-6 6v3h9l3-3" />
      <path d="m22 12-4.6 4.6a2 2 0 0 1-2.8 0l-5.2-5.2a2 2 0 0 1 0-2.8L14 4" />
    </svg>
  ),
  folder: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h16z" />
    </svg>
  ),
  folderPlus: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="M12 10v6" />
      <path d="M9 13h6" />
      <path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" />
    </svg>
  ),
  check: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  ),
  chevronRight: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <polyline points="9 18 15 12 9 6" />
    </svg>
  ),
  sparkles: (
    <svg className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z" />
    </svg>
  ),
  externalLink: (
    <svg className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
      <polyline points="15 3 21 3 21 9" />
      <line x1="10" x2="21" y1="14" y2="3" />
    </svg>
  ),
  x: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <path d="M18 6 6 18" />
      <path d="m6 6 12 12" />
    </svg>
  ),
};

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
  const [currentUrl, setCurrentUrl] = useState('');
  const [currentTitle, setCurrentTitle] = useState('');
  const [text, setText] = useState('');
  const [posting, setPosting] = useState(false);
  const [bookmarking, setBookmarking] = useState(false);
  const [bookmarked, setBookmarked] = useState(false);
  const [theme, setTheme] = useState<'light' | 'dark' | 'system'>('system');
  const [overlayEnabled, setOverlayEnabled] = useState(true);
  const [showSettings, setShowSettings] = useState(false);
  const [collectionModalItem, setCollectionModalItem] = useState<string | null>(null);
  const [addingToCollection, setAddingToCollection] = useState<string | null>(null);
  const [containingCollections, setContainingCollections] = useState<Set<string>>(new Set());
  const [tags, setTags] = useState<string[]>([]);
  const [bookmarkTags, setBookmarkTags] = useState<string[]>([]);
  const [showBookmarkTags, setShowBookmarkTags] = useState(false);
  const [tagSuggestions, setTagSuggestions] = useState<string[]>([]);

  useEffect(() => {
    checkSession();
    loadCurrentTab();
    loadSettings();

    browser.tabs.onActivated.addListener(loadCurrentTab);
    browser.tabs.onUpdated.addListener((_, info) => {
      if (info.url) loadCurrentTab();
    });

    return () => {
      browser.tabs.onActivated.removeListener(loadCurrentTab);
    };
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
    const t = await themeItem.getValue();
    setTheme(t);
    applyTheme(t);

    const overlay = await overlayEnabledItem.getValue();
    setOverlayEnabled(overlay);

    themeItem.watch((newTheme) => {
      setTheme(newTheme);
      applyTheme(newTheme);
    });

    overlayEnabledItem.watch((enabled) => {
      setOverlayEnabled(enabled);
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
      } else {
        const errorMessage = result.error
          ? `Failed to post annotation: ${result.error}`
          : 'Failed to post annotation';
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
      } else {
        const errorMessage = result.error
          ? `Failed to bookmark page: ${result.error}`
          : 'Failed to bookmark page';
        alert(errorMessage);
      }
    } catch (error) {
      console.error('Bookmark error:', error);
      alert('Error bookmarking page');
    } finally {
      setBookmarking(false);
    }
  }

  async function handleThemeChange(newTheme: 'light' | 'dark' | 'system') {
    await themeItem.setValue(newTheme);
  }

  async function handleOverlayToggle() {
    await overlayEnabledItem.setValue(!overlayEnabled);
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
      <div className="flex flex-col items-center justify-center h-screen p-8 text-center">
        <div className="w-14 h-14 rounded-2xl bg-[var(--accent-subtle)] flex items-center justify-center mb-5">
          <img src="/icons/logo.svg" alt="Margin" className="w-8 h-8" />
        </div>
        <h2 className="font-display text-xl font-bold tracking-tight mb-2">Welcome to Margin</h2>
        <p className="text-[var(--text-secondary)] text-sm mb-8 max-w-xs leading-relaxed">
          Annotate, highlight, and bookmark the web with your AT Protocol identity.
        </p>
        <button
          onClick={() => browser.tabs.create({ url: `${APP_URL}/login` })}
          className="px-8 py-2.5 bg-[var(--accent)] text-white rounded-xl text-sm font-semibold hover:bg-[var(--accent-hover)] transition-colors"
        >
          Sign In
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen">
      <header className="flex items-center justify-between px-4 py-2.5 border-b border-[var(--border)]">
        <div className="flex items-center gap-2">
          <img src="/icons/logo.svg" alt="Margin" className="w-5 h-5" />
          <span className="font-display font-bold text-sm tracking-tight">Margin</span>
        </div>
        <div className="flex items-center gap-1.5">
          <button
            onClick={() => browser.tabs.create({ url: APP_URL })}
            className="text-[11px] text-[var(--text-tertiary)] hover:text-[var(--accent)] px-2 py-1 rounded-md hover:bg-[var(--bg-hover)] transition-colors"
          >
            @{session.handle}
          </button>
          <button
            onClick={() => setShowSettings(!showSettings)}
            className="p-1.5 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)] rounded-lg transition-colors"
            title="Settings"
          >
            {Icons.settings}
          </button>
        </div>
      </header>

      {showSettings && (
        <div className="p-4 border-b border-[var(--border)] space-y-4 animate-slideDown">
          <div>
            <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">
              Theme
            </label>
            <div className="flex gap-1.5">
              {(['light', 'dark', 'system'] as const).map((t) => (
                <button
                  key={t}
                  onClick={() => handleThemeChange(t)}
                  className={`flex-1 py-2 px-3 text-xs font-medium rounded-lg transition-colors ${
                    theme === t
                      ? 'bg-[var(--accent)] text-white'
                      : 'bg-[var(--bg-card)] border border-[var(--border)] hover:bg-[var(--bg-hover)]'
                  }`}
                >
                  {t.charAt(0).toUpperCase() + t.slice(1)}
                </button>
              ))}
            </div>
          </div>

          <div className="flex items-center justify-between p-3 bg-[var(--bg-card)] border border-[var(--border)] rounded-lg">
            <div>
              <div className="text-sm font-medium">Page Overlay</div>
              <p className="text-[11px] text-[var(--text-tertiary)] mt-0.5">
                Show highlights and annotations on pages
              </p>
            </div>
            <button
              onClick={handleOverlayToggle}
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

          <button
            onClick={() => browser.tabs.create({ url: APP_URL })}
            className="w-full py-2 text-xs font-medium text-[var(--text-tertiary)] hover:text-[var(--accent)] rounded-lg hover:bg-[var(--bg-hover)] transition-colors flex items-center justify-center gap-1.5"
          >
            Open Margin {Icons.externalLink}
          </button>
        </div>
      )}

      <div className="flex border-b border-[var(--border)]">
        {(['page', 'bookmarks', 'highlights', 'collections'] as Tab[]).map((tab) => {
          const icons = {
            page: Icons.globe,
            bookmarks: Icons.bookmark,
            highlights: Icons.highlighter,
            collections: Icons.folder,
          };
          const labels = {
            page: 'Page',
            bookmarks: 'Bookmarks',
            highlights: 'Highlights',
            collections: 'Collections',
          };
          return (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`flex-1 py-2.5 text-[11px] font-medium flex items-center justify-center gap-1 border-b-2 transition-all ${
                activeTab === tab
                  ? 'border-[var(--accent)] text-[var(--accent)]'
                  : 'border-transparent text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
              }`}
            >
              {icons[tab]}
              <span className="hidden min-[340px]:inline">{labels[tab]}</span>
            </button>
          );
        })}
      </div>

      <div className="flex-1 overflow-y-auto">
        {activeTab === 'page' && (
          <div className="p-4">
            <div className="mb-4 p-3 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl">
              <div className="flex items-center gap-3">
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
                  <span className={`text-[var(--text-tertiary)] ${currentUrl ? 'hidden' : ''}`}>
                    {Icons.globe}
                  </span>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">{currentTitle || 'Untitled'}</div>
                  <div className="text-[11px] text-[var(--text-tertiary)] truncate">
                    {currentUrl ? new URL(currentUrl).hostname : ''}
                  </div>
                </div>
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
                  {bookmarked ? Icons.check : Icons.bookmark}
                </button>
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

            <div className="mb-6">
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
                  className="px-5 py-1.5 bg-[var(--accent)] text-white text-xs rounded-lg font-medium hover:bg-[var(--accent-hover)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                >
                  {posting ? 'Posting...' : 'Post'}
                </button>
              </div>
            </div>

            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold text-[var(--text-secondary)]">Activity</span>
                <span className="text-xs font-semibold bg-[var(--accent-subtle)] text-[var(--accent)] px-2 py-0.5 rounded-full">
                  {annotations.length + pageHighlights.length}
                </span>
              </div>
              <div className="flex items-center bg-[var(--bg-card)] border border-[var(--border)] rounded-lg p-0.5">
                {(['all', 'annotations', 'highlights'] as const).map((key) => (
                  <button
                    key={key}
                    onClick={() => setPageFilter(key)}
                    className={`px-2 py-1 text-[10px] font-medium rounded-md transition-all ${
                      pageFilter === key
                        ? 'bg-[var(--accent)] text-white shadow-sm'
                        : 'text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
                    }`}
                  >
                    {key === 'all' ? 'All' : key === 'annotations' ? 'Notes' : 'HL'}
                  </button>
                ))}
              </div>
            </div>

            {loadingAnnotations ? (
              <div className="flex items-center justify-center py-16">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-[var(--accent)] border-t-transparent" />
              </div>
            ) : annotations.length + pageHighlights.length === 0 ? (
              <div className="text-center py-16 text-[var(--text-tertiary)]">
                <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mx-auto mb-4">
                  {Icons.sparkles}
                </div>
                <p className="font-medium mb-1">No activity yet</p>
                <p className="text-xs">Be the first to annotate or highlight</p>
              </div>
            ) : (
              <div className="space-y-3">
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
                  <div className="text-center py-10 text-[var(--text-tertiary)]">
                    <p className="text-xs">
                      No {pageFilter === 'annotations' ? 'annotations' : 'highlights'} on this page
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {activeTab === 'bookmarks' && (
          <div className="p-4">
            {loadingBookmarks ? (
              <div className="flex items-center justify-center py-16">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-[var(--accent)] border-t-transparent" />
              </div>
            ) : bookmarks.length === 0 ? (
              <div className="text-center py-16 text-[var(--text-tertiary)]">
                <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mx-auto mb-4 text-[var(--accent)]">
                  {Icons.bookmark}
                </div>
                <p className="font-medium mb-1">No bookmarks yet</p>
                <p className="text-xs">Save pages to read later</p>
              </div>
            ) : (
              <div className="space-y-2">
                {bookmarks.map((item) => (
                  <div
                    key={item.uri || item.id}
                    className="flex items-center gap-3 p-3 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl hover:bg-[var(--bg-hover)] transition-all group"
                  >
                    <div className="w-9 h-9 rounded-lg bg-[var(--accent)]/15 flex items-center justify-center flex-shrink-0 text-[var(--accent)]">
                      {Icons.bookmark}
                    </div>
                    <a
                      href={item.source}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex-1 min-w-0"
                    >
                      <div className="text-sm font-medium truncate group-hover:text-[var(--accent)] transition-colors">
                        {item.title || item.source}
                      </div>
                      <div className="text-xs text-[var(--text-tertiary)] truncate">
                        {item.source ? new URL(item.source).hostname : ''}
                      </div>
                    </a>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        if (item.uri) openCollectionModal(item.uri);
                      }}
                      className="p-1.5 rounded-lg text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent)]/10 transition-colors"
                      title="Add to collection"
                    >
                      {Icons.folderPlus}
                    </button>
                    <a
                      href={item.source}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-[var(--text-tertiary)] group-hover:text-[var(--accent)]"
                    >
                      {Icons.chevronRight}
                    </a>
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
              <div className="text-center py-16 text-[var(--text-tertiary)]">
                <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mx-auto mb-4 text-[var(--accent)]">
                  {Icons.highlighter}
                </div>
                <p className="font-medium mb-1">No highlights yet</p>
                <p className="text-xs">Select text on any page to highlight</p>
              </div>
            ) : (
              <div className="space-y-3">
                {highlights.map((item) => (
                  <SidepanelHighlightCard
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
              <div className="text-center py-16 text-[var(--text-tertiary)]">
                <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mx-auto mb-4 text-[var(--accent)]">
                  {Icons.folder}
                </div>
                <p className="font-medium mb-1">No collections yet</p>
                <p className="text-xs">Organize your bookmarks into collections</p>
              </div>
            ) : (
              <div className="space-y-2">
                {collections.map((item) => (
                  <button
                    key={item.uri || item.id}
                    onClick={() =>
                      browser.tabs.create({
                        url: `${APP_URL}/collection/${encodeURIComponent(item.uri || item.id || '')}`,
                      })
                    }
                    className="w-full text-left p-4 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl hover:bg-[var(--bg-hover)] transition-all group flex items-center gap-3"
                  >
                    <div className="w-9 h-9 rounded-lg bg-[var(--accent)]/15 flex items-center justify-center flex-shrink-0 text-[var(--accent)] text-lg">
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
                    <div className="text-[var(--text-tertiary)] group-hover:text-[var(--accent)]">
                      {Icons.chevronRight}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {collectionModalItem && (
        <div
          className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4"
          onClick={() => setCollectionModalItem(null)}
        >
          <div
            className="bg-[var(--bg-primary)] border border-[var(--border)] rounded-2xl w-full max-w-sm max-h-[70vh] overflow-hidden shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-4 border-b border-[var(--border)]">
              <h3 className="font-semibold">Add to Collection</h3>
              <button
                onClick={() => setCollectionModalItem(null)}
                className="p-1 rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-tertiary)]"
              >
                {Icons.x}
              </button>
            </div>
            <div className="p-2 max-h-[50vh] overflow-y-auto">
              {collections.length === 0 ? (
                <div className="text-center py-8 text-[var(--text-tertiary)]">
                  <p className="text-sm">No collections yet</p>
                  <p className="text-xs mt-1">Create a collection on the web app first</p>
                </div>
              ) : (
                collections.map((col) => {
                  const colUri = col.uri || col.id || '';
                  const isAdding = addingToCollection === colUri;
                  const isInCollection = containingCollections.has(colUri);
                  return (
                    <button
                      key={colUri}
                      onClick={() => !isInCollection && handleAddToCollection(colUri)}
                      disabled={isAdding || isInCollection}
                      className={`w-full text-left p-3 rounded-xl flex items-center gap-3 transition-all ${
                        isInCollection
                          ? 'bg-emerald-400/10 cursor-default'
                          : 'hover:bg-[var(--bg-hover)]'
                      }`}
                    >
                      <div
                        className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 text-base ${
                          isInCollection
                            ? 'bg-emerald-400/15 text-emerald-400'
                            : 'bg-[var(--accent)]/15 text-[var(--accent)]'
                        }`}
                      >
                        <CollectionIcon icon={col.icon} size={16} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium truncate">{col.name}</div>
                        {col.description && (
                          <div className="text-xs text-[var(--text-tertiary)] truncate">
                            {col.description}
                          </div>
                        )}
                      </div>
                      {isAdding ? (
                        <div className="animate-spin rounded-full h-4 w-4 border-2 border-[var(--accent)] border-t-transparent" />
                      ) : isInCollection ? (
                        <span className="text-emerald-400">{Icons.check}</span>
                      ) : (
                        Icons.folderPlus
                      )}
                    </button>
                  );
                })
              )}
            </div>
          </div>
        </div>
      )}
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
    <div className="p-4 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl hover:bg-[var(--bg-hover)] transition-all">
      <div className="flex items-start gap-3">
        <div className="w-9 h-9 rounded-full bg-gradient-to-br from-[var(--accent)] to-[var(--accent-hover)] flex items-center justify-center text-white text-xs font-bold overflow-hidden flex-shrink-0">
          {author.avatar ? (
            <img src={author.avatar} alt={handle} className="w-full h-full object-cover" />
          ) : (
            handle[0]?.toUpperCase() || 'U'
          )}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <a
              href={`${APP_URL}/profile/${author.did || ''}`}
              target="_blank"
              rel="noopener"
              className="text-sm font-semibold hover:text-[var(--accent)] cursor-pointer transition-colors no-underline text-inherit"
            >
              @{handle}
            </a>
            <span className="text-xs text-[var(--text-tertiary)]">
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
                Highlight
              </span>
            )}
            <div className="ml-auto flex items-center gap-0.5">
              {isHighlight && isOwned && !showNoteInput && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowNoteInput(true);
                  }}
                  className="p-1 rounded text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent)]/10 transition-colors"
                  title="Add note (convert to annotation)"
                >
                  <svg
                    className="w-3.5 h-3.5"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    viewBox="0 0 24 24"
                  >
                    <path d="M12 19l7-7 3 3-7 7-3-3z" />
                    <path d="M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z" />
                    <path d="M2 2l7.586 7.586" />
                    <circle cx="11" cy="11" r="2" />
                  </svg>
                </button>
              )}
              {onAddToCollection && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onAddToCollection();
                  }}
                  className="p-1 rounded text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent)]/10 transition-colors"
                  title="Add to collection"
                >
                  {Icons.folderPlus}
                </button>
              )}
              {!isHighlight && (
                <a
                  href={`${APP_URL}/annotation/${encodeURIComponent(item.uri || item.id || '')}`}
                  target="_blank"
                  rel="noopener"
                  className="p-1 rounded text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent)]/10 transition-colors"
                  title="Reply"
                  onClick={(e) => e.stopPropagation()}
                >
                  <svg
                    className="w-3.5 h-3.5"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    viewBox="0 0 24 24"
                  >
                    <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z" />
                  </svg>
                </a>
              )}
            </div>
          </div>

          {quote && (
            <div
              className="text-sm text-[var(--text-secondary)] border-l-2 pl-3 mb-2 py-1 rounded-r italic cursor-pointer hover:opacity-80 transition-all"
              style={{
                borderColor: highlightColor,
                backgroundColor: `${highlightColor}12`,
              }}
              onClick={async (e) => {
                e.stopPropagation();
                const [tab] = await browser.tabs.query({ active: true, currentWindow: true });
                if (tab?.id) {
                  browser.tabs.sendMessage(tab.id, { type: 'SCROLL_TO_TEXT', text: quote });
                }
              }}
              title="Jump to text on page"
            >
              "{quote.length > 250 ? quote.slice(0, 250) + '...' : quote}"
            </div>
          )}

          {text && <div className="text-sm leading-relaxed">{text}</div>}

          {showNoteInput && (
            <div className="mt-2.5 flex gap-2 items-end">
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
                className="flex-1 p-2.5 bg-[var(--bg-primary)] border border-[var(--border)] rounded-lg text-xs resize-none focus:outline-none focus:border-[var(--accent)] focus:ring-1 focus:ring-[var(--accent)]/20 min-h-[60px]"
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
                    <svg
                      className="w-3.5 h-3.5"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      viewBox="0 0 24 24"
                    >
                      <line x1="22" x2="11" y1="2" y2="13" />
                      <polygon points="22 2 15 22 11 13 2 9 22 2" />
                    </svg>
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
                  {Icons.x}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function SidepanelHighlightCard({
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
    <div className="p-4 bg-[var(--bg-card)] border border-[var(--border)] rounded-xl hover:bg-[var(--bg-hover)] transition-all group">
      {quote && (
        <div
          className="text-sm leading-relaxed border-l-3 pl-3 mb-3 cursor-pointer"
          style={{
            borderColor: color,
            background: `linear-gradient(90deg, ${color}10, transparent)`,
          }}
          onClick={() => {
            if (source) browser.tabs.create({ url: source });
          }}
        >
          "{quote.length > 150 ? quote.slice(0, 150) + '...' : quote}"
        </div>
      )}

      <div className="flex items-center justify-between text-xs text-[var(--text-tertiary)]">
        <span className="flex items-center gap-1.5">
          {Icons.globe} {source ? new URL(source).hostname : ''}
        </span>
        <div className="flex items-center gap-1">
          {!showNoteInput && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowNoteInput(true);
              }}
              className="p-1 rounded text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent)]/10 transition-colors"
              title="Add note (convert to annotation)"
            >
              <svg
                className="w-3.5 h-3.5"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                viewBox="0 0 24 24"
              >
                <path d="M12 19l7-7 3 3-7 7-3-3z" />
                <path d="M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z" />
                <path d="M2 2l7.586 7.586" />
                <circle cx="11" cy="11" r="2" />
              </svg>
            </button>
          )}
          {onAddToCollection && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onAddToCollection();
              }}
              className="p-1 rounded text-[var(--text-tertiary)] hover:text-[var(--accent)] hover:bg-[var(--accent)]/10 transition-colors"
              title="Add to collection"
            >
              {Icons.folderPlus}
            </button>
          )}
          <button
            onClick={() => {
              if (source) browser.tabs.create({ url: source });
            }}
            className="group-hover:text-[var(--accent)]"
          >
            {Icons.chevronRight}
          </button>
        </div>
      </div>

      {showNoteInput && (
        <div className="mt-3 flex gap-2 items-end">
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
            className="flex-1 p-2.5 bg-[var(--bg-primary)] border border-[var(--border)] rounded-lg text-xs resize-none focus:outline-none focus:border-[var(--accent)] focus:ring-1 focus:ring-[var(--accent)]/20 min-h-[60px]"
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
                <svg
                  className="w-3.5 h-3.5"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  viewBox="0 0 24 24"
                >
                  <line x1="22" x2="11" y1="2" y2="13" />
                  <polygon points="22 2 15 22 11 13 2 9 22 2" />
                </svg>
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
              {Icons.x}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
