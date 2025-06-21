import React from 'react';
import { cn } from '../../utils/cn';
import { useStaggeredAnimation } from '../../hooks/useAnimations';

interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular' | 'rounded';
  width?: string | number;
  height?: string | number;
  animation?: 'pulse' | 'wave' | 'none';
  delay?: number;
}

export const Skeleton = ({
  className,
  variant = 'text',
  width,
  height,
  animation = 'pulse',
  delay = 0
}: SkeletonProps) => {
  const baseClasses = cn(
    'bg-theme-surface-alt',
    {
      'animate-pulse': animation === 'pulse',
      'animate-wave': animation === 'wave',
      'rounded-full': variant === 'circular',
      'rounded-lg': variant === 'rounded',
      'rounded': variant === 'rectangular',
      'h-4 rounded': variant === 'text'
    },
    className
  );

  const style: React.CSSProperties = {
    width: width || (variant === 'text' ? '100%' : '40px'),
    height: height || (variant === 'text' ? '1rem' : '40px'),
    animationDelay: delay ? `${delay}ms` : undefined
  };

  return <div className={baseClasses} style={style} />;
};

// Text skeleton with multiple lines
interface TextSkeletonProps {
  lines?: number;
  className?: string;
  lastLineWidth?: string;
  spacing?: 'sm' | 'md' | 'lg';
}

export const TextSkeleton = ({
  lines = 3,
  className,
  lastLineWidth = '75%',
  spacing = 'md'
}: TextSkeletonProps) => {
  const spacingClasses = {
    sm: 'space-y-1',
    md: 'space-y-2',
    lg: 'space-y-3'
  };

  return (
    <div className={cn(spacingClasses[spacing], className)}>
      {Array.from({ length: lines }).map((_, index) => (
        <Skeleton
          key={index}
          variant="text"
          width={index === lines - 1 ? lastLineWidth : '100%'}
          delay={index * 50}
        />
      ))}
    </div>
  );
};

// Avatar skeleton
interface AvatarSkeletonProps {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
}

export const AvatarSkeleton = ({ size = 'md', className }: AvatarSkeletonProps) => {
  const sizeClasses = {
    sm: 'w-8 h-8',
    md: 'w-12 h-12',
    lg: 'w-16 h-16',
    xl: 'w-24 h-24'
  };

  return (
    <Skeleton
      variant="circular"
      className={cn(sizeClasses[size], className)}
    />
  );
};

// Card skeleton
interface CardSkeletonProps {
  hasImage?: boolean;
  hasAvatar?: boolean;
  titleLines?: number;
  contentLines?: number;
  hasActions?: boolean;
  className?: string;
}

export const CardSkeleton = ({
  hasImage = false,
  hasAvatar = false,
  titleLines = 1,
  contentLines = 3,
  hasActions = false,
  className
}: CardSkeletonProps) => {
  return (
    <div className={cn('p-4 border border-theme-border-default rounded-lg', className)}>
      {hasImage && (
        <Skeleton variant="rectangular" className="w-full h-48 mb-4" />
      )}
      
      <div className="space-y-4">
        {hasAvatar && (
          <div className="flex items-center space-x-3">
            <AvatarSkeleton size="md" />
            <div className="flex-1">
              <Skeleton variant="text" width="60%" />
              <Skeleton variant="text" width="40%" className="mt-1" />
            </div>
          </div>
        )}
        
        <div className="space-y-2">
          <TextSkeleton lines={titleLines} spacing="sm" />
          <TextSkeleton lines={contentLines} lastLineWidth="80%" />
        </div>
        
        {hasActions && (
          <div className="flex space-x-2 pt-2">
            <Skeleton variant="rounded" width="80px" height="32px" />
            <Skeleton variant="rounded" width="80px" height="32px" />
          </div>
        )}
      </div>
    </div>
  );
};

// Table skeleton
interface TableSkeletonProps {
  rows?: number;
  columns?: number;
  hasHeader?: boolean;
  className?: string;
}

export const TableSkeleton = ({
  rows = 5,
  columns = 4,
  hasHeader = true,
  className
}: TableSkeletonProps) => {
  const animatedRows = useStaggeredAnimation(rows + (hasHeader ? 1 : 0), 50);

  return (
    <div className={cn('border border-theme-border-default rounded-lg overflow-hidden', className)}>
      <div className="divide-y divide-theme-border-light">
        {hasHeader && (
          <div className={cn(
            'grid gap-4 p-4 bg-theme-surface-alt',
            `grid-cols-${columns}`,
            'transition-opacity duration-200',
            animatedRows[0] ? 'opacity-100' : 'opacity-0'
          )}>
            {Array.from({ length: columns }).map((_, index) => (
              <Skeleton key={index} variant="text" width="80%" delay={index * 25} />
            ))}
          </div>
        )}
        
        {Array.from({ length: rows }).map((_, rowIndex) => (
          <div
            key={rowIndex}
            className={cn(
              'grid gap-4 p-4',
              `grid-cols-${columns}`,
              'transition-opacity duration-200',
              animatedRows[rowIndex + (hasHeader ? 1 : 0)] ? 'opacity-100' : 'opacity-0'
            )}
          >
            {Array.from({ length: columns }).map((_, colIndex) => (
              <Skeleton
                key={colIndex}
                variant="text"
                width={colIndex === 0 ? '90%' : '70%'}
                delay={colIndex * 25}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
};

// List skeleton
interface ListSkeletonProps {
  items?: number;
  hasImage?: boolean;
  hasSecondaryText?: boolean;
  className?: string;
}

export const ListSkeleton = ({
  items = 5,
  hasImage = false,
  hasSecondaryText = true,
  className
}: ListSkeletonProps) => {
  const animatedItems = useStaggeredAnimation(items, 75);

  return (
    <div className={cn('space-y-3', className)}>
      {Array.from({ length: items }).map((_, index) => (
        <div
          key={index}
          className={cn(
            'flex items-center space-x-3 p-3 border border-theme-border-default rounded-lg',
            'transition-opacity duration-200',
            animatedItems[index] ? 'opacity-100' : 'opacity-0'
          )}
        >
          {hasImage && <Skeleton variant="circular" width="40px" height="40px" />}
          
          <div className="flex-1 space-y-1">
            <Skeleton variant="text" width="70%" />
            {hasSecondaryText && (
              <Skeleton variant="text" width="50%" height="12px" />
            )}
          </div>
          
          <Skeleton variant="text" width="60px" height="20px" />
        </div>
      ))}
    </div>
  );
};

// Page skeleton layout
interface PageSkeletonProps {
  hasHeader?: boolean;
  hasSidebar?: boolean;
  contentType?: 'cards' | 'table' | 'list';
  className?: string;
}

export const PageSkeleton = ({
  hasHeader = true,
  hasSidebar = false,
  contentType = 'cards',
  className
}: PageSkeletonProps) => {
  return (
    <div className={cn('min-h-screen bg-theme-background', className)}>
      {hasHeader && (
        <div className="border-b border-theme-border-default p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Skeleton variant="circular" width="40px" height="40px" />
              <div>
                <Skeleton variant="text" width="200px" height="20px" />
                <Skeleton variant="text" width="150px" height="14px" className="mt-1" />
              </div>
            </div>
            <div className="flex space-x-2">
              <Skeleton variant="rounded" width="80px" height="36px" />
              <Skeleton variant="rounded" width="100px" height="36px" />
            </div>
          </div>
        </div>
      )}
      
      <div className="flex">
        {hasSidebar && (
          <div className="w-64 border-r border-theme-border-default p-4">
            <div className="space-y-3">
              {Array.from({ length: 6 }).map((_, index) => (
                <Skeleton key={index} variant="text" width="80%" delay={index * 50} />
              ))}
            </div>
          </div>
        )}
        
        <div className="flex-1 p-6">
          {contentType === 'cards' && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {Array.from({ length: 6 }).map((_, index) => (
                <CardSkeleton key={index} hasImage hasAvatar hasActions />
              ))}
            </div>
          )}
          
          {contentType === 'table' && (
            <TableSkeleton rows={8} columns={5} hasHeader />
          )}
          
          {contentType === 'list' && (
            <ListSkeleton items={10} hasImage hasSecondaryText />
          )}
        </div>
      </div>
    </div>
  );
};

// Shimmer effect for wave animation
const shimmerStyles = `
  @keyframes wave {
    0% {
      transform: translateX(-100%);
    }
    50% {
      transform: translateX(100%);
    }
    100% {
      transform: translateX(100%);
    }
  }
  
  .animate-wave {
    position: relative;
    overflow: hidden;
  }
  
  .animate-wave::after {
    content: '';
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    left: 0;
    transform: translateX(-100%);
    background: linear-gradient(
      90deg,
      transparent,
      rgba(255, 255, 255, 0.2),
      transparent
    );
    animation: wave 2s infinite;
  }
`;

// Inject shimmer styles
if (typeof document !== 'undefined') {
  const styleSheet = document.createElement('style');
  styleSheet.textContent = shimmerStyles;
  document.head.appendChild(styleSheet);
}

export default Skeleton;