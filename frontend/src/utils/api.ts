// API utility functions for authenticated requests

interface ApiRequestOptions extends RequestInit {
  requireAuth?: boolean;
}

// Use relative URL for production deployment
const API_BASE_URL = `${window.location.protocol}//${window.location.host}`;

export const apiRequest = async (
  endpoint: string, 
  options: ApiRequestOptions = {}
): Promise<Response> => {
  const { requireAuth = true, headers = {}, ...restOptions } = options;
  
  const requestHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  // Add custom headers
  Object.entries(headers).forEach(([key, value]) => {
    if (typeof value === 'string') {
      requestHeaders[key] = value;
    }
  });

  // Add Authorization header if auth is required
  if (requireAuth) {
    const token = localStorage.getItem('authToken');
    if (token) {
      requestHeaders['Authorization'] = `Bearer ${token}`;
    }
  }

  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE_URL}${endpoint}`;

  return fetch(url, {
    headers: requestHeaders,
    ...restOptions,
  });
};

export const apiGet = async (endpoint: string, requireAuth = true): Promise<any> => {
  const response = await apiRequest(endpoint, { 
    method: 'GET', 
    requireAuth 
  });
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }
  
  return response.json();
};

export const apiPost = async (
  endpoint: string, 
  data?: any, 
  requireAuth = true
): Promise<any> => {
  const response = await apiRequest(endpoint, {
    method: 'POST',
    body: data ? JSON.stringify(data) : undefined,
    requireAuth,
  });
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }
  
  return response.json();
};

export const apiPut = async (
  endpoint: string, 
  data?: any, 
  requireAuth = true
): Promise<any> => {
  const response = await apiRequest(endpoint, {
    method: 'PUT',
    body: data ? JSON.stringify(data) : undefined,
    requireAuth,
  });
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }
  
  return response.json();
};

export const apiDelete = async (
  endpoint: string, 
  requireAuth = true
): Promise<any> => {
  const response = await apiRequest(endpoint, {
    method: 'DELETE',
    requireAuth,
  });
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }
  
  return response.json();
}; 