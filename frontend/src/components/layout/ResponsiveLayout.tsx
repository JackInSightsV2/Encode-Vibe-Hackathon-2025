import React, { useState, useEffect } from 'react';
import { cn } from '../../utils/cn';
import { useIsMobile } from '../../hooks/useMediaQuery';

interface ResponsiveLayoutProps {
  sidebar?: React.ReactNode;
  header?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  sidebarClassName?: string;
  contentClassName?: string;
}

const ResponsiveLayout: React.FC<ResponsiveLayoutProps> = ({
  sidebar,
  header,
  children,
  className,
  sidebarClassName,
  contentClassName,
}) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const isMobile = useIsMobile();

  // Close sidebar when switching to desktop
  useEffect(() => {
    if (!isMobile) {
      setSidebarOpen(false);
    }
  }, [isMobile]);

  // Prevent body scroll when mobile sidebar is open
  useEffect(() => {
    if (isMobile && sidebarOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = 'unset';
    }

    return () => {
      document.body.style.overflow = 'unset';
    };
  }, [isMobile, sidebarOpen]);

  // Close sidebar on escape key
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && sidebarOpen) {
        setSidebarOpen(false);
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [sidebarOpen]);

  const toggleSidebar = () => {
    setSidebarOpen(!sidebarOpen);
  };

  return (
    <div className={cn('min-h-screen bg-gray-50', className)}>
      {/* Mobile menu overlay */}
      {isMobile && sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black bg-opacity-50 transition-opacity animate-fade-in"
          onClick={() => setSidebarOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      {sidebar && (
        <aside
          className={cn(
            'fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 transition-transform duration-300 ease-in-out',
            isMobile
              ? sidebarOpen
                ? 'translate-x-0'
                : '-translate-x-full'
              : 'translate-x-0',
            sidebarClassName
          )}
          aria-label="Sidebar navigation"
        >
          <div className="flex flex-col h-full">
            {/* Mobile sidebar header */}
            {isMobile && (
              <div className="flex items-center justify-between p-4 border-b border-gray-200">
                <h2 className="text-lg font-semibold text-gray-900">Menu</h2>
                <button
                  onClick={() => setSidebarOpen(false)}
                  className="p-2 text-gray-400 hover:text-gray-600 transition-colors rounded-md hover:bg-gray-100"
                  aria-label="Close sidebar"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            )}
            
            {/* Sidebar content */}
            <div className="flex-1 overflow-y-auto">
              {sidebar}
            </div>
          </div>
        </aside>
      )}

      {/* Main content */}
      <div
        className={cn(
          'flex flex-col transition-all duration-300',
          sidebar && !isMobile && 'ml-64'
        )}
      >
        {/* Header */}
        {header && (
          <header className="bg-white border-b border-gray-200 px-4 py-3 sm:px-6 sticky top-0 z-30">
            <div className="flex items-center justify-between">
              {/* Mobile menu button */}
              {isMobile && sidebar && (
                <button
                  onClick={toggleSidebar}
                  className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-md transition-colors mr-4"
                  aria-label="Open sidebar"
                  aria-expanded={sidebarOpen}
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                  </svg>
                </button>
              )}
              
              {/* Header content */}
              <div className="flex-1">
                {header}
              </div>
            </div>
          </header>
        )}

        {/* Page content */}
        <main className={cn('flex-1 p-4 sm:p-6', contentClassName)}>
          {children}
        </main>
      </div>
    </div>
  );
};

export { ResponsiveLayout };
export type { ResponsiveLayoutProps };