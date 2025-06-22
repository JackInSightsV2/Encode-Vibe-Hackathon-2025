# Mobile-Responsive Design & Progressive Web App - Refined Implementation Cycles

## Overview
Break down mobile-responsive design and PWA implementation into 4 manageable cycles, each delivering testable functionality with clear outcomes.

---

## **Cycle 17A: Mobile-First Responsive Foundation**
**Duration:** 5-6 hours | **Priority:** Critical

### Prerequisites
- React frontend development environment
- Understanding of responsive design principles
- Knowledge of CSS Grid/Flexbox and media queries

### Implementation Tasks
- [ ] Implement mobile-first CSS architecture
- [ ] Create responsive breakpoint system
- [ ] Redesign navigation for mobile devices
- [ ] Optimize touch targets and interactions
- [ ] Implement responsive typography and spacing
- [ ] Add mobile-optimized form components

### Code Deliverables
```css
/* frontend/src/styles/responsive.css */
:root {
  /* Breakpoints */
  --mobile: 320px;
  --tablet: 768px;
  --desktop: 1024px;
  --wide: 1440px;
  
  /* Touch targets */
  --touch-target-min: 44px;
  
  /* Spacing scale */
  --space-xs: 0.25rem;
  --space-sm: 0.5rem;
  --space-md: 1rem;
  --space-lg: 1.5rem;
  --space-xl: 2rem;
  
  /* Typography scale */
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.25rem;
}

/* Mobile-first base styles */
.container {
  width: 100%;
  padding: 0 var(--space-md);
}

/* Tablet styles */
@media (min-width: 768px) {
  .container {
    max-width: 768px;
    margin: 0 auto;
    padding: 0 var(--space-lg);
  }
}

/* Desktop styles */
@media (min-width: 1024px) {
  .container {
    max-width: 1200px;
    padding: 0 var(--space-xl);
  }
}

/* Touch-friendly components */
.touch-target {
  min-height: var(--touch-target-min);
  min-width: var(--touch-target-min);
  display: flex;
  align-items: center;
  justify-content: center;
}

.mobile-nav {
  display: block;
}

.desktop-nav {
  display: none;
}

@media (min-width: 768px) {
  .mobile-nav {
    display: none;
  }
  
  .desktop-nav {
    display: block;
  }
}
```

```typescript
// frontend/src/components/navigation/MobileNavigation.tsx
import React, { useState } from 'react';
import { Menu, X, Home, Settings, BarChart, AlertTriangle } from 'lucide-react';

interface MobileNavigationProps {
  currentPath: string;
  onNavigate: (path: string) => void;
}

export const MobileNavigation: React.FC<MobileNavigationProps> = ({
  currentPath,
  onNavigate
}) => {
  const [isOpen, setIsOpen] = useState(false);

  const navItems = [
    { path: '/', label: 'Dashboard', icon: Home },
    { path: '/analytics', label: 'Analytics', icon: BarChart },
    { path: '/alerts', label: 'Alerts', icon: AlertTriangle },
    { path: '/settings', label: 'Settings', icon: Settings },
  ];

  return (
    <>
      {/* Mobile Header */}
      <header className="md:hidden bg-blue-600 text-white p-4 flex justify-between items-center">
        <h1 className="text-lg font-semibold">QT-1 Admin</h1>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="touch-target p-2 hover:bg-blue-700 rounded"
          aria-label={isOpen ? 'Close menu' : 'Open menu'}
        >
          {isOpen ? <X size={24} /> : <Menu size={24} />}
        </button>
      </header>

      {/* Mobile Slide-out Menu */}
      <nav
        className={`md:hidden fixed inset-y-0 left-0 z-50 w-64 bg-white shadow-lg transform transition-transform duration-300 ${
          isOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="p-4 border-b">
          <h2 className="text-xl font-bold text-gray-800">Navigation</h2>
        </div>
        <ul className="py-4">
          {navItems.map(({ path, label, icon: Icon }) => (
            <li key={path}>
              <button
                onClick={() => {
                  onNavigate(path);
                  setIsOpen(false);
                }}
                className={`w-full flex items-center px-4 py-3 text-left hover:bg-gray-100 touch-target ${
                  currentPath === path ? 'bg-blue-50 text-blue-600 border-r-2 border-blue-600' : 'text-gray-700'
                }`}
              >
                <Icon size={20} className="mr-3" />
                {label}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      {/* Overlay */}
      {isOpen && (
        <div
          className="md:hidden fixed inset-0 bg-black bg-opacity-50 z-40"
          onClick={() => setIsOpen(false)}
        />
      )}

      {/* Bottom Tab Bar for Mobile */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 safe-area-inset-bottom">
        <div className="flex">
          {navItems.slice(0, 4).map(({ path, label, icon: Icon }) => (
            <button
              key={path}
              onClick={() => onNavigate(path)}
              className={`flex-1 flex flex-col items-center py-2 px-1 touch-target ${
                currentPath === path ? 'text-blue-600' : 'text-gray-500'
              }`}
            >
              <Icon size={20} />
              <span className="text-xs mt-1">{label}</span>
            </button>
          ))}
        </div>
      </nav>
    </>
  );
};
```

```typescript
// frontend/src/hooks/useResponsive.ts
import { useState, useEffect } from 'react';

export interface BreakpointState {
  isMobile: boolean;
  isTablet: boolean;
  isDesktop: boolean;
  width: number;
}

export const useResponsive = (): BreakpointState => {
  const [breakpoint, setBreakpoint] = useState<BreakpointState>({
    isMobile: false,
    isTablet: false,
    isDesktop: false,
    width: 0,
  });

  useEffect(() => {
    const updateBreakpoint = () => {
      const width = window.innerWidth;
      setBreakpoint({
        isMobile: width < 768,
        isTablet: width >= 768 && width < 1024,
        isDesktop: width >= 1024,
        width,
      });
    };

    updateBreakpoint();
    window.addEventListener('resize', updateBreakpoint);
    return () => window.removeEventListener('resize', updateBreakpoint);
  }, []);

  return breakpoint;
};
```

### Testing Requirements
- [ ] Test responsive breakpoints on various device sizes
- [ ] Test touch targets meet accessibility guidelines (44px minimum)
- [ ] Test mobile navigation usability
- [ ] Test responsive typography scales correctly
- [ ] Test form interactions on touch devices

### Acceptance Criteria
- [ ] All components responsive across mobile, tablet, desktop
- [ ] Touch targets minimum 44px for accessibility compliance
- [ ] Mobile navigation accessible and intuitive
- [ ] Typography scales appropriately across breakpoints
- [ ] Layout doesn't break on any common device size
- [ ] Performance impact <100ms for responsive adjustments

### Risk Mitigation
- Test on real devices, not just browser dev tools
- Use progressive enhancement approach
- Validate with accessibility tools (axe, WAVE)

---

## **Cycle 17B: Progressive Web App (PWA) Setup**
**Duration:** 4-5 hours | **Priority:** High

### Prerequisites
- Cycle 17A completed and tested
- Understanding of PWA concepts and Service Workers
- Knowledge of web app manifest specification

### Implementation Tasks
- [ ] Create web app manifest for PWA
- [ ] Implement service worker for caching
- [ ] Add app installation prompt
- [ ] Create offline fallback pages
- [ ] Add PWA icons and splash screens
- [ ] Implement basic offline functionality

### Code Deliverables
```json
// public/manifest.json
{
  "name": "QT-1 Middleware Admin",
  "short_name": "QT-1 Admin",
  "description": "Responsible AI Middleware Administration Dashboard",
  "start_url": "/",
  "display": "standalone",
  "theme_color": "#1e3a8a",
  "background_color": "#ffffff",
  "orientation": "portrait-primary",
  "scope": "/",
  "lang": "en",
  "categories": ["productivity", "utilities"],
  "icons": [
    {
      "src": "/icons/icon-72x72.png",
      "sizes": "72x72",
      "type": "image/png",
      "purpose": "maskable any"
    },
    {
      "src": "/icons/icon-96x96.png",
      "sizes": "96x96",
      "type": "image/png",
      "purpose": "maskable any"
    },
    {
      "src": "/icons/icon-128x128.png",
      "sizes": "128x128",
      "type": "image/png",
      "purpose": "maskable any"
    },
    {
      "src": "/icons/icon-144x144.png",
      "sizes": "144x144",
      "type": "image/png",
      "purpose": "maskable any"
    },
    {
      "src": "/icons/icon-152x152.png",
      "sizes": "152x152",
      "type": "image/png",
      "purpose": "maskable any"
    },
    {
      "src": "/icons/icon-192x192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "maskable any"
    },
    {
      "src": "/icons/icon-384x384.png",
      "sizes": "384x384",
      "type": "image/png",
      "purpose": "maskable any"
    },
    {
      "src": "/icons/icon-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "maskable any"
    }
  ],
  "shortcuts": [
    {
      "name": "Dashboard",
      "short_name": "Dashboard",
      "description": "View system dashboard",
      "url": "/dashboard",
      "icons": [{ "src": "/icons/dashboard-96x96.png", "sizes": "96x96" }]
    },
    {
      "name": "Analytics",
      "short_name": "Analytics",
      "description": "View analytics",
      "url": "/analytics",
      "icons": [{ "src": "/icons/analytics-96x96.png", "sizes": "96x96" }]
    }
  ]
}
```

```javascript
// public/sw.js - Service Worker
const CACHE_NAME = 'qt1-admin-v1';
const STATIC_CACHE = [
  '/',
  '/static/js/bundle.js',
  '/static/css/main.css',
  '/manifest.json',
  '/offline.html'
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(STATIC_CACHE))
      .then(() => self.skipWaiting())
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames
            .filter((name) => name !== CACHE_NAME)
            .map((name) => caches.delete(name))
        );
      })
      .then(() => self.clients.claim())
  );
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', (event) => {
  event.respondWith(
    caches.match(event.request)
      .then((response) => {
        // Return cached version or fetch from network
        return response || fetch(event.request)
          .then((fetchResponse) => {
            // Cache successful responses
            if (fetchResponse.status === 200) {
              const responseClone = fetchResponse.clone();
              caches.open(CACHE_NAME)
                .then((cache) => cache.put(event.request, responseClone));
            }
            return fetchResponse;
          })
          .catch(() => {
            // Fallback to offline page for navigation requests
            if (event.request.mode === 'navigate') {
              return caches.match('/offline.html');
            }
          });
      })
  );
});
```

```typescript
// frontend/src/components/PWAInstallPrompt.tsx
import React, { useState, useEffect } from 'react';
import { Download, X } from 'lucide-react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

export const PWAInstallPrompt: React.FC = () => {
  const [installPrompt, setInstallPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [showPrompt, setShowPrompt] = useState(false);

  useEffect(() => {
    const handleBeforeInstallPrompt = (e: BeforeInstallPromptEvent) => {
      e.preventDefault();
      setInstallPrompt(e);
      setShowPrompt(true);
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt as EventListener);

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt as EventListener);
    };
  }, []);

  const handleInstall = async () => {
    if (!installPrompt) return;

    await installPrompt.prompt();
    const choice = await installPrompt.userChoice;
    
    if (choice.outcome === 'accepted') {
      console.log('PWA installed');
    }
    
    setInstallPrompt(null);
    setShowPrompt(false);
  };

  const handleDismiss = () => {
    setShowPrompt(false);
    // Remember dismissal for this session
    sessionStorage.setItem('pwa-prompt-dismissed', 'true');
  };

  if (!showPrompt || sessionStorage.getItem('pwa-prompt-dismissed')) {
    return null;
  }

  return (
    <div className="fixed bottom-20 left-4 right-4 md:left-auto md:right-4 md:w-96 bg-white rounded-lg shadow-lg border border-gray-200 p-4 z-50">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <h3 className="font-semibold text-gray-900">Install QT-1 Admin</h3>
          <p className="text-sm text-gray-600 mt-1">
            Install our app for quick access and offline functionality.
          </p>
        </div>
        <button
          onClick={handleDismiss}
          className="ml-2 p-1 hover:bg-gray-100 rounded"
          aria-label="Dismiss"
        >
          <X size={16} />
        </button>
      </div>
      <div className="flex gap-2 mt-3">
        <button
          onClick={handleInstall}
          className="flex items-center px-3 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm font-medium"
        >
          <Download size={16} className="mr-1" />
          Install
        </button>
        <button
          onClick={handleDismiss}
          className="px-3 py-2 text-gray-600 hover:text-gray-800 text-sm"
        >
          Not now
        </button>
      </div>
    </div>
  );
};
```

```html
<!-- public/offline.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>QT-1 Admin - Offline</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            background-color: #f9fafb;
            color: #374151;
            text-align: center;
            padding: 1rem;
        }
        .offline-icon {
            width: 64px;
            height: 64px;
            margin-bottom: 1rem;
            opacity: 0.6;
        }
        h1 { margin-bottom: 0.5rem; color: #1f2937; }
        p { margin-bottom: 1rem; }
        .retry-btn {
            background: #3b82f6;
            color: white;
            border: none;
            padding: 0.75rem 1.5rem;
            border-radius: 0.5rem;
            cursor: pointer;
            font-size: 1rem;
        }
        .retry-btn:hover { background: #2563eb; }
    </style>
</head>
<body>
    <svg class="offline-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 5.636l-12.728 12.728m0-12.728l12.728 12.728"></path>
    </svg>
    <h1>You're offline</h1>
    <p>Check your internet connection and try again.</p>
    <button class="retry-btn" onclick="window.location.reload()">
        Try Again
    </button>
</body>
</html>
```

### Testing Requirements
- [ ] Test PWA installation on mobile devices
- [ ] Test offline functionality and fallbacks
- [ ] Test service worker caching behavior
- [ ] Test manifest validation
- [ ] Test app shortcuts and icons

### Acceptance Criteria
- [ ] PWA installable on iOS and Android devices
- [ ] Offline page displays when network unavailable
- [ ] App launches in standalone mode when installed
- [ ] Service worker caches critical resources
- [ ] Installation prompt appears on appropriate devices
- [ ] PWA audit score >90 in Lighthouse

### Risk Mitigation
- Test PWA features across different browsers and devices
- Implement gradual enhancement for offline features
- Use established PWA patterns and libraries

---

## **Cycle 17C: Touch-Optimized Components and Gestures**
**Duration:** 5-6 hours | **Priority:** Medium

### Prerequisites
- Cycles 17A and 17B completed
- Understanding of touch interaction patterns
- Knowledge of gesture handling in web applications

### Implementation Tasks
- [ ] Implement touch-optimized data tables
- [ ] Add swipe gestures for navigation and actions
- [ ] Create mobile-friendly form components
- [ ] Add pull-to-refresh functionality
- [ ] Implement touch-friendly charts and visualizations
- [ ] Add haptic feedback for supported devices

### Code Deliverables
```typescript
// frontend/src/components/mobile/MobileTable.tsx
import React, { useState } from 'react';
import { ChevronDown, ChevronRight, MoreHorizontal } from 'lucide-react';

interface MobileTableProps<T> {
  data: T[];
  columns: Array<{
    key: keyof T;
    label: string;
    primary?: boolean;
    secondary?: boolean;
    render?: (value: any, item: T) => React.ReactNode;
  }>;
  onRowTap?: (item: T) => void;
  onRowAction?: (action: string, item: T) => void;
}

export function MobileTable<T extends { id: string | number }>({
  data,
  columns,
  onRowTap,
  onRowAction
}: MobileTableProps<T>) {
  const [expandedRows, setExpandedRows] = useState<Set<string | number>>(new Set());

  const toggleRow = (id: string | number) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(id)) {
      newExpanded.delete(id);
    } else {
      newExpanded.add(id);
    }
    setExpandedRows(newExpanded);
  };

  const primaryColumn = columns.find(col => col.primary);
  const secondaryColumn = columns.find(col => col.secondary);
  const otherColumns = columns.filter(col => !col.primary && !col.secondary);

  return (
    <div className="space-y-2">
      {data.map((item) => {
        const isExpanded = expandedRows.has(item.id);
        
        return (
          <div
            key={item.id}
            className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden"
          >
            {/* Main row - always visible */}
            <div
              className="flex items-center p-4 touch-target"
              onClick={() => onRowTap?.(item)}
            >
              <div className="flex-1 min-w-0">
                {primaryColumn && (
                  <div className="font-medium text-gray-900 truncate">
                    {primaryColumn.render 
                      ? primaryColumn.render(item[primaryColumn.key], item)
                      : String(item[primaryColumn.key])
                    }
                  </div>
                )}
                {secondaryColumn && (
                  <div className="text-sm text-gray-500 truncate mt-1">
                    {secondaryColumn.render
                      ? secondaryColumn.render(item[secondaryColumn.key], item)
                      : String(item[secondaryColumn.key])
                    }
                  </div>
                )}
              </div>
              
              {otherColumns.length > 0 && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleRow(item.id);
                  }}
                  className="ml-2 p-1 hover:bg-gray-100 rounded touch-target"
                  aria-label={isExpanded ? 'Collapse' : 'Expand'}
                >
                  {isExpanded ? <ChevronDown size={20} /> : <ChevronRight size={20} />}
                </button>
              )}
              
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onRowAction?.('menu', item);
                }}
                className="ml-1 p-1 hover:bg-gray-100 rounded touch-target"
                aria-label="Actions"
              >
                <MoreHorizontal size={20} />
              </button>
            </div>

            {/* Expanded details */}
            {isExpanded && otherColumns.length > 0 && (
              <div className="px-4 pb-4 border-t border-gray-100">
                <div className="grid grid-cols-1 gap-3 mt-3">
                  {otherColumns.map((column) => (
                    <div key={String(column.key)} className="flex justify-between">
                      <span className="text-sm font-medium text-gray-700">
                        {column.label}:
                      </span>
                      <span className="text-sm text-gray-900">
                        {column.render
                          ? column.render(item[column.key], item)
                          : String(item[column.key])
                        }
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
```

```typescript
// frontend/src/hooks/useSwipeGesture.ts
import { useEffect, useRef } from 'react';

interface SwipeGestureConfig {
  onSwipeLeft?: () => void;
  onSwipeRight?: () => void;
  onSwipeUp?: () => void;
  onSwipeDown?: () => void;
  threshold?: number;
  preventScroll?: boolean;
}

export const useSwipeGesture = (config: SwipeGestureConfig) => {
  const elementRef = useRef<HTMLElement>(null);
  const touchStartRef = useRef<{ x: number; y: number } | null>(null);
  const { threshold = 50, preventScroll = false } = config;

  useEffect(() => {
    const element = elementRef.current;
    if (!element) return;

    const handleTouchStart = (e: TouchEvent) => {
      const touch = e.touches[0];
      touchStartRef.current = { x: touch.clientX, y: touch.clientY };
    };

    const handleTouchMove = (e: TouchEvent) => {
      if (preventScroll) {
        e.preventDefault();
      }
    };

    const handleTouchEnd = (e: TouchEvent) => {
      if (!touchStartRef.current) return;

      const touch = e.changedTouches[0];
      const deltaX = touch.clientX - touchStartRef.current.x;
      const deltaY = touch.clientY - touchStartRef.current.y;

      const absDeltaX = Math.abs(deltaX);
      const absDeltaY = Math.abs(deltaY);

      if (Math.max(absDeltaX, absDeltaY) < threshold) return;

      if (absDeltaX > absDeltaY) {
        // Horizontal swipe
        if (deltaX > 0) {
          config.onSwipeRight?.();
        } else {
          config.onSwipeLeft?.();
        }
      } else {
        // Vertical swipe
        if (deltaY > 0) {
          config.onSwipeDown?.();
        } else {
          config.onSwipeUp?.();
        }
      }

      touchStartRef.current = null;
    };

    element.addEventListener('touchstart', handleTouchStart, { passive: true });
    element.addEventListener('touchmove', handleTouchMove, { passive: !preventScroll });
    element.addEventListener('touchend', handleTouchEnd, { passive: true });

    return () => {
      element.removeEventListener('touchstart', handleTouchStart);
      element.removeEventListener('touchmove', handleTouchMove);
      element.removeEventListener('touchend', handleTouchEnd);
    };
  }, [config, threshold, preventScroll]);

  return elementRef;
};
```

```typescript
// frontend/src/components/mobile/PullToRefresh.tsx
import React, { useState, useRef, useEffect } from 'react';
import { RefreshCw } from 'lucide-react';

interface PullToRefreshProps {
  onRefresh: () => Promise<void>;
  children: React.ReactNode;
  threshold?: number;
  disabled?: boolean;
}

export const PullToRefresh: React.FC<PullToRefreshProps> = ({
  onRefresh,
  children,
  threshold = 80,
  disabled = false
}) => {
  const [pullDistance, setPullDistance] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [canRefresh, setCanRefresh] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const startYRef = useRef<number>(0);

  const handleTouchStart = (e: TouchEvent) => {
    if (disabled || window.scrollY > 0) return;
    startYRef.current = e.touches[0].clientY;
  };

  const handleTouchMove = (e: TouchEvent) => {
    if (disabled || isRefreshing || window.scrollY > 0) return;

    const currentY = e.touches[0].clientY;
    const distance = Math.max(0, currentY - startYRef.current);
    
    if (distance > 0) {
      e.preventDefault();
      setPullDistance(Math.min(distance, threshold * 1.5));
      setCanRefresh(distance >= threshold);
    }
  };

  const handleTouchEnd = async () => {
    if (disabled || isRefreshing) return;

    if (canRefresh) {
      setIsRefreshing(true);
      try {
        await onRefresh();
        // Haptic feedback if available
        if (navigator.vibrate) {
          navigator.vibrate(50);
        }
      } finally {
        setIsRefreshing(false);
      }
    }

    setPullDistance(0);
    setCanRefresh(false);
  };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    container.addEventListener('touchstart', handleTouchStart, { passive: true });
    container.addEventListener('touchmove', handleTouchMove, { passive: false });
    container.addEventListener('touchend', handleTouchEnd, { passive: true });

    return () => {
      container.removeEventListener('touchstart', handleTouchStart);
      container.removeEventListener('touchmove', handleTouchMove);
      container.removeEventListener('touchend', handleTouchEnd);
    };
  }, [disabled, isRefreshing, canRefresh]);

  const refreshIndicatorScale = Math.min(pullDistance / threshold, 1);
  const refreshIndicatorOpacity = pullDistance > 20 ? 1 : 0;

  return (
    <div ref={containerRef} className="relative">
      {/* Pull to refresh indicator */}
      <div
        className="absolute top-0 left-0 right-0 flex justify-center items-center transition-opacity duration-200"
        style={{
          transform: `translateY(${Math.max(0, pullDistance - 40)}px)`,
          opacity: refreshIndicatorOpacity,
          height: '40px'
        }}
      >
        <div
          className={`flex items-center justify-center w-8 h-8 rounded-full bg-blue-500 text-white transition-transform duration-200 ${
            isRefreshing ? 'animate-spin' : ''
          }`}
          style={{
            transform: `scale(${refreshIndicatorScale})`,
            backgroundColor: canRefresh ? '#10b981' : '#3b82f6'
          }}
        >
          <RefreshCw size={16} />
        </div>
      </div>

      {/* Content */}
      <div
        style={{
          transform: `translateY(${pullDistance}px)`,
          transition: pullDistance === 0 ? 'transform 0.3s ease-out' : 'none'
        }}
      >
        {children}
      </div>
    </div>
  );
};
```

### Testing Requirements
- [ ] Test swipe gestures on various touch devices
- [ ] Test pull-to-refresh functionality
- [ ] Test mobile table interactions
- [ ] Test haptic feedback where supported
- [ ] Test gesture conflicts with native browser behavior

### Acceptance Criteria
- [ ] Swipe gestures work consistently across touch devices
- [ ] Pull-to-refresh provides clear visual feedback
- [ ] Mobile tables display data clearly without horizontal scroll
- [ ] Touch interactions feel responsive (<100ms feedback)
- [ ] Gestures don't conflict with native scrolling
- [ ] Haptic feedback enhances touch interactions

### Risk Mitigation
- Test gestures on real devices with different screen sizes
- Implement gesture conflict resolution
- Provide visual feedback for all touch interactions

---

## **Cycle 17D: Performance Optimization and Testing**
**Duration:** 4-5 hours | **Priority:** Low

### Prerequisites
- Cycles 17A, 17B, and 17C completed
- Understanding of mobile performance optimization
- Knowledge of mobile testing strategies

### Implementation Tasks
- [ ] Optimize bundle size for mobile devices
- [ ] Implement image optimization and lazy loading
- [ ] Add mobile performance monitoring
- [ ] Create comprehensive mobile testing suite
- [ ] Optimize for slow network connections
- [ ] Add accessibility testing for mobile

### Code Deliverables
```typescript
// frontend/src/utils/imageOptimization.ts
interface ImageOptimizationOptions {
  quality?: number;
  format?: 'webp' | 'jpeg' | 'png';
  width?: number;
  height?: number;
  lazy?: boolean;
}

export const optimizeImageSrc = (
  src: string, 
  options: ImageOptimizationOptions = {}
): string => {
  const { quality = 80, format = 'webp', width, height } = options;
  
  // If it's already optimized or external, return as-is
  if (src.startsWith('http') || src.includes('optimized')) {
    return src;
  }
  
  const params = new URLSearchParams();
  params.set('q', quality.toString());
  params.set('f', format);
  if (width) params.set('w', width.toString());
  if (height) params.set('h', height.toString());
  
  return `/api/images/optimize?src=${encodeURIComponent(src)}&${params.toString()}`;
};

// Responsive image component
export const ResponsiveImage: React.FC<{
  src: string;
  alt: string;
  className?: string;
  lazy?: boolean;
  sizes?: string;
}> = ({ src, alt, className, lazy = true, sizes }) => {
  const [isLoaded, setIsLoaded] = useState(false);
  const [error, setError] = useState(false);

  const webpSrc = optimizeImageSrc(src, { format: 'webp' });
  const fallbackSrc = optimizeImageSrc(src, { format: 'jpeg' });

  return (
    <picture className={`block ${className}`}>
      <source srcSet={webpSrc} type="image/webp" sizes={sizes} />
      <img
        src={fallbackSrc}
        alt={alt}
        loading={lazy ? 'lazy' : 'eager'}
        sizes={sizes}
        className={`w-full h-auto transition-opacity duration-300 ${
          isLoaded ? 'opacity-100' : 'opacity-0'
        }`}
        onLoad={() => setIsLoaded(true)}
        onError={() => setError(true)}
      />
      {!isLoaded && !error && (
        <div className="bg-gray-200 animate-pulse w-full h-32 flex items-center justify-center">
          <span className="text-gray-400">Loading...</span>
        </div>
      )}
    </picture>
  );
};
```

```typescript
// frontend/src/utils/performanceMonitoring.ts
interface PerformanceMetrics {
  fcp: number; // First Contentful Paint
  lcp: number; // Largest Contentful Paint
  fid: number; // First Input Delay
  cls: number; // Cumulative Layout Shift
  ttfb: number; // Time to First Byte
}

export class MobilePerformanceMonitor {
  private metrics: Partial<PerformanceMetrics> = {};
  private observer: PerformanceObserver | null = null;

  constructor() {
    this.setupPerformanceObserver();
    this.measureNetworkSpeed();
  }

  private setupPerformanceObserver() {
    if ('PerformanceObserver' in window) {
      this.observer = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          switch (entry.entryType) {
            case 'paint':
              if (entry.name === 'first-contentful-paint') {
                this.metrics.fcp = entry.startTime;
              }
              break;
            case 'largest-contentful-paint':
              this.metrics.lcp = entry.startTime;
              break;
            case 'first-input':
              this.metrics.fid = entry.processingStart - entry.startTime;
              break;
            case 'layout-shift':
              if (!entry.hadRecentInput) {
                this.metrics.cls = (this.metrics.cls || 0) + entry.value;
              }
              break;
          }
        }
      });

      this.observer.observe({ entryTypes: ['paint', 'largest-contentful-paint', 'first-input', 'layout-shift'] });
    }
  }

  private measureNetworkSpeed() {
    if ('connection' in navigator) {
      const connection = (navigator as any).connection;
      console.log('Network type:', connection.effectiveType);
      console.log('Downlink speed:', connection.downlink, 'Mbps');
    }
  }

  public reportMetrics() {
    // Send metrics to analytics service
    const report = {
      ...this.metrics,
      timestamp: Date.now(),
      userAgent: navigator.userAgent,
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight
      }
    };

    // Report to analytics service
    if (process.env.NODE_ENV === 'production') {
      fetch('/api/analytics/performance', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(report)
      }).catch(console.error);
    }

    return report;
  }

  public destroy() {
    if (this.observer) {
      this.observer.disconnect();
    }
  }
}
```

```typescript
// frontend/src/hooks/useNetworkStatus.ts
export interface NetworkStatus {
  isOnline: boolean;
  isSlowConnection: boolean;
  effectiveType: string;
  downlink: number;
}

export const useNetworkStatus = (): NetworkStatus => {
  const [networkStatus, setNetworkStatus] = useState<NetworkStatus>({
    isOnline: navigator.onLine,
    isSlowConnection: false,
    effectiveType: 'unknown',
    downlink: 0
  });

  useEffect(() => {
    const updateNetworkStatus = () => {
      const connection = (navigator as any).connection;
      setNetworkStatus({
        isOnline: navigator.onLine,
        isSlowConnection: connection?.effectiveType === '2g' || connection?.effectiveType === 'slow-2g',
        effectiveType: connection?.effectiveType || 'unknown',
        downlink: connection?.downlink || 0
      });
    };

    updateNetworkStatus();

    window.addEventListener('online', updateNetworkStatus);
    window.addEventListener('offline', updateNetworkStatus);
    
    if ('connection' in navigator) {
      (navigator as any).connection.addEventListener('change', updateNetworkStatus);
    }

    return () => {
      window.removeEventListener('online', updateNetworkStatus);
      window.removeEventListener('offline', updateNetworkStatus);
      if ('connection' in navigator) {
        (navigator as any).connection.removeEventListener('change', updateNetworkStatus);
      }
    };
  }, []);

  return networkStatus;
};
```

```javascript
// e2e/mobile.spec.js - Mobile E2E Tests
const { test, expect, devices } = require('@playwright/test');

// Test on mobile devices
for (const deviceName of ['iPhone 13', 'Pixel 5', 'iPad']) {
  test.describe(`Mobile Tests - ${deviceName}`, () => {
    test.use({ ...devices[deviceName] });

    test('mobile navigation works', async ({ page }) => {
      await page.goto('/');
      
      // Test mobile menu
      await page.click('[aria-label="Open menu"]');
      await expect(page.locator('nav')).toBeVisible();
      
      // Test navigation
      await page.click('text=Analytics');
      await expect(page).toHaveURL('/analytics');
    });

    test('pull to refresh works', async ({ page }) => {
      await page.goto('/dashboard');
      
      // Simulate pull to refresh gesture
      await page.touchscreen.start({ x: 200, y: 100 });
      await page.touchscreen.move({ x: 200, y: 200 });
      await page.touchscreen.end();
      
      // Check for refresh indicator
      await expect(page.locator('[data-testid="refresh-indicator"]')).toBeVisible();
    });

    test('mobile table interactions', async ({ page }) => {
      await page.goto('/analytics');
      
      // Test table row expansion
      await page.click('[aria-label="Expand"]');
      await expect(page.locator('[data-testid="expanded-details"]')).toBeVisible();
      
      // Test swipe actions
      const row = page.locator('[data-testid="table-row"]').first();
      const box = await row.boundingBox();
      await page.touchscreen.start({ x: box.x + 10, y: box.y + box.height / 2 });
      await page.touchscreen.move({ x: box.x + box.width - 10, y: box.y + box.height / 2 });
      await page.touchscreen.end();
    });

    test('PWA functionality', async ({ page, context }) => {
      await page.goto('/');
      
      // Check manifest
      const manifest = await page.evaluate(() => {
        const link = document.querySelector('link[rel="manifest"]');
        return link ? link.href : null;
      });
      expect(manifest).toBeTruthy();
      
      // Check service worker registration
      const swRegistered = await page.evaluate(() => {
        return 'serviceWorker' in navigator;
      });
      expect(swRegistered).toBeTruthy();
    });

    test('performance metrics', async ({ page }) => {
      await page.goto('/');
      
      // Measure performance
      const metrics = await page.evaluate(() => {
        return new Promise((resolve) => {
          new PerformanceObserver((list) => {
            const entries = list.getEntries();
            const fcp = entries.find(e => e.name === 'first-contentful-paint');
            if (fcp) {
              resolve({ fcp: fcp.startTime });
            }
          }).observe({ entryTypes: ['paint'] });
        });
      });
      
      // FCP should be under 2 seconds on mobile
      expect(metrics.fcp).toBeLessThan(2000);
    });
  });
}
```

### Testing Requirements
- [ ] Test performance on various mobile devices
- [ ] Test with throttled network connections
- [ ] Test accessibility with screen readers
- [ ] Test PWA functionality across browsers
- [ ] Test offline behavior and service worker caching

### Acceptance Criteria
- [ ] Mobile performance scores >90 in Lighthouse
- [ ] App loads within 3 seconds on 3G networks
- [ ] Bundle size optimized for mobile (<1MB initial load)
- [ ] Images load progressively with optimization
- [ ] Accessibility score >95 for mobile interactions
- [ ] E2E tests pass on all target mobile devices

### Risk Mitigation
- Use real device testing for accurate performance measurement
- Implement progressive loading strategies
- Monitor performance metrics in production

---

## **Integration Testing Checklist**
After all cycles complete:
- [ ] Cross-device compatibility test (iOS, Android, tablets)
- [ ] PWA installation and offline functionality test
- [ ] Performance test under various network conditions
- [ ] Accessibility audit with assistive technologies
- [ ] Touch gesture conflicts and edge case testing

## **Success Metrics**
- Mobile responsiveness works on 100% of target devices
- PWA installs successfully on all supported platforms
- Touch interactions feel native and responsive
- Mobile Lighthouse score >90 (Performance, Accessibility, PWA)
- Offline functionality maintains core app capabilities
- Mobile user engagement increases by 40% post-implementation