import type { MarginSession } from './types';

export const apiUrlItem = storage.defineItem<string>('local:apiUrl', {
  fallback: 'https://margin.at',
});

export const cachedSessionItem = storage.defineItem<MarginSession | null>('local:cachedSession', {
  fallback: null,
});

export const overlayEnabledItem = storage.defineItem<boolean>('local:overlayEnabled', {
  fallback: true,
});

export const themeItem = storage.defineItem<'light' | 'dark' | 'system'>('local:theme', {
  fallback: 'system',
});
