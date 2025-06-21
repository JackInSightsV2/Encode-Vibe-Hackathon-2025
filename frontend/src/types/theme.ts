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
  special?: {
    neonGlow?: string;
    scanlines?: string;
    chrome?: string;
  };
}

export interface Theme {
  name: ThemeName;
  displayName: string;
  description: string;
  colors: ThemeColors;
  fonts: ThemeFonts;
  effects: ThemeEffects;
  custom?: Record<string, any>;
}

export interface ThemeContextType {
  theme: Theme;
  themeName: ThemeName;
  setTheme: (name: ThemeName) => void;
  isThemeLoading: boolean;
  isInitialized: boolean;
}

export interface ThemeConfig {
  [key: string]: Theme;
}

export interface FontLoadingState {
  [fontFamily: string]: {
    loaded: boolean;
    loading: boolean;
    error?: string;
  };
}