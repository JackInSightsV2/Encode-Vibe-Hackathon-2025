import React, { useState, useEffect } from 'react';
import { ArrowPathIcon as RefreshIcon, ServerIcon } from '@heroicons/react/24/outline';
import { Card } from '../ui/Card';
import { Button } from '../ui/Button';
import { ProviderCard } from './ProviderCard';
import { ProviderDetails } from './ProviderDetails';
import { providerService } from '../../services/providerService';

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

interface ProviderTestResult {
  provider: string;
  start_time: string;
  end_time: string;
  duration: number;
  config_validation?: {
    success: boolean;
    error?: string;
  };
  startup?: {
    success: boolean;
    error?: string;
  };
  health_check?: {
    success: boolean;
    error?: string;
  };
  functionality?: {
    success: boolean;
    error?: string;
  };
}

interface ProviderManagementProps {
  onProviderUpdate?: () => void;
}

export const ProviderManagement: React.FC<ProviderManagementProps> = ({
  onProviderUpdate
}) => {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<Map<string, ProviderTestResult>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadProviders = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const providersData = await providerService.getProviders();
      const healthData = await providerService.getProvidersHealth();
      const metricsData = await providerService.getProvidersMetrics();
      const configsData = await providerService.getProvidersConfigs();

      // Combine data from different sources
      const combinedProviders: Provider[] = Object.keys(providersData).map(name => {
        const provider = providersData[name];
        const health = healthData[name];
        const metrics = metricsData[name];
        const config = configsData[name];

        return {
          name,
          type: provider.type || 'unknown',
          enabled: config?.enabled || false,
          health: health?.health_score || 0,
          averageLatency: metrics?.average_latency_ms || 0,
          models: provider.models || [],
          version: provider.version || '1.0.0',
          status: health?.status || 'unhealthy',
          config
        };
      });

      setProviders(combinedProviders);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load providers');
      console.error('Failed to load providers:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleProvider = async (providerName: string, enabled: boolean) => {
    try {
      if (enabled) {
        await providerService.enableProvider(providerName);
      } else {
        await providerService.disableProvider(providerName);
      }

      // Update local state
      setProviders(prev => prev.map(p => 
        p.name === providerName ? { ...p, enabled } : p
      ));

      if (onProviderUpdate) {
        onProviderUpdate();
      }
    } catch (err) {
      console.error(`Failed to ${enabled ? 'enable' : 'disable'} provider:`, err);
      setError(`Failed to ${enabled ? 'enable' : 'disable'} provider ${providerName}`);
    }
  };

  const handleTestProvider = async (providerName: string) => {
    try {
      const provider = providers.find(p => p.name === providerName);
      if (!provider?.config) {
        throw new Error('Provider configuration not found');
      }

      const result = await providerService.testProvider(providerName, provider.config);
      setTestResults(prev => new Map(prev).set(providerName, result));
    } catch (err) {
      console.error('Failed to test provider:', err);
      setError(`Failed to test provider ${providerName}`);
    }
  };

  const handleConfigUpdate = async (providerName: string, config: ProviderConfig) => {
    try {
      await providerService.updateProviderConfig(providerName, config);
      
      // Update local state
      setProviders(prev => prev.map(p => 
        p.name === providerName ? { ...p, config } : p
      ));

      if (onProviderUpdate) {
        onProviderUpdate();
      }
    } catch (err) {
      console.error('Failed to update provider config:', err);
      setError(`Failed to update configuration for ${providerName}`);
    }
  };

  const handleRefresh = () => {
    loadProviders();
  };

  useEffect(() => {
    loadProviders();

    // Auto-refresh every 30 seconds
    const interval = setInterval(loadProviders, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Provider Management
        </h2>
        <div className="flex space-x-2">
          {error && (
            <div className="text-red-600 dark:text-red-400 text-sm">
              {error}
            </div>
          )}
          <Button onClick={handleRefresh} variant="outline">
            <RefreshIcon className="w-4 h-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Provider List */}
        <div className="lg:col-span-1">
          <Card title="Providers" className="h-full">
            <div className="space-y-2">
              {providers.length === 0 ? (
                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                  No providers configured
                </div>
              ) : (
                providers.map(provider => (
                  <ProviderCard
                    key={provider.name}
                    provider={provider}
                    selected={selectedProvider === provider.name}
                    onClick={() => setSelectedProvider(provider.name)}
                    onToggle={(enabled) => handleToggleProvider(provider.name, enabled)}
                    onTest={() => handleTestProvider(provider.name)}
                  />
                ))
              )}
            </div>
          </Card>
        </div>

        {/* Provider Details */}
        <div className="lg:col-span-2">
          {selectedProvider ? (
            (() => {
              const provider = providers.find(p => p.name === selectedProvider);
              return provider ? (
                <ProviderDetails
                  provider={provider}
                  testResult={testResults.get(selectedProvider)}
                  onConfigUpdate={(config) => handleConfigUpdate(selectedProvider, config)}
                />
              ) : (
                <Card>
                  <div className="text-center py-12">
                    <ServerIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                    <p className="text-gray-500 dark:text-gray-400">Provider not found</p>
                  </div>
                </Card>
              );
            })()
          ) : (
            <Card>
              <div className="text-center py-12">
                <ServerIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                <p className="text-gray-500 dark:text-gray-400">
                  Select a provider to view details
                </p>
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};