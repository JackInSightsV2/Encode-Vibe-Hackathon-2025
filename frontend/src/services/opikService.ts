/**
 * Service for interacting with the Opik integration API
 */

import { apiRequest } from '../utils/api';

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

interface OpikStatus {
  enabled: boolean;
  connected: boolean;
  project: string;
  base_url: string;
  batch_size: number;
  flush_interval: string;
  tracing: {
    enabled: boolean;
    sample_rate: number;
    trace_moderation: boolean;
    trace_providers: boolean;
    trace_security: boolean;
  };
  evaluations: {
    enabled: boolean;
    run_async: boolean;
    timeout: string;
    evaluators: string[];
  };
}

interface OpikTrace {
  id: string;
  name: string;
  start_time: string;
  end_time: string;
  duration: string;
  status: string;
  input: Record<string, any>;
  output: Record<string, any>;
  spans: number;
}

interface OpikEvaluations {
  regex_effectiveness: {
    score: number;
    trend: string;
    last_update: string;
    details: Record<string, any>;
  };
  llm_accuracy: {
    score: number;
    trend: string;
    last_update: string;
    details: Record<string, any>;
  };
  pii_coverage: {
    score: number;
    trend: string;
    last_update: string;
    details: Record<string, any>;
  };
  false_positive_rate: {
    score: number;
    trend: string;
    last_update: string;
    details: Record<string, any>;
  };
  response_time: {
    score: number;
    trend: string;
    last_update: string;
    details: Record<string, any>;
  };
}

interface OpikConfig {
  enabled: boolean;
  project_name: string;
  batch_size: number;
  flush_interval: string;
  base_url: string;
  tracing: {
    enabled: boolean;
    sample_rate: number;
    trace_moderation: boolean;
    trace_providers: boolean;
    trace_security: boolean;
  };
  evaluations: {
    enabled: boolean;
    run_async: boolean;
    timeout: number;
    evaluators: string[];
  };
}

interface OpikTestConnectionResult {
  connected: boolean;
  trace_id: string;
  project: string;
  timestamp: string;
  message: string;
}

export const opikService = {
  /**
   * Get Opik integration status
   */
  async getStatus(): Promise<OpikStatus> {
    const response = await apiRequest('/api/opik/status', {
      method: 'GET',
      requireAuth: false,
    });

    if (!response.ok) {
      throw new Error('Failed to fetch Opik status');
    }

    const data: ApiResponse<OpikStatus> = await response.json();
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch Opik status');
    }

    return data.data;
  },

  /**
   * Get recent traces from Opik
   */
  async getTraces(): Promise<OpikTrace[]> {
    const response = await apiRequest('/api/opik/traces', {
      method: 'GET',
      requireAuth: false,
    });

    if (!response.ok) {
      throw new Error('Failed to fetch Opik traces');
    }

    const data: ApiResponse<OpikTrace[]> = await response.json();
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch Opik traces');
    }

    return data.data;
  },

  /**
   * Get evaluation results from Opik
   */
  async getEvaluations(): Promise<OpikEvaluations> {
    const response = await apiRequest('/api/opik/evaluations', {
      method: 'GET',
      requireAuth: false,
    });

    if (!response.ok) {
      throw new Error('Failed to fetch Opik evaluations');
    }

    const data: ApiResponse<OpikEvaluations> = await response.json();
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch Opik evaluations');
    }

    return data.data;
  },

  /**
   * Get Opik configuration
   */
  async getConfig(): Promise<OpikConfig> {
    const response = await apiRequest('/api/opik/config', {
      method: 'GET',
      requireAuth: false,
    });

    if (!response.ok) {
      throw new Error('Failed to fetch Opik configuration');
    }

    const data: ApiResponse<OpikConfig> = await response.json();
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch Opik configuration');
    }

    return data.data;
  },

  /**
   * Update Opik configuration
   */
  async updateConfig(config: Partial<OpikConfig>): Promise<void> {
    const response = await apiRequest('/api/opik/config', {
      method: 'PUT',
      requireAuth: false,
      body: JSON.stringify(config),
    });

    if (!response.ok) {
      throw new Error('Failed to update Opik configuration');
    }

    const data: ApiResponse<any> = await response.json();
    if (!data.success) {
      throw new Error(data.error || 'Failed to update Opik configuration');
    }
  },

  /**
   * Test connection to Opik
   */
  async testConnection(): Promise<OpikTestConnectionResult> {
    const response = await apiRequest('/api/opik/test-connection', {
      method: 'POST',
      requireAuth: false,
    });

    if (!response.ok) {
      throw new Error('Failed to test Opik connection');
    }

    const data: ApiResponse<OpikTestConnectionResult> = await response.json();
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to test Opik connection');
    }

    return data.data;
  },
};

export type {
  OpikStatus,
  OpikTrace,
  OpikEvaluations,
  OpikConfig,
  OpikTestConnectionResult,
}; 