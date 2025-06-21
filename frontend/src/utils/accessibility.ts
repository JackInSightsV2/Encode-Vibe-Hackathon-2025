// Color contrast validation utilities
export interface ColorContrastOptions {
  background: string;
  foreground: string;
  level?: 'AA' | 'AAA';
  size?: 'normal' | 'large';
}

// Convert hex color to RGB
export const hexToRgb = (hex: string): { r: number; g: number; b: number } | null => {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result ? {
    r: parseInt(result[1], 16),
    g: parseInt(result[2], 16),
    b: parseInt(result[3], 16)
  } : null;
};

// Calculate relative luminance
export const getLuminance = (r: number, g: number, b: number): number => {
  const [rs, gs, bs] = [r, g, b].map(c => {
    c = c / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
};

// Calculate contrast ratio between two colors
export const getContrastRatio = (color1: string, color2: string): number => {
  const rgb1 = hexToRgb(color1);
  const rgb2 = hexToRgb(color2);
  
  if (!rgb1 || !rgb2) return 0;
  
  const lum1 = getLuminance(rgb1.r, rgb1.g, rgb1.b);
  const lum2 = getLuminance(rgb2.r, rgb2.g, rgb2.b);
  
  const brightest = Math.max(lum1, lum2);
  const darkest = Math.min(lum1, lum2);
  
  return (brightest + 0.05) / (darkest + 0.05);
};

// Check if color combination meets WCAG standards
export const isAccessibleColor = ({ 
  background, 
  foreground, 
  level = 'AA', 
  size = 'normal' 
}: ColorContrastOptions): boolean => {
  const ratio = getContrastRatio(background, foreground);
  
  if (level === 'AAA') {
    return size === 'large' ? ratio >= 4.5 : ratio >= 7;
  } else {
    return size === 'large' ? ratio >= 3 : ratio >= 4.5;
  }
};

// Get color contrast grade
export const getContrastGrade = (background: string, foreground: string): {
  ratio: number;
  gradeAA: boolean;
  gradeAAA: boolean;
  gradeAALarge: boolean;
  gradeAAALarge: boolean;
} => {
  const ratio = getContrastRatio(background, foreground);
  
  return {
    ratio,
    gradeAA: ratio >= 4.5,
    gradeAAA: ratio >= 7,
    gradeAALarge: ratio >= 3,
    gradeAAALarge: ratio >= 4.5
  };
};

// Generate accessible color suggestions
export const suggestAccessibleColors = (
  baseColor: string,
  targetContrast: number = 4.5
): string[] => {
  const suggestions: string[] = [];
  const baseRgb = hexToRgb(baseColor);
  
  if (!baseRgb) return suggestions;
  
  // Try different brightness levels
  for (let brightness = 0; brightness <= 255; brightness += 15) {
    const testColor = `#${brightness.toString(16).padStart(2, '0').repeat(3)}`;
    const ratio = getContrastRatio(baseColor, testColor);
    
    if (ratio >= targetContrast) {
      suggestions.push(testColor);
    }
  }
  
  return suggestions.slice(0, 5); // Return top 5 suggestions
};

// ARIA utilities
export interface AriaAttributes {
  role?: string;
  'aria-label'?: string;
  'aria-labelledby'?: string;
  'aria-describedby'?: string;
  'aria-expanded'?: boolean;
  'aria-hidden'?: boolean;
  'aria-live'?: 'polite' | 'assertive' | 'off';
  'aria-atomic'?: boolean;
  'aria-busy'?: boolean;
  'aria-controls'?: string;
  'aria-current'?: boolean | 'page' | 'step' | 'location' | 'date' | 'time';
  'aria-disabled'?: boolean;
  'aria-haspopup'?: boolean | 'false' | 'true' | 'menu' | 'listbox' | 'tree' | 'grid' | 'dialog';
  'aria-level'?: number;
  'aria-modal'?: boolean;
  'aria-multiline'?: boolean;
  'aria-multiselectable'?: boolean;
  'aria-orientation'?: 'horizontal' | 'vertical';
  'aria-pressed'?: boolean;
  'aria-readonly'?: boolean;
  'aria-required'?: boolean;
  'aria-selected'?: boolean;
  'aria-sort'?: 'none' | 'ascending' | 'descending' | 'other';
  'aria-valuemax'?: number;
  'aria-valuemin'?: number;
  'aria-valuenow'?: number;
  'aria-valuetext'?: string;
}

// Generate ARIA attributes for common components
export const generateAriaAttributes = {
  button: (label: string, expanded?: boolean, controls?: string): AriaAttributes => ({
    role: 'button',
    'aria-label': label,
    ...(expanded !== undefined && { 'aria-expanded': expanded }),
    ...(controls && { 'aria-controls': controls })
  }),
  
  navigation: (label: string): AriaAttributes => ({
    role: 'navigation',
    'aria-label': label
  }),
  
  tabList: (): AriaAttributes => ({
    role: 'tablist'
  }),
  
  tab: (selected: boolean, controls: string): AriaAttributes => ({
    role: 'tab',
    'aria-selected': selected,
    'aria-controls': controls
  }),
  
  tabPanel: (labelledby: string): AriaAttributes => ({
    role: 'tabpanel',
    'aria-labelledby': labelledby
  }),
  
  dialog: (labelledby?: string, describedby?: string): AriaAttributes => ({
    role: 'dialog',
    'aria-modal': true,
    ...(labelledby && { 'aria-labelledby': labelledby }),
    ...(describedby && { 'aria-describedby': describedby })
  }),
  
  listbox: (label: string, multiselectable?: boolean): AriaAttributes => ({
    role: 'listbox',
    'aria-label': label,
    ...(multiselectable && { 'aria-multiselectable': multiselectable })
  }),
  
  option: (selected: boolean): AriaAttributes => ({
    role: 'option',
    'aria-selected': selected
  }),
  
  progressbar: (value: number, min: number = 0, max: number = 100, label?: string): AriaAttributes => ({
    role: 'progressbar',
    'aria-valuenow': value,
    'aria-valuemin': min,
    'aria-valuemax': max,
    ...(label && { 'aria-label': label })
  }),
  
  status: (label?: string): AriaAttributes => ({
    role: 'status',
    'aria-live': 'polite',
    'aria-atomic': true,
    ...(label && { 'aria-label': label })
  }),
  
  alert: (label?: string): AriaAttributes => ({
    role: 'alert',
    'aria-live': 'assertive',
    'aria-atomic': true,
    ...(label && { 'aria-label': label })
  })
};

// Keyboard navigation utilities
export const KeyCodes = {
  TAB: 'Tab',
  ENTER: 'Enter',
  SPACE: ' ',
  ESCAPE: 'Escape',
  ARROW_UP: 'ArrowUp',
  ARROW_DOWN: 'ArrowDown',
  ARROW_LEFT: 'ArrowLeft',
  ARROW_RIGHT: 'ArrowRight',
  HOME: 'Home',
  END: 'End',
  PAGE_UP: 'PageUp',
  PAGE_DOWN: 'PageDown'
} as const;

// Focus management utilities
export const focusUtils = {
  // Get all focusable elements within a container
  getFocusableElements: (container: HTMLElement): HTMLElement[] => {
    const focusableSelectors = [
      'button:not([disabled])',
      '[href]',
      'input:not([disabled])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      '[tabindex]:not([tabindex="-1"]):not([disabled])',
      '[contenteditable="true"]'
    ].join(', ');
    
    return Array.from(container.querySelectorAll(focusableSelectors));
  },
  
  // Focus the first focusable element
  focusFirst: (container: HTMLElement): boolean => {
    const elements = focusUtils.getFocusableElements(container);
    if (elements.length > 0) {
      elements[0].focus();
      return true;
    }
    return false;
  },
  
  // Focus the last focusable element
  focusLast: (container: HTMLElement): boolean => {
    const elements = focusUtils.getFocusableElements(container);
    if (elements.length > 0) {
      elements[elements.length - 1].focus();
      return true;
    }
    return false;
  },
  
  // Check if element is visible and focusable
  isFocusable: (element: HTMLElement): boolean => {
    if (element.hidden || element.getAttribute('aria-hidden') === 'true') {
      return false;
    }
    
    const style = getComputedStyle(element);
    return style.display !== 'none' && style.visibility !== 'hidden';
  }
};

// Screen reader detection
export const getScreenReaderInfo = () => {
  const userAgent = navigator.userAgent.toLowerCase();
  const hasScreenReader = 
    userAgent.includes('nvda') || 
    userAgent.includes('jaws') || 
    userAgent.includes('sapi') ||
    userAgent.includes('dragon') ||
    // Check for reduced motion preference as indicator
    window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    
  return {
    hasScreenReader,
    prefersReducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    prefersHighContrast: window.matchMedia('(prefers-contrast: high)').matches,
    prefersColorScheme: window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  };
};

// Accessibility testing utilities
export const a11yTest = {
  // Check if all images have alt text
  checkImageAltText: (): { passed: boolean; issues: string[] } => {
    const images = document.querySelectorAll('img');
    const issues: string[] = [];
    
    images.forEach((img, index) => {
      if (!img.hasAttribute('alt')) {
        issues.push(`Image ${index + 1} missing alt attribute`);
      } else if (img.getAttribute('alt')?.trim() === '') {
        // Empty alt is valid for decorative images
        if (!img.hasAttribute('role') || img.getAttribute('role') !== 'presentation') {
          issues.push(`Image ${index + 1} has empty alt but no presentation role`);
        }
      }
    });
    
    return { passed: issues.length === 0, issues };
  },
  
  // Check for proper heading hierarchy
  checkHeadingHierarchy: (): { passed: boolean; issues: string[] } => {
    const headings = document.querySelectorAll('h1, h2, h3, h4, h5, h6');
    const issues: string[] = [];
    let lastLevel = 0;
    
    headings.forEach((heading, index) => {
      const level = parseInt(heading.tagName.charAt(1));
      
      if (index === 0 && level !== 1) {
        issues.push('Page should start with h1');
      } else if (level > lastLevel + 1) {
        issues.push(`Heading level jumps from h${lastLevel} to h${level}`);
      }
      
      lastLevel = level;
    });
    
    return { passed: issues.length === 0, issues };
  },
  
  // Check for sufficient color contrast
  checkColorContrast: (): { passed: boolean; issues: string[] } => {
    const issues: string[] = [];
    // This would require more complex analysis of computed styles
    // For now, return a placeholder
    return { passed: true, issues };
  },
  
  // Run all accessibility tests
  runAll: () => {
    const results = {
      imageAltText: a11yTest.checkImageAltText(),
      headingHierarchy: a11yTest.checkHeadingHierarchy(),
      colorContrast: a11yTest.checkColorContrast()
    };
    
    const allPassed = Object.values(results).every(result => result.passed);
    const allIssues = Object.values(results).flatMap(result => result.issues);
    
    return {
      passed: allPassed,
      issues: allIssues,
      details: results
    };
  }
};