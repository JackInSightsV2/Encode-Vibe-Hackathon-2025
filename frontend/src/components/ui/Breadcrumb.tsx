import React from 'react';
import { cn } from '../../utils/cn';
import { useIsMobile } from '../../hooks/useMediaQuery';

interface BreadcrumbItem {
  label: string;
  href?: string;
  onClick?: () => void;
  current?: boolean;
}

interface BreadcrumbProps {
  items: BreadcrumbItem[];
  className?: string;
  maxItems?: number;
  separator?: React.ReactNode;
}

const Breadcrumb: React.FC<BreadcrumbProps> = ({
  items,
  className,
  maxItems,
  separator,
}) => {
  const isMobile = useIsMobile();

  // On mobile, show only the last item if no maxItems specified
  const displayItems = isMobile && !maxItems 
    ? items.slice(-1) 
    : maxItems 
    ? items.slice(-maxItems) 
    : items;

  const defaultSeparator = (
    <svg
      className="flex-shrink-0 w-4 h-4 text-gray-400"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M9 5l7 7-7 7"
      />
    </svg>
  );

  return (
    <nav aria-label="Breadcrumb" className={cn('flex', className)}>
      <ol className="flex items-center space-x-1 sm:space-x-2">
        {/* Show ellipsis if items were truncated */}
        {maxItems && items.length > maxItems && (
          <>
            <li>
              <span className="text-gray-500 text-sm">...</span>
            </li>
            <li>
              <span className="text-gray-400">
                {separator || defaultSeparator}
              </span>
            </li>
          </>
        )}

        {displayItems.map((item, index) => (
          <li key={index} className="flex items-center">
            {index > 0 && (
              <span className="mx-1 sm:mx-2 text-gray-400">
                {separator || defaultSeparator}
              </span>
            )}
            
            {item.current ? (
              <span 
                className="text-sm font-medium text-gray-900 truncate max-w-[150px] sm:max-w-none"
                aria-current="page"
              >
                {item.label}
              </span>
            ) : item.href ? (
              <a
                href={item.href}
                className="text-sm text-gray-500 hover:text-gray-700 transition-colors truncate max-w-[100px] sm:max-w-[150px] md:max-w-none"
                onClick={(e) => {
                  if (item.onClick) {
                    e.preventDefault();
                    item.onClick();
                  }
                }}
              >
                {item.label}
              </a>
            ) : (
              <button
                onClick={item.onClick}
                className="text-sm text-gray-500 hover:text-gray-700 transition-colors truncate max-w-[100px] sm:max-w-[150px] md:max-w-none text-left"
              >
                {item.label}
              </button>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
};

// Mobile-optimized breadcrumb that shows a back button instead
const MobileBreadcrumb: React.FC<{
  currentPage: string;
  onBack?: () => void;
  className?: string;
}> = ({ currentPage, onBack, className }) => {
  return (
    <nav aria-label="Breadcrumb" className={cn('flex items-center space-x-2', className)}>
      {onBack && (
        <button
          onClick={onBack}
          className="p-2 text-gray-500 hover:text-gray-700 transition-colors -ml-2"
          aria-label="Go back"
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
              d="M15 19l-7-7 7-7"
            />
          </svg>
        </button>
      )}
      <h1 className="text-lg font-semibold text-gray-900 truncate">
        {currentPage}
      </h1>
    </nav>
  );
};

// Responsive breadcrumb that automatically switches between desktop and mobile views
const ResponsiveBreadcrumb: React.FC<BreadcrumbProps & {
  mobileTitle?: string;
  onBack?: () => void;
}> = ({ items, mobileTitle, onBack, ...props }) => {
  const isMobile = useIsMobile();
  
  if (isMobile) {
    const currentItem = items.find(item => item.current);
    const title = mobileTitle || currentItem?.label || items[items.length - 1]?.label;
    
    return (
      <MobileBreadcrumb
        currentPage={title}
        onBack={onBack}
        className={props.className}
      />
    );
  }
  
  return <Breadcrumb items={items} {...props} />;
};

export { Breadcrumb, MobileBreadcrumb, ResponsiveBreadcrumb };
export type { BreadcrumbProps, BreadcrumbItem };