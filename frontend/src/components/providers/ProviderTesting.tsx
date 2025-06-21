import React, { useState } from 'react';
import { 
  CheckCircleIcon, 
  XCircleIcon, 
  ClockIcon,
  PlayIcon,
  BeakerIcon
} from '@heroicons/react/24/outline';
import { Card } from '../ui/Card';
import { Button } from '../ui/Button';
import { Badge } from '../ui/Badge';
// import { Input } from '../ui/Input';
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
  config?: any;
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

interface ProviderTestingProps {
  provider: Provider;
  testResult?: ProviderTestResult;
}

interface TestStepProps {
  title: string;
  result?: { success: boolean; error?: string };
  description: string;
}

const TestStep: React.FC<TestStepProps> = ({ title, result, description }) => {
  const getIcon = () => {
    if (!result) return <ClockIcon className="w-5 h-5 text-gray-400" />;
    return result.success 
      ? <CheckCircleIcon className="w-5 h-5 text-green-500" />
      : <XCircleIcon className="w-5 h-5 text-red-500" />;
  };

  const getStatus = () => {
    if (!result) return 'pending';
    return result.success ? 'success' : 'error';
  };

  return (
    <div className="flex items-start space-x-3 p-4 border border-gray-200 dark:border-gray-700 rounded-lg">
      <div className="flex-shrink-0 mt-0.5">
        {getIcon()}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">{title}</h4>
          <Badge 
            variant={getStatus() === 'success' ? 'success' : getStatus() === 'error' ? 'danger' : 'secondary'}
            className="ml-2"
          >
            {result ? (result.success ? 'Passed' : 'Failed') : 'Pending'}
          </Badge>
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{description}</p>
        {result && !result.success && result.error && (
          <div className="mt-2 p-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-sm text-red-700 dark:text-red-300">
            {result.error}
          </div>
        )}
      </div>
    </div>
  );
};

export const ProviderTesting: React.FC<ProviderTestingProps> = ({ 
  provider, 
  testResult 
}) => {
  const [testMessage, setTestMessage] = useState('Hello, this is a test message. Please respond with a simple greeting.');
  const [isRunningTest, setIsRunningTest] = useState(false);
  const [liveTestResult, setLiveTestResult] = useState<any>(null);
  const [customTestLoading, setCustomTestLoading] = useState(false);

  const runLiveTest = async () => {
    try {
      setCustomTestLoading(true);
      const response = await providerService.sendTestRequest(provider.name, testMessage);
      setLiveTestResult(response);
    } catch (err) {
      setLiveTestResult({
        error: err instanceof Error ? err.message : 'Test failed'
      });
    } finally {
      setCustomTestLoading(false);
    }
  };

  const runFullTest = async () => {
    try {
      setIsRunningTest(true);
      if (provider.config) {
        await providerService.testProvider(provider.name, provider.config);
        // The parent component should handle updating the test result
      }
    } catch (err) {
      console.error('Failed to run test:', err);
    } finally {
      setIsRunningTest(false);
    }
  };

  const getOverallStatus = () => {
    if (!testResult) return 'Not tested';
    
    const tests = [
      testResult.config_validation,
      testResult.startup,
      testResult.health_check,
      testResult.functionality
    ];

    const allPassed = tests.every(test => test?.success);
    const anyFailed = tests.some(test => test && !test.success);

    if (allPassed) return 'All tests passed';
    if (anyFailed) return 'Some tests failed';
    return 'Tests incomplete';
  };

  const getOverallStatusColor = () => {
    if (!testResult) return 'text-gray-600 dark:text-gray-400';
    
    const tests = [
      testResult.config_validation,
      testResult.startup,
      testResult.health_check,
      testResult.functionality
    ];

    const allPassed = tests.every(test => test?.success);
    const anyFailed = tests.some(test => test && !test.success);

    if (allPassed) return 'text-green-600 dark:text-green-400';
    if (anyFailed) return 'text-red-600 dark:text-red-400';
    return 'text-yellow-600 dark:text-yellow-400';
  };

  return (
    <div className="space-y-6">
      {/* Test Overview */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Provider Testing
          </h3>
          <p className={`text-sm ${getOverallStatusColor()}`}>
            {getOverallStatus()}
            {testResult && (
              <span className="text-gray-500 dark:text-gray-400 ml-2">
                • Last tested: {new Date(testResult.start_time).toLocaleString()}
                • Duration: {testResult.duration}ms
              </span>
            )}
          </p>
        </div>
        <Button 
          onClick={runFullTest} 
          disabled={isRunningTest || !provider.config}
          className="flex items-center space-x-2"
        >
          {isRunningTest ? (
            <>
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
              <span>Testing...</span>
            </>
          ) : (
            <>
              <BeakerIcon className="w-4 h-4" />
              <span>Run Full Test</span>
            </>
          )}
        </Button>
      </div>

      {/* Test Steps */}
      <div className="space-y-3">
        <TestStep
          title="Configuration Validation"
          result={testResult?.config_validation}
          description="Validates the provider configuration format and required fields"
        />
        
        <TestStep
          title="Provider Startup"
          result={testResult?.startup}
          description="Tests if the provider can be initialized and started successfully"
        />
        
        <TestStep
          title="Health Check"
          result={testResult?.health_check}
          description="Verifies the provider's health endpoint responds correctly"
        />
        
        <TestStep
          title="Functionality Test"
          result={testResult?.functionality}
          description="Sends a test request to verify the provider can handle requests"
        />
      </div>

      {/* Live Test */}
      <Card title="Live Test">
        <div className="space-y-4">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Send a custom message to test the provider's real-time functionality.
          </p>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Test Message
            </label>
            <textarea
              className="w-full h-24 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-gray-100"
              value={testMessage}
              onChange={(e) => setTestMessage(e.target.value)}
              placeholder="Enter your test message..."
            />
          </div>

          <Button 
            onClick={runLiveTest} 
            disabled={customTestLoading || !provider.enabled || !testMessage.trim()}
            className="flex items-center space-x-2"
          >
            {customTestLoading ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                <span>Sending...</span>
              </>
            ) : (
              <>
                <PlayIcon className="w-4 h-4" />
                <span>Send Test Message</span>
              </>
            )}
          </Button>

          {liveTestResult && (
            <div className="mt-4">
              <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                Response:
              </h4>
              {liveTestResult.error ? (
                <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-sm text-red-700 dark:text-red-300">
                  Error: {liveTestResult.error}
                </div>
              ) : (
                <div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded">
                  <div className="text-sm text-green-800 dark:text-green-200">
                    <strong>Provider:</strong> {liveTestResult.provider || provider.name}
                  </div>
                  <div className="text-sm text-green-800 dark:text-green-200 mt-1">
                    <strong>Model:</strong> {liveTestResult.model || 'default'}
                  </div>
                  <div className="text-sm text-green-800 dark:text-green-200 mt-1">
                    <strong>Response:</strong>
                  </div>
                  <div className="mt-2 p-2 bg-white dark:bg-gray-800 rounded border text-gray-900 dark:text-gray-100">
                    {liveTestResult.message?.content || liveTestResult.response || 'No response content'}
                  </div>
                  {liveTestResult.usage && (
                    <div className="mt-2 text-xs text-green-700 dark:text-green-300">
                      Tokens: {liveTestResult.usage.input_tokens || 0} input, {liveTestResult.usage.output_tokens || 0} output
                      {liveTestResult.latency && ` • Latency: ${liveTestResult.latency}ms`}
                      {liveTestResult.cost && ` • Cost: $${liveTestResult.cost.toFixed(4)}`}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </Card>

      {/* Provider Status */}
      <Card title="Current Status">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className={`text-2xl font-bold ${provider.enabled ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
              {provider.enabled ? 'ON' : 'OFF'}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Status</div>
          </div>
          
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {(provider.health * 100).toFixed(0)}%
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Health</div>
          </div>
          
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {provider.averageLatency}ms
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Latency</div>
          </div>
          
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {provider.models.length}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Models</div>
          </div>
        </div>
      </Card>
    </div>
  );
};