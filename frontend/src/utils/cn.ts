import { clsx, type ClassValue } from 'clsx';

/**
 * Utility function for merging class names conditionally
 * Combines clsx functionality with Tailwind CSS class merging
 * 
 * @param inputs - Class values to merge
 * @returns Merged class string
 */
export function cn(...inputs: ClassValue[]): string {
  return clsx(inputs);
}

/**
 * Variant utility for creating component variant systems
 * 
 * @param base - Base classes that are always applied
 * @param variants - Object mapping variant keys to class values
 * @param defaultVariants - Default variant values
 * @returns Function that takes variant props and returns merged classes
 */
export function cva(
  base: string,
  options?: {
    variants?: Record<string, Record<string, string>>;
    defaultVariants?: Record<string, string>;
  }
) {
  return (props?: Record<string, string>) => {
    if (!options?.variants) return base;
    
    const { variants, defaultVariants } = options;
    const mergedProps = { ...defaultVariants, ...props };
    
    const variantClasses = Object.entries(mergedProps)
      .map(([key, value]) => {
        const variant = variants[key];
        return variant?.[value] || '';
      })
      .filter(Boolean);
    
    return cn(base, ...variantClasses);
  };
}