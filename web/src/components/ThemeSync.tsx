'use client';

import React from 'react';

const THEME_KEY = 'elmakina.theme';
const DEFAULT_THEME: 'light' | 'dark' = 'dark';

function applyTheme(theme: 'light' | 'dark') {
  if (typeof document === 'undefined') return;
  document.documentElement.classList.toggle('dark', theme === 'dark');
}

export function ThemeSync() {
  React.useEffect(() => {
    if (typeof window === 'undefined') return;
    const stored = localStorage.getItem(THEME_KEY) as 'light' | 'dark' | null;
    applyTheme(stored ?? DEFAULT_THEME);

    const handleStorage = (event: StorageEvent) => {
      if (event.key !== THEME_KEY) return;
      const nextTheme = (event.newValue as 'light' | 'dark' | null) ?? DEFAULT_THEME;
      applyTheme(nextTheme);
    };

    window.addEventListener('storage', handleStorage);
    return () => window.removeEventListener('storage', handleStorage);
  }, []);

  return null;
}
