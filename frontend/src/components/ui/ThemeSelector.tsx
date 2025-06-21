import { useState } from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { themes } from '../../themes';
import { ThemeName } from '../../types/theme';
import { cn } from '../../utils/cn';

interface ThemeSelectorProps {
  className?: string;
  showLabels?: boolean;
  size?: 'sm' | 'md' | 'lg';
  orientation?: 'horizontal' | 'vertical';
}

export function ThemeSelector({ 
  className, 
  showLabels = true, 
  size = 'md',
  orientation = 'horizontal' 
}: ThemeSelectorProps) {
  const { themeName, setTheme, isThemeLoading } = useTheme();
  const [hoveredTheme, setHoveredTheme] = useState<ThemeName | null>(null);

  const sizeClasses = {
    sm: 'w-6 h-6',
    md: 'w-8 h-8',
    lg: 'w-10 h-10',
  };

  const getThemePreviewColors = (theme: ThemeName) => {
    const themeConfig = themes[theme];
    return {
      primary: themeConfig.colors.primary,
      secondary: themeConfig.colors.secondary,
      accent: themeConfig.colors.accent,
      background: themeConfig.colors.background,
    };
  };

  const handleThemeChange = (newTheme: ThemeName) => {
    if (!isThemeLoading) {
      setTheme(newTheme);
    }
  };

  const renderThemeButton = (theme: ThemeName) => {
    const themeConfig = themes[theme];
    const colors = getThemePreviewColors(theme);
    const isSelected = themeName === theme;

    return (
      <button
        key={theme}
        onClick={() => handleThemeChange(theme)}
        onMouseEnter={() => setHoveredTheme(theme)}
        onMouseLeave={() => setHoveredTheme(null)}
        disabled={isThemeLoading}
        className={cn(
          'relative rounded-lg border-2 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2',
          sizeClasses[size],
          isSelected 
            ? 'border-theme-primary ring-2 ring-theme-primary ring-offset-2' 
            : 'border-gray-300 hover:border-gray-400',
          isThemeLoading && 'opacity-50 cursor-not-allowed',
          'group'
        )}
        aria-label={`Switch to ${themeConfig.displayName}`}
        title={themeConfig.description}
      >
        {/* Theme preview with multiple colors */}
        <div className="w-full h-full rounded-md overflow-hidden relative">
          {/* Background quadrant */}
          <div 
            className="absolute inset-0"
            style={{ backgroundColor: colors.background }}
          />
          
          {/* Primary color triangle */}
          <div 
            className="absolute inset-0"
            style={{ 
              background: `linear-gradient(135deg, ${colors.primary} 0%, ${colors.primary} 50%, transparent 50%)` 
            }}
          />
          
          {/* Secondary color triangle */}
          <div 
            className="absolute inset-0"
            style={{ 
              background: `linear-gradient(315deg, ${colors.secondary} 0%, ${colors.secondary} 50%, transparent 50%)` 
            }}
          />
          
          {/* Accent color dot */}
          <div 
            className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-1.5 h-1.5 rounded-full"
            style={{ backgroundColor: colors.accent }}
          />
          
          {/* Special effects for certain themes */}
          {theme === 'vibe' && (
            <div className={cn(
              'absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity',
              'bg-gradient-to-r from-pink-500/20 to-cyan-500/20'
            )} />
          )}
          
          {theme === 'encode' && (
            <div className={cn(
              'absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity',
              'shadow-inner'
            )}
            style={{ boxShadow: `inset 0 0 10px ${colors.accent}40` }}
            />
          )}
          
          {/* Loading overlay */}
          {isThemeLoading && isSelected && (
            <div className="absolute inset-0 bg-white/50 flex items-center justify-center">
              <div className="w-2 h-2 bg-gray-600 rounded-full animate-pulse" />
            </div>
          )}
        </div>
        
        {/* Selected indicator */}
        {isSelected && (
          <div className="absolute -top-1 -right-1 w-3 h-3 bg-theme-primary rounded-full border-2 border-white shadow-sm">
            <svg className="w-2 h-2 text-white absolute top-0.5 left-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
            </svg>
          </div>
        )}
      </button>
    );
  };

  return (
    <div className={cn('theme-selector', className)}>
      {/* Theme buttons */}
      <div className={cn(
        'flex gap-2',
        orientation === 'vertical' ? 'flex-col' : 'flex-row flex-wrap'
      )}>
        {Object.keys(themes).map((theme) => renderThemeButton(theme as ThemeName))}
      </div>
      
      {/* Theme labels */}
      {showLabels && (
        <div className={cn(
          'mt-3 space-y-1',
          orientation === 'horizontal' && 'grid grid-cols-2 gap-1'
        )}>
          {Object.entries(themes).map(([key, theme]) => (
            <button
              key={key}
              onClick={() => handleThemeChange(key as ThemeName)}
              disabled={isThemeLoading}
              className={cn(
                'text-left text-xs px-2 py-1 rounded transition-colors',
                themeName === key 
                  ? 'bg-theme-primary text-theme-text-inverse font-medium' 
                  : 'text-theme-text-secondary hover:bg-theme-surface-alt',
                isThemeLoading && 'opacity-50 cursor-not-allowed'
              )}
            >
              {theme.displayName}
            </button>
          ))}
        </div>
      )}
      
      {/* Current theme info */}
      {hoveredTheme && (
        <div className="mt-2 p-2 bg-theme-surface-alt rounded text-xs">
          <div className="font-medium text-theme-text-primary">
            {themes[hoveredTheme].displayName}
          </div>
          <div className="text-theme-text-secondary">
            {themes[hoveredTheme].description}
          </div>
        </div>
      )}
    </div>
  );
}

// Compact version for headers/toolbars
export function CompactThemeSelector({ className }: { className?: string }) {
  const { themeName } = useTheme();
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className={cn('relative', className)}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-2 px-3 py-1.5 text-sm bg-theme-surface border border-theme-border-default rounded-md hover:bg-theme-surface-alt transition-colors"
      >
        <div className="w-4 h-4 rounded border border-theme-border-light overflow-hidden">
          <div className="w-full h-full" style={{ backgroundColor: themes[themeName].colors.primary }} />
        </div>
        <span className="text-theme-text-secondary">
          {themes[themeName].displayName}
        </span>
        <svg className="w-4 h-4 text-theme-text-tertiary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      
      {isOpen && (
        <>
          <div 
            className="fixed inset-0 z-40" 
            onClick={() => setIsOpen(false)} 
          />
          <div className="absolute top-full left-0 mt-1 z-50 bg-theme-surface border border-theme-border-default rounded-md shadow-theme-lg p-2 min-w-[200px]">
            <ThemeSelector 
              showLabels={true} 
              size="sm" 
              orientation="vertical"
            />
          </div>
        </>
      )}
    </div>
  );
}

export default ThemeSelector;