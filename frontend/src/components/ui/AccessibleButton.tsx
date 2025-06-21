import { forwardRef, useState, useRef, useEffect } from 'react';
import { Button, ButtonProps } from './Button';
import { useGestureAnimation } from '../../hooks/useAnimations';
import { ScreenReaderOnly } from './ScreenReader';
import { cn } from '../../utils/cn';

interface AccessibleButtonProps extends ButtonProps {
  'aria-label'?: string;
  'aria-describedby'?: string;
  'aria-expanded'?: boolean;
  'aria-controls'?: string;
  'aria-pressed'?: boolean;
  'aria-haspopup'?: boolean | 'false' | 'true' | 'menu' | 'listbox' | 'tree' | 'grid' | 'dialog';
  description?: string;
  shortcut?: string;
  confirmAction?: boolean;
  confirmMessage?: string;
  tooltipDelay?: number;
  onPress?: () => void;
  onLongPress?: () => void;
  longPressDelay?: number;
  hapticFeedback?: boolean;
}

export const AccessibleButton = forwardRef<HTMLButtonElement, AccessibleButtonProps>(({
  children,
  className,
  disabled,
  description,
  shortcut,
  confirmAction = false,
  confirmMessage = 'Are you sure?',
  tooltipDelay = 500,
  onPress,
  onLongPress,
  longPressDelay = 500,
  hapticFeedback = false,
  onClick,
  ...props
}, ref) => {
  const [isPressed, setIsPressed] = useState(false);
  const [isFocused, setIsFocused] = useState(false);
  const [showTooltip, setShowTooltip] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  
  const longPressTimerRef = useRef<NodeJS.Timeout>();
  const tooltipTimerRef = useRef<NodeJS.Timeout>();
  const buttonRef = useRef<HTMLButtonElement>(null);
  
  const { scale, handlers } = useGestureAnimation(1);

  // Combine refs
  useEffect(() => {
    if (ref) {
      if (typeof ref === 'function') {
        ref(buttonRef.current);
      } else {
        ref.current = buttonRef.current;
      }
    }
  }, [ref]);

  // Handle long press
  const handleMouseDown = () => {
    setIsPressed(true);
    handlers.onMouseDown();
    onPress?.();
    
    if (hapticFeedback && navigator.vibrate) {
      navigator.vibrate(10);
    }
    
    if (onLongPress) {
      longPressTimerRef.current = setTimeout(() => {
        onLongPress();
        if (hapticFeedback && navigator.vibrate) {
          navigator.vibrate([20, 50, 20]);
        }
      }, longPressDelay);
    }
  };

  const handleMouseUp = () => {
    setIsPressed(false);
    handlers.onMouseUp();
    
    if (longPressTimerRef.current) {
      clearTimeout(longPressTimerRef.current);
    }
  };

  const handleMouseLeave = () => {
    setIsPressed(false);
    handlers.onMouseLeave();
    setShowTooltip(false);
    
    if (longPressTimerRef.current) {
      clearTimeout(longPressTimerRef.current);
    }
    if (tooltipTimerRef.current) {
      clearTimeout(tooltipTimerRef.current);
    }
  };

  const handleMouseEnter = () => {
    handlers.onMouseEnter();
    
    if (description || shortcut) {
      tooltipTimerRef.current = setTimeout(() => {
        setShowTooltip(true);
      }, tooltipDelay);
    }
  };

  const handleFocus = () => {
    setIsFocused(true);
    
    if (description || shortcut) {
      setShowTooltip(true);
    }
  };

  const handleBlur = () => {
    setIsFocused(false);
    setShowTooltip(false);
    
    if (tooltipTimerRef.current) {
      clearTimeout(tooltipTimerRef.current);
    }
  };

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (confirmAction && !showConfirm) {
      e.preventDefault();
      setShowConfirm(true);
      return;
    }
    
    if (showConfirm) {
      setShowConfirm(false);
    }
    
    onClick?.(e);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      setIsPressed(true);
      onPress?.();
    }
    
    // Handle shortcuts
    if (shortcut && e.key === shortcut.toLowerCase()) {
      e.preventDefault();
      handleClick(e as any);
    }
  };

  const handleKeyUp = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      setIsPressed(false);
    }
  };

  // Generate unique IDs for ARIA
  const descriptionId = description ? `btn-desc-${Math.random().toString(36).substr(2, 9)}` : undefined;
  const tooltipId = showTooltip ? `btn-tooltip-${Math.random().toString(36).substr(2, 9)}` : undefined;

  return (
    <div className="relative inline-block">
      <Button
        ref={buttonRef}
        className={cn(
          'transition-all duration-150 ease-out',
          'focus:ring-2 focus:ring-offset-2 focus:ring-theme-border-focus',
          'focus:outline-none',
          isPressed && 'transform scale-95',
          isFocused && 'ring-2 ring-theme-border-focus ring-offset-2',
          showConfirm && 'ring-2 ring-theme-status-warning ring-offset-2',
          className
        )}
        disabled={disabled}
        style={{ 
          transform: `scale(${scale})`,
          transformOrigin: 'center'
        }}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseLeave}
        onMouseEnter={handleMouseEnter}
        onFocus={handleFocus}
        onBlur={handleBlur}
        onClick={handleClick}
        onKeyDown={handleKeyDown}
        onKeyUp={handleKeyUp}
        aria-describedby={cn(
          props['aria-describedby'],
          descriptionId,
          tooltipId
        )}
        role="button"
        tabIndex={disabled ? -1 : 0}
        {...props}
      >
        {showConfirm ? confirmMessage : children}
        
        {/* Screen reader description */}
        {description && (
          <ScreenReaderOnly>
            <span id={descriptionId}>{description}</span>
          </ScreenReaderOnly>
        )}
        
        {/* Keyboard shortcut for screen readers */}
        {shortcut && (
          <ScreenReaderOnly>
            <span>Keyboard shortcut: {shortcut}</span>
          </ScreenReaderOnly>
        )}
      </Button>
      
      {/* Tooltip */}
      {showTooltip && (description || shortcut) && !disabled && (
        <div
          id={tooltipId}
          role="tooltip"
          className={cn(
            'absolute z-50 px-2 py-1 text-xs rounded',
            'bg-theme-surface border border-theme-border-default',
            'text-theme-text-primary shadow-theme-md',
            'bottom-full left-1/2 transform -translate-x-1/2 mb-2',
            'animate-fade-in'
          )}
        >
          {description && <div>{description}</div>}
          {shortcut && (
            <div className="text-theme-text-secondary">
              Press {shortcut}
            </div>
          )}
          
          {/* Tooltip arrow */}
          <div
            className="absolute top-full left-1/2 transform -translate-x-1/2"
            style={{
              width: 0,
              height: 0,
              borderLeft: '5px solid transparent',
              borderRight: '5px solid transparent',
              borderTop: '5px solid var(--color-border-default)'
            }}
          />
        </div>
      )}
    </div>
  );
});

AccessibleButton.displayName = 'AccessibleButton';

// Icon button variant with better accessibility
interface AccessibleIconButtonProps extends AccessibleButtonProps {
  icon: React.ReactNode;
  label: string;
  'aria-label': string;
}

export const AccessibleIconButton = forwardRef<HTMLButtonElement, AccessibleIconButtonProps>(({
  icon,
  label,
  className,
  ...props
}, ref) => {
  return (
    <AccessibleButton
      ref={ref}
      className={cn('p-2 rounded-full', className)}
      {...props}
    >
      <span aria-hidden="true">{icon}</span>
      <ScreenReaderOnly>{label}</ScreenReaderOnly>
    </AccessibleButton>
  );
});

AccessibleIconButton.displayName = 'AccessibleIconButton';

// Toggle button with proper ARIA states
interface AccessibleToggleButtonProps extends Omit<AccessibleButtonProps, 'aria-pressed'> {
  pressed: boolean;
  onToggle: (pressed: boolean) => void;
  pressedLabel?: string;
  unpressedLabel?: string;
}

export const AccessibleToggleButton = forwardRef<HTMLButtonElement, AccessibleToggleButtonProps>(({
  pressed,
  onToggle,
  pressedLabel,
  unpressedLabel,
  children,
  onClick,
  ...props
}, ref) => {
  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    onToggle(!pressed);
    onClick?.(e);
  };

  const getLabel = () => {
    if (pressed && pressedLabel) return pressedLabel;
    if (!pressed && unpressedLabel) return unpressedLabel;
    return typeof children === 'string' ? children : '';
  };

  return (
    <AccessibleButton
      ref={ref}
      aria-pressed={pressed}
      aria-label={getLabel()}
      onClick={handleClick}
      {...props}
    >
      {children}
    </AccessibleButton>
  );
});

AccessibleToggleButton.displayName = 'AccessibleToggleButton';

export default AccessibleButton;