import React, { useState } from 'react';
import {
  InformationCircleIcon,
  CogIcon,
  ChartBarIcon,
  BeakerIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon
} from '@heroicons/react/24/outline';
import { Card } from '../ui/Card';
import { ProviderOverview } from './ProviderOverview';
import { ProviderConfiguration } from './ProviderConfiguration';
import { ProviderMetrics } from './ProviderMetrics';
import { ProviderTesting } from './ProviderTesting';
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

interface ProviderDetailsProps {
  provider: Provider;
  testResult?: ProviderTestResult;
  onConfigUpdate: (config: ProviderConfig) => void;
}

type TabId = 'overview' | 'config' | 'metrics' | 'test';

export const ProviderDetails: React.FC<ProviderDetailsProps> = ({
  provider,
  testResult,
  onConfigUpdate
}) => {
  const [activeTab, setActiveTab] = useState<TabId>('overview');

  const tabs = [
    { 
      id: 'overview' as TabId, 
      label: 'Overview', 
      icon: <InformationCircleIcon className="w-4 h-4" /> 
    },
    { 
      id: 'config' as TabId, 
      label: 'Configuration', 
      icon: <CogIcon className="w-4 h-4" /> 
    },
    { 
      id: 'metrics' as TabId, 
      label: 'Metrics', 
      icon: <ChartBarIcon className="w-4 h-4" /> 
    },
    { 
      id: 'test' as TabId, 
      label: 'Testing', 
      icon: <BeakerIcon className="w-4 h-4" /> 
    },
  ];

  const getTestResultIcon = (result?: { success: boolean }) => {
    if (!result) return <ClockIcon className="w-4 h-4 text-gray-400" />;
    return result.success 
      ? <CheckCircleIcon className="w-4 h-4 text-green-500" />
      : <XCircleIcon className="w-4 h-4 text-red-500" />;
  };

  return (
    <Card className="h-full">
      {/* Header */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
              {provider.name}
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {provider.type} Provider • Version {provider.version}
            </p>
          </div>
          
          {/* Quick Status */}
          <div className="flex items-center space-x-4">
            <div className="text-right">
              <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                Health: {(provider.health * 100).toFixed(0)}%
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400">
                Latency: {provider.averageLatency}ms
              </div>
            </div>
            
            {testResult && (
              <div className="flex items-center space-x-1">
                {getTestResultIcon(testResult.config_validation)}
                {getTestResultIcon(testResult.startup)}
                {getTestResultIcon(testResult.health_check)}
                {getTestResultIcon(testResult.functionality)}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex space-x-8 px-6">
          {tabs.map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex items-center space-x-2 py-4 px-1 border-b-2 font-medium text-sm transition-colors',
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-200'
              )}
            >
              {tab.icon}
              <span>{tab.label}</span>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="p-6 flex-1 overflow-auto">
        {activeTab === 'overview' && (
          <ProviderOverview provider={provider} />
        )}
        {activeTab === 'config' && (
          <ProviderConfiguration 
            provider={provider} 
            onUpdate={onConfigUpdate}
          />
        )}
        {activeTab === 'metrics' && (
          <ProviderMetrics provider={provider} />
        )}
        {activeTab === 'test' && (
          <ProviderTesting 
            provider={provider}
            testResult={testResult}
          />
        )}
      </div>
    </Card>
  );
};