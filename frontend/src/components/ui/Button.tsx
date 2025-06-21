import React from 'react';
import { cn } from '../../utils/cn';
import { useIsTouchDevice } from '../../hooks/useMediaQuery';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  icon?: React.ReactNode;
  children: React.ReactNode;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = 'primary',
      size = 'md',
      loading = false,
      disabled = false,
      icon,
      children,
      className,
      ...props
    },
    ref
  ) => {
    const isTouchDevice = useIsTouchDevice();
    const baseClasses = 
      'inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed';

    const variantClasses = {
      primary: 
        'bg-primary-600 text-white hover:bg-primary-700 focus:ring-primary-500 active:bg-primary-800',
      secondary: 
        'bg-gray-600 text-white hover:bg-gray-700 focus:ring-gray-500 active:bg-gray-800',
      outline: 
        'border-2 border-primary-600 text-primary-600 bg-transparent hover:bg-primary-50 focus:ring-primary-500 active:bg-primary-100',
      ghost: 
        'text-gray-600 bg-transparent hover:bg-gray-100 focus:ring-gray-500 active:bg-gray-200',
      danger: 
        'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 active:bg-red-800',
    };

    const sizeClasses = {
      sm: isTouchDevice ? 'px-4 py-2.5 text-sm gap-1.5 min-h-[44px]' : 'px-3 py-1.5 text-sm gap-1.5',
      md: isTouchDevice ? 'px-6 py-3 text-base gap-2 min-h-[44px]' : 'px-4 py-2 text-base gap-2',
      lg: isTouchDevice ? 'px-8 py-4 text-lg gap-2.5 min-h-[48px]' : 'px-6 py-3 text-lg gap-2.5',
    };

    const classes = cn(
      baseClasses,
      variantClasses[variant],
      sizeClasses[size],
      className
    );

    return (
      <button
        ref={ref}
        className={classes}
        disabled={disabled || loading}
        {...props}
      >
        {loading && (
          <div className="animate-spin">
            <svg
              className={cn(
                'animate-spin',
                size === 'sm' ? 'w-4 h-4' : size === 'lg' ? 'w-5 h-5' : 'w-4 h-4'
              )}
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              />
            </svg>
          </div>
        )}
        {icon && !loading && (
          <span className={cn(size === 'sm' ? 'w-4 h-4' : size === 'lg' ? 'w-5 h-5' : 'w-4 h-4')}>
            {icon}
          </span>
        )}
        {children}
      </button>
    );
  }
);

Button.displayName = 'Button';

export { Button };
export type { ButtonProps };