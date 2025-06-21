import { useState, useEffect } from 'react';

/**
 * Custom hook for responsive design based on CSS media queries
 * @param query - CSS media query string
 * @returns boolean indicating if the query matches
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window !== 'undefined') {
      return window.matchMedia(query).matches;
    }
    return false;
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const media = window.matchMedia(query);
    
    // Set initial state
    if (media.matches !== matches) {
      setMatches(media.matches);
    }

    // Listen for changes
    const listener = (event: MediaQueryListEvent) => {
      setMatches(event.matches);
    };

    // Use the newer addEventListener if available, fallback to addListener
    if (media.addEventListener) {
      media.addEventListener('change', listener);
    } else {
      // @ts-ignore - for older browsers
      media.addListener(listener);
    }

    return () => {
      if (media.removeEventListener) {
        media.removeEventListener('change', listener);
      } else {
        // @ts-ignore - for older browsers
        media.removeListener(listener);
      }
    };
  }, [matches, query]);

  return matches;
}

/**
 * Predefined breakpoint hooks for common screen sizes
 * Based on Tailwind CSS breakpoints
 */

// Mobile first approach: these detect when screen is AT LEAST this size
export const useIsSm = () => useMediaQuery('(min-width: 640px)');   // sm and up
export const useIsMd = () => useMediaQuery('(min-width: 768px)');   // md and up  
export const useIsLg = () => useMediaQuery('(min-width: 1024px)');  // lg and up
export const useIsXl = () => useMediaQuery('(min-width: 1280px)');  // xl and up
export const useIs2Xl = () => useMediaQuery('(min-width: 1536px)'); // 2xl and up

// Detect mobile devices (below tablet)
export const useIsMobile = () => useMediaQuery('(max-width: 767px)');

// Detect tablet range
export const useIsTablet = () => useMediaQuery('(min-width: 768px) and (max-width: 1023px)');

// Detect desktop
export const useIsDesktop = () => useMediaQuery('(min-width: 1024px)');

// Detect touch devices
export const useIsTouchDevice = () => useMediaQuery('(hover: none) and (pointer: coarse)');

// Detect reduced motion preference for accessibility
export const usePrefersReducedMotion = () => useMediaQuery('(prefers-reduced-motion: reduce)');

// Detect dark mode preference
export const usePrefersDarkMode = () => useMediaQuery('(prefers-color-scheme: dark)');

/**
 * Responsive helper hook that returns an object with all breakpoint states
 */
export function useBreakpoints() {
  const isSm = useIsSm();
  const isMd = useIsMd();
  const isLg = useIsLg();
  const isXl = useIsXl();
  const is2Xl = useIs2Xl();
  const isMobile = useIsMobile();
  const isTablet = useIsTablet();
  const isDesktop = useIsDesktop();
  const isTouchDevice = useIsTouchDevice();

  return {
    isSm,
    isMd,
    isLg,
    isXl,
    is2Xl,
    isMobile,
    isTablet,
    isDesktop,
    isTouchDevice,
    // Convenience properties
    isSmallScreen: isMobile,
    isMediumScreen: isTablet,
    isLargeScreen: isDesktop,
  };
}

/**
 * Hook for responsive values based on breakpoints
 * @param values - Object with breakpoint keys and corresponding values
 * @returns The value for the current breakpoint
 */
export function useResponsiveValue<T>(values: {
  default: T;
  sm?: T;
  md?: T;
  lg?: T;
  xl?: T;
  '2xl'?: T;
}): T {
  const breakpoints = useBreakpoints();

  if (breakpoints.is2Xl && values['2xl'] !== undefined) {
    return values['2xl'];
  }
  if (breakpoints.isXl && values.xl !== undefined) {
    return values.xl;
  }
  if (breakpoints.isLg && values.lg !== undefined) {
    return values.lg;
  }
  if (breakpoints.isMd && values.md !== undefined) {
    return values.md;
  }
  if (breakpoints.isSm && values.sm !== undefined) {
    return values.sm;
  }
  
  return values.default;
}