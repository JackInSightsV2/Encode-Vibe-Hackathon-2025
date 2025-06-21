import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import { Theme, ThemeName, ThemeContextType } from '../types/theme';
import { themes, defaultTheme } from '../themes';
import { preloadThemeFonts } from '../utils/themeLoader';

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

interface ThemeProviderProps {
  children: ReactNode;
  defaultThemeName?: ThemeName;
}

const THEME_STORAGE_KEY = 'qt1-theme-preference';

export function ThemeProvider({ children, defaultThemeName = 'light' }: ThemeProviderProps) {
  const [themeName, setThemeName] = useState<ThemeName>(defaultThemeName);
  const [theme, setTheme] = useState<Theme>(defaultTheme);
  const [isThemeLoading, setIsThemeLoading] = useState(false);
  const [isInitialized, setIsInitialized] = useState(false);

  // Load saved theme preference on mount
  useEffect(() => {
    const loadSavedTheme = async () => {
      try {
        const savedTheme = localStorage.getItem(THEME_STORAGE_KEY) as ThemeName;
        const themeToLoad = savedTheme && themes[savedTheme] ? savedTheme : defaultThemeName;
        
        setIsThemeLoading(true);
        await applyTheme(themeToLoad);
        setIsInitialized(true);
      } catch (error) {
        console.warn('Failed to load saved theme:', error);
        await applyTheme(defaultThemeName);
        setIsInitialized(true);
      } finally {
        setIsThemeLoading(false);
      }
    };

    loadSavedTheme();
  }, [defaultThemeName]);

  const applyTheme = useCallback(async (newThemeName: ThemeName) => {
    const newTheme = themes[newThemeName];
    if (!newTheme) {
      console.warn(`Theme "${newThemeName}" not found, falling back to default`);
      return;
    }

    setIsThemeLoading(true);

    try {
      // Load theme fonts
      await preloadThemeFonts(newTheme.fonts);

      // Apply CSS custom properties
      const root = document.documentElement;
      
      // Colors
      root.style.setProperty('--color-primary', newTheme.colors.primary);
      root.style.setProperty('--color-secondary', newTheme.colors.secondary);
      root.style.setProperty('--color-accent', newTheme.colors.accent);
      root.style.setProperty('--color-background', newTheme.colors.background);
      root.style.setProperty('--color-surface', newTheme.colors.surface);
      root.style.setProperty('--color-surface-alt', newTheme.colors.surfaceAlt);
      
      // Text colors
      root.style.setProperty('--color-text-primary', newTheme.colors.text.primary);
      root.style.setProperty('--color-text-secondary', newTheme.colors.text.secondary);
      root.style.setProperty('--color-text-tertiary', newTheme.colors.text.tertiary);
      root.style.setProperty('--color-text-inverse', newTheme.colors.text.inverse);
      
      // Border colors
      root.style.setProperty('--color-border-default', newTheme.colors.border.default);
      root.style.setProperty('--color-border-light', newTheme.colors.border.light);
      root.style.setProperty('--color-border-focus', newTheme.colors.border.focus);
      
      // Status colors
      root.style.setProperty('--color-success', newTheme.colors.status.success);
      root.style.setProperty('--color-warning', newTheme.colors.status.warning);
      root.style.setProperty('--color-error', newTheme.colors.status.error);
      root.style.setProperty('--color-info', newTheme.colors.status.info);
      
      // Fonts
      root.style.setProperty('--font-heading', newTheme.fonts.heading);
      root.style.setProperty('--font-body', newTheme.fonts.body);
      root.style.setProperty('--font-mono', newTheme.fonts.mono);
      if (newTheme.fonts.display) {
        root.style.setProperty('--font-display', newTheme.fonts.display);
      }
      
      // Effects
      root.style.setProperty('--shadow-sm', newTheme.effects.shadows.sm);
      root.style.setProperty('--shadow-md', newTheme.effects.shadows.md);
      root.style.setProperty('--shadow-lg', newTheme.effects.shadows.lg);
      if (newTheme.effects.shadows.glow) {
        root.style.setProperty('--shadow-glow', newTheme.effects.shadows.glow);
      }
      
      // Gradients
      if (newTheme.effects.gradients) {
        if (newTheme.effects.gradients.primary) {
          root.style.setProperty('--gradient-primary', newTheme.effects.gradients.primary);
        }
        if (newTheme.effects.gradients.accent) {
          root.style.setProperty('--gradient-accent', newTheme.effects.gradients.accent);
        }
        if (newTheme.effects.gradients.background) {
          root.style.setProperty('--gradient-background', newTheme.effects.gradients.background);
        }
      }
      
      // Animations
      if (newTheme.effects.animations) {
        root.style.setProperty('--animation-duration', newTheme.effects.animations.duration);
        root.style.setProperty('--animation-easing', newTheme.effects.animations.easing);
      }
      
      // Special effects
      if (newTheme.effects.special) {
        if (newTheme.effects.special.neonGlow) {
          root.style.setProperty('--effect-neon-glow', newTheme.effects.special.neonGlow);
        }
        if (newTheme.effects.special.scanlines) {
          root.style.setProperty('--effect-scanlines', newTheme.effects.special.scanlines);
        }
        if (newTheme.effects.special.chrome) {
          root.style.setProperty('--effect-chrome', newTheme.effects.special.chrome);
        }
      }

      // Update body class for theme-specific styling
      document.body.className = document.body.className.replace(/theme-\w+/g, '');
      document.body.classList.add(`theme-${newThemeName}`);

      // Update state
      setTheme(newTheme);
      setThemeName(newThemeName);

      // Save to localStorage
      localStorage.setItem(THEME_STORAGE_KEY, newThemeName);
    } catch (error) {
      console.error('Failed to apply theme:', error);
    } finally {
      setIsThemeLoading(false);
    }
  }, []);

  const handleSetTheme = useCallback((newThemeName: ThemeName) => {
    applyTheme(newThemeName);
  }, [applyTheme]);

  const value: ThemeContextType = {
    theme,
    themeName,
    setTheme: handleSetTheme,
    isThemeLoading,
    isInitialized,
  };

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextType {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

export { ThemeContext };