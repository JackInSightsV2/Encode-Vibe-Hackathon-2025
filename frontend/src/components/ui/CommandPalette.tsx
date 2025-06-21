import { useState, useEffect, useRef, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { useKeyboardNavigation } from '../../hooks/useKeyboardNavigation';
import { useGlobalShortcut } from '../../hooks/useKeyboardShortcuts';
import { FocusTrap } from './FocusTrap';
import { cn } from '../../utils/cn';

interface Command {
  id: string;
  title: string;
  description?: string;
  category?: string;
  keywords?: string[];
  icon?: ReactNode;
  shortcut?: string;
  action: () => void;
  enabled?: boolean;
}

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
  commands: Command[];
  placeholder?: string;
  maxResults?: number;
  className?: string;
  recentCommands?: string[];
  onCommandExecute?: (command: Command) => void;
}

export const CommandPalette = ({
  isOpen,
  onClose,
  commands,
  placeholder = 'Type a command or search...',
  maxResults = 10,
  className,
  recentCommands = [],
  onCommandExecute
}: CommandPaletteProps) => {
  const [query, setQuery] = useState('');
  const [filteredCommands, setFilteredCommands] = useState<Command[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);

  const commandIds = filteredCommands.map(cmd => cmd.id);
  const { activeIndex, setActiveIndex, containerRef } = useKeyboardNavigation({
    items: commandIds,
    onSelect: (id) => {
      const command = filteredCommands.find(cmd => cmd.id === id);
      if (command) {
        executeCommand(command);
      }
    }
  });

  const executeCommand = (command: Command) => {
    if (command.enabled === false) return;
    
    command.action();
    onCommandExecute?.(command);
    onClose();
    setQuery('');
  };

  const filterCommands = (searchQuery: string) => {
    if (!searchQuery.trim()) {
      // Show recent commands first, then all commands
      const recent = commands.filter(cmd => recentCommands.includes(cmd.id));
      const others = commands.filter(cmd => !recentCommands.includes(cmd.id));
      return [...recent, ...others].slice(0, maxResults);
    }

    const normalizedQuery = searchQuery.toLowerCase().trim();
    
    const scored = commands
      .filter(cmd => cmd.enabled !== false)
      .map(command => {
        let score = 0;
        
        // Title exact match
        if (command.title.toLowerCase() === normalizedQuery) {
          score += 100;
        }
        // Title starts with query
        else if (command.title.toLowerCase().startsWith(normalizedQuery)) {
          score += 80;
        }
        // Title contains query
        else if (command.title.toLowerCase().includes(normalizedQuery)) {
          score += 60;
        }
        
        // Description match
        if (command.description?.toLowerCase().includes(normalizedQuery)) {
          score += 30;
        }
        
        // Keywords match
        if (command.keywords?.some(keyword => 
          keyword.toLowerCase().includes(normalizedQuery)
        )) {
          score += 40;
        }
        
        // Category match
        if (command.category?.toLowerCase().includes(normalizedQuery)) {
          score += 20;
        }
        
        // Recent commands bonus
        if (recentCommands.includes(command.id)) {
          score += 10;
        }

        return { command, score };
      })
      .filter(({ score }) => score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, maxResults)
      .map(({ command }) => command);

    return scored;
  };

  useEffect(() => {
    setFilteredCommands(filterCommands(query));
    setActiveIndex(0);
  }, [query, commands, recentCommands, maxResults]);

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(e.target.value);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    }
  };

  const groupedCommands = filteredCommands.reduce((groups, command) => {
    const category = command.category || 'Other';
    if (!groups[category]) {
      groups[category] = [];
    }
    groups[category].push(command);
    return groups;
  }, {} as Record<string, Command[]>);

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-50 backdrop-blur-sm"
        onClick={onClose}
      />
      
      {/* Command Palette */}
      <FocusTrap active={isOpen} onDeactivate={onClose}>
        <div className={cn(
          'relative w-full max-w-2xl mx-4',
          'bg-theme-surface border border-theme-border-default rounded-lg shadow-theme-lg',
          'animate-scale-in',
          className
        )}>
          {/* Search Input */}
          <div className="p-4 border-b border-theme-border-default">
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-theme-text-tertiary"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                ref={inputRef}
                type="text"
                value={query}
                onChange={handleInputChange}
                onKeyDown={handleKeyDown}
                placeholder={placeholder}
                className="w-full pl-10 pr-4 py-3 bg-transparent border-none outline-none text-theme-text-primary placeholder-theme-text-tertiary text-lg"
              />
            </div>
          </div>

          {/* Results */}
          <div
            ref={containerRef as React.RefObject<HTMLDivElement>}
            className="max-h-96 overflow-y-auto"
            tabIndex={0}
          >
            {filteredCommands.length === 0 ? (
              <div className="p-8 text-center text-theme-text-tertiary">
                <svg className="w-12 h-12 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                {query ? 'No commands found' : 'Start typing to search commands'}
              </div>
            ) : (
              <div className="py-2">
                {query === '' && recentCommands.length > 0 && (
                  <div className="px-4 py-2">
                    <h3 className="text-sm font-semibold text-theme-text-secondary mb-2">
                      Recent Commands
                    </h3>
                  </div>
                )}
                
                {Object.entries(groupedCommands).map(([category, categoryCommands]) => (
                  <div key={category}>
                    {query !== '' && Object.keys(groupedCommands).length > 1 && (
                      <div className="px-4 py-2 border-t border-theme-border-light first:border-t-0">
                        <h3 className="text-sm font-semibold text-theme-text-secondary">
                          {category}
                        </h3>
                      </div>
                    )}
                    
                    {categoryCommands.map((command, _index) => {
                      const globalIndex = filteredCommands.indexOf(command);
                      const isActive = globalIndex === activeIndex;
                      const isRecent = recentCommands.includes(command.id);
                      
                      return (
                        <button
                          key={command.id}
                          onClick={() => executeCommand(command)}
                          disabled={command.enabled === false}
                          className={cn(
                            'w-full px-4 py-3 text-left flex items-center space-x-3 transition-colors',
                            'hover:bg-theme-surface-alt focus:bg-theme-surface-alt',
                            isActive && 'bg-theme-primary text-theme-text-inverse',
                            command.enabled === false && 'opacity-50 cursor-not-allowed'
                          )}
                          tabIndex={-1}
                        >
                          {/* Icon */}
                          <div className="flex-shrink-0 w-5 h-5 flex items-center justify-center">
                            {command.icon || (
                              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" className="w-4 h-4">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                              </svg>
                            )}
                          </div>
                          
                          {/* Content */}
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center justify-between">
                              <span className="font-medium truncate">
                                {command.title}
                              </span>
                              <div className="flex items-center space-x-2 ml-2">
                                {isRecent && (
                                  <span className="text-xs px-1.5 py-0.5 bg-theme-accent text-theme-text-inverse rounded">
                                    Recent
                                  </span>
                                )}
                                {command.shortcut && (
                                  <kbd className="text-xs px-1.5 py-0.5 bg-theme-surface-alt border border-theme-border-default rounded">
                                    {command.shortcut}
                                  </kbd>
                                )}
                              </div>
                            </div>
                            {command.description && (
                              <p className="text-sm opacity-75 truncate mt-0.5">
                                {command.description}
                              </p>
                            )}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="p-3 border-t border-theme-border-default bg-theme-surface-alt">
            <div className="flex items-center justify-between text-xs text-theme-text-tertiary">
              <div className="flex items-center space-x-4">
                <span className="flex items-center space-x-1">
                  <kbd className="px-1 py-0.5 bg-theme-background border border-theme-border-default rounded">↑↓</kbd>
                  <span>Navigate</span>
                </span>
                <span className="flex items-center space-x-1">
                  <kbd className="px-1 py-0.5 bg-theme-background border border-theme-border-default rounded">↵</kbd>
                  <span>Select</span>
                </span>
                <span className="flex items-center space-x-1">
                  <kbd className="px-1 py-0.5 bg-theme-background border border-theme-border-default rounded">Esc</kbd>
                  <span>Close</span>
                </span>
              </div>
              <span>{filteredCommands.length} results</span>
            </div>
          </div>
        </div>
      </FocusTrap>
    </div>,
    document.body
  );
};

// Hook to manage command palette state
export const useCommandPalette = (_commands: Command[]) => {
  const [isOpen, setIsOpen] = useState(false);
  const [recentCommands, setRecentCommands] = useState<string[]>([]);

  // Register global shortcut
  useGlobalShortcut(
    'command-palette',
    { key: 'k', ctrlKey: true, description: 'Open command palette', category: 'Navigation' },
    () => setIsOpen(true)
  );

  const handleCommandExecute = (command: Command) => {
    setRecentCommands(prev => {
      const filtered = prev.filter(id => id !== command.id);
      return [command.id, ...filtered].slice(0, 5); // Keep last 5
    });
  };

  return {
    isOpen,
    setIsOpen,
    recentCommands,
    handleCommandExecute
  };
};

export default CommandPalette;