import React from 'react';

// Performance monitoring utility for dashboard operations
export class PerformanceMonitor {
  private static instance: PerformanceMonitor;
  private metrics: Map<string, PerformanceMetric> = new Map();
  private isEnabled: boolean = true;

  private constructor() {}

  static getInstance(): PerformanceMonitor {
    if (!PerformanceMonitor.instance) {
      PerformanceMonitor.instance = new PerformanceMonitor();
    }
    return PerformanceMonitor.instance;
  }

  enable(): void {
    this.isEnabled = true;
  }

  disable(): void {
    this.isEnabled = false;
  }

  startTimer(operation: string): string {
    if (!this.isEnabled) return '';
    
    const timerId = `${operation}_${Date.now()}_${Math.random()}`;
    const metric: PerformanceMetric = {
      operation,
      startTime: performance.now(),
      timerId,
    };
    
    this.metrics.set(timerId, metric);
    return timerId;
  }

  endTimer(timerId: string): number | null {
    if (!this.isEnabled || !timerId) return null;
    
    const metric = this.metrics.get(timerId);
    if (!metric) return null;
    
    const endTime = performance.now();
    const duration = endTime - metric.startTime;
    
    metric.endTime = endTime;
    metric.duration = duration;
    
    // Log performance warning if operation takes too long
    if (duration > 1000) { // 1 second threshold
      console.warn(`Slow operation detected: ${metric.operation} took ${duration.toFixed(2)}ms`);
    }
    
    this.metrics.delete(timerId);
    return duration;
  }

  measureFunction<T extends any[], R>(
    fn: (...args: T) => R,
    operation: string
  ): (...args: T) => R {
    return (...args: T): R => {
      const timerId = this.startTimer(operation);
      try {
        const result = fn(...args);
        this.endTimer(timerId);
        return result;
      } catch (error) {
        this.endTimer(timerId);
        throw error;
      }
    };
  }

  measureAsyncFunction<T extends any[], R>(
    fn: (...args: T) => Promise<R>,
    operation: string
  ): (...args: T) => Promise<R> {
    return async (...args: T): Promise<R> => {
      const timerId = this.startTimer(operation);
      try {
        const result = await fn(...args);
        this.endTimer(timerId);
        return result;
      } catch (error) {
        this.endTimer(timerId);
        throw error;
      }
    };
  }

  getMetrics(): PerformanceReport {
    const activeOperations = Array.from(this.metrics.values()).map(metric => ({
      operation: metric.operation,
      duration: performance.now() - metric.startTime,
      status: 'running',
    }));

    return {
      activeOperations,
      memoryUsage: this.getMemoryUsage(),
      timestamp: new Date().toISOString(),
    };
  }

  private getMemoryUsage(): MemoryUsage | null {
    if ('memory' in performance) {
      const memory = (performance as any).memory;
      return {
        usedJSHeapSize: memory.usedJSHeapSize,
        totalJSHeapSize: memory.totalJSHeapSize,
        jsHeapSizeLimit: memory.jsHeapSizeLimit,
      };
    }
    return null;
  }

  // Monitor React component render performance
  static withRenderMonitoring<P extends {}>(
    Component: React.ComponentType<P>,
    componentName: string
  ): React.ComponentType<P> {
    return (props: P) => {
      const monitor = PerformanceMonitor.getInstance();
      const timerId = monitor.startTimer(`render_${componentName}`);
      
      React.useEffect(() => {
        monitor.endTimer(timerId);
      });

      return React.createElement(Component, props);
    };
  }

  // Monitor dashboard refresh operations
  monitorDashboardRefresh(metrics: string[], timeRange: string): void {
    if (!this.isEnabled) return;
    
    const timerId = this.startTimer('dashboard_refresh');
    
    // Set up a timeout to warn about long refresh times
    setTimeout(() => {
      const metric = this.metrics.get(timerId);
      if (metric && !metric.endTime) {
        console.warn(`Dashboard refresh taking longer than expected. Metrics: ${metrics.join(', ')}, Range: ${timeRange}`);
      }
    }, 5000); // 5 second warning threshold
  }

  // Monitor chart rendering performance
  monitorChartRender(chartType: string, dataPoints: number): string {
    return this.startTimer(`chart_render_${chartType}_${dataPoints}_points`);
  }

  // Monitor export operations
  monitorExport(exportType: string, dataSize: number): string {
    return this.startTimer(`export_${exportType}_${dataSize}_records`);
  }

  // Check if dashboard is performing well
  isDashboardPerformant(): boolean {
    const report = this.getMetrics();
    
    // Check for long-running operations
    const hasSlowOperations = report.activeOperations.some(op => op.duration > 2000);
    
    // Check memory usage (if available)
    if (report.memoryUsage) {
      const memoryRatio = report.memoryUsage.usedJSHeapSize / report.memoryUsage.jsHeapSizeLimit;
      if (memoryRatio > 0.8) { // Using more than 80% of available heap
        return false;
      }
    }
    
    return !hasSlowOperations;
  }

  // Generate performance recommendations
  getPerformanceRecommendations(): string[] {
    const recommendations: string[] = [];
    const report = this.getMetrics();
    
    if (report.activeOperations.length > 10) {
      recommendations.push('Consider reducing the number of concurrent operations');
    }
    
    const slowOperations = report.activeOperations.filter(op => op.duration > 1000);
    if (slowOperations.length > 0) {
      recommendations.push('Some operations are running slowly. Consider optimizing data fetching or reducing refresh frequency');
    }
    
    if (report.memoryUsage) {
      const memoryRatio = report.memoryUsage.usedJSHeapSize / report.memoryUsage.jsHeapSizeLimit;
      if (memoryRatio > 0.7) {
        recommendations.push('High memory usage detected. Consider reducing the amount of cached data or time range');
      }
    }
    
    return recommendations;
  }

  // Clean up stale metrics (for memory management)
  cleanup(): void {
    const now = performance.now();
    const staleThreshold = 300000; // 5 minutes
    
    for (const [timerId, metric] of this.metrics.entries()) {
      if (now - metric.startTime > staleThreshold) {
        console.warn(`Cleaning up stale performance metric: ${metric.operation}`);
        this.metrics.delete(timerId);
      }
    }
  }
}

// Types
interface PerformanceMetric {
  operation: string;
  startTime: number;
  endTime?: number;
  duration?: number;
  timerId: string;
}

interface PerformanceReport {
  activeOperations: Array<{
    operation: string;
    duration: number;
    status: string;
  }>;
  memoryUsage: MemoryUsage | null;
  timestamp: string;
}

interface MemoryUsage {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
}

// Export singleton instance
export const performanceMonitor = PerformanceMonitor.getInstance();

// React hook for performance monitoring
export function usePerformanceMonitor(operationName: string, _dependencies: any[] = []) {
  const [isLoading, setIsLoading] = React.useState(false);
  const [duration, setDuration] = React.useState<number | null>(null);
  
  const startOperation = React.useCallback(() => {
    setIsLoading(true);
    setDuration(null);
    return performanceMonitor.startTimer(operationName);
  }, [operationName]);
  
  const endOperation = React.useCallback((timerId: string) => {
    const operationDuration = performanceMonitor.endTimer(timerId);
    setDuration(operationDuration);
    setIsLoading(false);
  }, []);
  
  React.useEffect(() => {
    // Clean up any stale metrics periodically
    const cleanup = setInterval(() => {
      performanceMonitor.cleanup();
    }, 60000); // Every minute
    
    return () => clearInterval(cleanup);
  }, []);
  
  return {
    isLoading,
    duration,
    startOperation,
    endOperation,
    isPerformant: performanceMonitor.isDashboardPerformant(),
    recommendations: performanceMonitor.getPerformanceRecommendations(),
  };
}

// Decorator for automatic performance monitoring
export function monitorPerformance(operationName: string) {
  return function <T extends (...args: any[]) => any>(
    _target: any,
    propertyName: string,
    descriptor: TypedPropertyDescriptor<T>
  ): TypedPropertyDescriptor<T> | void {
    const originalMethod = descriptor.value!;
    
    descriptor.value = function(this: any, ...args: any[]) {
      const timerId = performanceMonitor.startTimer(`${operationName}_${propertyName}`);
      try {
        const result = originalMethod.apply(this, args);
        if (result instanceof Promise) {
          return result.finally(() => performanceMonitor.endTimer(timerId));
        } else {
          performanceMonitor.endTimer(timerId);
          return result;
        }
      } catch (error) {
        performanceMonitor.endTimer(timerId);
        throw error;
      }
    } as T;
    
    return descriptor;
  };
}

export default PerformanceMonitor;