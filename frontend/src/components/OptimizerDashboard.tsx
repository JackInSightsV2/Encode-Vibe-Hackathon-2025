import React, { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import { Badge } from './ui/Badge';
import { Button } from './ui/Button';
import { Progress } from './ui/progress';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, BarChart, Bar, ScatterChart, Scatter } from 'recharts';
import { Activity, AlertTriangle, CheckCircle, TrendingUp, Zap, Eye, Clock, Users, Target } from 'lucide-react';
import { optimizerService } from '../services/optimizerService';

// Utility functions
const getStatusColor = (status: string) => {
  switch (status) {
    case 'active': case 'running': return 'bg-green-100 text-green-800';
    case 'paused': case 'pending': return 'bg-yellow-100 text-yellow-800';
    case 'stopped': case 'failed': return 'bg-red-100 text-red-800';
    case 'completed': return 'bg-blue-100 text-blue-800';
    default: return 'bg-gray-100 text-gray-800';
  }
};

const getSeverityColor = (severity: string) => {
  switch (severity) {
    case 'critical': return 'bg-red-100 text-red-800';
    case 'high': return 'bg-orange-100 text-orange-800';
    case 'medium': return 'bg-yellow-100 text-yellow-800';
    case 'low': return 'bg-green-100 text-green-800';
    default: return 'bg-gray-100 text-gray-800';
  }
};

interface OptimizerMetrics {
  detection_rate: number;
  false_positive_rate: number;
  response_time_ms: number;
  user_satisfaction: number;
  fitness_score: number;
}

interface Experiment {
  id: string;
  name: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  start_time: string;
  end_time?: string;
  control: Rule;
  variants: Rule[];
  results?: ExperimentResults;
  traffic_split: TrafficSplit;
}

interface Rule {
  id: string;
  name: string;
  type: 'regex' | 'semantic' | 'hybrid' | 'pii';
  parameters: Record<string, any>;
  enabled: boolean;
  priority: number;
  fitness: number;
}

interface ExperimentResults {
  winner?: VariantMetrics;
  significance?: SignificanceTest;
  confidence_level: number;
  summary: string;
  recommended_action: string;
  variant_metrics: Record<string, VariantMetrics>;
}

interface VariantMetrics {
  variant_id: string;
  sample_size: number;
  detection_rate: number;
  false_positive_rate: number;
  avg_response_time: number;
  user_satisfaction: number;
  fitness_score: number;
}

interface SignificanceTest {
  p_value: number;
  significant: boolean;
  improvement: number;
  method: string;
}

interface TrafficSplit {
  control: number;
  variants: Record<string, number>;
}

interface DriftAlert {
  id: string;
  metric_name: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  message: string;
  triggered_at: string;
  resolved_at?: string;
  status: 'active' | 'resolved' | 'muted';
  drift_score: number;
  threshold: number;
}

interface RolloutStatus {
  id: string;
  deployment_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'rolled_back';
  current_phase?: string;
  traffic_percent: number;
  started_at: string;
  completed_at?: string;
  strategy: 'immediate' | 'gradual' | 'canary' | 'blue_green';
}

const OptimizerDashboard: React.FC = () => {
  const [currentMetrics, setCurrentMetrics] = useState<OptimizerMetrics | null>(null);
  const [experiments, setExperiments] = useState<Experiment[]>([]);
  const [driftAlerts, setDriftAlerts] = useState<DriftAlert[]>([]);
  const [rolloutStatus, setRolloutStatus] = useState<RolloutStatus[]>([]);
  const [historicalData, setHistoricalData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTimeRange, setSelectedTimeRange] = useState('24h');

  // Simulate real-time data updates
  useEffect(() => {
    const interval = setInterval(() => {
      fetchDashboardData();
    }, 30000); // Update every 30 seconds

    fetchDashboardData();

    return () => clearInterval(interval);
  }, [selectedTimeRange]);

  const fetchDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Use the optimizer service to fetch real data
      const [metricsData, experimentsData, alertsData, rolloutsData, historicalDataResponse] = await Promise.all([
        optimizerService.getCurrentMetrics(),
        optimizerService.getExperiments(),
        optimizerService.getDriftAlerts(),
        optimizerService.getRolloutStatus(),
        optimizerService.getHistoricalMetrics(selectedTimeRange)
      ]);

      setCurrentMetrics(metricsData);
      setExperiments(experimentsData);
      setDriftAlerts(alertsData);
      setRolloutStatus(rolloutsData);
      setHistoricalData(historicalDataResponse);
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
      setError(error instanceof Error ? error.message : 'Failed to load optimizer data');
    } finally {
      setLoading(false);
    }
  }, [selectedTimeRange]);

  const getSeverityColorInternal = (severity: string) => {
    switch (severity) {
      case 'critical': return 'text-red-600 bg-red-50';
      case 'high': return 'text-orange-600 bg-orange-50';
      case 'medium': return 'text-yellow-600 bg-yellow-50';
      case 'low': return 'text-blue-600 bg-blue-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  const getStatusColorInternal = (status: string) => {
    switch (status) {
      case 'running': return 'text-green-600 bg-green-50';
      case 'completed': return 'text-blue-600 bg-blue-50';
      case 'failed': return 'text-red-600 bg-red-50';
      case 'pending': return 'text-yellow-600 bg-yellow-50';
      case 'rolled_back': return 'text-orange-600 bg-orange-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  const formatNumber = (num: number, decimals: number = 3) => {
    return num.toFixed(decimals);
  };

  const formatPercentage = (num: number) => {
    return `${(num * 100).toFixed(1)}%`;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-64 space-y-4">
        <div className="text-red-600 dark:text-red-400">
          <AlertTriangle className="h-8 w-8 mx-auto mb-2" />
          <p className="text-lg font-semibold">Failed to Load Optimizer Data</p>
          <p className="text-sm text-muted-foreground">{error}</p>
        </div>
        <Button onClick={fetchDashboardData} variant="outline">
          Retry
        </Button>
      </div>
    );
  }

  if (!currentMetrics) {
    return (
      <div className="flex flex-col items-center justify-center h-64 space-y-4">
        <div className="text-muted-foreground text-center">
          <TrendingUp className="h-8 w-8 mx-auto mb-2" />
          <p className="text-lg font-semibold">No Optimizer Data Available</p>
          <p className="text-sm">Start optimization experiments to see metrics here</p>
        </div>
        <Button onClick={fetchDashboardData} variant="outline">
          Refresh
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Self-Optimizing Rules Engine</h1>
          <p className="text-muted-foreground">
            AI-powered rule optimization using genetic algorithms and A/B testing
          </p>
        </div>
        <div className="flex space-x-2">
          <Button variant="outline" onClick={() => setSelectedTimeRange('1h')}>1H</Button>
          <Button variant="outline" onClick={() => setSelectedTimeRange('24h')}>24H</Button>
          <Button variant="outline" onClick={() => setSelectedTimeRange('7d')}>7D</Button>
          <Button variant="outline" onClick={() => setSelectedTimeRange('30d')}>30D</Button>
        </div>
      </div>

      {/* Key Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Detection Rate</p>
                <p className="text-2xl font-bold">{formatPercentage(currentMetrics?.detection_rate || 0)}</p>
              </div>
              <Target className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">False Positive Rate</p>
                <p className="text-2xl font-bold">{formatPercentage(currentMetrics?.false_positive_rate || 0)}</p>
              </div>
              <AlertTriangle className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Response Time</p>
                <p className="text-2xl font-bold">{formatNumber(currentMetrics?.response_time_ms || 0, 1)}ms</p>
              </div>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">User Satisfaction</p>
                <p className="text-2xl font-bold">{formatPercentage(currentMetrics?.user_satisfaction || 0)}</p>
              </div>
              <Users className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Fitness Score</p>
                <p className="text-2xl font-bold">{formatNumber(currentMetrics?.fitness_score || 0)}</p>
              </div>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Tabs */}
      <Tabs defaultValue="experiments" className="space-y-4">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="experiments">Experiments</TabsTrigger>
          <TabsTrigger value="rollouts">Rollouts</TabsTrigger>
          <TabsTrigger value="drift">Drift Detection</TabsTrigger>
          <TabsTrigger value="rules">Rules</TabsTrigger>
          <TabsTrigger value="analytics">Analytics</TabsTrigger>
          <TabsTrigger value="opik">Opik Integration</TabsTrigger>
        </TabsList>

        <TabsContent value="experiments">
          <ExperimentsTab experiments={experiments} />
        </TabsContent>

        <TabsContent value="rollouts">
          <RolloutsTab rollouts={rolloutStatus} />
        </TabsContent>

        <TabsContent value="drift">
          <DriftDetectionTab alerts={driftAlerts} />
        </TabsContent>

        <TabsContent value="rules">
          <RulesTab />
        </TabsContent>

        <TabsContent value="analytics">
          <AnalyticsTab historicalData={historicalData} />
        </TabsContent>

        <TabsContent value="opik">
          <OpikIntegrationTab />
        </TabsContent>
      </Tabs>
    </div>
  );
};

// Experiments Tab Component
const ExperimentsTab: React.FC<{ experiments: Experiment[] }> = ({ experiments }) => {
  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-semibold">A/B Testing Experiments</h2>
        <Button>
          <Zap className="w-4 h-4 mr-2" />
          Create Experiment
        </Button>
      </div>

      <div className="grid gap-4">
        {experiments.length === 0 ? (
          <Card>
            <CardContent className="p-6">
              <div className="text-center py-8">
                <p className="text-gray-500">No experiments currently running. Create one to get started!</p>
              </div>
            </CardContent>
          </Card>
        ) : (
          experiments.map((experiment) => (
            <Card key={experiment.id}>
              <CardHeader>
                <div className="flex justify-between items-start">
                  <div>
                    <CardTitle className="text-lg">{experiment.name}</CardTitle>
                    <CardDescription>
                      {experiment.variants.length} variants • {experiment.control.type} rule type
                    </CardDescription>
                  </div>
                  <Badge className={getStatusColor(experiment.status)}>
                    {experiment.status}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                  {/* Traffic Split */}
                  <div>
                    <h4 className="font-medium mb-2">Traffic Split</h4>
                    <div className="space-y-1">
                      <div className="flex justify-between">
                        <span className="text-sm">Control</span>
                        <span className="text-sm font-medium">{(experiment.traffic_split.control * 100).toFixed(0)}%</span>
                      </div>
                      {Object.entries(experiment.traffic_split.variants).map(([variantId, split]) => (
                        <div key={variantId} className="flex justify-between">
                          <span className="text-sm">{variantId}</span>
                          <span className="text-sm font-medium">{(split * 100).toFixed(0)}%</span>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Results */}
                  {experiment.results && (
                    <div>
                      <h4 className="font-medium mb-2">Results</h4>
                      <div className="space-y-1">
                        <div className="flex justify-between">
                          <span className="text-sm">Winner</span>
                          <span className="text-sm font-medium">
                            {experiment.results.winner?.variant_id || 'TBD'}
                          </span>
                        </div>
                        {experiment.results.significance && (
                          <>
                            <div className="flex justify-between">
                              <span className="text-sm">Significant</span>
                              <span className="text-sm font-medium">
                                {experiment.results.significance.significant ? 'Yes' : 'No'}
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-sm">Improvement</span>
                              <span className="text-sm font-medium">
                                {(experiment.results.significance.improvement * 100).toFixed(1)}%
                              </span>
                            </div>
                          </>
                        )}
                      </div>
                    </div>
                  )}

                  {/* Actions */}
                  <div>
                    <h4 className="font-medium mb-2">Actions</h4>
                    <div className="space-y-1">
                      {experiment.results && experiment.results.recommended_action && (
                        <p className="text-sm text-gray-600 mb-2">{experiment.results.recommended_action}</p>
                      )}
                      <div className="space-x-2">
                        <Button size="sm" variant="outline">View Details</Button>
                        {experiment.status === 'running' && (
                          <Button size="sm" variant="outline">Stop</Button>
                        )}
                        {experiment.status === 'completed' && experiment.results?.winner && (
                          <Button size="sm">Promote Winner</Button>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
};

// Rollouts Tab Component  
const RolloutsTab: React.FC<{ rollouts: RolloutStatus[] }> = ({ rollouts }) => {
  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-semibold">Deployment Rollouts</h2>
        <Button>
          <TrendingUp className="w-4 h-4 mr-2" />
          New Rollout
        </Button>
      </div>

      <div className="grid gap-4">
        {rollouts.length === 0 ? (
          <Card>
            <CardContent className="p-6">
              <div className="text-center py-8">
                <p className="text-gray-500">No active rollouts. Deploy a new rule version to get started!</p>
              </div>
            </CardContent>
          </Card>
        ) : (
          rollouts.map((rollout) => (
            <Card key={rollout.id}>
              <CardHeader>
                <div className="flex justify-between items-start">
                  <div>
                    <CardTitle className="text-lg">Rollout {rollout.id}</CardTitle>
                    <CardDescription>
                      {rollout.strategy} strategy • {rollout.current_phase || 'Initializing'}
                    </CardDescription>
                  </div>
                  <Badge className={getStatusColor(rollout.status)}>
                    {rollout.status}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {/* Progress */}
                  <div>
                    <div className="flex justify-between mb-2">
                      <span className="text-sm font-medium">Progress</span>
                      <span className="text-sm">{rollout.traffic_percent}%</span>
                    </div>
                    <Progress value={rollout.traffic_percent} className="h-2" />
                  </div>

                  {/* Timeline */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-gray-500">Started:</span>
                      <span className="ml-2">{new Date(rollout.started_at).toLocaleString()}</span>
                    </div>
                    {rollout.completed_at && (
                      <div>
                        <span className="text-gray-500">Completed:</span>
                        <span className="ml-2">{new Date(rollout.completed_at).toLocaleString()}</span>
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex space-x-2">
                    <Button size="sm" variant="outline">View Details</Button>
                    {rollout.status === 'running' && (
                      <>
                        <Button size="sm" variant="outline">Pause</Button>
                        <Button size="sm" variant="danger">Rollback</Button>
                      </>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
};

// Drift Detection Tab Component
const DriftDetectionTab: React.FC<{ alerts: DriftAlert[] }> = ({ alerts }) => {
  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-semibold">Safety Drift Detection</h2>
        <Button>
          <Eye className="w-4 h-4 mr-2" />
          Configure Detection
        </Button>
      </div>

      <div className="grid gap-4">
        {alerts.length === 0 ? (
          <Card>
            <CardContent className="p-6">
              <div className="text-center py-8">
                <CheckCircle className="w-8 h-8 text-green-600 mx-auto mb-2" />
                <p className="text-gray-500">No drift alerts detected. Your models are performing well!</p>
              </div>
            </CardContent>
          </Card>
        ) : (
          alerts.map((alert) => (
            <Card key={alert.id}>
              <CardHeader>
                <div className="flex justify-between items-start">
                  <div>
                    <CardTitle className="text-lg">{alert.metric_name}</CardTitle>
                    <CardDescription>{alert.message}</CardDescription>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Badge className={getSeverityColor(alert.severity)}>
                      {alert.severity}
                    </Badge>
                    <Badge className={getStatusColor(alert.status)}>
                      {alert.status}
                    </Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div>
                    <span className="text-sm text-gray-500">Drift Score:</span>
                    <span className="ml-2 font-medium">{alert.drift_score.toFixed(4)}</span>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500">Threshold:</span>
                    <span className="ml-2 font-medium">{alert.threshold.toFixed(4)}</span>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500">Triggered:</span>
                    <span className="ml-2 font-medium">
                      {new Date(alert.triggered_at).toLocaleString()}
                    </span>
                  </div>
                </div>
                {alert.status === 'active' && (
                  <div className="mt-4 flex space-x-2">
                    <Button size="sm">Investigate</Button>
                    <Button size="sm" variant="outline">Mute</Button>
                    <Button size="sm" variant="outline">Resolve</Button>
                  </div>
                )}
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
};

// Rules Tab Component
const RulesTab: React.FC = () => {
  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-semibold">Safety Rules Management</h2>
        <Button>
          <Zap className="w-4 h-4 mr-2" />
          Generate New Rule
        </Button>
      </div>
      
      <Card>
        <CardContent className="p-6">
          <div className="text-center py-8">
            <Activity className="w-8 h-8 text-blue-600 mx-auto mb-2" />
            <p className="text-gray-500">Rules management interface coming soon...</p>
            <p className="text-sm text-gray-400 mt-1">AI-generated rules with genetic optimization</p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

// Analytics Tab Component
const AnalyticsTab: React.FC<{ historicalData: any[] }> = ({ historicalData }) => {
  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold">Advanced Analytics</h2>
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Performance Distribution */}
        <Card>
          <CardHeader>
            <CardTitle>Performance Distribution</CardTitle>
            <CardDescription>Distribution of key metrics</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={historicalData.slice(-24)}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="timestamp" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="detection_rate" fill="#10b981" />
                <Bar dataKey="user_satisfaction" fill="#8b5cf6" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Correlation Analysis */}
        <Card>
          <CardHeader>
            <CardTitle>Metrics Correlation</CardTitle>
            <CardDescription>Relationship between response time and detection rate</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <ScatterChart data={historicalData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="response_time_ms" name="Response Time (ms)" />
                <YAxis dataKey="detection_rate" name="Detection Rate" />
                <Tooltip cursor={{ strokeDasharray: '3 3' }} />
                <Scatter name="Metrics" fill="#3b82f6" />
              </ScatterChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

// Opik Integration Tab Component
const OpikIntegrationTab: React.FC = () => {
  const [opikConfig, setOpikConfig] = useState({
    enabled: false,
    apiKey: '',
    baseUrl: 'https://api.opik.com',
    projectName: 'qt1-self-optimizing-rules',
    batchSize: 100,
    flushInterval: 5,
    tracing: {
      enabled: true,
      sampleRate: 1.0,
      traceModeration: true,
      traceProviders: true,
      traceSecurity: true,
    },
    evaluations: {
      enabled: true,
      runAsync: true,
      timeout: 30,
      evaluators: ['accuracy', 'drift_detection', 'performance']
    }
  });

  const [connectionStatus, setConnectionStatus] = useState('disconnected');
  const [testResults, setTestResults] = useState<any>(null);

  const handleConfigChange = (key: string, value: any) => {
    setOpikConfig(prev => ({
      ...prev,
      [key]: value
    }));
  };

  const handleNestedConfigChange = (section: string, key: string, value: any) => {
    setOpikConfig(prev => ({
      ...prev,
      [section]: {
        ...(prev[section as keyof typeof prev] as any),
        [key]: value
      }
    }));
  };

  const testConnection = async () => {
    setConnectionStatus('testing');
    try {
      // Test Opik connection
      const response = await fetch('/api/safety-cockpit/test-connection', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          baseUrl: opikConfig.baseUrl,
          apiKey: opikConfig.apiKey,
          projectName: opikConfig.projectName
        })
      });
      
      const result = await response.json();
      if (result.success) {
        setConnectionStatus('connected');
        setTestResults(result.data);
      } else {
        setConnectionStatus('error');
        setTestResults({ error: result.error });
      }
    } catch (error) {
      setConnectionStatus('error');
      setTestResults({ error: 'Connection failed' });
    }
  };

  const saveConfiguration = async () => {
    try {
      const response = await fetch('/api/safety-cockpit/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ opik: opikConfig })
      });
      
      if (response.ok) {
        alert('Configuration saved successfully!');
      } else {
        alert('Failed to save configuration');
      }
    } catch (error) {
      alert('Error saving configuration');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold">Opik Integration</h2>
          <p className="text-muted-foreground mt-1">
            Configure Opik for tracing and evaluation of self-optimizing rules
          </p>
        </div>
        <div className="flex space-x-2">
          <Button variant="outline" onClick={testConnection} disabled={connectionStatus === 'testing'}>
            {connectionStatus === 'testing' ? 'Testing...' : 'Test Connection'}
          </Button>
          <Button onClick={saveConfiguration}>
            Save Configuration
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Connection Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>Connection Settings</CardTitle>
            <CardDescription>Configure your Opik connection parameters</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">Enable Opik Integration</label>
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={opikConfig.enabled}
                  onChange={(e) => handleConfigChange('enabled', e.target.checked)}
                  className="rounded"
                />
                <span>Enable Opik tracing and evaluation</span>
              </label>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">API Key</label>
              <input
                type="password"
                value={opikConfig.apiKey}
                onChange={(e) => handleConfigChange('apiKey', e.target.value)}
                placeholder="Enter your Opik API key"
                className="w-full px-3 py-2 border border-gray-300 rounded-md"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Base URL</label>
              <input
                type="url"
                value={opikConfig.baseUrl}
                onChange={(e) => handleConfigChange('baseUrl', e.target.value)}
                placeholder="https://api.opik.com"
                className="w-full px-3 py-2 border border-gray-300 rounded-md"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Project Name</label>
              <input
                type="text"
                value={opikConfig.projectName}
                onChange={(e) => handleConfigChange('projectName', e.target.value)}
                placeholder="qt1-self-optimizing-rules"
                className="w-full px-3 py-2 border border-gray-300 rounded-md"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2">Batch Size</label>
                <input
                  type="number"
                  value={opikConfig.batchSize}
                  onChange={(e) => handleConfigChange('batchSize', parseInt(e.target.value))}
                  min="1"
                  max="1000"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Flush Interval (seconds)</label>
                <input
                  type="number"
                  value={opikConfig.flushInterval}
                  onChange={(e) => handleConfigChange('flushInterval', parseInt(e.target.value))}
                  min="1"
                  max="60"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Tracing Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>Tracing Configuration</CardTitle>
            <CardDescription>Configure what gets traced in Opik</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={opikConfig.tracing.enabled}
                  onChange={(e) => handleNestedConfigChange('tracing', 'enabled', e.target.checked)}
                  className="rounded"
                />
                <span>Enable tracing</span>
              </label>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Sample Rate</label>
              <input
                type="number"
                value={opikConfig.tracing.sampleRate}
                onChange={(e) => handleNestedConfigChange('tracing', 'sampleRate', parseFloat(e.target.value))}
                min="0"
                max="1"
                step="0.1"
                className="w-full px-3 py-2 border border-gray-300 rounded-md"
              />
              <p className="text-xs text-gray-500 mt-1">0.0 = no tracing, 1.0 = trace everything</p>
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium">Trace Categories</label>
              <div className="space-y-2">
                <label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={opikConfig.tracing.traceModeration}
                    onChange={(e) => handleNestedConfigChange('tracing', 'traceModeration', e.target.checked)}
                    className="rounded"
                  />
                  <span>Moderation events</span>
                </label>
                <label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={opikConfig.tracing.traceProviders}
                    onChange={(e) => handleNestedConfigChange('tracing', 'traceProviders', e.target.checked)}
                    className="rounded"
                  />
                  <span>Provider interactions</span>
                </label>
                <label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={opikConfig.tracing.traceSecurity}
                    onChange={(e) => handleNestedConfigChange('tracing', 'traceSecurity', e.target.checked)}
                    className="rounded"
                  />
                  <span>Security events</span>
                </label>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Evaluation Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>Evaluation Configuration</CardTitle>
            <CardDescription>Configure automatic evaluations for rule optimization</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={opikConfig.evaluations.enabled}
                  onChange={(e) => handleNestedConfigChange('evaluations', 'enabled', e.target.checked)}
                  className="rounded"
                />
                <span>Enable automatic evaluations</span>
              </label>
            </div>

            <div>
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={opikConfig.evaluations.runAsync}
                  onChange={(e) => handleNestedConfigChange('evaluations', 'runAsync', e.target.checked)}
                  className="rounded"
                />
                <span>Run evaluations asynchronously</span>
              </label>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Evaluation Timeout (seconds)</label>
              <input
                type="number"
                value={opikConfig.evaluations.timeout}
                onChange={(e) => handleNestedConfigChange('evaluations', 'timeout', parseInt(e.target.value))}
                min="1"
                max="300"
                className="w-full px-3 py-2 border border-gray-300 rounded-md"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Evaluators</label>
              <div className="space-y-2">
                {['accuracy', 'drift_detection', 'performance', 'false_positive_rate', 'user_satisfaction'].map((evaluator) => (
                  <label key={evaluator} className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      checked={opikConfig.evaluations.evaluators.includes(evaluator)}
                      onChange={(e) => {
                        const evaluators = e.target.checked
                          ? [...opikConfig.evaluations.evaluators, evaluator]
                          : opikConfig.evaluations.evaluators.filter(ev => ev !== evaluator);
                        handleNestedConfigChange('evaluations', 'evaluators', evaluators);
                      }}
                      className="rounded"
                    />
                    <span className="capitalize">{evaluator.replace('_', ' ')}</span>
                  </label>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Connection Status & Test Results */}
        <Card>
          <CardHeader>
            <CardTitle>Connection Status</CardTitle>
            <CardDescription>Current status of your Opik integration</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <div className={`w-3 h-3 rounded-full ${
                  connectionStatus === 'connected' ? 'bg-green-500' :
                  connectionStatus === 'testing' ? 'bg-yellow-500' :
                  connectionStatus === 'error' ? 'bg-red-500' : 'bg-gray-500'
                }`}></div>
                <span className="capitalize">{connectionStatus}</span>
              </div>

              {testResults && (
                <div className="bg-gray-50 p-4 rounded-md">
                  <h4 className="font-medium mb-2">Test Results</h4>
                  {testResults.error ? (
                    <p className="text-red-600 text-sm">{testResults.error}</p>
                  ) : (
                    <div className="text-sm space-y-1">
                      <p>Connection successful!</p>
                      <p>Project: {testResults.project || 'Unknown'}</p>
                      <p>Status: {testResults.status || 'Active'}</p>
                    </div>
                  )}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default OptimizerDashboard; 