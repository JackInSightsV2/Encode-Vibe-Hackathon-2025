// Provider service for API communication
class ProviderService {
  private baseUrl: string;

  constructor() {
    this.baseUrl = process.env.REACT_APP_API_URL || '/api';
  }

  // Get all providers
  async getProviders(): Promise<Record<string, any>> {
    const response = await fetch(`${this.baseUrl}/providers`);
    if (!response.ok) {
      throw new Error(`Failed to fetch providers: ${response.statusText}`);
    }
    return response.json();
  }

  // Get provider health information
  async getProvidersHealth(): Promise<Record<string, any>> {
    const response = await fetch(`${this.baseUrl}/providers/health`);
    if (!response.ok) {
      throw new Error(`Failed to fetch provider health: ${response.statusText}`);
    }
    return response.json();
  }

  // Get provider metrics
  async getProvidersMetrics(): Promise<Record<string, any>> {
    const response = await fetch(`${this.baseUrl}/providers/metrics`);
    if (!response.ok) {
      throw new Error(`Failed to fetch provider metrics: ${response.statusText}`);
    }
    return response.json();
  }

  // Get provider configurations
  async getProvidersConfigs(): Promise<Record<string, any>> {
    const response = await fetch(`${this.baseUrl}/providers/configs`);
    if (!response.ok) {
      throw new Error(`Failed to fetch provider configs: ${response.statusText}`);
    }
    return response.json();
  }

  // Get specific provider
  async getProvider(name: string): Promise<any> {
    const response = await fetch(`${this.baseUrl}/providers/${name}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch provider ${name}: ${response.statusText}`);
    }
    return response.json();
  }

  // Enable provider
  async enableProvider(name: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/providers/${name}/enable`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(`Failed to enable provider ${name}: ${response.statusText}`);
    }
  }

  // Disable provider
  async disableProvider(name: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/providers/${name}/disable`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(`Failed to disable provider ${name}: ${response.statusText}`);
    }
  }

  // Update provider configuration
  async updateProviderConfig(name: string, config: any): Promise<void> {
    const response = await fetch(`${this.baseUrl}/providers/${name}/config`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config),
    });
    if (!response.ok) {
      throw new Error(`Failed to update provider config ${name}: ${response.statusText}`);
    }
  }

  // Test provider
  async testProvider(name: string, config: any): Promise<any> {
    const response = await fetch(`${this.baseUrl}/providers/${name}/test`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config),
    });
    if (!response.ok) {
      throw new Error(`Failed to test provider ${name}: ${response.statusText}`);
    }
    return response.json();
  }

  // Get provider health details
  async getProviderHealth(name: string): Promise<any> {
    const response = await fetch(`${this.baseUrl}/providers/${name}/health`);
    if (!response.ok) {
      throw new Error(`Failed to fetch provider health ${name}: ${response.statusText}`);
    }
    return response.json();
  }

  // Get provider metrics details
  async getProviderMetrics(name: string, timeRange = '1h'): Promise<any> {
    const response = await fetch(`${this.baseUrl}/providers/${name}/metrics?range=${timeRange}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch provider metrics ${name}: ${response.statusText}`);
    }
    return response.json();
  }

  // Get routing statistics
  async getRoutingStats(): Promise<any> {
    const response = await fetch(`${this.baseUrl}/routing/stats`);
    if (!response.ok) {
      throw new Error(`Failed to fetch routing stats: ${response.statusText}`);
    }
    return response.json();
  }

  // Get circuit breaker states
  async getCircuitBreakerStates(): Promise<Record<string, any>> {
    const response = await fetch(`${this.baseUrl}/circuit-breakers`);
    if (!response.ok) {
      throw new Error(`Failed to fetch circuit breaker states: ${response.statusText}`);
    }
    return response.json();
  }

  // Reset circuit breaker
  async resetCircuitBreaker(name: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/circuit-breakers/${name}/reset`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(`Failed to reset circuit breaker ${name}: ${response.statusText}`);
    }
  }

  // Export provider configuration
  async exportConfig(): Promise<Blob> {
    const response = await fetch(`${this.baseUrl}/providers/config/export`);
    if (!response.ok) {
      throw new Error(`Failed to export config: ${response.statusText}`);
    }
    return response.blob();
  }

  // Import provider configuration
  async importConfig(file: File): Promise<void> {
    const formData = new FormData();
    formData.append('config', file);

    const response = await fetch(`${this.baseUrl}/providers/config/import`, {
      method: 'POST',
      body: formData,
    });
    if (!response.ok) {
      throw new Error(`Failed to import config: ${response.statusText}`);
    }
  }

  // Save configuration
  async saveConfig(): Promise<void> {
    const response = await fetch(`${this.baseUrl}/providers/config/save`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(`Failed to save config: ${response.statusText}`);
    }
  }

  // Reload configuration
  async reloadConfig(): Promise<void> {
    const response = await fetch(`${this.baseUrl}/providers/config/reload`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(`Failed to reload config: ${response.statusText}`);
    }
  }

  // Enable hot reload
  async enableHotReload(): Promise<void> {
    const response = await fetch(`${this.baseUrl}/providers/config/hot-reload/enable`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(`Failed to enable hot reload: ${response.statusText}`);
    }
  }

  // Disable hot reload
  async disableHotReload(): Promise<void> {
    const response = await fetch(`${this.baseUrl}/providers/config/hot-reload/disable`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(`Failed to disable hot reload: ${response.statusText}`);
    }
  }

  // Get provider types
  async getProviderTypes(): Promise<string[]> {
    const response = await fetch(`${this.baseUrl}/providers/types`);
    if (!response.ok) {
      throw new Error(`Failed to fetch provider types: ${response.statusText}`);
    }
    return response.json();
  }

  // Validate provider configuration
  async validateConfig(config: any): Promise<any> {
    const response = await fetch(`${this.baseUrl}/providers/config/validate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config),
    });
    if (!response.ok) {
      throw new Error(`Failed to validate config: ${response.statusText}`);
    }
    return response.json();
  }

  // Send test request to provider
  async sendTestRequest(providerName: string, message: string): Promise<any> {
    const response = await fetch(`${this.baseUrl}/providers/${providerName}/test-request`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        user_id: 'test',
        session_id: 'test',
        message: message,
      }),
    });
    if (!response.ok) {
      throw new Error(`Failed to send test request: ${response.statusText}`);
    }
    return response.json();
  }

  // Get system health
  async getSystemHealth(): Promise<any> {
    const response = await fetch(`${this.baseUrl}/health`);
    if (!response.ok) {
      throw new Error(`Failed to fetch system health: ${response.statusText}`);
    }
    return response.json();
  }

  // Get system status
  async getSystemStatus(): Promise<any> {
    const response = await fetch(`${this.baseUrl}/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch system status: ${response.statusText}`);
    }
    return response.json();
  }
}

export const providerService = new ProviderService();