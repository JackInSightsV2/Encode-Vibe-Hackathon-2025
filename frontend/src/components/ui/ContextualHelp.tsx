import { useState, useRef, useEffect, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { AdvancedTooltip } from './AdvancedTooltip';
import { AccessibleButton } from './AccessibleButton';
import { Modal } from './Modal';
import { cn } from '../../utils/cn';

interface HelpItem {
  id: string;
  title: string;
  content: ReactNode;
  category?: string;
  keywords?: string[];
  selector?: string;
  placement?: 'top' | 'bottom' | 'left' | 'right';
}

interface ContextualHelpProps {
  items: HelpItem[];
  children: ReactNode;
  className?: string;
}

// Help tooltip for specific elements
interface HelpTooltipProps {
  content: ReactNode;
  title?: string;
  placement?: 'top' | 'bottom' | 'left' | 'right';
  children: ReactNode;
  className?: string;
}

export const HelpTooltip = ({
  content,
  title,
  placement = 'top',
  children,
  className
}: HelpTooltipProps) => {
  return (
    <div className={cn('relative inline-block', className)}>
      {children}
      <AdvancedTooltip
        content={
          <div className="max-w-xs">
            {title && (
              <div className="font-semibold text-theme-text-primary mb-2 border-b border-theme-border-light pb-1">
                {title}
              </div>
            )}
            <div className="text-sm text-theme-text-secondary">
              {content}
            </div>
          </div>
        }
        placement={placement}
        delay={500}
        interactive={true}
        maxWidth={300}
      >
        <button
          className="ml-1 text-theme-text-tertiary hover:text-theme-text-secondary transition-colors"
          aria-label="Help"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </button>
      </AdvancedTooltip>
    </div>
  );
};

// Guided tour system
interface TourStep {
  id: string;
  title: string;
  content: ReactNode;
  selector: string;
  placement?: 'top' | 'bottom' | 'left' | 'right';
  highlightPadding?: number;
  beforeShow?: () => void;
  afterShow?: () => void;
  beforeHide?: () => void;
  afterHide?: () => void;
}

interface GuidedTourProps {
  steps: TourStep[];
  isActive: boolean;
  onComplete: () => void;
  onSkip: () => void;
  showProgress?: boolean;
  className?: string;
}

export const GuidedTour = ({
  steps,
  isActive,
  onComplete,
  onSkip,
  showProgress = true,
  className
}: GuidedTourProps) => {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [highlightPosition, setHighlightPosition] = useState({ x: 0, y: 0, width: 0, height: 0 });
  const overlayRef = useRef<HTMLDivElement>(null);

  const currentStep = steps[currentStepIndex];
  const isLastStep = currentStepIndex === steps.length - 1;

  const updateHighlight = () => {
    if (!currentStep?.selector) return;

    const element = document.querySelector(currentStep.selector) as HTMLElement;
    if (!element) return;

    const rect = element.getBoundingClientRect();
    const padding = currentStep.highlightPadding || 8;

    setHighlightPosition({
      x: rect.left - padding,
      y: rect.top - padding,
      width: rect.width + padding * 2,
      height: rect.height + padding * 2
    });

    // Scroll element into view if needed
    element.scrollIntoView({ behavior: 'smooth', block: 'center' });
  };

  useEffect(() => {
    if (isActive && currentStep) {
      currentStep.beforeShow?.();
      updateHighlight();
      
      const handleResize = () => updateHighlight();
      window.addEventListener('resize', handleResize);
      
      return () => {
        window.removeEventListener('resize', handleResize);
        currentStep.afterHide?.();
      };
    }
  }, [isActive, currentStepIndex, currentStep]);

  const nextStep = () => {
    currentStep?.afterShow?.();
    
    if (isLastStep) {
      onComplete();
    } else {
      setCurrentStepIndex(prev => prev + 1);
    }
  };

  const prevStep = () => {
    if (currentStepIndex > 0) {
      currentStep?.beforeHide?.();
      setCurrentStepIndex(prev => prev - 1);
    }
  };

  const skipTour = () => {
    currentStep?.afterHide?.();
    onSkip();
  };

  if (!isActive || !currentStep) return null;

  return createPortal(
    <div className={cn('fixed inset-0 z-50', className)}>
      {/* Overlay with highlight cutout */}
      <div 
        ref={overlayRef}
        className="absolute inset-0 bg-black bg-opacity-50"
        style={{
          clipPath: `polygon(
            0% 0%, 
            0% 100%, 
            ${highlightPosition.x}px 100%, 
            ${highlightPosition.x}px ${highlightPosition.y}px, 
            ${highlightPosition.x + highlightPosition.width}px ${highlightPosition.y}px, 
            ${highlightPosition.x + highlightPosition.width}px ${highlightPosition.y + highlightPosition.height}px, 
            ${highlightPosition.x}px ${highlightPosition.y + highlightPosition.height}px, 
            ${highlightPosition.x}px 100%, 
            100% 100%, 
            100% 0%
          )`
        }}
      />

      {/* Highlighted element border */}
      <div
        className="absolute border-2 border-theme-primary rounded-lg animate-pulse"
        style={{
          left: highlightPosition.x,
          top: highlightPosition.y,
          width: highlightPosition.width,
          height: highlightPosition.height
        }}
      />

      {/* Tour popup */}
      <div className="absolute max-w-sm bg-theme-surface border border-theme-border-default rounded-lg shadow-theme-lg p-4">
        <div className="mb-4">
          <h3 className="text-lg font-semibold text-theme-text-primary mb-2">
            {currentStep.title}
          </h3>
          <div className="text-theme-text-secondary">
            {currentStep.content}
          </div>
        </div>

        {showProgress && (
          <div className="mb-4">
            <div className="flex justify-between text-sm text-theme-text-tertiary mb-1">
              <span>Step {currentStepIndex + 1} of {steps.length}</span>
              <span>{Math.round(((currentStepIndex + 1) / steps.length) * 100)}%</span>
            </div>
            <div className="w-full bg-theme-surface-alt rounded-full h-2">
              <div
                className="bg-theme-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${((currentStepIndex + 1) / steps.length) * 100}%` }}
              />
            </div>
          </div>
        )}

        <div className="flex justify-between items-center">
          <button
            onClick={skipTour}
            className="text-sm text-theme-text-tertiary hover:text-theme-text-secondary transition-colors"
          >
            Skip Tour
          </button>

          <div className="flex space-x-2">
            {currentStepIndex > 0 && (
              <AccessibleButton
                variant="outline"
                size="sm"
                onClick={prevStep}
              >
                Previous
              </AccessibleButton>
            )}
            
            <AccessibleButton
              variant="primary"
              size="sm"
              onClick={nextStep}
            >
              {isLastStep ? 'Finish' : 'Next'}
            </AccessibleButton>
          </div>
        </div>
      </div>
    </div>,
    document.body
  );
};

// Help center modal
interface HelpCenterProps {
  isOpen: boolean;
  onClose: () => void;
  items: HelpItem[];
  searchable?: boolean;
  categorized?: boolean;
}

export const HelpCenter = ({
  isOpen,
  onClose,
  items,
  searchable = true,
  categorized = true
}: HelpCenterProps) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [selectedItem, setSelectedItem] = useState<HelpItem | null>(null);

  const categories = categorized ? [...new Set(items.map(item => item.category).filter(Boolean))] : [];

  const filteredItems = items.filter(item => {
    const matchesSearch = !searchTerm || 
      item.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
      item.keywords?.some(keyword => keyword.toLowerCase().includes(searchTerm.toLowerCase()));
    
    const matchesCategory = !selectedCategory || item.category === selectedCategory;
    
    return matchesSearch && matchesCategory;
  });

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Help Center"
      size="lg"
    >
      <div className="flex h-96">
        {/* Sidebar */}
        <div className="w-1/3 border-r border-theme-border-default pr-4">
          {searchable && (
            <div className="mb-4">
              <input
                type="text"
                placeholder="Search help..."
                className="w-full px-3 py-2 border border-theme-border-default rounded-md"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
          )}

          {categorized && (
            <div className="mb-4">
              <button
                onClick={() => setSelectedCategory(null)}
                className={cn(
                  'block w-full text-left px-2 py-1 rounded',
                  !selectedCategory ? 'bg-theme-primary text-theme-text-inverse' : 'hover:bg-theme-surface-alt'
                )}
              >
                All Items
              </button>
              {categories.map(category => (
                <button
                  key={category}
                  onClick={() => setSelectedCategory(category!)}
                  className={cn(
                    'block w-full text-left px-2 py-1 rounded mt-1',
                    selectedCategory === category ? 'bg-theme-primary text-theme-text-inverse' : 'hover:bg-theme-surface-alt'
                  )}
                >
                  {category}
                </button>
              ))}
            </div>
          )}

          <div className="space-y-1 max-h-48 overflow-y-auto">
            {filteredItems.map(item => (
              <button
                key={item.id}
                onClick={() => setSelectedItem(item)}
                className={cn(
                  'block w-full text-left px-2 py-2 rounded text-sm',
                  selectedItem?.id === item.id ? 'bg-theme-primary text-theme-text-inverse' : 'hover:bg-theme-surface-alt'
                )}
              >
                {item.title}
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 pl-4">
          {selectedItem ? (
            <div>
              <h3 className="text-xl font-semibold text-theme-text-primary mb-4">
                {selectedItem.title}
              </h3>
              <div className="text-theme-text-secondary">
                {selectedItem.content}
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-theme-text-tertiary">
              <div className="text-center">
                <svg className="w-12 h-12 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <p>Select a help topic to get started</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </Modal>
  );
};

// Contextual help overlay
export const ContextualHelp = ({ items, children, className }: ContextualHelpProps) => {
  const [isHelpMode, setIsHelpMode] = useState(false);
  const [_highlightedElements, setHighlightedElements] = useState<HTMLElement[]>([]);

  useEffect(() => {
    if (isHelpMode) {
      const elements = items
        .filter(item => item.selector)
        .map(item => document.querySelector(item.selector!) as HTMLElement)
        .filter(Boolean);
      
      setHighlightedElements(elements);
      
      // Add help indicators to elements
      elements.forEach((element, _index) => {
        const indicator = document.createElement('div');
        indicator.className = 'help-indicator';
        indicator.style.cssText = `
          position: absolute;
          top: -8px;
          right: -8px;
          width: 20px;
          height: 20px;
          background: var(--color-primary);
          color: white;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 12px;
          font-weight: bold;
          z-index: 1000;
          cursor: pointer;
        `;
        indicator.textContent = '?';
        indicator.onclick = () => {
          // Show help for this element
          const helpItem = items.find(item => item.selector === `[data-help-id="${item.id}"]`);
          if (helpItem) {
            // Implementation for showing help details
          }
        };
        
        element.style.position = 'relative';
        element.appendChild(indicator);
      });
    } else {
      // Remove help indicators
      document.querySelectorAll('.help-indicator').forEach(indicator => {
        indicator.remove();
      });
    }

    return () => {
      document.querySelectorAll('.help-indicator').forEach(indicator => {
        indicator.remove();
      });
    };
  }, [isHelpMode, items]);

  return (
    <div className={className}>
      {children}
      
      {/* Help mode toggle */}
      <button
        onClick={() => setIsHelpMode(!isHelpMode)}
        className={cn(
          'fixed bottom-4 right-4 p-3 rounded-full shadow-lg z-50',
          'bg-theme-primary text-theme-text-inverse',
          'hover:bg-opacity-90 transition-all',
          isHelpMode && 'animate-pulse'
        )}
        aria-label={isHelpMode ? 'Exit help mode' : 'Enter help mode'}
      >
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </button>
    </div>
  );
};

export default ContextualHelp;