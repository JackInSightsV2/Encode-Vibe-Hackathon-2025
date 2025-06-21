import React from 'react';
import { cn } from '../../utils/cn';
import { useIsTouchDevice } from '../../hooks/useMediaQuery';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helper?: string;
  icon?: React.ReactNode;
  variant?: 'default' | 'filled';
  className?: string;
}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  (
    {
      label,
      error,
      helper,
      icon,
      variant = 'default',
      className,
      id,
      ...props
    },
    ref
  ) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
    const isTouchDevice = useIsTouchDevice();

    const inputClasses = cn(
      'w-full rounded-lg border transition-colors',
      isTouchDevice ? 'px-4 py-3 text-base min-h-[44px]' : 'px-3 py-2 text-sm',
      'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent',
      'disabled:opacity-50 disabled:cursor-not-allowed',
      error
        ? 'border-red-300 bg-red-50 text-red-900 placeholder-red-300 focus:ring-red-500'
        : variant === 'filled'
        ? 'border-gray-200 bg-gray-50 focus:bg-white'
        : 'border-gray-300 bg-white',
      icon && (isTouchDevice ? 'pl-12' : 'pl-10'),
      className
    );

    return (
      <div className="space-y-1">
        {label && (
          <label 
            htmlFor={inputId}
            className="block text-sm font-medium text-gray-700"
          >
            {label}
          </label>
        )}
        
        <div className="relative">
          {icon && (
            <div className={cn(
              'absolute inset-y-0 left-0 flex items-center pointer-events-none',
              isTouchDevice ? 'pl-4' : 'pl-3'
            )}>
              <span className={cn(
                'text-gray-400',
                isTouchDevice ? 'w-5 h-5' : 'w-4 h-4'
              )}>
                {icon}
              </span>
            </div>
          )}
          
          <input
            ref={ref}
            id={inputId}
            className={inputClasses}
            aria-invalid={error ? 'true' : 'false'}
            aria-describedby={
              error
                ? `${inputId}-error`
                : helper
                ? `${inputId}-helper`
                : undefined
            }
            {...props}
          />
        </div>
        
        {error && (
          <p
            id={`${inputId}-error`}
            className="text-sm text-red-600"
            role="alert"
          >
            {error}
          </p>
        )}
        
        {helper && !error && (
          <p
            id={`${inputId}-helper`}
            className="text-sm text-gray-500"
          >
            {helper}
          </p>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';

export { Input };
export type { InputProps };