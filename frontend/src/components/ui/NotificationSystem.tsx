import { useState, useEffect, useRef, createContext, useContext, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { cn } from '../../utils/cn';

export type NotificationType = 'success' | 'error' | 'warning' | 'info';

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message?: string;
  duration?: number;
  persistent?: boolean;
  actions?: NotificationAction[];
  onDismiss?: () => void;
  icon?: ReactNode;
}

interface NotificationAction {
  label: string;
  onClick: () => void;
  variant?: 'primary' | 'secondary';
}

interface NotificationContextType {
  notifications: Notification[];
  addNotification: (notification: Omit<Notification, 'id'>) => string;
  removeNotification: (id: string) => void;
  clearAll: () => void;
}

const NotificationContext = createContext<NotificationContextType | null>(null);

export const useNotifications = () => {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error('useNotifications must be used within NotificationProvider');
  }
  return context;
};

interface NotificationProviderProps {
  children: ReactNode;
  maxNotifications?: number;
  position?: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left' | 'top-center' | 'bottom-center';
  className?: string;
}

export const NotificationProvider = ({
  children,
  maxNotifications = 5,
  position = 'top-right',
  className
}: NotificationProviderProps) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const timeoutRefs = useRef<Map<string, NodeJS.Timeout>>(new Map());

  const addNotification = (notification: Omit<Notification, 'id'>) => {
    const id = Math.random().toString(36).substr(2, 9);
    const newNotification: Notification = {
      ...notification,
      id,
      duration: notification.duration ?? (notification.persistent ? undefined : 5000)
    };

    setNotifications(prev => {
      const updated = [newNotification, ...prev];
      if (updated.length > maxNotifications) {
        // Remove oldest notifications
        const removed = updated.slice(maxNotifications);
        removed.forEach(notif => {
          const timeout = timeoutRefs.current.get(notif.id);
          if (timeout) {
            clearTimeout(timeout);
            timeoutRefs.current.delete(notif.id);
          }
        });
        return updated.slice(0, maxNotifications);
      }
      return updated;
    });

    // Auto-dismiss after duration
    if (newNotification.duration && !newNotification.persistent) {
      const timeout = setTimeout(() => {
        removeNotification(id);
      }, newNotification.duration);
      timeoutRefs.current.set(id, timeout);
    }

    return id;
  };

  const removeNotification = (id: string) => {
    setNotifications(prev => prev.filter(notif => notif.id !== id));
    
    const timeout = timeoutRefs.current.get(id);
    if (timeout) {
      clearTimeout(timeout);
      timeoutRefs.current.delete(id);
    }

    const notification = notifications.find(n => n.id === id);
    notification?.onDismiss?.();
  };

  const clearAll = () => {
    notifications.forEach(notif => {
      const timeout = timeoutRefs.current.get(notif.id);
      if (timeout) {
        clearTimeout(timeout);
      }
    });
    timeoutRefs.current.clear();
    setNotifications([]);
  };

  useEffect(() => {
    return () => {
      // Cleanup timeouts on unmount
      timeoutRefs.current.forEach(timeout => clearTimeout(timeout));
      timeoutRefs.current.clear();
    };
  }, []);

  const positionClasses = {
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4',
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'top-center': 'top-4 left-1/2 -translate-x-1/2',
    'bottom-center': 'bottom-4 left-1/2 -translate-x-1/2'
  };

  return (
    <NotificationContext.Provider value={{ notifications, addNotification, removeNotification, clearAll }}>
      {children}
      {typeof document !== 'undefined' && createPortal(
        <div className={cn('fixed z-50 pointer-events-none', positionClasses[position], className)}>
          <div className="space-y-2 w-80">
            {notifications.map((notification) => (
              <NotificationItem
                key={notification.id}
                notification={notification}
                onRemove={() => removeNotification(notification.id)}
              />
            ))}
          </div>
        </div>,
        document.body
      )}
    </NotificationContext.Provider>
  );
};

interface NotificationItemProps {
  notification: Notification;
  onRemove: () => void;
}

const NotificationItem = ({ notification, onRemove }: NotificationItemProps) => {
  const [isVisible, setIsVisible] = useState(false);
  const [isExiting, setIsExiting] = useState(false);

  useEffect(() => {
    // Animate in
    requestAnimationFrame(() => {
      setIsVisible(true);
    });
  }, []);

  const handleDismiss = () => {
    setIsExiting(true);
    setTimeout(() => {
      onRemove();
    }, 300); // Match animation duration
  };

  const typeStyles = {
    success: 'border-green-200 bg-green-50 text-green-800',
    error: 'border-red-200 bg-red-50 text-red-800',
    warning: 'border-yellow-200 bg-yellow-50 text-yellow-800',
    info: 'border-blue-200 bg-blue-50 text-blue-800'
  };

  const typeIcons = {
    success: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
      </svg>
    ),
    error: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
      </svg>
    ),
    warning: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
      </svg>
    ),
    info: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
      </svg>
    )
  };

  return (
    <div
      className={cn(
        'pointer-events-auto relative w-full border-l-4 p-4 shadow-lg rounded-r-lg',
        'transform transition-all duration-300 ease-in-out',
        typeStyles[notification.type],
        {
          'translate-x-full opacity-0': !isVisible || isExiting,
          'translate-x-0 opacity-100': isVisible && !isExiting
        }
      )}
    >
      <div className="flex items-start">
        <div className="flex-shrink-0">
          {notification.icon || typeIcons[notification.type]}
        </div>
        
        <div className="ml-3 flex-1">
          <h4 className="text-sm font-semibold">
            {notification.title}
          </h4>
          {notification.message && (
            <p className="text-sm mt-1 opacity-90">
              {notification.message}
            </p>
          )}
          
          {notification.actions && notification.actions.length > 0 && (
            <div className="mt-3 flex space-x-2">
              {notification.actions.map((action, index) => (
                <button
                  key={index}
                  onClick={action.onClick}
                  className={cn(
                    'text-xs px-2 py-1 rounded font-medium transition-colors',
                    action.variant === 'primary'
                      ? 'bg-current text-white opacity-90 hover:opacity-100'
                      : 'bg-white bg-opacity-20 hover:bg-opacity-30'
                  )}
                >
                  {action.label}
                </button>
              ))}
            </div>
          )}
        </div>
        
        {!notification.persistent && (
          <button
            onClick={handleDismiss}
            className="flex-shrink-0 ml-4 text-current opacity-60 hover:opacity-80 transition-opacity"
            aria-label="Dismiss notification"
          >
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>
        )}
      </div>
    </div>
  );
};

// Convenience hooks for different notification types
export const useToast = () => {
  const { addNotification } = useNotifications();

  return {
    success: (title: string, message?: string, options?: Partial<Notification>) =>
      addNotification({ type: 'success', title, message, ...options }),
    
    error: (title: string, message?: string, options?: Partial<Notification>) =>
      addNotification({ type: 'error', title, message, ...options }),
    
    warning: (title: string, message?: string, options?: Partial<Notification>) =>
      addNotification({ type: 'warning', title, message, ...options }),
    
    info: (title: string, message?: string, options?: Partial<Notification>) =>
      addNotification({ type: 'info', title, message, ...options }),
    
    custom: (notification: Omit<Notification, 'id'>) =>
      addNotification(notification)
  };
};

// Progress notification component
interface ProgressNotificationProps {
  title: string;
  progress: number;
  message?: string;
  onCancel?: () => void;
}

export const ProgressNotification = ({ title, progress, message, onCancel }: ProgressNotificationProps) => {
  const { addNotification, removeNotification } = useNotifications();
  const notificationIdRef = useRef<string | null>(null);

  useEffect(() => {
    if (notificationIdRef.current) {
      removeNotification(notificationIdRef.current);
    }

    const actions: NotificationAction[] = [];
    if (onCancel) {
      actions.push({
        label: 'Cancel',
        onClick: onCancel,
        variant: 'secondary'
      });
    }

    notificationIdRef.current = addNotification({
      type: 'info',
      title,
      message: message || `${Math.round(progress)}% complete`,
      persistent: true,
      actions,
      icon: (
        <div className="relative w-5 h-5">
          <svg className="w-5 h-5 transform -rotate-90" viewBox="0 0 20 20">
            <circle
              cx="10"
              cy="10"
              r="8"
              stroke="currentColor"
              strokeWidth="2"
              fill="none"
              className="opacity-20"
            />
            <circle
              cx="10"
              cy="10"
              r="8"
              stroke="currentColor"
              strokeWidth="2"
              fill="none"
              strokeDasharray={`${progress * 0.5} ${50 - progress * 0.5}`}
              className="transition-all duration-300"
            />
          </svg>
        </div>
      )
    });

    // Cleanup on unmount or when progress reaches 100%
    return () => {
      if (notificationIdRef.current && progress >= 100) {
        setTimeout(() => {
          if (notificationIdRef.current) {
            removeNotification(notificationIdRef.current);
          }
        }, 1000);
      }
    };
  }, [progress, title, message, onCancel, addNotification, removeNotification]);

  return null;
};

// Notification queue for batch operations
export class NotificationQueue {
  private notifications: Array<Omit<Notification, 'id'>> = [];
  private addNotification: (notification: Omit<Notification, 'id'>) => string;

  constructor(addNotification: (notification: Omit<Notification, 'id'>) => string) {
    this.addNotification = addNotification;
  }

  add(notification: Omit<Notification, 'id'>) {
    this.notifications.push(notification);
    return this;
  }

  flush(delay = 0) {
    this.notifications.forEach((notification, index) => {
      setTimeout(() => {
        this.addNotification(notification);
      }, delay * index);
    });
    this.notifications = [];
    return this;
  }

  clear() {
    this.notifications = [];
    return this;
  }
}

export default NotificationProvider;