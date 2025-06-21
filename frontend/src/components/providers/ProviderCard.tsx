import React from 'react';
import { CheckCircleIcon, ExclamationTriangleIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
// import { Switch } from '../ui/Switch';
import { cn } from '../../utils/cn';

interface Provider {
  name: string;
  type: string;
  enabled: boolean;
  health: number;
  averageLatency: number;
  models: string[];
  version: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
}

interface ProviderCardProps {
  provider: Provider;
  selected: boolean;
  onClick: () => void;
  onToggle: (enabled: boolean) => void;
  onTest: () => void;
}

export const ProviderCard: React.FC<ProviderCardProps> = ({
  provider,
  selected,
  onClick,
  onToggle,
  onTest
}) => {
  const getHealthColor = (health: number) => {
    if (health >= 0.8) return 'text-green-600 dark:text-green-400';
    if (health >= 0.6) return 'text-yellow-600 dark:text-yellow-400';
    return 'text-red-600 dark:text-red-400';
  };

  const getHealthIcon = (health: number) => {
    const className = "w-5 h-5";
    if (health >= 0.8) return <CheckCircleIcon className={className} />;
    if (health >= 0.6) return <ExclamationTriangleIcon className={className} />;
    return <XCircleIcon className={className} />;
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'text-green-600 dark:text-green-400';
      case 'degraded':
        return 'text-yellow-600 dark:text-yellow-400';
      case 'unhealthy':
        return 'text-red-600 dark:text-red-400';
      default:
        return 'text-gray-600 dark:text-gray-400';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type.toLowerCase()) {
      case 'openai':
        return '🤖';
      case 'anthropic':
        return '🧠';
      case 'local':
        return '🏠';
      case 'mock':
        return '🎭';
      default:
        return '⚡';
    }
  };

  return (
    <div
      className={cn(
        'p-4 border rounded-lg cursor-pointer transition-all hover:shadow-md',
        'dark:border-gray-700 dark:hover:border-gray-600',
        selected
          ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 dark:border-blue-400'
          : 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'
      )}
      onClick={onClick}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-2">
          <span className="text-lg">{getTypeIcon(provider.type)}</span>
          <div>
            <h3 className="font-medium text-gray-900 dark:text-gray-100">
              {provider.name}
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {provider.type} • v{provider.version}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Badge 
            variant={provider.enabled ? 'success' : 'secondary'}
            className="text-xs"
          >
            {provider.enabled ? 'Enabled' : 'Disabled'}
          </Badge>
          <button
            onClick={(e: any) => {
              e.stopPropagation();
              onToggle(!provider.enabled);
            }}
            className={`w-10 h-6 rounded-full transition-colors ${
              provider.enabled ? 'bg-blue-600' : 'bg-gray-300'
            }`}
          >
            <div className={`w-4 h-4 bg-white rounded-full transition-transform ${
              provider.enabled ? 'translate-x-5' : 'translate-x-1'
            }`} />
          </button>
        </div>
      </div>

      <div className="space-y-2">
        {/* Health Status */}
        <div className="flex items-center justify-between">
          <div className={cn('flex items-center space-x-1', getHealthColor(provider.health))}>
            {getHealthIcon(provider.health)}
            <span className="text-sm font-medium">
              Health: {(provider.health * 100).toFixed(0)}%
            </span>
          </div>
          <div className={cn('text-sm font-medium', getStatusColor(provider.status))}>
            {provider.status.charAt(0).toUpperCase() + provider.status.slice(1)}
          </div>
        </div>

        {/* Performance Metrics */}
        <div className="grid grid-cols-2 gap-2 text-xs text-gray-600 dark:text-gray-400">
          <div>
            <span className="font-medium">Latency:</span>
            <div className="font-mono">{provider.averageLatency}ms</div>
          </div>
          <div>
            <span className="font-medium">Models:</span>
            <div className="font-mono">{provider.models.length}</div>
          </div>
        </div>

        {/* Models Preview */}
        {provider.models.length > 0 && (
          <div className="text-xs text-gray-500 dark:text-gray-400">
            <span className="font-medium">Available:</span>
            <div className="truncate mt-1">
              {provider.models.slice(0, 2).join(', ')}
              {provider.models.length > 2 && ` +${provider.models.length - 2} more`}
            </div>
          </div>
        )}

        {/* Test Button */}
        <div className="pt-2">
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              onTest();
            }}
            className="w-full"
          >
            Test Provider
          </Button>
        </div>
      </div>
    </div>
  );
};