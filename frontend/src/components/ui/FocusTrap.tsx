import { useEffect, useRef, ReactNode } from 'react';
import { useFocusManagement } from '../../hooks/useKeyboardNavigation';

interface FocusTrapProps {
  children: ReactNode;
  active: boolean;
  restoreFocus?: boolean;
  initialFocus?: HTMLElement | (() => HTMLElement | null);
  fallbackFocus?: HTMLElement | (() => HTMLElement | null);
  onDeactivate?: () => void;
  clickOutsideDeactivates?: boolean;
  escapeDeactivates?: boolean;
}

export const FocusTrap = ({ 
  children, 
  active, 
  restoreFocus = true,
  initialFocus,
  fallbackFocus,
  onDeactivate,
  clickOutsideDeactivates = false,
  escapeDeactivates = true
}: FocusTrapProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const previousActiveElement = useRef<HTMLElement | null>(null);
  const { getFocusableElements } = useFocusManagement();

  useEffect(() => {
    if (!active) return;

    const container = containerRef.current;
    if (!container) return;

    // Store the previously focused element
    previousActiveElement.current = document.activeElement as HTMLElement;

    // Focus management
    const setInitialFocus = () => {
      let elementToFocus: HTMLElement | null = null;

      // Try initial focus element
      if (initialFocus) {
        elementToFocus = typeof initialFocus === 'function' ? initialFocus() : initialFocus;
      }

      // Try first focusable element in container
      if (!elementToFocus) {
        const focusableElements = getFocusableElements(container);
        elementToFocus = focusableElements[0] || null;
      }

      // Try fallback focus element
      if (!elementToFocus && fallbackFocus) {
        elementToFocus = typeof fallbackFocus === 'function' ? fallbackFocus() : fallbackFocus;
      }

      // Focus the container itself as last resort
      if (!elementToFocus) {
        container.tabIndex = -1;
        elementToFocus = container;
      }

      if (elementToFocus) {
        elementToFocus.focus();
      }
    };

    // Set initial focus
    setInitialFocus();

    // Handle tab key for focus trapping
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && escapeDeactivates) {
        event.preventDefault();
        onDeactivate?.();
        return;
      }

      if (event.key !== 'Tab') return;

      const focusableElements = getFocusableElements(container);
      if (focusableElements.length === 0) return;

      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];
      const activeElement = document.activeElement as HTMLElement;

      if (event.shiftKey) {
        // Shift + Tab (moving backwards)
        if (activeElement === firstElement || !container.contains(activeElement)) {
          event.preventDefault();
          lastElement.focus();
        }
      } else {
        // Tab (moving forwards)
        if (activeElement === lastElement || !container.contains(activeElement)) {
          event.preventDefault();
          firstElement.focus();
        }
      }
    };

    // Handle click outside
    const handleClickOutside = (event: MouseEvent) => {
      if (clickOutsideDeactivates && container && !container.contains(event.target as Node)) {
        onDeactivate?.();
      }
    };

    // Add event listeners
    document.addEventListener('keydown', handleKeyDown);
    if (clickOutsideDeactivates) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      if (clickOutsideDeactivates) {
        document.removeEventListener('mousedown', handleClickOutside);
      }
    };
  }, [
    active, 
    initialFocus, 
    fallbackFocus, 
    onDeactivate, 
    clickOutsideDeactivates, 
    escapeDeactivates,
    getFocusableElements
  ]);

  // Restore focus when deactivating
  useEffect(() => {
    return () => {
      if (restoreFocus && previousActiveElement.current) {
        // Use setTimeout to ensure any pending focus changes are completed
        setTimeout(() => {
          if (previousActiveElement.current && document.body.contains(previousActiveElement.current)) {
            previousActiveElement.current.focus();
          }
        }, 0);
      }
    };
  }, [restoreFocus]);

  if (!active) {
    return <>{children}</>;
  }

  return (
    <div
      ref={containerRef}
      data-focus-trap="true"
      role="dialog"
      aria-modal="true"
    >
      {children}
    </div>
  );
};

// Hook for managing focus trap state
export const useFocusTrap = (initialActive = false) => {
  const isActive = useRef(initialActive);

  const activate = () => {
    isActive.current = true;
  };

  const deactivate = () => {
    isActive.current = false;
  };

  const toggle = () => {
    isActive.current = !isActive.current;
  };

  return {
    isActive: isActive.current,
    activate,
    deactivate,
    toggle
  };
};

export default FocusTrap;