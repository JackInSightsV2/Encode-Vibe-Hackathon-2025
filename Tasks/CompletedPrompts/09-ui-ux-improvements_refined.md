# UI/UX Improvements - Refined Implementation Cycles

## Overview
Break down UI/UX improvements into 5 manageable cycles, from design system to advanced interactions.

---

## **Cycle 9A: Design System & Component Library**
**Duration:** 6-8 hours | **Priority:** High

### Prerequisites
- Frontend development environment ready
- Understanding of design system principles

### Implementation Tasks
- [ ] Create design system foundation with Tailwind CSS
- [ ] Build reusable UI component library
- [ ] Implement consistent color palette and typography
- [ ] Create component documentation
- [ ] Add component testing

### Code Deliverables
```typescript
// frontend/src/components/ui/Button.tsx
interface ButtonProps {
    variant: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
    size: 'sm' | 'md' | 'lg';
    loading?: boolean;
    disabled?: boolean;
    icon?: React.ReactNode;
    children: React.ReactNode;
    onClick?: () => void;
    className?: string;
}

const Button: React.FC<ButtonProps> = ({
    variant = 'primary',
    size = 'md',
    loading = false,
    disabled = false,
    icon,
    children,
    onClick,
    className,
    ...props
}) => {
    const baseClasses = 'inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2';
    
    const variantClasses = {
        primary: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500',
        secondary: 'bg-gray-600 text-white hover:bg-gray-700 focus:ring-gray-500',
        outline: 'border-2 border-blue-600 text-blue-600 hover:bg-blue-50 focus:ring-blue-500',
        ghost: 'text-gray-600 hover:bg-gray-100 focus:ring-gray-500',
        danger: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500',
    };
    
    const sizeClasses = {
        sm: 'px-3 py-1.5 text-sm',
        md: 'px-4 py-2 text-base',
        lg: 'px-6 py-3 text-lg',
    };
    
    const classes = cn(
        baseClasses,
        variantClasses[variant],
        sizeClasses[size],
        disabled && 'opacity-50 cursor-not-allowed',
        className
    );
    
    return (
        <button
            className={classes}
            disabled={disabled || loading}
            onClick={onClick}
            {...props}
        >
            {loading && <Spinner className="w-4 h-4 mr-2" />}
            {icon && !loading && <span className="mr-2">{icon}</span>}
            {children}
        </button>
    );
};

// frontend/src/components/ui/Card.tsx
interface CardProps {
    title?: string;
    subtitle?: string;
    actions?: React.ReactNode;
    children: React.ReactNode;
    className?: string;
    padding?: 'none' | 'sm' | 'md' | 'lg';
}

const Card: React.FC<CardProps> = ({
    title,
    subtitle,
    actions,
    children,
    className,
    padding = 'md'
}) => {
    const paddingClasses = {
        none: '',
        sm: 'p-4',
        md: 'p-6',
        lg: 'p-8'
    };
    
    return (
        <div className={cn(
            'bg-white rounded-xl shadow-sm border border-gray-200',
            paddingClasses[padding],
            className
        )}>
            {(title || subtitle || actions) && (
                <div className="flex items-center justify-between mb-4">
                    <div>
                        {title && <h3 className="text-lg font-semibold text-gray-900">{title}</h3>}
                        {subtitle && <p className="text-sm text-gray-500 mt-1">{subtitle}</p>}
                    </div>
                    {actions && <div className="flex space-x-2">{actions}</div>}
                </div>
            )}
            {children}
        </div>
    );
};

// frontend/src/components/ui/Input.tsx
interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
    label?: string;
    error?: string;
    helper?: string;
    icon?: React.ReactNode;
    variant?: 'default' | 'filled';
}

const Input: React.FC<InputProps> = ({
    label,
    error,
    helper,
    icon,
    variant = 'default',
    className,
    ...props
}) => {
    const inputClasses = cn(
        'w-full rounded-lg border px-3 py-2 text-sm transition-colors',
        'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
        error
            ? 'border-red-300 bg-red-50 text-red-900 placeholder-red-300'
            : variant === 'filled'
            ? 'border-gray-200 bg-gray-50'
            : 'border-gray-300 bg-white',
        icon && 'pl-10',
        className
    );
    
    return (
        <div>
            {label && (
                <label className="block text-sm font-medium text-gray-700 mb-2">
                    {label}
                </label>
            )}
            <div className="relative">
                {icon && (
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center">
                        {icon}
                    </div>
                )}
                <input className={inputClasses} {...props} />
            </div>
            {error && (
                <p className="mt-1 text-sm text-red-600">{error}</p>
            )}
            {helper && !error && (
                <p className="mt-1 text-sm text-gray-500">{helper}</p>
            )}
        </div>
    );
};
```

### Component Library Structure
```
frontend/src/components/ui/
├── Button.tsx
├── Input.tsx
├── Card.tsx
├── Badge.tsx
├── Modal.tsx
├── Toast.tsx
├── Spinner.tsx
├── Tooltip.tsx
├── Dropdown.tsx
└── index.ts
```

### Testing Requirements
- [ ] Unit tests for all components
- [ ] Visual regression tests
- [ ] Accessibility tests
- [ ] Storybook documentation

### Acceptance Criteria
- [ ] All components follow consistent design patterns
- [ ] Color palette is accessible (WCAG AA)
- [ ] Typography scales appropriately
- [ ] Components are fully typed with TypeScript
- [ ] Documentation is complete and helpful

### Risk Mitigation
- Test components across different browsers
- Ensure accessibility from the start
- Use established design patterns

---

## **Cycle 9B: Responsive Design Implementation**
**Duration:** 6-8 hours | **Priority:** High

### Prerequisites
- Cycle 9A completed and tested
- Understanding of responsive design principles

### Implementation Tasks
- [ ] Implement mobile-first responsive design
- [ ] Create responsive navigation patterns
- [ ] Optimize data tables for mobile
- [ ] Add responsive breakpoints throughout
- [ ] Test across different screen sizes

### Code Deliverables
```typescript
// frontend/src/components/layout/ResponsiveLayout.tsx
interface ResponsiveLayoutProps {
    sidebar?: React.ReactNode;
    header?: React.ReactNode;
    children: React.ReactNode;
}

const ResponsiveLayout: React.FC<ResponsiveLayoutProps> = ({
    sidebar,
    header,
    children
}) => {
    const [sidebarOpen, setSidebarOpen] = useState(false);
    const isMobile = useMediaQuery('(max-width: 768px)');
    
    return (
        <div className="min-h-screen bg-gray-50">
            {/* Mobile menu overlay */}
            {isMobile && sidebarOpen && (
                <div 
                    className="fixed inset-0 z-40 bg-black bg-opacity-50"
                    onClick={() => setSidebarOpen(false)}
                />
            )}
            
            {/* Sidebar */}
            {sidebar && (
                <aside className={cn(
                    'fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 transition-transform duration-300',
                    isMobile 
                        ? sidebarOpen ? 'translate-x-0' : '-translate-x-full'
                        : 'translate-x-0'
                )}>
                    {sidebar}
                </aside>
            )}
            
            {/* Main content */}
            <div className={cn(
                'flex flex-col',
                sidebar && !isMobile && 'ml-64'
            )}>
                {/* Header */}
                {header && (
                    <header className="bg-white border-b border-gray-200 px-4 py-3 sm:px-6">
                        <div className="flex items-center justify-between">
                            {isMobile && sidebar && (
                                <button
                                    onClick={() => setSidebarOpen(true)}
                                    className="p-2 text-gray-500 hover:text-gray-700"
                                >
                                    <MenuIcon className="w-6 h-6" />
                                </button>
                            )}
                            {header}
                        </div>
                    </header>
                )}
                
                {/* Page content */}
                <main className="flex-1 p-4 sm:p-6">
                    {children}
                </main>
            </div>
        </div>
    );
};

// frontend/src/components/ui/ResponsiveTable.tsx
interface ResponsiveTableProps<T> {
    data: T[];
    columns: Column<T>[];
    loading?: boolean;
    mobileCardRenderer?: (item: T) => React.ReactNode;
}

function ResponsiveTable<T>({ 
    data, 
    columns, 
    loading, 
    mobileCardRenderer 
}: ResponsiveTableProps<T>) {
    const isMobile = useMediaQuery('(max-width: 768px)');
    
    if (loading) {
        return <TableSkeleton />;
    }
    
    if (isMobile && mobileCardRenderer) {
        return (
            <div className="space-y-3">
                {data.map((item, index) => (
                    <Card key={index} padding="sm">
                        {mobileCardRenderer(item)}
                    </Card>
                ))}
            </div>
        );
    }
    
    return (
        <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                    <tr>
                        {columns.map((column, index) => (
                            <th
                                key={index}
                                className={cn(
                                    'px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider',
                                    column.hidden?.mobile && 'hidden sm:table-cell'
                                )}
                            >
                                {column.header}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                    {data.map((item, rowIndex) => (
                        <tr key={rowIndex}>
                            {columns.map((column, colIndex) => (
                                <td
                                    key={colIndex}
                                    className={cn(
                                        'px-6 py-4 whitespace-nowrap text-sm text-gray-900',
                                        column.hidden?.mobile && 'hidden sm:table-cell'
                                    )}
                                >
                                    {column.render ? column.render(item) : String(item[column.key])}
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

// Custom hook for media queries
function useMediaQuery(query: string): boolean {
    const [matches, setMatches] = useState(false);
    
    useEffect(() => {
        const media = window.matchMedia(query);
        if (media.matches !== matches) {
            setMatches(media.matches);
        }
        
        const listener = () => setMatches(media.matches);
        media.addListener(listener);
        return () => media.removeListener(listener);
    }, [matches, query]);
    
    return matches;
}
```

### Testing Requirements
- [ ] Test on various screen sizes
- [ ] Test touch interactions on mobile
- [ ] Test responsive navigation
- [ ] Cross-browser compatibility testing

### Acceptance Criteria
- [ ] Design works on screens from 320px to 2560px
- [ ] Navigation is accessible on all screen sizes
- [ ] Tables are usable on mobile devices
- [ ] Touch targets meet accessibility guidelines
- [ ] No horizontal scrolling on mobile

### Risk Mitigation
- Test on real devices, not just browser dev tools
- Use progressive enhancement approach
- Ensure core functionality works without JavaScript

---

## **Cycle 9C: Dark Mode & Theme System**
**Duration:** 5-6 hours | **Priority:** Medium

### Prerequisites
- Cycle 9B completed and tested
- Understanding of CSS custom properties

### Implementation Tasks
- [ ] Create theme context and provider
- [ ] Implement CSS custom properties for theming
- [ ] Add dark mode toggle component
- [ ] Update all components for theme compatibility
- [ ] Persist theme preference

### Code Deliverables
```typescript
// frontend/src/contexts/ThemeContext.tsx
type Theme = 'light' | 'dark' | 'system';

interface ThemeContextType {
    theme: Theme;
    setTheme: (theme: Theme) => void;
    resolvedTheme: 'light' | 'dark';
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [theme, setTheme] = useState<Theme>(() => {
        const stored = localStorage.getItem('qt1-theme') as Theme;
        return stored || 'system';
    });
    
    const [resolvedTheme, setResolvedTheme] = useState<'light' | 'dark'>('light');
    
    useEffect(() => {
        const root = document.documentElement;
        
        const updateTheme = () => {
            let resolved: 'light' | 'dark';
            
            if (theme === 'system') {
                resolved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
            } else {
                resolved = theme;
            }
            
            setResolvedTheme(resolved);
            root.setAttribute('data-theme', resolved);
            
            // Update CSS custom properties
            if (resolved === 'dark') {
                root.style.setProperty('--bg-primary', '#0f172a');
                root.style.setProperty('--bg-secondary', '#1e293b');
                root.style.setProperty('--text-primary', '#f1f5f9');
                root.style.setProperty('--text-secondary', '#cbd5e1');
                root.style.setProperty('--border-color', '#334155');
            } else {
                root.style.setProperty('--bg-primary', '#ffffff');
                root.style.setProperty('--bg-secondary', '#f8fafc');
                root.style.setProperty('--text-primary', '#1e293b');
                root.style.setProperty('--text-secondary', '#64748b');
                root.style.setProperty('--border-color', '#e2e8f0');
            }
        };
        
        updateTheme();
        
        // Listen for system theme changes
        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
        mediaQuery.addListener(updateTheme);
        
        return () => mediaQuery.removeListener(updateTheme);
    }, [theme]);
    
    const handleSetTheme = (newTheme: Theme) => {
        setTheme(newTheme);
        localStorage.setItem('qt1-theme', newTheme);
    };
    
    return (
        <ThemeContext.Provider value={{ theme, setTheme: handleSetTheme, resolvedTheme }}>
            {children}
        </ThemeContext.Provider>
    );
};

// frontend/src/components/ui/ThemeToggle.tsx
const ThemeToggle: React.FC = () => {
    const { theme, setTheme, resolvedTheme } = useTheme();
    
    const toggleTheme = () => {
        if (theme === 'light') {
            setTheme('dark');
        } else if (theme === 'dark') {
            setTheme('system');
        } else {
            setTheme('light');
        }
    };
    
    const getIcon = () => {
        if (theme === 'system') {
            return <ComputerIcon className="w-4 h-4" />;
        }
        return resolvedTheme === 'dark' 
            ? <MoonIcon className="w-4 h-4" />
            : <SunIcon className="w-4 h-4" />;
    };
    
    const getLabel = () => {
        if (theme === 'system') return 'System';
        return resolvedTheme === 'dark' ? 'Dark' : 'Light';
    };
    
    return (
        <Button
            variant="ghost"
            size="sm"
            onClick={toggleTheme}
            className="flex items-center space-x-2"
        >
            {getIcon()}
            <span className="hidden sm:inline">{getLabel()}</span>
        </Button>
    );
};
```

### CSS Custom Properties
```css
/* frontend/src/styles/themes.css */
:root {
    /* Light theme (default) */
    --bg-primary: #ffffff;
    --bg-secondary: #f8fafc;
    --bg-tertiary: #f1f5f9;
    --text-primary: #1e293b;
    --text-secondary: #64748b;
    --text-tertiary: #94a3b8;
    --border-color: #e2e8f0;
    --border-focus: #3b82f6;
    --accent-primary: #3b82f6;
    --accent-secondary: #10b981;
    --accent-danger: #ef4444;
    --accent-warning: #f59e0b;
    --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
    --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1);
    --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1);
}

[data-theme="dark"] {
    --bg-primary: #0f172a;
    --bg-secondary: #1e293b;
    --bg-tertiary: #334155;
    --text-primary: #f1f5f9;
    --text-secondary: #cbd5e1;
    --text-tertiary: #94a3b8;
    --border-color: #334155;
    --border-focus: #60a5fa;
    --accent-primary: #60a5fa;
    --accent-secondary: #34d399;
    --accent-danger: #f87171;
    --accent-warning: #fbbf24;
    --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.3);
    --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.4);
    --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.4);
}

/* Update Tailwind config to use custom properties */
/* tailwind.config.js */
module.exports = {
    theme: {
        extend: {
            colors: {
                primary: 'var(--bg-primary)',
                secondary: 'var(--bg-secondary)',
                tertiary: 'var(--bg-tertiary)',
            },
            textColor: {
                primary: 'var(--text-primary)',
                secondary: 'var(--text-secondary)',
                tertiary: 'var(--text-tertiary)',
            },
            borderColor: {
                primary: 'var(--border-color)',
                focus: 'var(--border-focus)',
            },
        },
    },
};
```

### Testing Requirements
- [ ] Test theme switching functionality
- [ ] Test system theme detection
- [ ] Test theme persistence
- [ ] Visual testing for both themes

### Acceptance Criteria
- [ ] Theme switching works smoothly
- [ ] All components support both themes
- [ ] System theme preference is respected
- [ ] Theme choice persists across sessions
- [ ] No flash of incorrect theme on load

### Risk Mitigation
- Test theme switching thoroughly
- Ensure contrast ratios meet accessibility standards
- Validate theme persistence across browsers

---

## **Cycle 9D: Accessibility & Micro-interactions**
**Duration:** 6-7 hours | **Priority:** High

### Prerequisites
- Cycle 9C completed and tested
- Understanding of accessibility guidelines

### Implementation Tasks
- [ ] Add comprehensive ARIA labels and roles
- [ ] Implement keyboard navigation
- [ ] Add micro-interactions and animations
- [ ] Create focus management system
- [ ] Add screen reader support

### Code Deliverables
```typescript
// frontend/src/hooks/useKeyboardNavigation.ts
export const useKeyboardNavigation = (items: string[], onSelect: (id: string) => void) => {
    const [activeIndex, setActiveIndex] = useState(0);
    
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            switch (event.key) {
                case 'ArrowDown':
                    event.preventDefault();
                    setActiveIndex(prev => (prev + 1) % items.length);
                    break;
                case 'ArrowUp':
                    event.preventDefault();
                    setActiveIndex(prev => (prev - 1 + items.length) % items.length);
                    break;
                case 'Enter':
                case ' ':
                    event.preventDefault();
                    onSelect(items[activeIndex]);
                    break;
                case 'Escape':
                    // Handle escape
                    break;
            }
        };
        
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [items, activeIndex, onSelect]);
    
    return { activeIndex, setActiveIndex };
};

// frontend/src/components/ui/AccessibleButton.tsx
interface AccessibleButtonProps extends ButtonProps {
    'aria-label'?: string;
    'aria-describedby'?: string;
    'aria-expanded'?: boolean;
    'aria-controls'?: string;
}

const AccessibleButton: React.FC<AccessibleButtonProps> = (props) => {
    const [isPressed, setIsPressed] = useState(false);
    const [isFocused, setIsFocused] = useState(false);
    
    return (
        <Button
            {...props}
            className={cn(
                props.className,
                'focus:ring-2 focus:ring-offset-2 focus:ring-blue-500',
                'transform transition-transform duration-75',
                isPressed && 'scale-95',
                isFocused && 'ring-2 ring-blue-500'
            )}
            onMouseDown={() => setIsPressed(true)}
            onMouseUp={() => setIsPressed(false)}
            onMouseLeave={() => setIsPressed(false)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            role="button"
            tabIndex={props.disabled ? -1 : 0}
        />
    );
};

// frontend/src/components/ui/AnimatedCounter.tsx
interface AnimatedCounterProps {
    value: number;
    duration?: number;
    formatter?: (value: number) => string;
}

const AnimatedCounter: React.FC<AnimatedCounterProps> = ({
    value,
    duration = 1000,
    formatter = (v) => v.toString()
}) => {
    const [displayValue, setDisplayValue] = useState(0);
    const [isVisible, setIsVisible] = useState(false);
    const ref = useRef<HTMLSpanElement>(null);
    
    // Intersection observer for triggering animation
    useEffect(() => {
        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting) {
                    setIsVisible(true);
                }
            },
            { threshold: 0.1 }
        );
        
        if (ref.current) {
            observer.observe(ref.current);
        }
        
        return () => observer.disconnect();
    }, []);
    
    // Animate counter
    useEffect(() => {
        if (!isVisible) return;
        
        const startTime = Date.now();
        const startValue = displayValue;
        const difference = value - startValue;
        
        const animate = () => {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // Easing function
            const easeOutQuart = 1 - Math.pow(1 - progress, 4);
            const current = startValue + difference * easeOutQuart;
            
            setDisplayValue(Math.round(current));
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }, [value, isVisible, duration]);
    
    return (
        <span 
            ref={ref}
            className="tabular-nums"
            aria-live="polite"
            aria-label={`Count: ${formatter(value)}`}
        >
            {formatter(displayValue)}
        </span>
    );
};

// frontend/src/components/ui/FocusTrap.tsx
interface FocusTrapProps {
    children: React.ReactNode;
    active: boolean;
    restoreFocus?: boolean;
}

const FocusTrap: React.FC<FocusTrapProps> = ({ 
    children, 
    active, 
    restoreFocus = true 
}) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const previousActiveElement = useRef<HTMLElement | null>(null);
    
    useEffect(() => {
        if (!active) return;
        
        // Store the previously focused element
        previousActiveElement.current = document.activeElement as HTMLElement;
        
        const container = containerRef.current;
        if (!container) return;
        
        // Find all focusable elements
        const focusableElements = container.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        
        const firstElement = focusableElements[0] as HTMLElement;
        const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;
        
        // Focus the first element
        if (firstElement) {
            firstElement.focus();
        }
        
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key !== 'Tab') return;
            
            if (event.shiftKey) {
                // Shift + Tab
                if (document.activeElement === firstElement) {
                    event.preventDefault();
                    lastElement?.focus();
                }
            } else {
                // Tab
                if (document.activeElement === lastElement) {
                    event.preventDefault();
                    firstElement?.focus();
                }
            }
        };
        
        document.addEventListener('keydown', handleKeyDown);
        
        return () => {
            document.removeEventListener('keydown', handleKeyDown);
            
            // Restore focus
            if (restoreFocus && previousActiveElement.current) {
                previousActiveElement.current.focus();
            }
        };
    }, [active, restoreFocus]);
    
    return <div ref={containerRef}>{children}</div>;
};
```

### Testing Requirements
- [ ] Accessibility testing with screen readers
- [ ] Keyboard navigation testing
- [ ] Color contrast validation
- [ ] Focus management testing

### Acceptance Criteria
- [ ] WCAG 2.1 AA compliance achieved
- [ ] All interactive elements are keyboard accessible
- [ ] Screen reader announces content correctly
- [ ] Focus indicators are clearly visible
- [ ] Animations respect reduced-motion preferences

### Risk Mitigation
- Test with actual assistive technologies
- Validate with automated accessibility tools
- Get feedback from users with disabilities

---

## **Cycle 9E: Advanced Interactions & Polish**
**Duration:** 6-8 hours | **Priority:** Low

### Prerequisites
- Cycles 9A-9D completed and tested
- Advanced React knowledge

### Implementation Tasks
- [ ] Add drag-and-drop functionality
- [ ] Implement advanced tooltips and popovers
- [ ] Create keyboard shortcuts system
- [ ] Add advanced loading states and skeletons
- [ ] Implement contextual help system

### Code Deliverables
```typescript
// frontend/src/hooks/useDragAndDrop.ts
interface DragAndDropOptions<T> {
    onDrop: (draggedItem: T, targetItem: T) => void;
    onReorder?: (items: T[]) => void;
    enabled?: boolean;
}

export const useDragAndDrop = <T extends { id: string }>(
    items: T[],
    options: DragAndDropOptions<T>
) => {
    const [draggedItem, setDraggedItem] = useState<T | null>(null);
    const [dragOverItem, setDragOverItem] = useState<T | null>(null);
    
    const handleDragStart = (item: T) => (event: React.DragEvent) => {
        if (!options.enabled) return;
        setDraggedItem(item);
        event.dataTransfer.effectAllowed = 'move';
    };
    
    const handleDragOver = (item: T) => (event: React.DragEvent) => {
        if (!options.enabled || !draggedItem) return;
        event.preventDefault();
        setDragOverItem(item);
        event.dataTransfer.dropEffect = 'move';
    };
    
    const handleDrop = (item: T) => (event: React.DragEvent) => {
        if (!options.enabled || !draggedItem) return;
        event.preventDefault();
        
        if (draggedItem.id !== item.id) {
            options.onDrop(draggedItem, item);
        }
        
        setDraggedItem(null);
        setDragOverItem(null);
    };
    
    const handleDragEnd = () => {
        setDraggedItem(null);
        setDragOverItem(null);
    };
    
    return {
        draggedItem,
        dragOverItem,
        handleDragStart,
        handleDragOver,
        handleDrop,
        handleDragEnd,
    };
};

// frontend/src/components/ui/Tooltip.tsx
interface TooltipProps {
    content: React.ReactNode;
    children: React.ReactNode;
    placement?: 'top' | 'bottom' | 'left' | 'right';
    delay?: number;
    disabled?: boolean;
}

const Tooltip: React.FC<TooltipProps> = ({
    content,
    children,
    placement = 'top',
    delay = 300,
    disabled = false
}) => {
    const [isVisible, setIsVisible] = useState(false);
    const timeoutRef = useRef<NodeJS.Timeout>();
    
    const showTooltip = () => {
        if (disabled) return;
        timeoutRef.current = setTimeout(() => setIsVisible(true), delay);
    };
    
    const hideTooltip = () => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
        }
        setIsVisible(false);
    };
    
    return (
        <div 
            className="relative inline-block"
            onMouseEnter={showTooltip}
            onMouseLeave={hideTooltip}
            onFocus={showTooltip}
            onBlur={hideTooltip}
        >
            {children}
            {isVisible && (
                <div
                    className={cn(
                        'absolute z-50 px-2 py-1 text-sm text-white bg-gray-900 rounded-md shadow-lg',
                        'animate-in fade-in-0 zoom-in-95 duration-200',
                        {
                            'bottom-full left-1/2 transform -translate-x-1/2 mb-2': placement === 'top',
                            'top-full left-1/2 transform -translate-x-1/2 mt-2': placement === 'bottom',
                            'right-full top-1/2 transform -translate-y-1/2 mr-2': placement === 'left',
                            'left-full top-1/2 transform -translate-y-1/2 ml-2': placement === 'right',
                        }
                    )}
                    role="tooltip"
                >
                    {content}
                    {/* Arrow */}
                    <div
                        className={cn(
                            'absolute w-2 h-2 bg-gray-900 transform rotate-45',
                            {
                                'top-full left-1/2 -translate-x-1/2 -mt-1': placement === 'top',
                                'bottom-full left-1/2 -translate-x-1/2 -mb-1': placement === 'bottom',
                                'top-1/2 left-full -translate-y-1/2 -ml-1': placement === 'left',
                                'top-1/2 right-full -translate-y-1/2 -mr-1': placement === 'right',
                            }
                        )}
                    />
                </div>
            )}
        </div>
    );
};

// frontend/src/hooks/useKeyboardShortcuts.ts
interface Shortcut {
    key: string;
    ctrlKey?: boolean;
    shiftKey?: boolean;
    altKey?: boolean;
    action: () => void;
    description?: string;
}

export const useKeyboardShortcuts = (shortcuts: Shortcut[]) => {
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            const matchingShortcut = shortcuts.find(shortcut => {
                return (
                    shortcut.key.toLowerCase() === event.key.toLowerCase() &&
                    !!shortcut.ctrlKey === event.ctrlKey &&
                    !!shortcut.shiftKey === event.shiftKey &&
                    !!shortcut.altKey === event.altKey
                );
            });
            
            if (matchingShortcut) {
                event.preventDefault();
                matchingShortcut.action();
            }
        };
        
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [shortcuts]);
};

// frontend/src/components/ui/LoadingSkeleton.tsx
interface SkeletonProps {
    className?: string;
    lines?: number;
    avatar?: boolean;
    button?: boolean;
}

const Skeleton: React.FC<SkeletonProps> = ({
    className,
    lines = 1,
    avatar = false,
    button = false
}) => {
    if (avatar) {
        return (
            <div className="flex items-center space-x-4 animate-pulse">
                <div className="w-10 h-10 bg-gray-300 rounded-full"></div>
                <div className="flex-1 space-y-2">
                    <div className="h-4 bg-gray-300 rounded w-3/4"></div>
                    <div className="h-3 bg-gray-300 rounded w-1/2"></div>
                </div>
            </div>
        );
    }
    
    if (button) {
        return (
            <div className="animate-pulse">
                <div className="h-10 bg-gray-300 rounded w-24"></div>
            </div>
        );
    }
    
    return (
        <div className={cn('animate-pulse space-y-3', className)}>
            {Array.from({ length: lines }).map((_, index) => (
                <div
                    key={index}
                    className={cn(
                        'h-4 bg-gray-300 rounded',
                        index === lines - 1 && 'w-3/4' // Last line is shorter
                    )}
                />
            ))}
        </div>
    );
};
```

### Testing Requirements
- [ ] Test drag-and-drop functionality
- [ ] Test keyboard shortcuts
- [ ] Test tooltip positioning
- [ ] Performance testing for animations

### Acceptance Criteria
- [ ] Drag-and-drop works smoothly on touch devices
- [ ] Keyboard shortcuts are discoverable
- [ ] Tooltips don't interfere with interactions
- [ ] Loading states provide good user feedback
- [ ] All animations respect accessibility preferences

### Risk Mitigation
- Test on various devices and browsers
- Ensure animations don't cause seizures
- Provide alternatives for complex interactions

---

## **Integration Testing**
**Duration:** 3-4 hours

### Comprehensive UI Testing
- [ ] Cross-browser compatibility testing
- [ ] Mobile device testing
- [ ] Accessibility compliance testing
- [ ] Performance testing with animations
- [ ] User acceptance testing

### Success Metrics
- [ ] Lighthouse accessibility score >95
- [ ] Mobile-friendly test passes
- [ ] Page load time <2 seconds
- [ ] Smooth 60fps animations
- [ ] Zero critical accessibility violations

---

## **Rollback Plan**
If any cycle fails:
1. Revert to previous UI version
2. Disable problematic features via feature flags
3. Use simplified UI components temporarily
4. Disable animations if performance issues occur