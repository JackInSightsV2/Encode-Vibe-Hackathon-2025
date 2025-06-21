import React from 'react';
import { cn } from '../../utils/cn';
import { AlertTriangle, CheckCircle, Info, XCircle } from 'lucide-react';

interface AlertProps {
  children: React.ReactNode;
  variant?: 'default' | 'success' | 'warning' | 'error' | 'info';
  className?: string;
}

interface AlertDescriptionProps {
  children: React.ReactNode;
  className?: string;
}

const alertVariants = {
  default: {
    container: 'border-gray-200 bg-gray-50 text-gray-900',
    icon: Info,
    iconColor: 'text-gray-600'
  },
  success: {
    container: 'border-green-200 bg-green-50 text-green-900',
    icon: CheckCircle,
    iconColor: 'text-green-600'
  },
  warning: {
    container: 'border-yellow-200 bg-yellow-50 text-yellow-900',
    icon: AlertTriangle,
    iconColor: 'text-yellow-600'
  },
  error: {
    container: 'border-red-200 bg-red-50 text-red-900',
    icon: XCircle,
    iconColor: 'text-red-600'
  },
  info: {
    container: 'border-blue-200 bg-blue-50 text-blue-900',
    icon: Info,
    iconColor: 'text-blue-600'
  }
};

export const Alert: React.FC<AlertProps> = ({
  children,
  variant = 'default',
  className
}) => {
  const { container, icon: Icon, iconColor } = alertVariants[variant];

  return (
    <div
      className={cn(
        'relative w-full rounded-lg border p-4',
        container,
        className
      )}
    >
      <div className="flex items-start space-x-3">
        <Icon className={cn('w-5 h-5 mt-0.5 flex-shrink-0', iconColor)} />
        <div className="flex-1 min-w-0">
          {children}
        </div>
      </div>
    </div>
  );
};

export const AlertDescription: React.FC<AlertDescriptionProps> = ({
  children,
  className
}) => {
  return (
    <div className={cn('text-sm leading-relaxed', className)}>
      {children}
    </div>
  );
};