import { useState, useEffect, useCallback, useRef } from 'react';

export interface KeyboardNavigationOptions {
  items: string[];
  onSelect: (id: string) => void;
  initialIndex?: number;
  loop?: boolean;
  disabled?: boolean;
  orientation?: 'horizontal' | 'vertical';
}

export const useKeyboardNavigation = ({
  items,
  onSelect,
  initialIndex = 0,
  loop = true,
  disabled = false,
  orientation = 'vertical'
}: KeyboardNavigationOptions) => {
  const [activeIndex, setActiveIndex] = useState(initialIndex);
  const containerRef = useRef<HTMLElement>(null);

  const moveNext = useCallback(() => {
    if (disabled || items.length === 0) return;
    
    setActiveIndex(prev => {
      if (prev >= items.length - 1) {
        return loop ? 0 : prev;
      }
      return prev + 1;
    });
  }, [disabled, items.length, loop]);

  const movePrevious = useCallback(() => {
    if (disabled || items.length === 0) return;
    
    setActiveIndex(prev => {
      if (prev <= 0) {
        return loop ? items.length - 1 : prev;
      }
      return prev - 1;
    });
  }, [disabled, items.length, loop]);

  const moveFirst = useCallback(() => {
    if (disabled || items.length === 0) return;
    setActiveIndex(0);
  }, [disabled, items.length]);

  const moveLast = useCallback(() => {
    if (disabled || items.length === 0) return;
    setActiveIndex(items.length - 1);
  }, [disabled, items.length]);

  const select = useCallback((index?: number) => {
    if (disabled || items.length === 0) return;
    
    const targetIndex = index !== undefined ? index : activeIndex;
    if (targetIndex >= 0 && targetIndex < items.length) {
      onSelect(items[targetIndex]);
    }
  }, [disabled, items, activeIndex, onSelect]);

  useEffect(() => {
    if (disabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      // Only handle keyboard navigation if the container or its children have focus
      if (containerRef.current && !containerRef.current.contains(document.activeElement)) {
        return;
      }

      const isVertical = orientation === 'vertical';
      const nextKey = isVertical ? 'ArrowDown' : 'ArrowRight';
      const prevKey = isVertical ? 'ArrowUp' : 'ArrowLeft';

      switch (event.key) {
        case nextKey:
          event.preventDefault();
          moveNext();
          break;
        case prevKey:
          event.preventDefault();
          movePrevious();
          break;
        case 'Home':
          event.preventDefault();
          moveFirst();
          break;
        case 'End':
          event.preventDefault();
          moveLast();
          break;
        case 'Enter':
        case ' ':
          event.preventDefault();
          select();
          break;
        case 'Escape':
          // Allow parent components to handle escape
          break;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [disabled, orientation, moveNext, movePrevious, moveFirst, moveLast, select]);

  // Reset active index when items change
  useEffect(() => {
    if (activeIndex >= items.length) {
      setActiveIndex(Math.max(0, items.length - 1));
    }
  }, [items.length, activeIndex]);

  return {
    activeIndex,
    setActiveIndex,
    moveNext,
    movePrevious,
    moveFirst,
    moveLast,
    select,
    containerRef
  };
};

// Hook for managing focus within a component
export const useFocusManagement = () => {
  const focusableSelectors = [
    'button:not([disabled])',
    '[href]',
    'input:not([disabled])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"]):not([disabled])',
    '[contenteditable="true"]'
  ].join(', ');

  const getFocusableElements = useCallback((container: HTMLElement): HTMLElement[] => {
    return Array.from(container.querySelectorAll(focusableSelectors));
  }, [focusableSelectors]);

  const getFirstFocusableElement = useCallback((container: HTMLElement): HTMLElement | null => {
    const elements = getFocusableElements(container);
    return elements[0] || null;
  }, [getFocusableElements]);

  const getLastFocusableElement = useCallback((container: HTMLElement): HTMLElement | null => {
    const elements = getFocusableElements(container);
    return elements[elements.length - 1] || null;
  }, [getFocusableElements]);

  const focusFirst = useCallback((container: HTMLElement): boolean => {
    const element = getFirstFocusableElement(container);
    if (element) {
      element.focus();
      return true;
    }
    return false;
  }, [getFirstFocusableElement]);

  const focusLast = useCallback((container: HTMLElement): boolean => {
    const element = getLastFocusableElement(container);
    if (element) {
      element.focus();
      return true;
    }
    return false;
  }, [getLastFocusableElement]);

  return {
    getFocusableElements,
    getFirstFocusableElement,
    getLastFocusableElement,
    focusFirst,
    focusLast
  };
};

// Hook for managing roving tabindex pattern
export const useRovingTabIndex = (_items: string[], activeIndex: number) => {
  const getTabIndex = useCallback((index: number): number => {
    return index === activeIndex ? 0 : -1;
  }, [activeIndex]);

  const getAriaSelected = useCallback((index: number): boolean => {
    return index === activeIndex;
  }, [activeIndex]);

  return {
    getTabIndex,
    getAriaSelected
  };
};