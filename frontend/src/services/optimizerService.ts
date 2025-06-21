/**
 * Service for interacting with the Self-Optimizing Rules Engine API
 */

import { apiGet, apiPost, apiPut, apiDelete } from '../utils/api';

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

interface OptimizerMetrics {
  detection_rate: number;
  false_positive_rate: number;
  response_time_ms: number;
  user_satisfaction: number;
  fitness_score: number;
}

interface ExperimentConfig {
  name: string;
  objectives: string[];
  control_rule_id: string;
  variant_rules: any[];
  traffic_split: {
    control: number;
    variants: Record<string, number>;
  };
  duration_hours: number;
  min_sample_size: number;
  early_stopping_enabled: boolean;
}

interface DriftDetectionConfig {
  metric_name: string;
  enabled: boolean;
  sensitivity: 'low' | 'medium' | 'high';
  methods: string[];
  thresholds: {
    alert: number;
    critical: number;
  };
}

class OptimizerService {
  private baseUrl: string;

  constructor(baseUrl: string = '/api/optimizer') {
    this.baseUrl = baseUrl;
  }

  /**
   * Fetch current optimizer metrics
   */
  async getCurrentMetrics(): Promise<any> {
    const data: ApiResponse<any> = await apiGet(`${this.baseUrl}/status`);
    
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch optimizer metrics');
    }
    
    return data.data;
  }

  /**
   * Fetch historical metrics data
   */
  async getHistoricalMetrics(timeRange: string): Promise<any[]> {
    const data: ApiResponse<any[]> = await apiGet(`${this.baseUrl}/metrics/historical?range=${timeRange}`);
    
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch historical metrics');
    }
    
    return data.data;
  }

  /**
   * Fetch active experiments
   */
  async getExperiments(): Promise<any[]> {
    const data: ApiResponse<any[]> = await apiGet(`${this.baseUrl}/experiments`);
    
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch experiments');
    }
    
    return data.data;
  }

  /**
   * Create a new experiment
   */
  async createExperiment(config: ExperimentConfig): Promise<any> {
    try {
      const data: ApiResponse<any> = await apiPost(`${this.baseUrl}/experiments`, config);
      
      if (!data.success) {
        throw new Error(data.error || 'Failed to create experiment');
      }
      
      return data.data;
    } catch (error) {
      console.error('Error creating experiment:', error);
      throw error;
    }
  }

  /**
   * Fetch drift detection alerts
   */
  async getDriftAlerts(): Promise<any[]> {
    const data: ApiResponse<any[]> = await apiGet(`${this.baseUrl}/drift/alerts`);
    
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch drift alerts');
    }
    
    return data.data;
  }

  /**
   * Fetch rollout status
   */
  async getRolloutStatus(): Promise<any[]> {
    const data: ApiResponse<any[]> = await apiGet(`${this.baseUrl}/rollouts`);
    
    if (!data.success || !data.data) {
      throw new Error(data.error || 'Failed to fetch rollout status');
    }
    
    return data.data;
  }

  /**
   * Generate new rule variants using genetic algorithms
   */
  async generateRuleVariants(baseRuleId: string, count: number = 3): Promise<any[]> {
    try {
      const data: ApiResponse<any[]> = await apiPost(`${this.baseUrl}/rules/generate`, 
        { base_rule_id: baseRuleId, count });
      
      if (!data.success || !data.data) {
        throw new Error(data.error || 'Failed to generate rule variants');
      }
      
      return data.data;
    } catch (error) {
      console.error('Error generating rule variants:', error);
      throw error;
    }
  }

  /**
   * Start a rollout
   */
  async startRollout(deploymentId: string, strategy: string): Promise<any> {
    try {
      const data: ApiResponse<any> = await apiPost(`${this.baseUrl}/rollouts`, 
        { deployment_id: deploymentId, strategy });
      
      if (!data.success) {
        throw new Error(data.error || 'Failed to start rollout');
      }
      
      return data.data;
    } catch (error) {
      console.error('Error starting rollout:', error);
      throw error;
    }
  }

  /**
   * Trigger a rollback
   */
  async triggerRollback(rolloutId: string): Promise<any> {
    try {
      const data: ApiResponse<any> = await apiPost(`${this.baseUrl}/rollouts/${rolloutId}/rollback`);
      
      if (!data.success) {
        throw new Error(data.error || 'Failed to trigger rollback');
      }
      
      return data.data;
    } catch (error) {
      console.error('Error triggering rollback:', error);
      throw error;
    }
  }

  /**
   * Configure drift detection
   */
  async configureDriftDetection(config: DriftDetectionConfig): Promise<any> {
    try {
      const data: ApiResponse<any> = await apiPost(`${this.baseUrl}/drift/config`, config);
      
      if (!data.success) {
        throw new Error(data.error || 'Failed to configure drift detection');
      }
      
      return data.data;
    } catch (error) {
      console.error('Error configuring drift detection:', error);
      throw error;
    }
  }
}

// Export singleton instance
export const optimizerService = new OptimizerService();
export default OptimizerService;