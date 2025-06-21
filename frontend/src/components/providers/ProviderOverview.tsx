import React from 'react';
import { 
  CheckCircleIcon, 
  XCircleIcon, 
  ClockIcon,
  CpuChipIcon,
  GlobeAltIcon,
  TagIcon,
  CurrencyDollarIcon
} from '@heroicons/react/24/outline';
import { Badge } from '../ui/Badge';
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
  config?: ProviderConfig;
}

interface ProviderConfig {
  name: string;
  type: string;
  enabled: boolean;
  version: string;
  priority: number;
  api_key?: string;
  base_url: string;
  models: string[];
  timeout: number;
  max_retries: number;
  retry_delay: number;
  tags: string[];
  dependencies: string[];
  health_check?: {
    interval: number;
    timeout: number;
    max_failures: number;
    custom_endpoint?: string;
    test_message?: string;
  };
  rate_limit?: {
    requests_per_second: number;
    requests_per_minute: number;
    requests_per_hour: number;
    tokens_per_minute: number;
    burst_size: number;
  };
  pricing?: {
    input_token_cost: number;
    output_token_cost: number;
    request_cost: number;
    currency: string;
  };
  settings?: Record<string, any>;
}

interface ProviderOverviewProps {
  provider: Provider;
}

export const ProviderOverview: React.FC<ProviderOverviewProps> = ({ provider }) => {
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

  const getStatusIcon = (status: string) => {
    const className = "w-5 h-5";
    switch (status) {
      case 'healthy':
        return <CheckCircleIcon className={cn(className, 'text-green-500')} />;
      case 'degraded':
        return <ClockIcon className={cn(className, 'text-yellow-500')} />;
      case 'unhealthy':
        return <XCircleIcon className={cn(className, 'text-red-500')} />;
      default:
        return <XCircleIcon className={cn(className, 'text-gray-500')} />;
    }
  };

  const formatCurrency = (amount: number, currency = 'USD') => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
      minimumFractionDigits: 4,
      maximumFractionDigits: 6
    }).format(amount);
  };

  return (
    <div className="space-y-6">
      {/* Status Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Status</p>
              <div className="flex items-center space-x-2 mt-1">
                {getStatusIcon(provider.status)}
                <span className={cn('font-medium capitalize', getStatusColor(provider.status))}>
                  {provider.status}
                </span>
              </div>
            </div>
            <Badge variant={provider.enabled ? 'success' : 'secondary'}>
              {provider.enabled ? 'Enabled' : 'Disabled'}
            </Badge>
          </div>
        </div>

        <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Health Score</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">
                {(provider.health * 100).toFixed(0)}%
              </p>
            </div>
            <div className={cn(
              'w-12 h-12 rounded-full flex items-center justify-center',
              provider.health >= 0.8 ? 'bg-green-100 dark:bg-green-900' :
              provider.health >= 0.6 ? 'bg-yellow-100 dark:bg-yellow-900' :
              'bg-red-100 dark:bg-red-900'
            )}>
              <div className={cn(
                'w-6 h-6 rounded-full',
                provider.health >= 0.8 ? 'bg-green-500' :
                provider.health >= 0.6 ? 'bg-yellow-500' :
                'bg-red-500'
              )} />
            </div>
          </div>
        </div>

        <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Avg Latency</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">
                {provider.averageLatency}ms
              </p>
            </div>
            <ClockIcon className="w-8 h-8 text-gray-400" />
          </div>
        </div>
      </div>

      {/* Configuration Details */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Basic Information */}
        <div className="space-y-4">
          <h4 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Basic Information
          </h4>

          <div className="space-y-3">
            <div className="flex items-center space-x-3">
              <CpuChipIcon className="w-5 h-5 text-gray-400" />
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Type</p>
                <p className="text-sm text-gray-600 dark:text-gray-400 capitalize">{provider.type}</p>
              </div>
            </div>

            <div className="flex items-center space-x-3">
              <GlobeAltIcon className="w-5 h-5 text-gray-400" />
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Base URL</p>
                <p className="text-sm text-gray-600 dark:text-gray-400 font-mono">
                  {provider.config?.base_url || 'Not configured'}
                </p>
              </div>
            </div>

            {provider.config?.priority !== undefined && (
              <div className="flex items-center space-x-3">
                <TagIcon className="w-5 h-5 text-gray-400" />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Priority</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">{provider.config.priority}</p>
                </div>
              </div>
            )}

            {provider.config?.timeout && (
              <div className="flex items-center space-x-3">
                <ClockIcon className="w-5 h-5 text-gray-400" />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Timeout</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">{provider.config.timeout}ms</p>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Models and Tags */}
        <div className="space-y-4">
          <h4 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Models & Configuration
          </h4>

          <div className="space-y-3">
            <div>
              <p className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                Available Models ({provider.models.length})
              </p>
              <div className="flex flex-wrap gap-1">
                {provider.models.length > 0 ? (
                  provider.models.map((model, index) => (
                    <Badge key={index} variant="outline" className="text-xs">
                      {model}
                    </Badge>
                  ))
                ) : (
                  <span className="text-sm text-gray-500 dark:text-gray-400">No models configured</span>
                )}
              </div>
            </div>

            {provider.config?.tags && provider.config.tags.length > 0 && (
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">Tags</p>
                <div className="flex flex-wrap gap-1">
                  {provider.config.tags.map((tag, index) => (
                    <Badge key={index} variant="secondary" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {provider.config?.dependencies && provider.config.dependencies.length > 0 && (
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">Dependencies</p>
                <div className="flex flex-wrap gap-1">
                  {provider.config.dependencies.map((dep, index) => (
                    <Badge key={index} variant="outline" className="text-xs">
                      {dep}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Pricing Information */}
      {provider.config?.pricing && (
        <div className="space-y-4">
          <h4 className="text-lg font-medium text-gray-900 dark:text-gray-100 flex items-center space-x-2">
            <CurrencyDollarIcon className="w-5 h-5" />
            <span>Pricing</span>
          </h4>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Input Tokens</p>
              <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                {formatCurrency(provider.config.pricing.input_token_cost, provider.config.pricing.currency)}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">per 1K tokens</p>
            </div>

            <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Output Tokens</p>
              <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                {formatCurrency(provider.config.pricing.output_token_cost, provider.config.pricing.currency)}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">per 1K tokens</p>
            </div>

            <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Per Request</p>
              <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                {formatCurrency(provider.config.pricing.request_cost, provider.config.pricing.currency)}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400">base cost</p>
            </div>
          </div>
        </div>
      )}

      {/* Rate Limiting */}
      {provider.config?.rate_limit && (
        <div className="space-y-4">
          <h4 className="text-lg font-medium text-gray-900 dark:text-gray-100">Rate Limits</h4>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {provider.config.rate_limit.requests_per_second}
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">req/sec</p>
            </div>

            <div className="text-center">
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {provider.config.rate_limit.requests_per_minute}
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">req/min</p>
            </div>

            <div className="text-center">
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {provider.config.rate_limit.requests_per_hour}
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">req/hour</p>
            </div>

            <div className="text-center">
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {provider.config.rate_limit.tokens_per_minute}
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">tokens/min</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};