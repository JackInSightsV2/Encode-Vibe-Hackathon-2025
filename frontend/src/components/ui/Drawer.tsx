import React, { useEffect, useRef } from 'react';
import { cn } from '../../utils/cn';

interface DrawerProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  position?: 'left' | 'right' | 'top' | 'bottom';
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
  children: React.ReactNode;
  className?: string;
  closeOnOverlayClick?: boolean;
  closeOnEscape?: boolean;
}

const Drawer: React.FC<DrawerProps> = ({
  isOpen,
  onClose,
  title,
  position = 'right',
  size = 'md',
  children,
  className,
  closeOnOverlayClick = true,
  closeOnEscape = true,
}) => {
  const drawerRef = useRef<HTMLDivElement>(null);
  const previousActiveElement = useRef<HTMLElement | null>(null);

  // Focus management
  useEffect(() => {
    if (isOpen) {
      previousActiveElement.current = document.activeElement as HTMLElement;
      
      const timer = setTimeout(() => {
        if (drawerRef.current) {
          drawerRef.current.focus();
        }
      }, 100);

      return () => clearTimeout(timer);
    } else {
      if (previousActiveElement.current) {
        previousActiveElement.current.focus();
      }
    }
  }, [isOpen]);

  // Escape key handler
  useEffect(() => {
    if (!closeOnEscape) return;

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose, closeOnEscape]);

  // Prevent body scroll when drawer is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = 'unset';
    }

    return () => {
      document.body.style.overflow = 'unset';
    };
  }, [isOpen]);

  if (!isOpen) return null;

  const getPositionClasses = () => {
    const baseClasses = 'fixed bg-white shadow-xl transition-transform duration-300 ease-in-out';
    
    switch (position) {
      case 'left':
        return cn(baseClasses, 'inset-y-0 left-0');
      case 'right':
        return cn(baseClasses, 'inset-y-0 right-0');
      case 'top':
        return cn(baseClasses, 'inset-x-0 top-0');
      case 'bottom':
        return cn(baseClasses, 'inset-x-0 bottom-0');
      default:
        return cn(baseClasses, 'inset-y-0 right-0');
    }
  };

  const getSizeClasses = () => {
    if (position === 'top' || position === 'bottom') {
      // Horizontal drawers
      switch (size) {
        case 'sm': return 'h-64';
        case 'md': return 'h-80';
        case 'lg': return 'h-96';
        case 'xl': return 'h-1/2';
        case 'full': return 'h-full';
        default: return 'h-80';
      }
    } else {
      // Vertical drawers
      switch (size) {
        case 'sm': return 'w-64';
        case 'md': return 'w-80';
        case 'lg': return 'w-96';
        case 'xl': return 'w-1/2';
        case 'full': return 'w-full';
        default: return 'w-80';
      }
    }
  };

  const getTransformClasses = () => {
    if (!isOpen) return '';
    
    switch (position) {
      case 'left':
        return 'translate-x-0';
      case 'right':
        return 'translate-x-0';
      case 'top':
        return 'translate-y-0';
      case 'bottom':
        return 'translate-y-0';
      default:
        return 'translate-x-0';
    }
  };

  return (
    <div className="fixed inset-0 z-50">
      {/* Overlay */}
      <div
        className="fixed inset-0 bg-black bg-opacity-50 transition-opacity animate-fade-in"
        onClick={closeOnOverlayClick ? onClose : undefined}
        aria-hidden="true"
      />
      
      {/* Drawer content */}
      <div
        ref={drawerRef}
        className={cn(
          getPositionClasses(),
          getSizeClasses(),
          getTransformClasses(),
          'z-50 flex flex-col',
          className
        )}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? 'drawer-title' : undefined}
      >
        {/* Header */}
        {title && (
          <div className="flex items-center justify-between p-4 border-b border-gray-200 flex-shrink-0">
            <h2
              id="drawer-title"
              className="text-lg font-semibold text-gray-900"
            >
              {title}
            </h2>
            <button
              onClick={onClose}
              className="p-2 text-gray-400 hover:text-gray-600 transition-colors rounded-md hover:bg-gray-100"
              aria-label="Close drawer"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        )}
        
        {/* Content */}
        <div className="flex-1 overflow-y-auto">
          {children}
        </div>
      </div>
    </div>
  );
};

// Drawer sub-components for better organization
const DrawerHeader: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({
  className,
  children,
  ...props
}) => (
  <div
    className={cn('flex flex-col space-y-1.5 p-4 border-b border-gray-200', className)}
    {...props}
  >
    {children}
  </div>
);

const DrawerTitle: React.FC<React.HTMLAttributes<HTMLHeadingElement>> = ({
  className,
  children,
  ...props
}) => (
  <h2
    className={cn('text-lg font-semibold text-gray-900', className)}
    {...props}
  >
    {children}
  </h2>
);

const DrawerContent: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({
  className,
  children,
  ...props
}) => (
  <div
    className={cn('flex-1 p-4 overflow-y-auto', className)}
    {...props}
  >
    {children}
  </div>
);

const DrawerFooter: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({
  className,
  children,
  ...props
}) => (
  <div
    className={cn('flex items-center justify-end space-x-2 p-4 border-t border-gray-200', className)}
    {...props}
  >
    {children}
  </div>
);

export { Drawer, DrawerHeader, DrawerTitle, DrawerContent, DrawerFooter };
export type { DrawerProps };