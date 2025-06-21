import React from 'react';
import { useIntersectionAnimation, useCountAnimation } from '../../hooks/useAnimations';
import { cn } from '../../utils/cn';

interface AnimatedCounterProps {
  value: number;
  duration?: number;
  formatter?: (value: number) => string;
  className?: string;
  prefix?: string;
  suffix?: string;
  decimals?: number;
  separator?: string;
  trigger?: boolean;
  easing?: 'linear' | 'easeOutQuart' | 'easeOutCubic' | 'easeOutCirc';
  'aria-label'?: string;
  'aria-describedby'?: string;
}

export const AnimatedCounter = ({
  value,
  duration = 1000,
  formatter,
  className,
  prefix = '',
  suffix = '',
  decimals = 0,
  separator = ',',
  trigger: externalTrigger,
  easing = 'easeOutQuart',
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedby
}: AnimatedCounterProps) => {
  const { ref, isVisible } = useIntersectionAnimation({ threshold: 0.1 });
  const triggerAnimation = externalTrigger !== undefined ? externalTrigger : isVisible;
  
  const animatedValue = useCountAnimation(value, duration, triggerAnimation, easing);

  const formatNumber = (num: number): string => {
    if (formatter) {
      return formatter(num);
    }

    // Apply decimals
    const rounded = Number(num.toFixed(decimals));
    
    // Convert to string and add separators
    const parts = rounded.toString().split('.');
    parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, separator);
    
    return parts.join('.');
  };

  const displayValue = `${prefix}${formatNumber(animatedValue)}${suffix}`;
  const finalValue = `${prefix}${formatNumber(value)}${suffix}`;

  return (
    <span
      ref={ref as React.RefObject<HTMLSpanElement>}
      className={cn(
        'tabular-nums transition-all duration-200',
        'font-mono text-theme-text-primary',
        className
      )}
      aria-live="polite"
      aria-label={ariaLabel || `Count: ${finalValue}`}
      aria-describedby={ariaDescribedby}
      role="status"
    >
      {displayValue}
    </span>
  );
};

// Preset counter components for common use cases
export const MetricCounter = ({ 
  value, 
  label, 
  icon,
  trend,
  ...props 
}: AnimatedCounterProps & { 
  label: string; 
  icon?: React.ReactNode;
  trend?: 'up' | 'down' | 'stable';
}) => {
  const getTrendColor = () => {
    switch (trend) {
      case 'up': return 'text-theme-status-success';
      case 'down': return 'text-theme-status-error';
      default: return 'text-theme-text-secondary';
    }
  };

  const getTrendIcon = () => {
    switch (trend) {
      case 'up':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 17l9.2-9.2M17 17V7h-10" />
          </svg>
        );
      case 'down':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 7l-9.2 9.2M7 7v10h10" />
          </svg>
        );
      default:
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
          </svg>
        );
    }
  };

  return (
    <div className="bg-theme-surface p-4 rounded-lg border border-theme-border-default">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-theme-text-secondary">{label}</span>
        {icon && <span className="text-theme-text-tertiary">{icon}</span>}
      </div>
      
      <div className="flex items-end justify-between">
        <AnimatedCounter
          value={value}
          className="text-2xl font-bold text-theme-text-primary"
          {...props}
        />
        
        {trend && (
          <div className={cn('flex items-center space-x-1 text-sm', getTrendColor())}>
            {getTrendIcon()}
          </div>
        )}
      </div>
    </div>
  );
};

// Progress counter component
export const ProgressCounter = ({
  current,
  total,
  label,
  showPercentage = true,
  ...props
}: Omit<AnimatedCounterProps, 'value'> & {
  current: number;
  total: number;
  label?: string;
  showPercentage?: boolean;
}) => {
  const percentage = Math.round((current / total) * 100);

  return (
    <div className="space-y-2">
      {label && (
        <div className="flex justify-between items-center">
          <span className="text-sm font-medium text-theme-text-secondary">{label}</span>
          {showPercentage && (
            <AnimatedCounter
              value={percentage}
              suffix="%"
              className="text-sm font-semibold text-theme-text-primary"
              {...props}
            />
          )}
        </div>
      )}
      
      <div className="flex items-center space-x-2">
        <AnimatedCounter
          value={current}
          className="text-lg font-bold text-theme-text-primary"
          {...props}
        />
        <span className="text-theme-text-tertiary">/</span>
        <span className="text-lg font-bold text-theme-text-secondary">{total}</span>
      </div>
      
      {/* Progress bar */}
      <div className="w-full bg-theme-surface-alt rounded-full h-2">
        <div
          className="bg-theme-primary h-2 rounded-full transition-all duration-1000 ease-out"
          style={{ width: `${Math.min(percentage, 100)}%` }}
          role="progressbar"
          aria-valuenow={percentage}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={`Progress: ${percentage}%`}
        />
      </div>
    </div>
  );
};

export default AnimatedCounter;