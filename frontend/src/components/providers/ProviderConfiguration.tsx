import React, { useState } from 'react';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Card } from '../ui/Card';

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

interface ProviderConfigurationProps {
  provider: Provider;
  onUpdate: (config: ProviderConfig) => void;
}

export const ProviderConfiguration: React.FC<ProviderConfigurationProps> = ({
  provider,
  onUpdate
}) => {
  const [config, setConfig] = useState<ProviderConfig>(
    provider.config || {
      name: provider.name,
      type: provider.type,
      enabled: provider.enabled,
      version: provider.version,
      priority: 1,
      base_url: '',
      models: [],
      timeout: 30000,
      max_retries: 3,
      retry_delay: 1000,
      tags: [],
      dependencies: []
    }
  );

  const [isEditing, setIsEditing] = useState(false);

  const handleSave = () => {
    onUpdate(config);
    setIsEditing(false);
  };

  const handleCancel = () => {
    setConfig(provider.config || config);
    setIsEditing(false);
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Configuration
        </h3>
        <div className="space-x-2">
          {isEditing ? (
            <>
              <Button variant="outline" onClick={handleCancel}>
                Cancel
              </Button>
              <Button onClick={handleSave}>
                Save Changes
              </Button>
            </>
          ) : (
            <Button onClick={() => setIsEditing(true)}>
              Edit Configuration
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Basic Settings */}
        <Card title="Basic Settings">
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Provider Name
              </label>
              <Input
                value={config.name}
                onChange={(e) => setConfig({ ...config, name: e.target.value })}
                disabled={!isEditing}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Base URL
              </label>
              <Input
                value={config.base_url}
                onChange={(e) => setConfig({ ...config, base_url: e.target.value })}
                disabled={!isEditing}
                placeholder="https://api.example.com/v1"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                API Key
              </label>
              <Input
                type="password"
                value={config.api_key || ''}
                onChange={(e) => setConfig({ ...config, api_key: e.target.value })}
                disabled={!isEditing}
                placeholder="Enter API key..."
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Priority
              </label>
              <Input
                type="number"
                value={config.priority}
                onChange={(e) => setConfig({ ...config, priority: parseInt(e.target.value) })}
                disabled={!isEditing}
                min="0"
                max="100"
              />
            </div>
          </div>
        </Card>

        {/* Timeout & Retry Settings */}
        <Card title="Timeout & Retry">
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Timeout (ms)
              </label>
              <Input
                type="number"
                value={config.timeout}
                onChange={(e) => setConfig({ ...config, timeout: parseInt(e.target.value) })}
                disabled={!isEditing}
                min="1000"
                max="300000"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Max Retries
              </label>
              <Input
                type="number"
                value={config.max_retries}
                onChange={(e) => setConfig({ ...config, max_retries: parseInt(e.target.value) })}
                disabled={!isEditing}
                min="0"
                max="10"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Retry Delay (ms)
              </label>
              <Input
                type="number"
                value={config.retry_delay}
                onChange={(e) => setConfig({ ...config, retry_delay: parseInt(e.target.value) })}
                disabled={!isEditing}
                min="100"
                max="60000"
              />
            </div>
          </div>
        </Card>
      </div>

      {/* Models */}
      <Card title="Supported Models">
        <div className="space-y-3">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Models (one per line)
          </label>
          <textarea
            className="w-full h-32 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-gray-100"
            value={config.models.join('\n')}
            onChange={(e) => setConfig({ 
              ...config, 
              models: e.target.value.split('\n').filter(model => model.trim() !== '') 
            })}
            disabled={!isEditing}
            placeholder="gpt-4&#10;gpt-3.5-turbo&#10;text-davinci-003"
          />
        </div>
      </Card>

      {/* Tags */}
      <Card title="Tags & Dependencies">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Tags (comma-separated)
            </label>
            <Input
              value={config.tags.join(', ')}
              onChange={(e) => setConfig({ 
                ...config, 
                tags: e.target.value.split(',').map(tag => tag.trim()).filter(tag => tag !== '') 
              })}
              disabled={!isEditing}
              placeholder="production, high-priority, gpt"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Dependencies (comma-separated)
            </label>
            <Input
              value={config.dependencies.join(', ')}
              onChange={(e) => setConfig({ 
                ...config, 
                dependencies: e.target.value.split(',').map(dep => dep.trim()).filter(dep => dep !== '') 
              })}
              disabled={!isEditing}
              placeholder="auth-service, rate-limiter"
            />
          </div>
        </div>
      </Card>

      {/* JSON Configuration */}
      {isEditing && (
        <Card title="Advanced Configuration (JSON)">
          <div className="space-y-3">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Raw Configuration
            </label>
            <textarea
              className="w-full h-64 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-gray-100 font-mono text-sm"
              value={JSON.stringify(config, null, 2)}
              onChange={(e) => {
                try {
                  const parsed = JSON.parse(e.target.value);
                  setConfig(parsed);
                } catch (err) {
                  // Invalid JSON, ignore for now
                }
              }}
              disabled={!isEditing}
            />
          </div>
        </Card>
      )}
    </div>
  );
};