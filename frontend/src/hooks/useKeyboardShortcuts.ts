import React, { useEffect, useCallback, useRef, useState } from 'react';

interface Shortcut {
  key: string;
  ctrlKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
  metaKey?: boolean;
  action: () => void;
  description?: string;
  category?: string;
  preventDefault?: boolean;
  stopPropagation?: boolean;
  enabled?: boolean;
}

interface KeyboardShortcutsOptions {
  shortcuts: Shortcut[];
  enabled?: boolean;
  target?: HTMLElement | Document;
  capture?: boolean;
}

export const useKeyboardShortcuts = ({
  shortcuts,
  enabled = true,
  target,
  capture = false
}: KeyboardShortcutsOptions) => {
  const shortcutsRef = useRef(shortcuts);
  shortcutsRef.current = shortcuts;

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;

    const activeShortcuts = shortcutsRef.current.filter(shortcut => {
      if (shortcut.enabled === false) return false;
      
      const keyMatches = shortcut.key.toLowerCase() === event.key.toLowerCase();
      const ctrlMatches = !!shortcut.ctrlKey === event.ctrlKey;
      const shiftMatches = !!shortcut.shiftKey === event.shiftKey;
      const altMatches = !!shortcut.altKey === event.altKey;
      const metaMatches = !!shortcut.metaKey === event.metaKey;

      return keyMatches && ctrlMatches && shiftMatches && altMatches && metaMatches;
    });

    if (activeShortcuts.length > 0) {
      const shortcut = activeShortcuts[0]; // Take first match
      
      if (shortcut.preventDefault !== false) {
        event.preventDefault();
      }
      
      if (shortcut.stopPropagation) {
        event.stopPropagation();
      }
      
      shortcut.action();
    }
  }, [enabled]);

  useEffect(() => {
    const targetElement = target || document;
    
    targetElement.addEventListener('keydown', handleKeyDown as EventListener, capture);
    
    return () => {
      targetElement.removeEventListener('keydown', handleKeyDown as EventListener, capture);
    };
  }, [handleKeyDown, target, capture]);

  return {
    shortcuts: shortcutsRef.current
  };
};

// Global shortcuts manager
class ShortcutsManager {
  private shortcuts: Map<string, Shortcut> = new Map();
  private listeners: Set<(shortcuts: Shortcut[]) => void> = new Set();

  register(id: string, shortcut: Shortcut) {
    this.shortcuts.set(id, shortcut);
    this.notifyListeners();
  }

  unregister(id: string) {
    this.shortcuts.delete(id);
    this.notifyListeners();
  }

  getAll(): Shortcut[] {
    return Array.from(this.shortcuts.values());
  }

  getByCategory(category: string): Shortcut[] {
    return this.getAll().filter(s => s.category === category);
  }

  subscribe(listener: (shortcuts: Shortcut[]) => void) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  private notifyListeners() {
    this.listeners.forEach(listener => listener(this.getAll()));
  }

  formatShortcut(shortcut: Shortcut): string {
    const parts: string[] = [];
    
    if (shortcut.ctrlKey) parts.push('Ctrl');
    if (shortcut.metaKey) parts.push('⌘');
    if (shortcut.altKey) parts.push('Alt');
    if (shortcut.shiftKey) parts.push('Shift');
    
    parts.push(shortcut.key.toUpperCase());
    
    return parts.join(' + ');
  }
}

export const shortcutsManager = new ShortcutsManager();

// Hook to register a global shortcut
export const useGlobalShortcut = (
  id: string,
  shortcut: Omit<Shortcut, 'action'>,
  action: () => void,
  deps: React.DependencyList = []
) => {
  const actionRef = useRef(action);
  actionRef.current = action;

  useEffect(() => {
    const fullShortcut: Shortcut = {
      ...shortcut,
      action: () => actionRef.current()
    };
    
    shortcutsManager.register(id, fullShortcut);
    
    return () => {
      shortcutsManager.unregister(id);
    };
  }, [id, ...deps]);
};

// Hook to listen to all global shortcuts
export const useGlobalShortcuts = () => {
  const [shortcuts, setShortcuts] = useState<Shortcut[]>([]);

  useEffect(() => {
    setShortcuts(shortcutsManager.getAll());
    
    const unsubscribe = shortcutsManager.subscribe(setShortcuts);
    return () => {
      unsubscribe();
    };
  }, []);

  useKeyboardShortcuts({
    shortcuts,
    enabled: true
  });

  return shortcuts;
};

// Common shortcut configurations
export const commonShortcuts = {
  copy: { key: 'c', ctrlKey: true, description: 'Copy' },
  paste: { key: 'v', ctrlKey: true, description: 'Paste' },
  cut: { key: 'x', ctrlKey: true, description: 'Cut' },
  undo: { key: 'z', ctrlKey: true, description: 'Undo' },
  redo: { key: 'y', ctrlKey: true, description: 'Redo' },
  save: { key: 's', ctrlKey: true, description: 'Save' },
  find: { key: 'f', ctrlKey: true, description: 'Find' },
  selectAll: { key: 'a', ctrlKey: true, description: 'Select All' },
  escape: { key: 'Escape', description: 'Close/Cancel' },
  enter: { key: 'Enter', description: 'Confirm/Submit' },
  delete: { key: 'Delete', description: 'Delete' },
  backspace: { key: 'Backspace', description: 'Delete/Back' },
  
  // Navigation
  arrowUp: { key: 'ArrowUp', description: 'Move Up' },
  arrowDown: { key: 'ArrowDown', description: 'Move Down' },
  arrowLeft: { key: 'ArrowLeft', description: 'Move Left' },
  arrowRight: { key: 'ArrowRight', description: 'Move Right' },
  home: { key: 'Home', description: 'Go to Start' },
  end: { key: 'End', description: 'Go to End' },
  pageUp: { key: 'PageUp', description: 'Page Up' },
  pageDown: { key: 'PageDown', description: 'Page Down' },
  
  // Function keys
  f1: { key: 'F1', description: 'Help' },
  f5: { key: 'F5', description: 'Refresh' },
  f11: { key: 'F11', description: 'Fullscreen' },
  
  // Custom app shortcuts
  commandPalette: { key: 'k', ctrlKey: true, description: 'Command Palette' },
  quickSearch: { key: '/', description: 'Quick Search' },
  newItem: { key: 'n', ctrlKey: true, description: 'New Item' },
  settings: { key: ',', ctrlKey: true, description: 'Settings' },
  help: { key: '?', shiftKey: true, description: 'Help' }
};

// Hook for sequence shortcuts (like vim-style commands)
export const useSequenceShortcuts = (
  sequences: Array<{
    sequence: string[];
    action: () => void;
    description?: string;
    timeout?: number;
  }>,
  enabled = true
) => {
  const [currentSequence, setCurrentSequence] = useState<string[]>([]);
  const timeoutRef = useRef<NodeJS.Timeout>();

  const resetSequence = useCallback(() => {
    setCurrentSequence([]);
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
  }, []);

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;

    const key = event.key.toLowerCase();
    const newSequence = [...currentSequence, key];

    // Check if any sequence starts with this
    const matchingSequences = sequences.filter(seq => 
      seq.sequence.slice(0, newSequence.length).every((k, i) => k === newSequence[i])
    );

    if (matchingSequences.length === 0) {
      resetSequence();
      return;
    }

    // Check for exact match
    const exactMatch = matchingSequences.find(seq => 
      seq.sequence.length === newSequence.length
    );

    if (exactMatch) {
      event.preventDefault();
      exactMatch.action();
      resetSequence();
    } else {
      // Continue sequence
      setCurrentSequence(newSequence);
      
      // Set timeout to reset sequence
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      
      timeoutRef.current = setTimeout(resetSequence, 2000);
    }
  }, [enabled, currentSequence, sequences, resetSequence]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return {
    currentSequence,
    resetSequence
  };
};

export default useKeyboardShortcuts;