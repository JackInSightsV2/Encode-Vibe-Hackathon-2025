import { useState, useRef, useEffect, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { cn } from '../../utils/cn';
import { usePageTransition } from '../../hooks/useAnimations';

interface TooltipProps {
  content: ReactNode;
  children: ReactNode;
  placement?: 'top' | 'bottom' | 'left' | 'right' | 'auto';
  delay?: number;
  disabled?: boolean;
  interactive?: boolean;
  arrow?: boolean;
  maxWidth?: number;
  offset?: number;
  className?: string;
  contentClassName?: string;
  trigger?: 'hover' | 'click' | 'focus';
  onShow?: () => void;
  onHide?: () => void;
}

export const AdvancedTooltip = ({
  content,
  children,
  placement = 'top',
  delay = 300,
  disabled = false,
  interactive = false,
  arrow = true,
  maxWidth = 300,
  offset = 8,
  className,
  contentClassName,
  trigger = 'hover',
  onShow,
  onHide
}: TooltipProps) => {
  const [isVisible, setIsVisible] = useState(false);
  const [actualPlacement, setActualPlacement] = useState(placement);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  
  const timeoutRef = useRef<NodeJS.Timeout>();
  const triggerRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  
  const { shouldRender, isEntered } = usePageTransition(isVisible, 200);

  const showTooltip = () => {
    if (disabled) return;
    
    clearTimeout(timeoutRef.current);
    timeoutRef.current = setTimeout(() => {
      setIsVisible(true);
      onShow?.();
    }, delay);
  };

  const hideTooltip = () => {
    clearTimeout(timeoutRef.current);
    if (!interactive || !isVisible) {
      setIsVisible(false);
      onHide?.();
    }
  };

  const toggleTooltip = () => {
    if (isVisible) {
      hideTooltip();
    } else {
      showTooltip();
    }
  };

  const calculatePosition = () => {
    if (!triggerRef.current || !tooltipRef.current) return;

    const triggerRect = triggerRef.current.getBoundingClientRect();
    const tooltipRect = tooltipRef.current.getBoundingClientRect();
    const viewport = {
      width: window.innerWidth,
      height: window.innerHeight
    };

    let finalPlacement = placement;
    let x = 0;
    let y = 0;

    // Auto placement calculation
    if (placement === 'auto') {
      const spaceTop = triggerRect.top;
      const spaceBottom = viewport.height - triggerRect.bottom;
      const spaceLeft = triggerRect.left;
      const spaceRight = viewport.width - triggerRect.right;

      const maxSpace = Math.max(spaceTop, spaceBottom, spaceLeft, spaceRight);
      
      if (maxSpace === spaceTop) finalPlacement = 'top';
      else if (maxSpace === spaceBottom) finalPlacement = 'bottom';
      else if (maxSpace === spaceLeft) finalPlacement = 'left';
      else finalPlacement = 'right';
    }

    // Calculate position based on placement
    switch (finalPlacement) {
      case 'top':
        x = triggerRect.left + triggerRect.width / 2 - tooltipRect.width / 2;
        y = triggerRect.top - tooltipRect.height - offset;
        
        // Adjust if tooltip would go off screen
        if (y < 0) {
          finalPlacement = 'bottom';
          y = triggerRect.bottom + offset;
        }
        break;
        
      case 'bottom':
        x = triggerRect.left + triggerRect.width / 2 - tooltipRect.width / 2;
        y = triggerRect.bottom + offset;
        
        if (y + tooltipRect.height > viewport.height) {
          finalPlacement = 'top';
          y = triggerRect.top - tooltipRect.height - offset;
        }
        break;
        
      case 'left':
        x = triggerRect.left - tooltipRect.width - offset;
        y = triggerRect.top + triggerRect.height / 2 - tooltipRect.height / 2;
        
        if (x < 0) {
          finalPlacement = 'right';
          x = triggerRect.right + offset;
        }
        break;
        
      case 'right':
        x = triggerRect.right + offset;
        y = triggerRect.top + triggerRect.height / 2 - tooltipRect.height / 2;
        
        if (x + tooltipRect.width > viewport.width) {
          finalPlacement = 'left';
          x = triggerRect.left - tooltipRect.width - offset;
        }
        break;
    }

    // Ensure tooltip doesn't go off screen horizontally
    if (x < 8) x = 8;
    if (x + tooltipRect.width > viewport.width - 8) {
      x = viewport.width - tooltipRect.width - 8;
    }

    // Ensure tooltip doesn't go off screen vertically
    if (y < 8) y = 8;
    if (y + tooltipRect.height > viewport.height - 8) {
      y = viewport.height - tooltipRect.height - 8;
    }

    setPosition({ x, y });
    setActualPlacement(finalPlacement);
  };

  useEffect(() => {
    if (isVisible) {
      calculatePosition();
      
      const handleResize = () => calculatePosition();
      const handleScroll = () => calculatePosition();
      
      window.addEventListener('resize', handleResize);
      window.addEventListener('scroll', handleScroll, true);
      
      return () => {
        window.removeEventListener('resize', handleResize);
        window.removeEventListener('scroll', handleScroll, true);
      };
    }
  }, [isVisible]);

  const handleMouseEnter = () => {
    if (trigger === 'hover') showTooltip();
  };

  const handleMouseLeave = () => {
    if (trigger === 'hover') hideTooltip();
  };

  const handleClick = () => {
    if (trigger === 'click') toggleTooltip();
  };

  const handleFocus = () => {
    if (trigger === 'focus' || trigger === 'hover') showTooltip();
  };

  const handleBlur = () => {
    if (trigger === 'focus' || trigger === 'hover') hideTooltip();
  };

  const handleTooltipMouseEnter = () => {
    if (interactive) {
      clearTimeout(timeoutRef.current);
    }
  };

  const handleTooltipMouseLeave = () => {
    if (interactive) {
      hideTooltip();
    }
  };

  const triggerProps = {
    onMouseEnter: handleMouseEnter,
    onMouseLeave: handleMouseLeave,
    onClick: handleClick,
    onFocus: handleFocus,
    onBlur: handleBlur
  };

  const getArrowPosition = () => {
    if (!triggerRef.current || !arrow) return {};
    
    switch (actualPlacement) {
      case 'top':
        return {
          top: '100%',
          left: '50%',
          transform: 'translateX(-50%) translateY(-50%) rotate(45deg)',
          borderBottom: 'none',
          borderRight: 'none'
        };
      case 'bottom':
        return {
          bottom: '100%',
          left: '50%',
          transform: 'translateX(-50%) translateY(50%) rotate(45deg)',
          borderTop: 'none',
          borderLeft: 'none'
        };
      case 'left':
        return {
          left: '100%',
          top: '50%',
          transform: 'translateY(-50%) translateX(-50%) rotate(45deg)',
          borderTop: 'none',
          borderRight: 'none'
        };
      case 'right':
        return {
          right: '100%',
          top: '50%',
          transform: 'translateY(-50%) translateX(50%) rotate(45deg)',
          borderBottom: 'none',
          borderLeft: 'none'
        };
      default:
        return {};
    }
  };

  return (
    <>
      <div
        ref={triggerRef}
        className={cn('inline-block', className)}
        {...triggerProps}
      >
        {children}
      </div>
      
      {shouldRender && createPortal(
        <div
          ref={tooltipRef}
          className={cn(
            'fixed z-50 px-3 py-2 text-sm rounded-lg shadow-lg pointer-events-none',
            'bg-theme-surface border border-theme-border-default text-theme-text-primary',
            'transition-all duration-200',
            isEntered ? 'opacity-100 scale-100' : 'opacity-0 scale-95',
            interactive && 'pointer-events-auto',
            contentClassName
          )}
          style={{
            left: position.x,
            top: position.y,
            maxWidth
          }}
          onMouseEnter={handleTooltipMouseEnter}
          onMouseLeave={handleTooltipMouseLeave}
          role="tooltip"
          aria-hidden={!isVisible}
        >
          {content}
          
          {arrow && (
            <div
              className="absolute w-2 h-2 bg-theme-surface border border-theme-border-default"
              style={getArrowPosition()}
            />
          )}
        </div>,
        document.body
      )}
    </>
  );
};

// Popover component (larger, more interactive tooltip)
interface PopoverProps extends Omit<TooltipProps, 'trigger'> {
  title?: string;
  footer?: ReactNode;
  closable?: boolean;
  onClose?: () => void;
}

export const Popover = ({
  title,
  footer,
  closable = true,
  onClose,
  ...tooltipProps
}: PopoverProps) => {
  const [_isOpen, setIsOpen] = useState(false);

  const handleClose = () => {
    setIsOpen(false);
    onClose?.();
  };

  return (
    <AdvancedTooltip
      {...tooltipProps}
      trigger="click"
      interactive={true}
      maxWidth={400}
      content={
        <div className="max-w-sm">
          {(title || closable) && (
            <div className="flex items-center justify-between mb-3 pb-2 border-b border-theme-border-light">
              {title && (
                <h3 className="font-semibold text-theme-text-primary">{title}</h3>
              )}
              {closable && (
                <button
                  onClick={handleClose}
                  className="text-theme-text-tertiary hover:text-theme-text-primary transition-colors"
                  aria-label="Close"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              )}
            </div>
          )}
          
          <div className="text-theme-text-secondary">
            {tooltipProps.content}
          </div>
          
          {footer && (
            <div className="mt-3 pt-2 border-t border-theme-border-light">
              {footer}
            </div>
          )}
        </div>
      }
    />
  );
};

export default AdvancedTooltip;