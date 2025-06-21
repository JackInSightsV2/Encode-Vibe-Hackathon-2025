import React from 'react';
import { cn } from '../../utils/cn';

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'secondary' | 'success' | 'warning' | 'danger' | 'outline';
  size?: 'sm' | 'md' | 'lg';
  children: React.ReactNode;
}

const Badge: React.FC<BadgeProps> = ({
  variant = 'default',
  size = 'md',
  className,
  children,
  ...props
}) => {
  const baseClasses = 
    'inline-flex items-center font-medium rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2';

  const variantClasses = {
    default: 
      'bg-primary-100 text-primary-800 border border-primary-200',
    secondary: 
      'bg-gray-100 text-gray-800 border border-gray-200',
    success: 
      'bg-green-100 text-green-800 border border-green-200',
    warning: 
      'bg-yellow-100 text-yellow-800 border border-yellow-200',
    danger: 
      'bg-red-100 text-red-800 border border-red-200',
    outline: 
      'border border-gray-300 text-gray-700 bg-transparent',
  };

  const sizeClasses = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-0.5 text-sm',
    lg: 'px-3 py-1 text-sm',
  };

  return (
    <span
      className={cn(
        baseClasses,
        variantClasses[variant],
        sizeClasses[size],
        className
      )}
      {...props}
    >
      {children}
    </span>
  );
};

export { Badge };
export type { BadgeProps };