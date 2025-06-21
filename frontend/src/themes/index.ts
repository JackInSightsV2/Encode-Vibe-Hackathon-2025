import { Theme, ThemeConfig } from '../types/theme';

// Light Mode Theme
const lightTheme: Theme = {
  name: 'light',
  displayName: 'Light Mode',
  description: 'Clean and professional light theme',
  colors: {
    primary: '#3B82F6',
    secondary: '#64748B',
    accent: '#06B6D4',
    background: '#FFFFFF',
    surface: '#F8FAFC',
    surfaceAlt: '#F1F5F9',
    text: {
      primary: '#0F172A',
      secondary: '#475569',
      tertiary: '#64748B',
      inverse: '#FFFFFF',
    },
    border: {
      default: '#E2E8F0',
      light: '#F1F5F9',
      focus: '#3B82F6',
    },
    status: {
      success: '#10B981',
      warning: '#F59E0B',
      error: '#EF4444',
      info: '#3B82F6',
    },
  },
  fonts: {
    heading: 'Inter',
    body: 'Inter',
    mono: 'Monaco, Menlo, monospace',
  },
  effects: {
    shadows: {
      sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
      md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
      lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
    },
    animations: {
      duration: '200ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  },
};

// Dark Mode Theme
const darkTheme: Theme = {
  name: 'dark',
  displayName: 'Dark Mode',
  description: 'Elegant dark theme for low-light environments',
  colors: {
    primary: '#60A5FA',
    secondary: '#94A3B8',
    accent: '#22D3EE',
    background: '#0F172A',
    surface: '#1E293B',
    surfaceAlt: '#334155',
    text: {
      primary: '#F1F5F9',
      secondary: '#CBD5E1',
      tertiary: '#94A3B8',
      inverse: '#0F172A',
    },
    border: {
      default: '#334155',
      light: '#475569',
      focus: '#60A5FA',
    },
    status: {
      success: '#22C55E',
      warning: '#FDE047',
      error: '#F87171',
      info: '#60A5FA',
    },
  },
  fonts: {
    heading: 'Inter',
    body: 'Inter',
    mono: 'Monaco, Menlo, monospace',
  },
  effects: {
    shadows: {
      sm: '0 1px 2px 0 rgb(0 0 0 / 0.3)',
      md: '0 4px 6px -1px rgb(0 0 0 / 0.3), 0 2px 4px -2px rgb(0 0 0 / 0.3)',
      lg: '0 10px 15px -3px rgb(0 0 0 / 0.3), 0 4px 6px -4px rgb(0 0 0 / 0.3)',
      glow: '0 0 20px rgb(96 165 250 / 0.3)',
    },
    animations: {
      duration: '200ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  },
};

// Vibe Mode Theme (70s Vice City)
const vibeTheme: Theme = {
  name: 'vibe',
  displayName: 'Vibe Mode',
  description: '70s Vice City retro-futuristic theme',
  colors: {
    primary: '#FF0080',
    secondary: '#00FFD1',
    accent: '#FF6F00',
    background: '#1E003B',
    surface: '#2E004F',
    surfaceAlt: '#3E005F',
    text: {
      primary: '#FFFFFF',
      secondary: '#E879F9',
      tertiary: '#C084FC',
      inverse: '#1E003B',
    },
    border: {
      default: '#7C3AED',
      light: '#A855F7',
      focus: '#FF0080',
    },
    status: {
      success: '#00FFD1',
      warning: '#FF6F00',
      error: '#FF0080',
      info: '#8B5CF6',
    },
  },
  fonts: {
    heading: 'Orbitron',
    body: 'Montserrat',
    mono: 'JetBrains Mono',
    display: 'Press Start 2P',
  },
  effects: {
    shadows: {
      sm: '0 1px 2px 0 rgb(255 0 128 / 0.2)',
      md: '0 4px 6px -1px rgb(255 0 128 / 0.3), 0 2px 4px -2px rgb(0 255 209 / 0.2)',
      lg: '0 10px 15px -3px rgb(255 0 128 / 0.4), 0 4px 6px -4px rgb(0 255 209 / 0.3)',
      glow: '0 0 30px rgb(255 0 128 / 0.5), 0 0 60px rgb(0 255 209 / 0.3)',
    },
    gradients: {
      primary: 'linear-gradient(135deg, #FF0080 0%, #7C3AED 100%)',
      accent: 'linear-gradient(135deg, #00FFD1 0%, #0EA5E9 100%)',
      background: 'linear-gradient(135deg, #1E003B 0%, #0F0F23 100%)',
    },
    animations: {
      duration: '300ms',
      easing: 'cubic-bezier(0.34, 1.56, 0.64, 1)',
    },
    special: {
      neonGlow: 'drop-shadow(0 0 10px currentColor)',
      scanlines: 'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(255,255,255,0.03) 2px, rgba(255,255,255,0.03) 4px)',
      chrome: 'linear-gradient(135deg, #C4B5FD 0%, #8B5CF6 25%, #7C3AED 50%, #6D28D9 75%, #5B21B6 100%)',
    },
  },
};

// Encode Mode Theme (London Tech)
const encodeTheme: Theme = {
  name: 'encode',
  displayName: 'Encode Mode',
  description: 'London tech dark theme with neon accents',
  colors: {
    primary: '#3B82F6',
    secondary: '#94A3B8',
    accent: '#06B6D4',
    background: '#0F172A',
    surface: '#1E293B',
    surfaceAlt: '#334155',
    text: {
      primary: '#F8FAFC',
      secondary: '#E2E8F0',
      tertiary: '#CBD5E1',
      inverse: '#0F172A',
    },
    border: {
      default: '#475569',
      light: '#64748B',
      focus: '#06B6D4',
    },
    status: {
      success: '#10B981',
      warning: '#F59E0B',
      error: '#EF4444',
      info: '#06B6D4',
    },
  },
  fonts: {
    heading: 'JetBrains Mono',
    body: 'Source Sans Pro',
    mono: 'JetBrains Mono',
  },
  effects: {
    shadows: {
      sm: '0 1px 2px 0 rgb(6 182 212 / 0.1)',
      md: '0 4px 6px -1px rgb(6 182 212 / 0.2), 0 2px 4px -2px rgb(59 130 246 / 0.1)',
      lg: '0 10px 15px -3px rgb(6 182 212 / 0.3), 0 4px 6px -4px rgb(59 130 246 / 0.2)',
      glow: '0 0 20px rgb(6 182 212 / 0.4)',
    },
    gradients: {
      primary: 'linear-gradient(135deg, #3B82F6 0%, #06B6D4 100%)',
      accent: 'linear-gradient(135deg, #06B6D4 0%, #0891B2 100%)',
    },
    animations: {
      duration: '250ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
    special: {
      neonGlow: 'drop-shadow(0 0 8px rgb(6 182 212 / 0.6))',
    },
  },
};

// Opik Mode Theme (Clean Developer UI)
const opikTheme: Theme = {
  name: 'opik',
  displayName: 'Opik Mode',
  description: 'Clean developer-focused UI inspired by Opik',
  colors: {
    primary: '#22C55E',
    secondary: '#1F2937',
    accent: '#F97316',
    background: '#FFFFFF',
    surface: '#FFFFFF',
    surfaceAlt: '#F3F4F6',
    text: {
      primary: '#111827',
      secondary: '#374151',
      tertiary: '#6B7280',
      inverse: '#FFFFFF',
    },
    border: {
      default: '#D1D5DB',
      light: '#E5E7EB',
      focus: '#22C55E',
    },
    status: {
      success: '#22C55E',
      warning: '#F59E0B',
      error: '#EF4444',
      info: '#3B82F6',
    },
  },
  fonts: {
    heading: 'Poppins',
    body: 'Inter',
    mono: 'Fira Code',
  },
  effects: {
    shadows: {
      sm: '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
      md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
      lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
    },
    animations: {
      duration: '150ms',
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  },
};

export const themes: ThemeConfig = {
  light: lightTheme,
  dark: darkTheme,
  vibe: vibeTheme,
  encode: encodeTheme,
  opik: opikTheme,
};

export const defaultTheme = lightTheme;

export const getTheme = (name: string): Theme => {
  return themes[name] || defaultTheme;
};

export const getThemeNames = (): string[] => {
  return Object.keys(themes);
};