import { useEffect, useState } from 'react';

// Screen reader only content component
interface ScreenReaderOnlyProps {
  children: React.ReactNode;
  as?: keyof JSX.IntrinsicElements;
  className?: string;
}

export const ScreenReaderOnly = ({ 
  children, 
  as: Component = 'span',
  className = ''
}: ScreenReaderOnlyProps) => {
  return (
    <Component
      className={`sr-only ${className}`}
      style={{
        position: 'absolute',
        width: '1px',
        height: '1px',
        padding: '0',
        margin: '-1px',
        overflow: 'hidden',
        clip: 'rect(0, 0, 0, 0)',
        whiteSpace: 'nowrap',
        border: '0'
      }}
    >
      {children}
    </Component>
  );
};

// Live region component for dynamic content announcements
interface LiveRegionProps {
  children: React.ReactNode;
  politeness?: 'polite' | 'assertive' | 'off';
  atomic?: boolean;
  relevant?: 'additions' | 'removals' | 'text' | 'all';
  busy?: boolean;
  className?: string;
}

export const LiveRegion = ({
  children,
  politeness = 'polite',
  atomic = false,
  relevant = 'additions',
  busy = false,
  className = ''
}: LiveRegionProps) => {
  return (
    <div
      aria-live={politeness}
      aria-atomic={atomic}
      aria-relevant={relevant}
      aria-busy={busy}
      className={className}
    >
      {children}
    </div>
  );
};

// Status announcer for dynamic status updates
interface StatusAnnouncerProps {
  message: string;
  priority?: 'polite' | 'assertive';
  clearAfter?: number;
}

export const StatusAnnouncer = ({ 
  message, 
  priority = 'polite',
  clearAfter = 5000 
}: StatusAnnouncerProps) => {
  const [announcement, setAnnouncement] = useState(message);

  useEffect(() => {
    setAnnouncement(message);
    
    if (clearAfter > 0) {
      const timer = setTimeout(() => {
        setAnnouncement('');
      }, clearAfter);
      
      return () => clearTimeout(timer);
    }
  }, [message, clearAfter]);

  return (
    <LiveRegion 
      politeness={priority}
      className="sr-only"
    >
      {announcement}
    </LiveRegion>
  );
};

// Progress announcer for loading states
interface ProgressAnnouncerProps {
  value: number;
  max: number;
  label?: string;
  announceEvery?: number; // Announce every N percent
}

export const ProgressAnnouncer = ({
  value,
  max,
  label = 'Progress',
  announceEvery = 10
}: ProgressAnnouncerProps) => {
  const [lastAnnounced, setLastAnnounced] = useState(0);
  const [announcement, setAnnouncement] = useState('');

  const percentage = Math.round((value / max) * 100);

  useEffect(() => {
    const shouldAnnounce = 
      percentage >= lastAnnounced + announceEvery || 
      percentage === 100 || 
      percentage === 0;

    if (shouldAnnounce) {
      setAnnouncement(`${label}: ${percentage}% complete`);
      setLastAnnounced(percentage);
      
      // Clear announcement after 2 seconds
      const timer = setTimeout(() => setAnnouncement(''), 2000);
      return () => clearTimeout(timer);
    }
  }, [value, max, percentage, lastAnnounced, announceEvery, label]);

  return (
    <LiveRegion politeness="polite" className="sr-only">
      {announcement}
    </LiveRegion>
  );
};

// Skip link component for keyboard navigation
interface SkipLinkProps {
  href: string;
  children: React.ReactNode;
  className?: string;
}

export const SkipLink = ({ href, children, className = '' }: SkipLinkProps) => {
  return (
    <a
      href={href}
      className={`
        sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 
        focus:z-50 focus:px-4 focus:py-2 focus:bg-theme-primary 
        focus:text-theme-text-inverse focus:rounded-md focus:shadow-lg
        focus:outline-none focus:ring-2 focus:ring-theme-border-focus
        ${className}
      `}
      style={{
        position: 'absolute',
        left: '-10000px',
        top: 'auto',
        width: '1px',
        height: '1px',
        overflow: 'hidden'
      }}
      onFocus={(e) => {
        e.currentTarget.style.position = 'absolute';
        e.currentTarget.style.left = '1rem';
        e.currentTarget.style.top = '1rem';
        e.currentTarget.style.width = 'auto';
        e.currentTarget.style.height = 'auto';
        e.currentTarget.style.overflow = 'visible';
      }}
      onBlur={(e) => {
        e.currentTarget.style.position = 'absolute';
        e.currentTarget.style.left = '-10000px';
        e.currentTarget.style.top = 'auto';
        e.currentTarget.style.width = '1px';
        e.currentTarget.style.height = '1px';
        e.currentTarget.style.overflow = 'hidden';
      }}
    >
      {children}
    </a>
  );
};

// Skip navigation component
export const SkipNavigation = () => {
  return (
    <>
      <SkipLink href="#main-content">Skip to main content</SkipLink>
      <SkipLink href="#navigation">Skip to navigation</SkipLink>
      <SkipLink href="#footer">Skip to footer</SkipLink>
    </>
  );
};

// Hook for managing announcements
export const useAnnouncements = () => {
  const [announcements, setAnnouncements] = useState<string[]>([]);

  const announce = (message: string, _priority: 'polite' | 'assertive' = 'polite') => {
    setAnnouncements(prev => [...prev, message]);
    
    // Clear announcement after 5 seconds
    setTimeout(() => {
      setAnnouncements(prev => prev.filter(msg => msg !== message));
    }, 5000);
  };

  const clear = () => {
    setAnnouncements([]);
  };

  return {
    announcements,
    announce,
    clear
  };
};

// Global announcements provider
interface AnnouncementsProviderProps {
  children: React.ReactNode;
}

export const AnnouncementsProvider = ({ children }: AnnouncementsProviderProps) => {
  const { announcements } = useAnnouncements();

  return (
    <>
      {children}
      <div aria-live="polite" aria-atomic="false" className="sr-only">
        {announcements.map((announcement, index) => (
          <div key={index}>{announcement}</div>
        ))}
      </div>
    </>
  );
};

// Custom hook for managing focus announcements
export const useFocusAnnouncement = () => {
  const announce = (element: HTMLElement) => {
    const label = 
      element.getAttribute('aria-label') ||
      element.getAttribute('aria-labelledby') ||
      element.textContent ||
      element.getAttribute('title') ||
      'Interactive element';

    const role = element.getAttribute('role') || element.tagName.toLowerCase();
    
    return `${label}, ${role}`;
  };

  const announceNavigation = (direction: 'next' | 'previous', total: number, current: number) => {
    return `${direction === 'next' ? 'Next' : 'Previous'} item. ${current} of ${total}`;
  };

  return {
    announce,
    announceNavigation
  };
};

export default ScreenReaderOnly;