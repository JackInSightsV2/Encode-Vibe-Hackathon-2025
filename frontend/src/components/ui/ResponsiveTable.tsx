import React from 'react';
import { cn } from '../../utils/cn';
import { useIsMobile } from '../../hooks/useMediaQuery';
import { Card } from './Card';

interface Column<T> {
  key: keyof T;
  header: string;
  render?: (item: T) => React.ReactNode;
  hidden?: {
    mobile?: boolean;
    tablet?: boolean;
  };
  sortable?: boolean;
  className?: string;
}

interface ResponsiveTableProps<T> {
  data: T[];
  columns: Column<T>[];
  loading?: boolean;
  className?: string;
  mobileCardRenderer?: (item: T, index: number) => React.ReactNode;
  onSort?: (key: keyof T, direction: 'asc' | 'desc') => void;
  sortKey?: keyof T;
  sortDirection?: 'asc' | 'desc';
  emptyState?: React.ReactNode;
}

function ResponsiveTable<T extends Record<string, any>>({
  data,
  columns,
  loading = false,
  className,
  mobileCardRenderer,
  onSort,
  sortKey,
  sortDirection = 'asc',
  emptyState
}: ResponsiveTableProps<T>) {
  const isMobile = useIsMobile();

  // Loading state
  if (loading) {
    return (
      <div className="space-y-3">
        {isMobile ? (
          // Mobile loading skeleton
          Array.from({ length: 3 }).map((_, index) => (
            <Card key={index} padding="sm">
              <div className="animate-pulse space-y-3">
                <div className="h-4 bg-gray-200 rounded w-3/4"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2"></div>
                <div className="h-3 bg-gray-200 rounded w-2/3"></div>
              </div>
            </Card>
          ))
        ) : (
          // Desktop loading skeleton
          <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
            <div className="animate-pulse">
              <div className="border-b border-gray-200 bg-gray-50">
                <div className="flex">
                  {columns.map((_, index) => (
                    <div key={index} className="px-6 py-4 flex-1">
                      <div className="h-4 bg-gray-200 rounded w-20"></div>
                    </div>
                  ))}
                </div>
              </div>
              {Array.from({ length: 5 }).map((_, rowIndex) => (
                <div key={rowIndex} className="border-b border-gray-100">
                  <div className="flex">
                    {columns.map((_, colIndex) => (
                      <div key={colIndex} className="px-6 py-4 flex-1">
                        <div className="h-4 bg-gray-200 rounded w-16"></div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    );
  }

  // Empty state
  if (data.length === 0) {
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-12 text-center">
        {emptyState || (
          <>
            <svg
              className="mx-auto h-12 w-12 text-gray-400 mb-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1}
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
            <h3 className="text-sm font-medium text-gray-900 mb-1">No data available</h3>
            <p className="text-sm text-gray-500">There are no items to display at this time.</p>
          </>
        )}
      </div>
    );
  }

  // Handle sort
  const handleSort = (key: keyof T) => {
    if (!onSort) return;
    
    const newDirection = sortKey === key && sortDirection === 'asc' ? 'desc' : 'asc';
    onSort(key, newDirection);
  };

  // Mobile card view
  if (isMobile && mobileCardRenderer) {
    return (
      <div className={cn('space-y-3', className)}>
        {data.map((item, index) => (
          <Card key={index} padding="sm" className="transition-shadow hover:shadow-md">
            {mobileCardRenderer(item, index)}
          </Card>
        ))}
      </div>
    );
  }

  // Mobile fallback - simplified table view
  if (isMobile) {
    const visibleColumns = columns.filter(col => !col.hidden?.mobile);
    
    return (
      <div className={cn('space-y-3', className)}>
        {data.map((item, index) => (
          <Card key={index} padding="sm" className="transition-shadow hover:shadow-md">
            <div className="space-y-2">
              {visibleColumns.map((column) => (
                <div key={String(column.key)} className="flex justify-between items-start">
                  <span className="text-sm font-medium text-gray-500 min-w-0 flex-1">
                    {column.header}:
                  </span>
                  <span className="text-sm text-gray-900 ml-2 text-right">
                    {column.render ? column.render(item) : String(item[column.key] || '')}
                  </span>
                </div>
              ))}
            </div>
          </Card>
        ))}
      </div>
    );
  }

  // Desktop table view
  return (
    <div className={cn('bg-white border border-gray-200 rounded-lg overflow-hidden', className)}>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              {columns.map((column) => (
                <th
                  key={String(column.key)}
                  className={cn(
                    'px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider',
                    column.hidden?.tablet && 'hidden lg:table-cell',
                    column.sortable && 'cursor-pointer hover:bg-gray-100 select-none',
                    column.className
                  )}
                  onClick={column.sortable ? () => handleSort(column.key) : undefined}
                >
                  <div className="flex items-center space-x-1">
                    <span>{column.header}</span>
                    {column.sortable && (
                      <svg
                        className={cn(
                          'w-4 h-4 transition-transform',
                          sortKey === column.key
                            ? sortDirection === 'asc'
                              ? 'transform rotate-0'
                              : 'transform rotate-180'
                            : 'text-gray-400'
                        )}
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
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {data.map((item, rowIndex) => (
              <tr
                key={rowIndex}
                className="hover:bg-gray-50 transition-colors"
              >
                {columns.map((column) => (
                  <td
                    key={String(column.key)}
                    className={cn(
                      'px-6 py-4 whitespace-nowrap text-sm text-gray-900',
                      column.hidden?.tablet && 'hidden lg:table-cell',
                      column.className
                    )}
                  >
                    {column.render ? column.render(item) : String(item[column.key] || '')}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// Table loading skeleton component
const TableSkeleton: React.FC<{ rows?: number; columns?: number }> = ({ 
  rows = 5, 
  columns = 4 
}) => {
  return (
    <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div className="animate-pulse">
        {/* Header skeleton */}
        <div className="border-b border-gray-200 bg-gray-50">
          <div className="flex">
            {Array.from({ length: columns }).map((_, index) => (
              <div key={index} className="px-6 py-4 flex-1">
                <div className="h-4 bg-gray-200 rounded w-20"></div>
              </div>
            ))}
          </div>
        </div>
        
        {/* Rows skeleton */}
        {Array.from({ length: rows }).map((_, rowIndex) => (
          <div key={rowIndex} className="border-b border-gray-100">
            <div className="flex">
              {Array.from({ length: columns }).map((_, colIndex) => (
                <div key={colIndex} className="px-6 py-4 flex-1">
                  <div className="h-4 bg-gray-200 rounded w-16"></div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export { ResponsiveTable, TableSkeleton };
export type { ResponsiveTableProps, Column };