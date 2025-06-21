import { useState, useEffect, useRef, useMemo, ReactNode } from 'react';
import { cn } from '../../utils/cn';

interface VirtualizedListProps<T> {
  items: T[];
  itemHeight: number | ((index: number, item: T) => number);
  containerHeight: number;
  renderItem: (item: T, index: number, style: React.CSSProperties) => ReactNode;
  overscan?: number;
  className?: string;
  onScroll?: (scrollTop: number) => void;
  scrollToIndex?: number;
  horizontal?: boolean;
  cache?: boolean;
}

interface VirtualItem {
  index: number;
  start: number;
  size: number;
}

export function VirtualizedList<T>({
  items,
  itemHeight,
  containerHeight,
  renderItem,
  overscan = 3,
  className,
  onScroll,
  scrollToIndex,
  horizontal = false,
  cache = true
}: VirtualizedListProps<T>) {
  const [scrollTop, setScrollTop] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const heightCache = useRef<Map<number, number>>(new Map());

  const getItemHeight = (index: number): number => {
    if (typeof itemHeight === 'number') return itemHeight;
    
    if (cache && heightCache.current.has(index)) {
      return heightCache.current.get(index)!;
    }
    
    const height = itemHeight(index, items[index]);
    if (cache) {
      heightCache.current.set(index, height);
    }
    
    return height;
  };

  const virtualItems = useMemo(() => {
    const itemCount = items.length;
    const virtualItems: VirtualItem[] = [];
    let start = 0;

    for (let i = 0; i < itemCount; i++) {
      const size = getItemHeight(i);
      virtualItems.push({
        index: i,
        start,
        size
      });
      start += size;
    }

    return virtualItems;
  }, [items, itemHeight, cache]);

  const totalSize = virtualItems.reduce((acc, item) => acc + item.size, 0);

  const visibleRange = useMemo(() => {
    const containerSize = containerHeight;
    const scrollOffset = scrollTop;

    let start = 0;
    let end = virtualItems.length - 1;

    // Find first visible item
    for (let i = 0; i < virtualItems.length; i++) {
      const item = virtualItems[i];
      if (item.start + item.size >= scrollOffset) {
        start = Math.max(0, i - overscan);
        break;
      }
    }

    // Find last visible item
    for (let i = start; i < virtualItems.length; i++) {
      const item = virtualItems[i];
      if (item.start > scrollOffset + containerSize) {
        end = Math.min(virtualItems.length - 1, i + overscan);
        break;
      }
    }

    return { start, end };
  }, [virtualItems, scrollTop, containerHeight, overscan]);

  const visibleItems = useMemo(() => {
    const items: VirtualItem[] = [];
    for (let i = visibleRange.start; i <= visibleRange.end; i++) {
      if (virtualItems[i]) {
        items.push(virtualItems[i]);
      }
    }
    return items;
  }, [virtualItems, visibleRange]);

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const scrollTop = e.currentTarget.scrollTop;
    setScrollTop(scrollTop);
    onScroll?.(scrollTop);
  };

  // Scroll to specific index
  useEffect(() => {
    if (scrollToIndex !== undefined && containerRef.current && virtualItems[scrollToIndex]) {
      const item = virtualItems[scrollToIndex];
      containerRef.current.scrollTop = item.start;
    }
  }, [scrollToIndex, virtualItems]);

  const style: React.CSSProperties = horizontal
    ? { width: containerHeight, height: totalSize, overflowX: 'auto', overflowY: 'hidden' }
    : { height: containerHeight, overflowY: 'auto' };

  return (
    <div
      ref={containerRef}
      className={cn('relative', className)}
      style={style}
      onScroll={handleScroll}
    >
      <div
        style={horizontal
          ? { display: 'flex', width: totalSize }
          : { height: totalSize }
        }
      >
        {visibleItems.map((virtualItem) => {
          const item = items[virtualItem.index];
          const itemStyle: React.CSSProperties = horizontal
            ? {
                position: 'absolute',
                left: virtualItem.start,
                width: virtualItem.size,
                height: '100%'
              }
            : {
                position: 'absolute',
                top: virtualItem.start,
                height: virtualItem.size,
                width: '100%'
              };

          return (
            <div key={virtualItem.index} style={itemStyle}>
              {renderItem(item, virtualItem.index, itemStyle)}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// Hook for infinite loading
export const useInfiniteScroll = <T,>(
  loadMore: () => Promise<T[]>,
  hasMore: boolean,
  threshold = 0.8
) => {
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<T[]>([]);

  const handleScroll = async (scrollTop: number, containerHeight: number, totalHeight: number) => {
    if (loading || !hasMore) return;

    const scrollPercentage = (scrollTop + containerHeight) / totalHeight;
    
    if (scrollPercentage >= threshold) {
      setLoading(true);
      try {
        const newItems = await loadMore();
        setItems(prev => [...prev, ...newItems]);
      } catch (error) {
        console.error('Error loading more items:', error);
      } finally {
        setLoading(false);
      }
    }
  };

  return {
    items,
    setItems,
    loading,
    handleScroll
  };
};

// Grid virtualization
interface VirtualizedGridProps<T> {
  items: T[];
  itemWidth: number;
  itemHeight: number;
  containerWidth: number;
  containerHeight: number;
  renderItem: (item: T, index: number, style: React.CSSProperties) => ReactNode;
  gap?: number;
  overscan?: number;
  className?: string;
}

export function VirtualizedGrid<T>({
  items,
  itemWidth,
  itemHeight,
  containerWidth,
  containerHeight,
  renderItem,
  gap = 0,
  overscan = 3,
  className
}: VirtualizedGridProps<T>) {
  const [scrollTop, setScrollTop] = useState(0);
  const [_scrollLeft, setScrollLeft] = useState(0);

  const columnsCount = Math.floor(containerWidth / (itemWidth + gap));
  const rowsCount = Math.ceil(items.length / columnsCount);

  const visibleRowStart = Math.max(0, Math.floor(scrollTop / (itemHeight + gap)) - overscan);
  const visibleRowEnd = Math.min(
    rowsCount - 1,
    Math.ceil((scrollTop + containerHeight) / (itemHeight + gap)) + overscan
  );

  const visibleItems = [];
  for (let row = visibleRowStart; row <= visibleRowEnd; row++) {
    for (let col = 0; col < columnsCount; col++) {
      const index = row * columnsCount + col;
      if (index < items.length) {
        visibleItems.push({
          item: items[index],
          index,
          x: col * (itemWidth + gap),
          y: row * (itemHeight + gap)
        });
      }
    }
  }

  const totalHeight = rowsCount * (itemHeight + gap) - gap;

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop);
    setScrollLeft(e.currentTarget.scrollLeft);
  };

  return (
    <div
      className={cn('relative overflow-auto', className)}
      style={{ width: containerWidth, height: containerHeight }}
      onScroll={handleScroll}
    >
      <div style={{ height: totalHeight, position: 'relative' }}>
        {visibleItems.map(({ item, index, x, y }) => {
          const style: React.CSSProperties = {
            position: 'absolute',
            left: x,
            top: y,
            width: itemWidth,
            height: itemHeight
          };

          return (
            <div key={index} style={style}>
              {renderItem(item, index, style)}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// Table virtualization
interface VirtualizedTableProps<T> {
  data: T[];
  columns: Array<{
    key: keyof T;
    header: string;
    width: number;
    render?: (value: any, item: T, index: number) => ReactNode;
  }>;
  rowHeight: number;
  containerHeight: number;
  className?: string;
  onRowClick?: (item: T, index: number) => void;
}

export function VirtualizedTable<T extends Record<string, any>>({
  data,
  columns,
  rowHeight,
  containerHeight,
  className,
  onRowClick
}: VirtualizedTableProps<T>) {
  // const _totalWidth = columns.reduce((sum, col) => sum + col.width, 0);

  const renderRow = (item: T, index: number, style: React.CSSProperties) => (
    <div
      style={style}
      className={cn(
        'flex border-b border-theme-border-light hover:bg-theme-surface-alt transition-colors',
        onRowClick && 'cursor-pointer'
      )}
      onClick={() => onRowClick?.(item, index)}
    >
      {columns.map((column) => (
        <div
          key={String(column.key)}
          style={{ width: column.width }}
          className="px-4 py-2 flex items-center overflow-hidden"
        >
          {column.render
            ? column.render(item[column.key], item, index)
            : String(item[column.key] || '')
          }
        </div>
      ))}
    </div>
  );

  return (
    <div className={cn('border border-theme-border-default rounded-lg overflow-hidden', className)}>
      {/* Header */}
      <div className="flex bg-theme-surface-alt border-b border-theme-border-default">
        {columns.map((column) => (
          <div
            key={String(column.key)}
            style={{ width: column.width }}
            className="px-4 py-3 font-semibold text-theme-text-primary"
          >
            {column.header}
          </div>
        ))}
      </div>

      {/* Virtualized body */}
      <VirtualizedList
        items={data}
        itemHeight={rowHeight}
        containerHeight={containerHeight - 48} // Subtract header height
        renderItem={renderRow}
        overscan={5}
      />
    </div>
  );
}

export default VirtualizedList;