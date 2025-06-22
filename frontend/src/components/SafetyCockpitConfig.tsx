import React, { useState, useEffect } from 'react';
import { opikService } from '../services/opikService';
// import { getCockpitService, EventFilter, EventType } from '../services/cockpitService';

// Configuration interfaces
interface OpikConfig {
  enabled: boolean;
  api_key: string;
  project_name: string;
  batch_size: number;
  flush_interval: string;
  base_url?: string;
}

interface EvaluatorConfig {
  name: string;
  type: string;
  enabled: boolean;
  threshold: number;
  config: Record<string, any>;
}

interface SafetyConfig {
  opik: OpikConfig;
  evaluators: EvaluatorConfig[];
  realtime_evaluation: boolean;
  event_retention_hours: number;
  max_events_buffer: number;
  threat_sensitivity: number;
  auto_block_high_threats: boolean;
}

export const SafetyCockpitConfig: React.FC = () => {
  const [config, setConfig] = useState<SafetyConfig>({
    opik: {
      enabled: true,
      api_key: '',
      project_name: 'qt1-safety-cockpit',
      batch_size: 100,
      flush_interval: '5s',
      base_url: ''
    },
    evaluators: [
      {
        name: 'moderation_accuracy',
        type: 'accuracy',
        enabled: true,
        threshold: 0.95,
        config: {}
      },
      {
        name: 'moderation_latency',
        type: 'latency',
        enabled: true,
        threshold: 500,
        config: { threshold: '500ms' }
      },
      {
        name: 'threat_detection_recall',
        type: 'recall',
        enabled: true,
        threshold: 0.90,
        config: { window: '1h' }
      },
      {
        name: 'false_positive_rate',
        type: 'false_positive_rate',
        enabled: true,
        threshold: 0.05,
        config: { window: '1h' }
      }
    ],
    realtime_evaluation: true,
    event_retention_hours: 24,
    max_events_buffer: 10000,
    threat_sensitivity: 0.8,
    auto_block_high_threats: true
  });

  const [activeTab, setActiveTab] = useState<'opik' | 'evaluators' | 'general'>('general');
  const [isLoading, setIsLoading] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');

  // Load configuration on mount
  useEffect(() => {
    loadConfiguration();
  }, []);

  const loadConfiguration = async () => {
    setIsLoading(true);
    try {
      // Load Opik configuration from the new API
      const opikConfig = await opikService.getConfig();
      setConfig(prev => ({
        ...prev,
        opik: {
          enabled: opikConfig.enabled,
          api_key: '', // Don't show API key
          project_name: opikConfig.project_name,
          batch_size: opikConfig.batch_size,
          flush_interval: opikConfig.flush_interval,
          base_url: opikConfig.base_url
        }
      }));
    } catch (error) {
      console.error('Failed to load configuration:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const saveConfiguration = async () => {
    setSaveStatus('saving');
    try {
      // Save Opik configuration using the new API
      await opikService.updateConfig({
        enabled: config.opik.enabled,
        batch_size: config.opik.batch_size,
        flush_interval: config.opik.flush_interval
      });

      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 2000);
    } catch (error) {
      console.error('Failed to save configuration:', error);
      setSaveStatus('error');
    }
  };

  const updateOpikConfig = (updates: Partial<OpikConfig>) => {
    setConfig(prev => ({
      ...prev,
      opik: { ...prev.opik, ...updates }
    }));
  };

  const updateEvaluator = (index: number, updates: Partial<EvaluatorConfig>) => {
    setConfig(prev => ({
      ...prev,
      evaluators: prev.evaluators.map((evaluator, i) =>
        i === index ? { ...evaluator, ...updates } : evaluator
      )
    }));
  };

  const addEvaluator = () => {
    const newEvaluator: EvaluatorConfig = {
      name: 'custom_evaluator',
      type: 'custom',
      enabled: true,
      threshold: 0.8,
      config: {}
    };

    setConfig(prev => ({
      ...prev,
      evaluators: [...prev.evaluators, newEvaluator]
    }));
  };

  const removeEvaluator = (index: number) => {
    setConfig(prev => ({
      ...prev,
      evaluators: prev.evaluators.filter((_, i) => i !== index)
    }));
  };

  const resetToDefaults = () => {
    if (confirm('Are you sure you want to reset all configuration to defaults? This cannot be undone.')) {
      loadConfiguration();
    }
  };

  const testConnection = async () => {
    try {
      const result = await opikService.testConnection();
      alert(`Connection test successful!\nProject: ${result.project}\nTrace ID: ${result.trace_id}`);
    } catch (error) {
      alert(`Connection test failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-white">Loading configuration...</div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-white">Safety Cockpit Configuration</h2>
          <p className="text-gray-400">Configure AI safety monitoring and evaluation settings</p>
        </div>
        
        <div className="flex items-center space-x-3">
          <button
            onClick={resetToDefaults}
            className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded-lg transition-colors"
          >
            Reset to Defaults
          </button>
          <button
            onClick={saveConfiguration}
            disabled={saveStatus === 'saving'}
            className={`px-6 py-2 rounded-lg transition-colors ${
              saveStatus === 'saving'
                ? 'bg-gray-600 cursor-not-allowed'
                : saveStatus === 'saved'
                ? 'bg-green-600 hover:bg-green-700'
                : saveStatus === 'error'
                ? 'bg-red-600 hover:bg-red-700'
                : 'bg-blue-600 hover:bg-blue-700'
            } text-white`}
          >
            {saveStatus === 'saving' ? 'Saving...' :
             saveStatus === 'saved' ? '✓ Saved' :
             saveStatus === 'error' ? '✗ Error' : 'Save Configuration'}
          </button>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="border-b border-gray-700">
        <nav className="-mb-px flex space-x-8">
          {[
            { id: 'general', label: 'General Settings', icon: '⚙️' },
            { id: 'opik', label: 'Opik Integration', icon: '🔗' },
            { id: 'evaluators', label: 'Evaluators', icon: '📊' }
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`py-2 px-1 border-b-2 font-medium text-sm flex items-center space-x-2 ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-400'
                  : 'border-transparent text-gray-400 hover:text-gray-300 hover:border-gray-300'
              }`}
            >
              <span>{tab.icon}</span>
              <span>{tab.label}</span>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="space-y-6">
        {activeTab === 'general' && (
          <GeneralSettings
            config={config}
            onChange={setConfig}
          />
        )}

        {activeTab === 'opik' && (
          <OpikSettings
            config={config.opik}
            onChange={updateOpikConfig}
            onTestConnection={testConnection}
          />
        )}

        {activeTab === 'evaluators' && (
          <EvaluatorSettings
            evaluators={config.evaluators}
            onUpdateEvaluator={updateEvaluator}
            onAddEvaluator={addEvaluator}
            onRemoveEvaluator={removeEvaluator}
          />
        )}
      </div>
    </div>
  );
};

// General Settings Component
interface GeneralSettingsProps {
  config: SafetyConfig;
  onChange: (config: SafetyConfig) => void;
}

const GeneralSettings: React.FC<GeneralSettingsProps> = ({ config, onChange }) => {
  return (
    <div className="space-y-6">
      <div className="bg-gray-800 rounded-lg p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Event Management</h3>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Event Retention (hours)
            </label>
            <input
              type="number"
              value={config.event_retention_hours}
              onChange={(e) => onChange({
                ...config,
                event_retention_hours: parseInt(e.target.value) || 24
              })}
              className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              min="1"
              max="168"
            />
            <p className="text-xs text-gray-400 mt-1">
              How long to keep events in memory (1-168 hours)
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Max Events Buffer
            </label>
            <input
              type="number"
              value={config.max_events_buffer}
              onChange={(e) => onChange({
                ...config,
                max_events_buffer: parseInt(e.target.value) || 10000
              })}
              className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              min="100"
              max="100000"
            />
            <p className="text-xs text-gray-400 mt-1">
              Maximum number of events to buffer
            </p>
          </div>
        </div>
      </div>

      <div className="bg-gray-800 rounded-lg p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Threat Detection</h3>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Threat Sensitivity: {config.threat_sensitivity}
            </label>
            <input
              type="range"
              min="0.1"
              max="1.0"
              step="0.1"
              value={config.threat_sensitivity}
              onChange={(e) => onChange({
                ...config,
                threat_sensitivity: parseFloat(e.target.value)
              })}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-gray-400">
              <span>Low Sensitivity</span>
              <span>High Sensitivity</span>
            </div>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="auto-block"
              checked={config.auto_block_high_threats}
              onChange={(e) => onChange({
                ...config,
                auto_block_high_threats: e.target.checked
              })}
              className="rounded bg-gray-700 border-gray-600"
            />
            <label htmlFor="auto-block" className="ml-2 text-sm text-gray-300">
              Automatically block high-severity threats
            </label>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="realtime-eval"
              checked={config.realtime_evaluation}
              onChange={(e) => onChange({
                ...config,
                realtime_evaluation: e.target.checked
              })}
              className="rounded bg-gray-700 border-gray-600"
            />
            <label htmlFor="realtime-eval" className="ml-2 text-sm text-gray-300">
              Enable real-time evaluation
            </label>
          </div>
        </div>
      </div>
    </div>
  );
};

// Opik Settings Component
interface OpikSettingsProps {
  config: OpikConfig;
  onChange: (config: Partial<OpikConfig>) => void;
  onTestConnection: () => void;
}

const OpikSettings: React.FC<OpikSettingsProps> = ({ config, onChange, onTestConnection }) => {
  return (
    <div className="space-y-6">
      <div className="bg-gray-800 rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-white">Opik Integration</h3>
          <button
            onClick={onTestConnection}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm"
          >
            Test Connection
          </button>
        </div>

        <div className="space-y-4">
          <div className="flex items-center">
            <input
              type="checkbox"
              id="opik-enabled"
              checked={config.enabled}
              onChange={(e) => onChange({ enabled: e.target.checked })}
              className="rounded bg-gray-700 border-gray-600"
            />
            <label htmlFor="opik-enabled" className="ml-2 text-sm text-gray-300">
              Enable Opik integration
            </label>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                API Key
              </label>
              <input
                type="password"
                value={config.api_key}
                onChange={(e) => onChange({ api_key: e.target.value })}
                placeholder="Enter Opik API key"
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Project Name
              </label>
              <input
                type="text"
                value={config.project_name}
                onChange={(e) => onChange({ project_name: e.target.value })}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Base URL (optional)
              </label>
              <input
                type="url"
                value={config.base_url || ''}
                onChange={(e) => onChange({ base_url: e.target.value })}
                placeholder="https://api.opik.co"
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Batch Size
              </label>
              <input
                type="number"
                value={config.batch_size}
                onChange={(e) => onChange({ batch_size: parseInt(e.target.value) || 100 })}
                min="1"
                max="1000"
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              />
            </div>

            <div className="md:col-span-2">
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Flush Interval
              </label>
              <select
                value={config.flush_interval}
                onChange={(e) => onChange({ flush_interval: e.target.value })}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              >
                <option value="1s">1 second</option>
                <option value="5s">5 seconds</option>
                <option value="10s">10 seconds</option>
                <option value="30s">30 seconds</option>
                <option value="1m">1 minute</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Evaluator Settings Component
interface EvaluatorSettingsProps {
  evaluators: EvaluatorConfig[];
  onUpdateEvaluator: (index: number, updates: Partial<EvaluatorConfig>) => void;
  onAddEvaluator: () => void;
  onRemoveEvaluator: (index: number) => void;
}

const EvaluatorSettings: React.FC<EvaluatorSettingsProps> = ({
  evaluators,
  onUpdateEvaluator,
  onAddEvaluator,
  onRemoveEvaluator
}) => {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-white">Evaluator Configuration</h3>
        <button
          onClick={onAddEvaluator}
          className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg text-sm"
        >
          + Add Evaluator
        </button>
      </div>

      <div className="space-y-4">
        {evaluators.map((evaluator, index) => (
          <div key={index} className="bg-gray-800 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <input
                  type="checkbox"
                  checked={evaluator.enabled}
                  onChange={(e) => onUpdateEvaluator(index, { enabled: e.target.checked })}
                  className="rounded bg-gray-700 border-gray-600"
                />
                <div>
                  <h4 className="font-medium text-white">{evaluator.name}</h4>
                  <p className="text-sm text-gray-400 capitalize">{evaluator.type} evaluator</p>
                </div>
              </div>
              
              <button
                onClick={() => onRemoveEvaluator(index)}
                className="text-red-400 hover:text-red-300 text-sm"
              >
                Remove
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Name
                </label>
                <input
                  type="text"
                  value={evaluator.name}
                  onChange={(e) => onUpdateEvaluator(index, { name: e.target.value })}
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Type
                </label>
                <select
                  value={evaluator.type}
                  onChange={(e) => onUpdateEvaluator(index, { type: e.target.value })}
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
                >
                  <option value="accuracy">Accuracy</option>
                  <option value="precision">Precision</option>
                  <option value="recall">Recall</option>
                  <option value="f1_score">F1 Score</option>
                  <option value="latency">Latency</option>
                  <option value="false_positive_rate">False Positive Rate</option>
                  <option value="custom">Custom</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Threshold
                </label>
                <input
                  type="number"
                  value={evaluator.threshold}
                  onChange={(e) => onUpdateEvaluator(index, { threshold: parseFloat(e.target.value) })}
                  step="0.01"
                  min="0"
                  max="1"
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
                />
              </div>
            </div>

            {/* Additional config for specific evaluator types */}
            {evaluator.type === 'latency' && (
              <div className="mt-4">
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Latency Threshold (ms)
                </label>
                <input
                  type="number"
                  value={evaluator.config.threshold?.replace('ms', '') || '500'}
                  onChange={(e) => onUpdateEvaluator(index, {
                    config: { ...evaluator.config, threshold: `${e.target.value}ms` }
                  })}
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
                />
              </div>
            )}

            {(evaluator.type === 'precision' || evaluator.type === 'recall') && (
              <div className="mt-4">
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Time Window
                </label>
                <select
                  value={evaluator.config.window || '1h'}
                  onChange={(e) => onUpdateEvaluator(index, {
                    config: { ...evaluator.config, window: e.target.value }
                  })}
                  className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
                >
                  <option value="15m">15 minutes</option>
                  <option value="30m">30 minutes</option>
                  <option value="1h">1 hour</option>
                  <option value="6h">6 hours</option>
                  <option value="24h">24 hours</option>
                </select>
              </div>
            )}
          </div>
        ))}
      </div>

      {evaluators.length === 0 && (
        <div className="text-center py-8 text-gray-400">
          No evaluators configured. Click "Add Evaluator" to get started.
        </div>
      )}
    </div>
  );
};

export default SafetyCockpitConfig;