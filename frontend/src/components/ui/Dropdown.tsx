import React, { useState, useRef, useEffect } from 'react';
import { cn } from '../../utils/cn';

interface DropdownItem {
  id: string;
  label: string;
  icon?: React.ReactNode;
  disabled?: boolean;
  onClick?: () => void;
}

interface DropdownProps {
  trigger: React.ReactNode;
  items: DropdownItem[];
  placement?: 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end';
  className?: string;
  menuClassName?: string;
}

const Dropdown: React.FC<DropdownProps> = ({
  trigger,
  items,
  placement = 'bottom-start',
  className,
  menuClassName
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setActiveIndex(-1);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Keyboard navigation
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      const enabledItems = items.filter(item => !item.disabled);
      
      switch (event.key) {
        case 'ArrowDown':
          event.preventDefault();
          setActiveIndex(prev => {
            const currentIndex = enabledItems.findIndex((_, index) => index === prev);
            return currentIndex < enabledItems.length - 1 ? currentIndex + 1 : 0;
          });
          break;
        case 'ArrowUp':
          event.preventDefault();
          setActiveIndex(prev => {
            const currentIndex = enabledItems.findIndex((_, index) => index === prev);
            return currentIndex > 0 ? currentIndex - 1 : enabledItems.length - 1;
          });
          break;
        case 'Enter':
        case ' ':
          event.preventDefault();
          if (activeIndex >= 0 && activeIndex < enabledItems.length) {
            const item = enabledItems[activeIndex];
            item.onClick?.();
            setIsOpen(false);
            setActiveIndex(-1);
          }
          break;
        case 'Escape':
          setIsOpen(false);
          setActiveIndex(-1);
          break;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, activeIndex, items]);

  const handleToggle = () => {
    setIsOpen(!isOpen);
    if (!isOpen) {
      setActiveIndex(-1);
    }
  };

  const handleItemClick = (item: DropdownItem) => {
    if (item.disabled) return;
    
    item.onClick?.();
    setIsOpen(false);
    setActiveIndex(-1);
  };

  const getPlacementClasses = () => {
    switch (placement) {
      case 'bottom-start':
        return 'top-full left-0 mt-1';
      case 'bottom-end':
        return 'top-full right-0 mt-1';
      case 'top-start':
        return 'bottom-full left-0 mb-1';
      case 'top-end':
        return 'bottom-full right-0 mb-1';
      default:
        return 'top-full left-0 mt-1';
    }
  };

  return (
    <div ref={dropdownRef} className={cn('relative inline-block', className)}>
      {/* Trigger */}
      <div onClick={handleToggle} className="cursor-pointer">
        {trigger}
      </div>

      {/* Menu */}
      {isOpen && (
        <div
          ref={menuRef}
          className={cn(
            'absolute z-50 min-w-[12rem] bg-white border border-gray-200 rounded-md shadow-lg animate-scale-in',
            getPlacementClasses(),
            menuClassName
          )}
          role="menu"
          aria-orientation="vertical"
        >
          <div className="py-1">
            {items.map((item, index) => {
              const isActive = activeIndex === index;
              
              return (
                <button
                  key={item.id}
                  className={cn(
                    'w-full text-left px-4 py-2 text-sm transition-colors flex items-center gap-2',
                    item.disabled
                      ? 'text-gray-400 cursor-not-allowed'
                      : isActive
                      ? 'bg-gray-100 text-gray-900'
                      : 'text-gray-700 hover:bg-gray-50'
                  )}
                  onClick={() => handleItemClick(item)}
                  disabled={item.disabled}
                  role="menuitem"
                  aria-disabled={item.disabled}
                >
                  {item.icon && (
                    <span className="w-4 h-4 flex-shrink-0">
                      {item.icon}
                    </span>
                  )}
                  {item.label}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};

// Simple Select component built on Dropdown
interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface SelectProps {
  options: SelectOption[];
  value?: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

const Select: React.FC<SelectProps> = ({
  options,
  value,
  onChange,
  placeholder = 'Select an option',
  disabled = false,
  className
}) => {
  const selectedOption = options.find(option => option.value === value);

  const dropdownItems: DropdownItem[] = options.map(option => ({
    id: option.value,
    label: option.label,
    disabled: option.disabled,
    onClick: () => onChange(option.value)
  }));

  const trigger = (
    <div
      className={cn(
        'w-full px-3 py-2 text-left bg-white border border-gray-300 rounded-lg shadow-sm transition-colors',
        'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent',
        disabled && 'opacity-50 cursor-not-allowed',
        className
      )}
    >
      <div className="flex items-center justify-between">
        <span className={cn('text-sm', !selectedOption && 'text-gray-500')}>
          {selectedOption ? selectedOption.label : placeholder}
        </span>
        <svg
          className="w-4 h-4 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M19 9l-7 7-7-7"
          />
        </svg>
      </div>
    </div>
  );

  return (
    <Dropdown
      trigger={trigger}
      items={dropdownItems}
      className="w-full"
    />
  );
};

export { Dropdown, Select };
export type { DropdownProps, DropdownItem, SelectProps, SelectOption };