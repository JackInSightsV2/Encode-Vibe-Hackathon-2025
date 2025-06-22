# UI Theme Implementation - Refined Implementation Cycles

## Overview
Break down theme implementation into 5 manageable cycles, implementing five distinct themes with proper architecture, testing, and validation.

---

## **Cycle 21A: Theme Architecture & Foundation**
**Duration:** 6-8 hours | **Priority:** High

### Prerequisites
- Frontend development environment ready
- Understanding of CSS custom properties
- Tailwind CSS configured

### Implementation Tasks
- [ ] Create theme context and provider system
- [ ] Implement CSS custom properties architecture
- [ ] Set up theme configuration structure
- [ ] Create theme persistence mechanism
- [ ] Build theme type definitions

### Code Deliverables
```typescript
// frontend/src/types/theme.ts
export type ThemeName = 'light' | 'dark' | 'vibe' | 'encode' | 'opik';

export interface ThemeColors {
  primary: string;
  secondary: string;
  accent: string;
  background: string;
  surface: string;
  surfaceAlt: string;
  text: {
    primary: string;
    secondary: string;
    tertiary: string;
    inverse: string;
  };
  border: {
    default: string;
    light: string;
    focus: string;
  };
  status: {
    success: string;
    warning: string;
    error: string;
    info: string;
  };
}

export interface ThemeFonts {
  heading: string;
  body: string;
  mono: string;
  display?: string;
}

export interface ThemeEffects {
  shadows: {
    sm: string;
    md: string;
    lg: string;
    glow?: string;
  };
  gradients?: {
    primary?: string;
    accent?: string;
    background?: string;
  };
  animations?: {
    duration: string;
    easing: string;
  };
}

export interface Theme {
  name: ThemeName;
  colors: ThemeColors;
  fonts: ThemeFonts;
  effects: ThemeEffects;
  custom?: Record<string, any>;
}

// frontend/src/contexts/ThemeContext.tsx
import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { Theme, ThemeName } from '../types/theme';
import { themes } from '../themes';

interface ThemeContextType {
  theme: Theme;
  themeName: ThemeName;
  setTheme: (name: ThemeName) => void;
  isThemeLoading: boolean;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export const ThemeProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [themeName, setThemeName] = useState<ThemeName>(() => {
    // Check localStorage first, then default to light
    const stored = localStorage.getItem('qt1-theme') as ThemeName;
    return stored && themes[stored] ? stored : 'light';
  });
  
  const [isThemeLoading, setIsThemeLoading] = useState(true);
  const [theme, setThemeState] = useState<Theme>(themes[themeName]);

  useEffect(() => {
    const applyTheme = async () => {
      setIsThemeLoading(true);
      const selectedTheme = themes[themeName];
      
      // Apply CSS custom properties
      const root = document.documentElement;
      
      // Colors
      root.style.setProperty('--color-primary', selectedTheme.colors.primary);
      root.style.setProperty('--color-secondary', selectedTheme.colors.secondary);
      root.style.setProperty('--color-accent', selectedTheme.colors.accent);
      root.style.setProperty('--color-background', selectedTheme.colors.background);
      root.style.setProperty('--color-surface', selectedTheme.colors.surface);
      root.style.setProperty('--color-surface-alt', selectedTheme.colors.surfaceAlt);
      
      // Text colors
      root.style.setProperty('--text-primary', selectedTheme.colors.text.primary);
      root.style.setProperty('--text-secondary', selectedTheme.colors.text.secondary);
      root.style.setProperty('--text-tertiary', selectedTheme.colors.text.tertiary);
      root.style.setProperty('--text-inverse', selectedTheme.colors.text.inverse);
      
      // Border colors
      root.style.setProperty('--border-default', selectedTheme.colors.border.default);
      root.style.setProperty('--border-light', selectedTheme.colors.border.light);
      root.style.setProperty('--border-focus', selectedTheme.colors.border.focus);
      
      // Status colors
      root.style.setProperty('--status-success', selectedTheme.colors.status.success);
      root.style.setProperty('--status-warning', selectedTheme.colors.status.warning);
      root.style.setProperty('--status-error', selectedTheme.colors.status.error);
      root.style.setProperty('--status-info', selectedTheme.colors.status.info);
      
      // Effects
      root.style.setProperty('--shadow-sm', selectedTheme.effects.shadows.sm);
      root.style.setProperty('--shadow-md', selectedTheme.effects.shadows.md);
      root.style.setProperty('--shadow-lg', selectedTheme.effects.shadows.lg);
      
      if (selectedTheme.effects.shadows.glow) {
        root.style.setProperty('--shadow-glow', selectedTheme.effects.shadows.glow);
      }
      
      // Gradients
      if (selectedTheme.effects.gradients) {
        Object.entries(selectedTheme.effects.gradients).forEach(([key, value]) => {
          root.style.setProperty(`--gradient-${key}`, value);
        });
      }
      
      // Load theme-specific fonts
      await loadThemeFonts(selectedTheme);
      
      // Apply fonts
      root.style.setProperty('--font-heading', selectedTheme.fonts.heading);
      root.style.setProperty('--font-body', selectedTheme.fonts.body);
      root.style.setProperty('--font-mono', selectedTheme.fonts.mono);
      if (selectedTheme.fonts.display) {
        root.style.setProperty('--font-display', selectedTheme.fonts.display);
      }
      
      // Set theme attribute for CSS selectors
      root.setAttribute('data-theme', themeName);
      
      setThemeState(selectedTheme);
      setIsThemeLoading(false);
    };
    
    applyTheme();
  }, [themeName]);

  const handleSetTheme = (name: ThemeName) => {
    setThemeName(name);
    localStorage.setItem('qt1-theme', name);
  };

  return (
    <ThemeContext.Provider value={{
      theme,
      themeName,
      setTheme: handleSetTheme,
      isThemeLoading
    }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

// frontend/src/utils/themeLoader.ts
const fontUrls: Record<string, string> = {
  'Inter': 'https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap',
  'Poppins': 'https://fonts.googleapis.com/css2?family=Poppins:wght@400;500;600;700;800&display=swap',
  'JetBrains Mono': 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap',
  'Roboto Mono': 'https://fonts.googleapis.com/css2?family=Roboto+Mono:wght@400;500;600;700&display=swap',
  'IBM Plex Mono': 'https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600;700&display=swap',
  'Montserrat': 'https://fonts.googleapis.com/css2?family=Montserrat:wght@400;500;600;700;800&display=swap',
  'Fira Code': 'https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500;600;700&display=swap',
  'Press Start 2P': 'https://fonts.googleapis.com/css2?family=Press+Start+2P&display=swap',
  'Space Mono': 'https://fonts.googleapis.com/css2?family=Space+Mono:wght@400;700&display=swap',
};

const loadedFonts = new Set<string>();

export async function loadThemeFonts(theme: Theme): Promise<void> {
  const fontsToLoad = [
    theme.fonts.heading,
    theme.fonts.body,
    theme.fonts.mono,
    theme.fonts.display
  ].filter(Boolean);

  const loadPromises = fontsToLoad.map(async (fontFamily) => {
    // Extract the first font name from the font stack
    const primaryFont = fontFamily.split(',')[0].trim().replace(/['"]/g, '');
    
    if (loadedFonts.has(primaryFont) || !fontUrls[primaryFont]) {
      return;
    }

    // Create and inject link element
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = fontUrls[primaryFont];
    
    return new Promise((resolve) => {
      link.onload = () => {
        loadedFonts.add(primaryFont);
        resolve(undefined);
      };
      link.onerror = () => resolve(undefined); // Fail silently
      document.head.appendChild(link);
    });
  });

  await Promise.all(loadPromises);
}
```

### Theme Persistence
```typescript
// frontend/src/hooks/useThemePersistence.ts
export const useThemePersistence = () => {
  const { themeName, setTheme } = useTheme();
  
  useEffect(() => {
    // Save theme preference
    localStorage.setItem('qt1-theme', themeName);
    
    // Sync across tabs
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'qt1-theme' && e.newValue) {
        setTheme(e.newValue as ThemeName);
      }
    };
    
    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, [themeName, setTheme]);
};
```

### Testing Requirements
- [ ] Theme context properly provides theme data
- [ ] Theme persistence works across sessions
- [ ] CSS custom properties are applied correctly
- [ ] Font loading doesn't block rendering
- [ ] Theme switching is smooth

### Acceptance Criteria
- [ ] Theme system architecture is solid and extensible
- [ ] All theme properties are accessible via context
- [ ] Theme changes persist across page reloads
- [ ] No flash of unstyled content
- [ ] Fonts load asynchronously without blocking

### Risk Mitigation
- Test theme switching performance
- Ensure fallback fonts are specified
- Validate localStorage availability
- Handle font loading failures gracefully

---

## **Cycle 21B: Light & Dark Theme Implementation**
**Duration:** 5-6 hours | **Priority:** High

### Prerequisites
- Cycle 21A completed and tested
- Understanding of color theory and accessibility

### Implementation Tasks
- [ ] Implement Light theme with professional colors
- [ ] Implement Dark theme with proper contrast
- [ ] Create theme-specific component styles
- [ ] Test color contrast ratios
- [ ] Ensure smooth transitions

### Code Deliverables
```typescript
// frontend/src/themes/light.ts
import { Theme } from '../types/theme';

export const lightTheme: Theme = {
  name: 'light',
  colors: {
    primary: '#3B82F6',
    secondary: '#64748B',
    accent: '#10B981',
    background: '#FFFFFF',
    surface: '#F8FAFC',
    surfaceAlt: '#F1F5F9',
    text: {
      primary: '#0F172A',
      secondary: '#475569',
      tertiary: '#94A3B8',
      inverse: '#FFFFFF'
    },
    border: {
      default: '#E2E8F0',
      light: '#F1F5F9',
      focus: '#3B82F6'
    },
    status: {
      success: '#10B981',
      warning: '#F59E0B',
      error: '#EF4444',
      info: '#3B82F6'
    }
  },
  fonts: {
    heading: 'Inter, system-ui, -apple-system, sans-serif',
    body: 'Inter, system-ui, -apple-system, sans-serif',
    mono: 'Monaco, Menlo, Consolas, monospace'
  },
  effects: {
    shadows: {
      sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
      md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
      lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)'
    },
    animations: {
      duration: '200ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)'
    }
  }
};

// frontend/src/themes/dark.ts
export const darkTheme: Theme = {
  name: 'dark',
  colors: {
    primary: '#60A5FA',
    secondary: '#94A3B8',
    accent: '#34D399',
    background: '#0F172A',
    surface: '#1E293B',
    surfaceAlt: '#334155',
    text: {
      primary: '#F1F5F9',
      secondary: '#CBD5E1',
      tertiary: '#94A3B8',
      inverse: '#0F172A'
    },
    border: {
      default: '#334155',
      light: '#1E293B',
      focus: '#60A5FA'
    },
    status: {
      success: '#34D399',
      warning: '#FBBF24',
      error: '#F87171',
      info: '#60A5FA'
    }
  },
  fonts: {
    heading: 'Inter, system-ui, -apple-system, sans-serif',
    body: 'Inter, system-ui, -apple-system, sans-serif',
    mono: 'Monaco, Menlo, Consolas, monospace'
  },
  effects: {
    shadows: {
      sm: '0 1px 2px 0 rgba(0, 0, 0, 0.3)',
      md: '0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -1px rgba(0, 0, 0, 0.3)',
      lg: '0 10px 15px -3px rgba(0, 0, 0, 0.4), 0 4px 6px -2px rgba(0, 0, 0, 0.3)'
    },
    animations: {
      duration: '200ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)'
    }
  }
};

// frontend/src/themes/themeTransitions.css
[data-theme] {
  transition: 
    background-color var(--animation-duration) var(--animation-easing),
    color var(--animation-duration) var(--animation-easing),
    border-color var(--animation-duration) var(--animation-easing);
}

/* Prevent transitions on theme load */
.theme-loading * {
  transition: none !important;
}

/* Component-specific theme styles */
[data-theme="light"] {
  --button-hover-brightness: 0.95;
  --input-background: var(--color-surface);
  --card-hover-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
}

[data-theme="dark"] {
  --button-hover-brightness: 1.1;
  --input-background: var(--color-surface-alt);
  --card-hover-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
}
```

### Component Theme Support
```typescript
// frontend/src/components/ui/ThemedButton.tsx
import { cn } from '../../utils/cn';
import { useTheme } from '../../contexts/ThemeContext';

interface ThemedButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
}

export const ThemedButton: React.FC<ThemedButtonProps> = ({
  variant = 'primary',
  size = 'md',
  className,
  children,
  ...props
}) => {
  const { theme } = useTheme();
  
  const baseClasses = 'inline-flex items-center justify-center font-medium rounded-lg transition-all duration-[var(--animation-duration)] ease-[var(--animation-easing)]';
  
  const variantClasses = {
    primary: 'bg-[var(--color-primary)] text-[var(--text-inverse)] hover:brightness-[var(--button-hover-brightness)]',
    secondary: 'bg-[var(--color-secondary)] text-[var(--text-inverse)] hover:brightness-[var(--button-hover-brightness)]',
    ghost: 'bg-transparent text-[var(--text-primary)] hover:bg-[var(--color-surface-alt)]'
  };
  
  const sizeClasses = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg'
  };
  
  return (
    <button
      className={cn(
        baseClasses,
        variantClasses[variant],
        sizeClasses[size],
        'focus:outline-none focus:ring-2 focus:ring-[var(--border-focus)] focus:ring-offset-2 focus:ring-offset-[var(--color-background)]',
        className
      )}
      {...props}
    >
      {children}
    </button>
  );
};
```

### Testing Requirements
- [ ] Test WCAG AA contrast ratios for all color combinations
- [ ] Verify theme transitions are smooth
- [ ] Test component appearance in both themes
- [ ] Validate hover and focus states
- [ ] Check status color visibility

### Acceptance Criteria
- [ ] Light theme meets accessibility standards
- [ ] Dark theme has proper contrast ratios
- [ ] All components work in both themes
- [ ] Theme transitions are smooth
- [ ] No color contrast issues

### Risk Mitigation
- Use automated contrast checking tools
- Test with users who have visual impairments
- Ensure critical UI elements are clearly visible
- Provide high contrast alternatives if needed

---

## **Cycle 21C: Vibe Mode (70s Vice City) Implementation**
**Duration:** 6-8 hours | **Priority:** Medium

### Prerequisites
- Cycle 21B completed and tested
- Understanding of retro design aesthetics

### Implementation Tasks
- [ ] Implement Vibe theme with neon colors
- [ ] Create gradient effects and animations
- [ ] Add retro-futuristic UI elements
- [ ] Implement chrome/metallic effects
- [ ] Create theme-specific components

### Code Deliverables
```typescript
// frontend/src/themes/vibe.ts
export const vibeTheme: Theme = {
  name: 'vibe',
  colors: {
    primary: '#FF0080', // Electric magenta
    secondary: '#00FFD1', // Neon aqua
    accent: '#FF6F00', // Sunset gold
    background: '#1E003B', // Deep indigo
    surface: '#2E004F', // Dark purple
    surfaceAlt: '#3D0066', // Lighter purple
    text: {
      primary: '#FFFFFF',
      secondary: '#FFB3E6',
      tertiary: '#B366FF',
      inverse: '#1E003B'
    },
    border: {
      default: '#FF0080',
      light: '#660033',
      focus: '#00FFD1'
    },
    status: {
      success: '#00FFD1',
      warning: '#FFD700',
      error: '#FF0080',
      info: '#00B3FF'
    }
  },
  fonts: {
    heading: 'Montserrat, Avant Garde, sans-serif',
    body: 'Montserrat, Avenir Next, sans-serif',
    mono: 'Fira Code, monospace',
    display: 'Press Start 2P, cursive'
  },
  effects: {
    shadows: {
      sm: '0 2px 4px rgba(255, 0, 128, 0.3)',
      md: '0 4px 8px rgba(255, 0, 128, 0.4), 0 0 20px rgba(0, 255, 209, 0.2)',
      lg: '0 8px 16px rgba(255, 0, 128, 0.5), 0 0 40px rgba(0, 255, 209, 0.3)',
      glow: '0 0 20px rgba(255, 0, 128, 0.6), 0 0 40px rgba(0, 255, 209, 0.4)'
    },
    gradients: {
      primary: 'linear-gradient(135deg, #FF0080 0%, #FF6F00 100%)',
      accent: 'linear-gradient(135deg, #00FFD1 0%, #00B3FF 100%)',
      background: 'linear-gradient(180deg, #1E003B 0%, #2E004F 50%, #3D0066 100%)'
    },
    animations: {
      duration: '300ms',
      easing: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)'
    }
  },
  custom: {
    neonGlow: true,
    scanlines: true,
    chromeEffects: true
  }
};

// frontend/src/components/theme/VibeEffects.tsx
export const VibeEffects: React.FC = () => {
  const { themeName } = useTheme();
  
  if (themeName !== 'vibe') return null;
  
  return (
    <>
      {/* Scanlines overlay */}
      <div className="fixed inset-0 pointer-events-none z-50 opacity-10">
        <div className="scanlines"></div>
      </div>
      
      {/* Gradient overlay */}
      <div className="fixed inset-0 pointer-events-none z-40 opacity-20">
        <div className="w-full h-full bg-gradient-to-b from-transparent via-[#FF0080] to-transparent animate-gradient-shift"></div>
      </div>
    </>
  );
};

// frontend/src/styles/vibe.css
@keyframes neon-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.8; }
}

@keyframes gradient-shift {
  0% { transform: translateY(-100%); }
  100% { transform: translateY(100%); }
}

@keyframes chrome-shine {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}

[data-theme="vibe"] {
  /* Neon text effects */
  .neon-text {
    text-shadow: 
      0 0 10px currentColor,
      0 0 20px currentColor,
      0 0 40px currentColor;
    animation: neon-pulse 2s ease-in-out infinite;
  }
  
  /* Chrome button effects */
  .chrome-button {
    background: linear-gradient(
      135deg,
      #FF0080 0%,
      #FF6F00 25%,
      #FFD700 50%,
      #FF6F00 75%,
      #FF0080 100%
    );
    background-size: 400% 100%;
    animation: chrome-shine 3s linear infinite;
    border: 2px solid #00FFD1;
    box-shadow: 
      0 0 20px rgba(255, 0, 128, 0.6),
      inset 0 0 20px rgba(255, 255, 255, 0.2);
  }
  
  /* Scanlines effect */
  .scanlines {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: repeating-linear-gradient(
      0deg,
      transparent,
      transparent 2px,
      rgba(255, 255, 255, 0.03) 2px,
      rgba(255, 255, 255, 0.03) 4px
    );
  }
  
  /* Wireframe borders */
  .wireframe {
    border: 1px solid #00FFD1;
    position: relative;
    
    &::before,
    &::after {
      content: '';
      position: absolute;
      background: #00FFD1;
      width: 10px;
      height: 10px;
    }
    
    &::before {
      top: -5px;
      left: -5px;
    }
    
    &::after {
      bottom: -5px;
      right: -5px;
    }
  }
}

// frontend/src/components/theme/VibeButton.tsx
export const VibeButton: React.FC<ButtonProps> = ({ children, ...props }) => {
  const { themeName } = useTheme();
  
  if (themeName !== 'vibe') {
    return <ThemedButton {...props}>{children}</ThemedButton>;
  }
  
  return (
    <button
      className="chrome-button px-6 py-3 text-white font-bold uppercase tracking-wider rounded-lg transform transition-transform hover:scale-105"
      {...props}
    >
      <span className="neon-text">{children}</span>
    </button>
  );
};
```

### Testing Requirements
- [ ] Test neon glow effects performance
- [ ] Verify gradient animations are smooth
- [ ] Test chrome effects across browsers
- [ ] Ensure text remains readable
- [ ] Validate animation performance

### Acceptance Criteria
- [ ] Vibe theme captures 70s aesthetic
- [ ] Neon effects work without performance issues
- [ ] Chrome effects render correctly
- [ ] Animations are smooth at 60fps
- [ ] Theme is usable despite decorative elements

### Risk Mitigation
- Provide option to disable animations
- Ensure core functionality isn't obscured
- Test on lower-end devices
- Optimize animation performance

---

## **Cycle 21D: London Encode & Comet Opik Themes**
**Duration:** 6-8 hours | **Priority:** Medium

### Prerequisites
- Cycle 21C completed and tested
- Understanding of modern tech aesthetics

### Implementation Tasks
- [ ] Implement London Encode dark tech theme
- [ ] Implement Comet Opik clean developer theme
- [ ] Create theme-specific UI patterns
- [ ] Add subtle animations and effects
- [ ] Test theme consistency

### Code Deliverables
```typescript
// frontend/src/themes/encode.ts
export const encodeTheme: Theme = {
  name: 'encode',
  colors: {
    primary: '#3B82F6', // Bold Azure blue
    secondary: '#94A3B8', // Cool gray
    accent: '#06B6D4', // Electric cyan
    background: '#0F172A', // Dark navy-blue
    surface: '#1E293B', // Dark panels
    surfaceAlt: '#334155', // Lighter panels
    text: {
      primary: '#F1F5F9',
      secondary: '#CBD5E1',
      tertiary: '#94A3B8',
      inverse: '#0F172A'
    },
    border: {
      default: '#334155',
      light: '#475569',
      focus: '#06B6D4'
    },
    status: {
      success: '#10B981',
      warning: '#F59E0B',
      error: '#EF4444',
      info: '#06B6D4'
    }
  },
  fonts: {
    heading: 'JetBrains Mono, Space Mono, monospace',
    body: 'Roboto Mono, Source Sans Pro, sans-serif',
    mono: 'IBM Plex Mono, monospace'
  },
  effects: {
    shadows: {
      sm: '0 2px 4px rgba(6, 182, 212, 0.1)',
      md: '0 4px 8px rgba(6, 182, 212, 0.15), 0 0 20px rgba(59, 130, 246, 0.1)',
      lg: '0 8px 16px rgba(6, 182, 212, 0.2), 0 0 40px rgba(59, 130, 246, 0.15)',
      glow: '0 0 20px rgba(6, 182, 212, 0.4), 0 0 40px rgba(59, 130, 246, 0.2)'
    },
    gradients: {
      primary: 'linear-gradient(135deg, #3B82F6 0%, #06B6D4 100%)',
      accent: 'linear-gradient(135deg, #1E293B 0%, #334155 100%)'
    },
    animations: {
      duration: '250ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)'
    }
  },
  custom: {
    neonGlow: true,
    techGrid: true
  }
};

// frontend/src/themes/opik.ts
export const opikTheme: Theme = {
  name: 'opik',
  colors: {
    primary: '#22C55E', // Vibrant teal from Opik
    secondary: '#1F2937', // Dark charcoal
    accent: '#F97316', // Warm orange
    background: '#FFFFFF', // Crisp white
    surface: '#F9FAFB', // Very light gray
    surfaceAlt: '#F3F4F6', // Light gray panels
    text: {
      primary: '#111827',
      secondary: '#4B5563',
      tertiary: '#9CA3AF',
      inverse: '#FFFFFF'
    },
    border: {
      default: '#E5E7EB',
      light: '#F3F4F6',
      focus: '#22C55E'
    },
    status: {
      success: '#22C55E',
      warning: '#F97316',
      error: '#EF4444',
      info: '#3B82F6'
    }
  },
  fonts: {
    heading: 'Poppins, Inter, sans-serif',
    body: 'Inter, Roboto, sans-serif',
    mono: 'Fira Code, monospace'
  },
  effects: {
    shadows: {
      sm: '0 1px 3px rgba(0, 0, 0, 0.08)',
      md: '0 4px 6px rgba(0, 0, 0, 0.1)',
      lg: '0 10px 15px rgba(0, 0, 0, 0.12)'
    },
    animations: {
      duration: '200ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)'
    }
  },
  custom: {
    cleanDesign: true,
    subtleAnimations: true
  }
};

// frontend/src/components/theme/EncodeEffects.tsx
export const EncodeEffects: React.FC = () => {
  const { themeName } = useTheme();
  
  if (themeName !== 'encode') return null;
  
  return (
    <div className="fixed inset-0 pointer-events-none z-30 opacity-5">
      <div className="tech-grid"></div>
    </div>
  );
};

// frontend/src/styles/encode.css
[data-theme="encode"] {
  /* Tech grid background */
  .tech-grid {
    width: 100%;
    height: 100%;
    background-image: 
      linear-gradient(rgba(6, 182, 212, 0.1) 1px, transparent 1px),
      linear-gradient(90deg, rgba(6, 182, 212, 0.1) 1px, transparent 1px);
    background-size: 50px 50px;
  }
  
  /* Neon outline buttons */
  .neon-outline {
    border: 1px solid var(--color-accent);
    box-shadow: 
      0 0 5px rgba(6, 182, 212, 0.5),
      inset 0 0 5px rgba(6, 182, 212, 0.1);
    transition: all var(--animation-duration) var(--animation-easing);
    
    &:hover {
      box-shadow: 
        0 0 15px rgba(6, 182, 212, 0.7),
        inset 0 0 15px rgba(6, 182, 212, 0.2);
    }
  }
  
  /* Code-style elements */
  .code-style {
    font-family: var(--font-mono);
    background: rgba(6, 182, 212, 0.1);
    border-left: 3px solid var(--color-accent);
    padding-left: 1rem;
  }
}

// frontend/src/styles/opik.css
[data-theme="opik"] {
  /* Clean card design */
  .clean-card {
    background: var(--color-surface);
    border: 1px solid var(--border-default);
    border-radius: 0.75rem;
    transition: all var(--animation-duration) var(--animation-easing);
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: var(--shadow-lg);
    }
  }
  
  /* Ghost buttons with lift */
  .ghost-lift {
    background: transparent;
    border: 2px solid transparent;
    transition: all var(--animation-duration) var(--animation-easing);
    
    &:hover {
      background: var(--color-surface-alt);
      border-color: var(--color-primary);
      transform: translateY(-1px);
    }
  }
  
  /* Subtle gradients */
  .subtle-gradient {
    background: linear-gradient(
      135deg,
      var(--color-background) 0%,
      var(--color-surface) 100%
    );
  }
}
```

### Testing Requirements
- [ ] Test tech grid performance in Encode theme
- [ ] Verify neon effects don't overwhelm content
- [ ] Test Opik theme's clean aesthetics
- [ ] Ensure all interactive elements are visible
- [ ] Validate font loading and fallbacks

### Acceptance Criteria
- [ ] Encode theme reflects tech-focused aesthetic
- [ ] Opik theme maintains clean, professional look
- [ ] Both themes are fully functional
- [ ] Special effects enhance rather than distract
- [ ] Performance remains optimal

### Risk Mitigation
- Keep effects subtle and optional
- Test on various screen sizes
- Ensure readability is maintained
- Provide fallbacks for custom fonts

---

## **Cycle 21E: Theme Selector & Integration**
**Duration:** 6-8 hours | **Priority:** High

### Prerequisites
- Cycles 21A-21D completed and tested
- All themes implemented and working

### Implementation Tasks
- [ ] Create theme selector for login screen
- [ ] Build theme preview component
- [ ] Implement theme switcher in app
- [ ] Add keyboard shortcuts
- [ ] Complete integration testing

### Code Deliverables
```typescript
// frontend/src/components/auth/ThemeSelector.tsx
import { useTheme } from '../../contexts/ThemeContext';
import { themes } from '../../themes';
import { motion, AnimatePresence } from 'framer-motion';

interface ThemeSelectorProps {
  showLabels?: boolean;
  className?: string;
}

export const ThemeSelector: React.FC<ThemeSelectorProps> = ({ 
  showLabels = true,
  className 
}) => {
  const { themeName, setTheme } = useTheme();
  const [isExpanded, setIsExpanded] = useState(false);
  
  const themeOptions = [
    { name: 'light', label: 'Light', icon: '☀️', preview: '#FFFFFF' },
    { name: 'dark', label: 'Dark', icon: '🌙', preview: '#0F172A' },
    { name: 'vibe', label: 'Vibe', icon: '🌆', preview: '#FF0080' },
    { name: 'encode', label: 'Encode', icon: '💻', preview: '#06B6D4' },
    { name: 'opik', label: 'Opik', icon: '🟢', preview: '#22C55E' }
  ];
  
  return (
    <div className={cn('relative', className)}>
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--border-default)] hover:border-[var(--border-focus)] transition-colors"
        aria-label="Select theme"
        aria-expanded={isExpanded}
      >
        <span className="text-lg">
          {themeOptions.find(t => t.name === themeName)?.icon}
        </span>
        {showLabels && (
          <span className="text-[var(--text-primary)]">
            {themeOptions.find(t => t.name === themeName)?.label}
          </span>
        )}
        <ChevronDownIcon className={cn(
          'w-4 h-4 transition-transform',
          isExpanded && 'rotate-180'
        )} />
      </button>
      
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="absolute top-full mt-2 right-0 bg-[var(--color-surface)] border border-[var(--border-default)] rounded-lg shadow-lg overflow-hidden z-50"
          >
            {themeOptions.map((theme) => (
              <ThemeOption
                key={theme.name}
                theme={theme}
                isActive={themeName === theme.name}
                onClick={() => {
                  setTheme(theme.name as ThemeName);
                  setIsExpanded(false);
                }}
              />
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

// frontend/src/components/auth/ThemeOption.tsx
interface ThemeOptionProps {
  theme: {
    name: string;
    label: string;
    icon: string;
    preview: string;
  };
  isActive: boolean;
  onClick: () => void;
}

const ThemeOption: React.FC<ThemeOptionProps> = ({ theme, isActive, onClick }) => {
  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full px-4 py-3 flex items-center gap-3 hover:bg-[var(--color-surface-alt)] transition-colors',
        isActive && 'bg-[var(--color-surface-alt)]'
      )}
    >
      <div 
        className="w-8 h-8 rounded-full border-2 border-[var(--border-default)] flex items-center justify-center text-sm"
        style={{ backgroundColor: theme.preview }}
      >
        {theme.icon}
      </div>
      <div className="flex-1 text-left">
        <div className="font-medium text-[var(--text-primary)]">{theme.label}</div>
        {isActive && (
          <div className="text-xs text-[var(--text-secondary)]">Active</div>
        )}
      </div>
      {isActive && (
        <CheckIcon className="w-5 h-5 text-[var(--color-primary)]" />
      )}
    </button>
  );
};

// frontend/src/components/auth/LoginThemeSelector.tsx
export const LoginThemeSelector: React.FC = () => {
  const { setTheme } = useTheme();
  const [selectedTheme, setSelectedTheme] = useState<ThemeName>('light');
  
  const themes = [
    {
      name: 'light',
      label: 'Light Mode',
      description: 'Clean and professional',
      preview: (
        <div className="w-full h-32 bg-white rounded-lg p-4">
          <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
          <div className="h-3 bg-gray-300 rounded w-1/2"></div>
        </div>
      )
    },
    {
      name: 'dark',
      label: 'Dark Mode',
      description: 'Easy on the eyes',
      preview: (
        <div className="w-full h-32 bg-gray-900 rounded-lg p-4">
          <div className="h-4 bg-gray-700 rounded w-3/4 mb-2"></div>
          <div className="h-3 bg-gray-600 rounded w-1/2"></div>
        </div>
      )
    },
    {
      name: 'vibe',
      label: 'Vibe Mode',
      description: '70s Vice City style',
      preview: (
        <div className="w-full h-32 bg-gradient-to-br from-purple-900 to-pink-600 rounded-lg p-4 relative overflow-hidden">
          <div className="absolute inset-0 bg-gradient-to-t from-cyan-400/20 to-transparent"></div>
          <div className="h-4 bg-pink-400 rounded w-3/4 mb-2 relative z-10"></div>
          <div className="h-3 bg-cyan-400 rounded w-1/2 relative z-10"></div>
        </div>
      )
    },
    {
      name: 'encode',
      label: 'London Encode',
      description: 'Dark tech aesthetic',
      preview: (
        <div className="w-full h-32 bg-slate-900 rounded-lg p-4 border border-cyan-500/30">
          <div className="h-4 bg-blue-600 rounded w-3/4 mb-2"></div>
          <div className="h-3 bg-cyan-500 rounded w-1/2"></div>
        </div>
      )
    },
    {
      name: 'opik',
      label: 'Comet Opik',
      description: 'Clean developer UI',
      preview: (
        <div className="w-full h-32 bg-gray-50 rounded-lg p-4 border border-green-500/30">
          <div className="h-4 bg-green-500 rounded w-3/4 mb-2"></div>
          <div className="h-3 bg-orange-500 rounded w-1/2"></div>
        </div>
      )
    }
  ];
  
  return (
    <div className="w-full max-w-4xl mx-auto p-6">
      <h3 className="text-lg font-semibold mb-4">Choose Your Theme</h3>
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
        {themes.map((theme) => (
          <button
            key={theme.name}
            onClick={() => {
              setSelectedTheme(theme.name as ThemeName);
              setTheme(theme.name as ThemeName);
            }}
            className={cn(
              'relative group cursor-pointer transition-all',
              selectedTheme === theme.name && 'ring-2 ring-[var(--color-primary)] ring-offset-2'
            )}
          >
            <div className="space-y-2">
              {theme.preview}
              <div className="text-center">
                <div className="font-medium text-sm">{theme.label}</div>
                <div className="text-xs text-[var(--text-secondary)]">{theme.description}</div>
              </div>
            </div>
            {selectedTheme === theme.name && (
              <div className="absolute top-2 right-2 w-6 h-6 bg-[var(--color-primary)] rounded-full flex items-center justify-center">
                <CheckIcon className="w-4 h-4 text-white" />
              </div>
            )}
          </button>
        ))}
      </div>
    </div>
  );
};

// frontend/src/hooks/useThemeShortcuts.ts
export const useThemeShortcuts = () => {
  const { themeName, setTheme } = useTheme();
  
  const themeOrder: ThemeName[] = ['light', 'dark', 'vibe', 'encode', 'opik'];
  
  useKeyboardShortcuts([
    {
      key: 't',
      ctrlKey: true,
      shiftKey: true,
      action: () => {
        const currentIndex = themeOrder.indexOf(themeName);
        const nextIndex = (currentIndex + 1) % themeOrder.length;
        setTheme(themeOrder[nextIndex]);
      },
      description: 'Cycle through themes'
    },
    {
      key: '1',
      altKey: true,
      action: () => setTheme('light'),
      description: 'Switch to Light theme'
    },
    {
      key: '2',
      altKey: true,
      action: () => setTheme('dark'),
      description: 'Switch to Dark theme'
    },
    {
      key: '3',
      altKey: true,
      action: () => setTheme('vibe'),
      description: 'Switch to Vibe theme'
    },
    {
      key: '4',
      altKey: true,
      action: () => setTheme('encode'),
      description: 'Switch to Encode theme'
    },
    {
      key: '5',
      altKey: true,
      action: () => setTheme('opik'),
      description: 'Switch to Opik theme'
    }
  ]);
};
```

### Integration Updates
```typescript
// frontend/src/App.tsx updates
import { ThemeProvider } from './contexts/ThemeContext';
import { ThemeSelector } from './components/auth/ThemeSelector';
import { VibeEffects } from './components/theme/VibeEffects';
import { EncodeEffects } from './components/theme/EncodeEffects';

function App() {
  return (
    <ThemeProvider>
      <div className="min-h-screen bg-[var(--color-background)] text-[var(--text-primary)]">
        {/* Theme-specific effects */}
        <VibeEffects />
        <EncodeEffects />
        
        {/* Rest of the app */}
        <AppContent />
      </div>
    </ThemeProvider>
  );
}

// Update login screen
function LoginScreen() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-[var(--color-background)]">
      {/* Theme selector at top */}
      <div className="absolute top-4 right-4">
        <ThemeSelector />
      </div>
      
      {/* Login form */}
      <div className="w-full max-w-md">
        <LoginThemeSelector />
        <LoginForm />
      </div>
    </div>
  );
}
```

### Testing Requirements
- [ ] Test theme selector functionality
- [ ] Verify theme previews are accurate
- [ ] Test keyboard shortcuts
- [ ] Ensure theme persists after login
- [ ] Validate theme switching performance

### Acceptance Criteria
- [ ] Theme selector works on login screen
- [ ] All themes can be selected and previewed
- [ ] Theme choice persists across sessions
- [ ] Keyboard shortcuts work correctly
- [ ] No flash of wrong theme on load

### Risk Mitigation
- Test theme switching extensively
- Ensure graceful fallbacks
- Validate performance impact
- Test across different browsers

---

## **Integration Testing & Validation**
**Duration:** 4-5 hours

### Comprehensive Theme Testing
- [ ] Cross-browser compatibility (Chrome, Firefox, Safari, Edge)
- [ ] Mobile responsive testing for all themes
- [ ] Accessibility testing (WCAG AA compliance)
- [ ] Performance testing with theme switching
- [ ] User acceptance testing

### Validation Checklist
- [ ] All 5 themes fully implemented
- [ ] Theme selection from login screen works
- [ ] Themes persist across sessions
- [ ] All components properly themed
- [ ] Special effects work without issues
- [ ] Keyboard shortcuts functional
- [ ] No performance degradation
- [ ] Smooth transitions between themes

### Success Metrics
- [ ] 100% theme coverage across components
- [ ] <100ms theme switching time
- [ ] Zero accessibility violations
- [ ] 60fps animations in all themes
- [ ] Positive user feedback on all themes

---

## **Rollback Plan**
If any cycle fails:
1. Revert to default theme only
2. Disable problematic theme temporarily
3. Use CSS fallbacks for custom properties
4. Disable theme-specific effects if performance issues
5. Fall back to system color scheme detection