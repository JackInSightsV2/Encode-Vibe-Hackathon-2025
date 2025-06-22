import React, { useState, useEffect, useCallback } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import { Badge } from './ui/Badge';
import { Button } from './ui/Button';
import { Progress } from './ui/progress';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, BarChart, Bar, ScatterChart, Scatter } from 'recharts';
import { Activity, AlertTriangle, CheckCircle, TrendingUp, Zap, Eye, Clock, Users, Target } from 'lucide-react';
import { optimizerService } from '../services/optimizerService';
import { opikService } from '../services/opikService';

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
  const [opikStatus, setOpikStatus] = useState<any>(null);
  const [traces, setTraces] = useState<any[]>([]);
  const [evaluations, setEvaluations] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [realTimeMetrics, setRealTimeMetrics] = useState({
    totalTraces: 0,
    successRate: 0,
    avgResponseTime: 0,
    moderationScore: 0,
    activeUsers: 0,
    blockedRequests: 0
  });

  // Load Opik data on component mount and refresh every 10 seconds
  useEffect(() => {
    loadOpikData();
    const interval = setInterval(loadOpikData, 10000);
    return () => clearInterval(interval);
  }, []);

  const loadOpikData = async () => {
    try {
      const [statusData, tracesData, evaluationsData] = await Promise.all([
        opikService.getStatus(),
        opikService.getTraces(),
        opikService.getEvaluations()
      ]);

      setOpikStatus(statusData);
      setTraces(tracesData);
      setEvaluations(evaluationsData);

      // Calculate real-time metrics from traces
      if (tracesData && tracesData.length > 0) {
        const recent = tracesData.slice(0, 50); // Last 50 traces
        const successfulTraces = recent.filter((t: any) => t.output?.allowed !== false);
        const blockedTraces = recent.filter((t: any) => t.output?.allowed === false);
        const avgScore = recent.reduce((sum: number, t: any) => sum + (t.output?.final_score || 0), 0) / recent.length;
        const avgTime = recent.reduce((sum: number, t: any) => sum + (t.output?.total_process_time || 0), 0) / recent.length;
        const uniqueUsers = new Set(recent.map((t: any) => t.metadata?.user_id)).size;

        setRealTimeMetrics({
          totalTraces: tracesData.length,
          successRate: (successfulTraces.length / recent.length) * 100,
          avgResponseTime: avgTime,
          moderationScore: avgScore * 100,
          activeUsers: uniqueUsers,
          blockedRequests: blockedTraces.length
        });
      }

      setLoading(false);
    } catch (error) {
      console.error('Failed to load Opik data:', error);
      setLoading(false);
    }
  };

  const getTraceChartData = () => {
    if (!traces || traces.length === 0) return [];
    
    // Group traces by hour for the last 24 hours
    const hourlyData: { [key: string]: { allowed: number; blocked: number; total: number } } = {};
    const now = new Date();
    
    // Initialize last 24 hours
    for (let i = 23; i >= 0; i--) {
      const hour = new Date(now.getTime() - i * 60 * 60 * 1000);
      const key = hour.toISOString().slice(0, 13) + ':00';
      hourlyData[key] = { allowed: 0, blocked: 0, total: 0 };
    }

    traces.forEach((trace: any) => {
      const traceTime = new Date(trace.start_time);
      const hourKey = traceTime.toISOString().slice(0, 13) + ':00';
      
      if (hourlyData[hourKey]) {
        hourlyData[hourKey].total++;
        if (trace.output?.allowed === false) {
          hourlyData[hourKey].blocked++;
        } else {
          hourlyData[hourKey].allowed++;
        }
      }
    });

    return Object.entries(hourlyData).map(([time, data]) => ({
      time: new Date(time).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }),
      allowed: data.allowed,
      blocked: data.blocked,
      total: data.total
    }));
  };

  const getModerationScoreDistribution = () => {
    if (!traces || traces.length === 0) return [];
    
    const buckets = { '0-20': 0, '21-40': 0, '41-60': 0, '61-80': 0, '81-100': 0 };
    
    traces.forEach((trace: any) => {
      const score = (trace.output?.final_score || 0) * 100;
      if (score <= 20) buckets['0-20']++;
      else if (score <= 40) buckets['21-40']++;
      else if (score <= 60) buckets['41-60']++;
      else if (score <= 80) buckets['61-80']++;
      else buckets['81-100']++;
    });

    return Object.entries(buckets).map(([range, count]) => ({
      range,
      count,
      percentage: traces.length > 0 ? (count / traces.length * 100).toFixed(1) : '0'
    }));
  };

  const getTopUsers = () => {
    if (!traces || traces.length === 0) return [];
    
    const userStats: { [key: string]: { requests: number; blocked: number; avgScore: number } } = {};
    
    traces.forEach((trace: any) => {
      const userId = trace.metadata?.user_id || 'unknown';
      if (!userStats[userId]) {
        userStats[userId] = { requests: 0, blocked: 0, avgScore: 0 };
      }
      userStats[userId].requests++;
      if (trace.output?.allowed === false) {
        userStats[userId].blocked++;
      }
      userStats[userId].avgScore += trace.output?.final_score || 0;
    });

    return Object.entries(userStats)
      .map(([userId, stats]) => ({
        userId,
        requests: stats.requests,
        blocked: stats.blocked,
        blockRate: (stats.blocked / stats.requests * 100).toFixed(1),
        avgScore: ((stats.avgScore / stats.requests) * 100).toFixed(1)
      }))
      .sort((a, b) => b.requests - a.requests)
      .slice(0, 5);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        <span className="ml-3 text-lg">Loading Opik Analytics...</span>
      </div>
    );
  }

  const chartData = getTraceChartData();
  const scoreDistribution = getModerationScoreDistribution();
  const topUsers = getTopUsers();

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
            🚀 Opik Analytics Dashboard
          </h2>
          <p className="text-muted-foreground mt-1">
            Real-time moderation insights powered by Comet Opik
          </p>
        </div>
        <div className="flex items-center space-x-4">
          <Badge className={`${opikStatus?.connected ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
            {opikStatus?.connected ? '🟢 Connected' : '🔴 Disconnected'}
          </Badge>
          <Button onClick={loadOpikData} variant="outline" size="sm">
            🔄 Refresh
          </Button>
        </div>
      </div>

      {/* Real-time Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
        <Card className="bg-gradient-to-br from-blue-50 to-blue-100 border-blue-200">
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-blue-600">Total Traces</p>
                <p className="text-2xl font-bold text-blue-900">{realTimeMetrics.totalTraces.toLocaleString()}</p>
              </div>
              <Activity className="h-8 w-8 text-blue-500" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-green-50 to-green-100 border-green-200">
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-green-600">Success Rate</p>
                <p className="text-2xl font-bold text-green-900">{realTimeMetrics.successRate.toFixed(1)}%</p>
              </div>
              <CheckCircle className="h-8 w-8 text-green-500" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-purple-50 to-purple-100 border-purple-200">
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-purple-600">Avg Response</p>
                <p className="text-2xl font-bold text-purple-900">{realTimeMetrics.avgResponseTime.toFixed(0)}ms</p>
              </div>
              <Clock className="h-8 w-8 text-purple-500" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-orange-50 to-orange-100 border-orange-200">
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-orange-600">Mod Score</p>
                <p className="text-2xl font-bold text-orange-900">{realTimeMetrics.moderationScore.toFixed(1)}</p>
              </div>
              <Target className="h-8 w-8 text-orange-500" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-indigo-50 to-indigo-100 border-indigo-200">
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-indigo-600">Active Users</p>
                <p className="text-2xl font-bold text-indigo-900">{realTimeMetrics.activeUsers}</p>
              </div>
              <Users className="h-8 w-8 text-indigo-500" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-red-50 to-red-100 border-red-200">
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-red-600">Blocked</p>
                <p className="text-2xl font-bold text-red-900">{realTimeMetrics.blockedRequests}</p>
              </div>
              <AlertTriangle className="h-8 w-8 text-red-500" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Trace Timeline Chart */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <TrendingUp className="h-5 w-5" />
              <span>Request Timeline (24h)</span>
            </CardTitle>
            <CardDescription>Allowed vs Blocked requests over time</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Line type="monotone" dataKey="allowed" stroke="#22c55e" strokeWidth={2} name="Allowed" />
                <Line type="monotone" dataKey="blocked" stroke="#ef4444" strokeWidth={2} name="Blocked" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Moderation Score Distribution */}
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Target className="h-5 w-5" />
              <span>Moderation Score Distribution</span>
            </CardTitle>
            <CardDescription>Distribution of moderation confidence scores</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={scoreDistribution}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="range" />
                <YAxis />
                <Tooltip formatter={(value, name) => [value, name === 'count' ? 'Requests' : name]} />
                <Bar dataKey="count" fill="#8884d8" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* Tables Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Traces */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Eye className="h-5 w-5" />
              <span>Recent Traces</span>
            </CardTitle>
            <CardDescription>Latest moderation requests</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3 max-h-80 overflow-y-auto">
              {traces.slice(0, 10).map((trace: any, index: number) => (
                <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2">
                      <Badge className={trace.output?.allowed === false ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'}>
                        {trace.output?.allowed === false ? '🚫 Blocked' : '✅ Allowed'}
                      </Badge>
                      <span className="text-sm text-gray-600">
                        {trace.metadata?.user_id || 'Unknown'}
                      </span>
                    </div>
                    <p className="text-xs text-gray-500 mt-1">
                      Score: {((trace.output?.final_score || 0) * 100).toFixed(1)} | 
                      Time: {trace.output?.total_process_time || 0}ms |
                      Layers: {trace.output?.layers_checked || 0}
                    </p>
                  </div>
                  <div className="text-xs text-gray-400">
                    {new Date(trace.start_time).toLocaleTimeString()}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Top Users */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Users className="h-5 w-5" />
              <span>Top Active Users</span>
            </CardTitle>
            <CardDescription>Users with most requests</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {topUsers.map((user: any, index: number) => (
                <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2">
                      <span className="font-medium">{user.userId}</span>
                      <Badge variant="outline">{user.requests} requests</Badge>
                    </div>
                    <div className="flex space-x-4 text-xs text-gray-500 mt-1">
                      <span>Block Rate: {user.blockRate}%</span>
                      <span>Avg Score: {user.avgScore}</span>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="text-sm font-medium">{user.blocked} blocked</div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Configuration Status */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Zap className="h-5 w-5" />
            <span>Integration Status</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-4 bg-green-50 rounded-lg">
              <div className="text-2xl font-bold text-green-600">{opikStatus?.enabled ? '✅' : '❌'}</div>
              <div className="text-sm text-green-700">Opik Enabled</div>
            </div>
            <div className="text-center p-4 bg-blue-50 rounded-lg">
              <div className="text-2xl font-bold text-blue-600">{opikStatus?.tracing?.enabled ? '✅' : '❌'}</div>
              <div className="text-sm text-blue-700">Tracing Active</div>
            </div>
            <div className="text-center p-4 bg-purple-50 rounded-lg">
              <div className="text-2xl font-bold text-purple-600">
                {opikStatus?.tracing?.sample_rate ? `${(opikStatus.tracing.sample_rate * 100).toFixed(0)}%` : '0%'}
              </div>
              <div className="text-sm text-purple-700">Sample Rate</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default OptimizerDashboard; 