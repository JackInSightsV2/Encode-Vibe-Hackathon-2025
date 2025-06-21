/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Theme-aware color system
        theme: {
          primary: 'var(--color-primary)',
          secondary: 'var(--color-secondary)',
          accent: 'var(--color-accent)',
          background: 'var(--color-background)',
          surface: 'var(--color-surface)',
          'surface-alt': 'var(--color-surface-alt)',
          text: {
            primary: 'var(--color-text-primary)',
            secondary: 'var(--color-text-secondary)',
            tertiary: 'var(--color-text-tertiary)',
            inverse: 'var(--color-text-inverse)',
          },
          border: {
            DEFAULT: 'var(--color-border-default)',
            light: 'var(--color-border-light)',
            focus: 'var(--color-border-focus)',
          },
          status: {
            success: 'var(--color-success)',
            warning: 'var(--color-warning)',
            error: 'var(--color-error)',
            info: 'var(--color-info)',
          },
        },
        // Legacy compatibility
        primary: {
          50: 'var(--color-primary-50, #eff6ff)',
          100: 'var(--color-primary-100, #dbeafe)',
          200: 'var(--color-primary-200, #bfdbfe)',
          300: 'var(--color-primary-300, #93c5fd)',
          400: 'var(--color-primary-400, #60a5fa)',
          500: 'var(--color-primary, #3b82f6)',
          600: 'var(--color-primary-600, #2563eb)',
          700: 'var(--color-primary-700, #1d4ed8)',
          800: 'var(--color-primary-800, #1e40af)',
          900: 'var(--color-primary-900, #1e3a8a)',
        },
        gray: {
          50: 'var(--color-gray-50, #f8fafc)',
          100: 'var(--color-gray-100, #f1f5f9)',
          200: 'var(--color-gray-200, #e2e8f0)',
          300: 'var(--color-gray-300, #cbd5e1)',
          400: 'var(--color-gray-400, #94a3b8)',
          500: 'var(--color-gray-500, #64748b)',
          600: 'var(--color-gray-600, #475569)',
          700: 'var(--color-gray-700, #334155)',
          800: 'var(--color-gray-800, #1e293b)',
          900: 'var(--color-gray-900, #0f172a)',
        },
        background: 'var(--color-background, var(--background))',
        foreground: 'var(--color-text-primary, var(--foreground))',
        card: {
          DEFAULT: 'var(--color-surface, var(--card))',
          foreground: 'var(--color-text-primary, var(--card-foreground))',
        },
        popover: {
          DEFAULT: 'var(--color-surface, var(--popover))',
          foreground: 'var(--color-text-primary, var(--popover-foreground))',
        },
        muted: {
          DEFAULT: 'var(--color-surface-alt, var(--muted))',
          foreground: 'var(--color-text-secondary, var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'var(--color-accent, var(--accent))',
          foreground: 'var(--color-text-inverse, var(--accent-foreground))',
        },
        destructive: {
          DEFAULT: 'var(--color-error, var(--destructive))',
          foreground: 'var(--color-text-inverse, var(--destructive-foreground))',
        },
        border: 'var(--color-border-default, var(--border))',
        input: 'var(--color-border-default, var(--input))',
        ring: 'var(--color-border-focus, var(--ring))',
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontFamily: {
        heading: 'var(--font-heading, ui-sans-serif, system-ui, sans-serif)',
        body: 'var(--font-body, ui-sans-serif, system-ui, sans-serif)',
        mono: 'var(--font-mono, ui-monospace, monospace)',
        display: 'var(--font-display, ui-sans-serif, system-ui, sans-serif)',
      },
      boxShadow: {
        'theme-sm': 'var(--shadow-sm)',
        'theme-md': 'var(--shadow-md)',
        'theme-lg': 'var(--shadow-lg)',
        'theme-glow': 'var(--shadow-glow, var(--shadow-md))',
      },
      backgroundImage: {
        'gradient-primary': 'var(--gradient-primary, linear-gradient(135deg, var(--color-primary) 0%, var(--color-secondary) 100%))',
        'gradient-accent': 'var(--gradient-accent, linear-gradient(135deg, var(--color-accent) 0%, var(--color-primary) 100%))',
        'gradient-background': 'var(--gradient-background, linear-gradient(135deg, var(--color-background) 0%, var(--color-surface) 100%))',
      },
      keyframes: {
        'accordion-down': {
          from: { height: 0 },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: 0 },
        },
        'fade-in': {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-in': {
          '0%': { opacity: '0', transform: 'translateX(-10px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        'scale-in': {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
        'neon-pulse': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.8' },
        },
        'glitch': {
          '0%': { transform: 'translate(0)' },
          '20%': { transform: 'translate(-2px, 2px)' },
          '40%': { transform: 'translate(-2px, -2px)' },
          '60%': { transform: 'translate(2px, 2px)' },
          '80%': { transform: 'translate(2px, -2px)' },
          '100%': { transform: 'translate(0)' },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
        'fade-in': 'fade-in var(--animation-duration, 200ms) var(--animation-easing, ease-out)',
        'slide-in': 'slide-in var(--animation-duration, 200ms) var(--animation-easing, ease-out)',
        'scale-in': 'scale-in var(--animation-duration, 200ms) var(--animation-easing, ease-out)',
        'neon-pulse': 'neon-pulse 2s ease-in-out infinite',
        'glitch': 'glitch 0.3s ease-in-out',
      },
    },
  },
  plugins: [],
}