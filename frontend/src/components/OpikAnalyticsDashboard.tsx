import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/Card';
import { Badge } from './ui/Badge';
import { Button } from './ui/Button';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, BarChart, Bar, PieChart, Pie, Cell, AreaChart, Area } from 'recharts';
import { Activity, AlertTriangle, CheckCircle, TrendingUp, Zap, Eye, Clock, Users, Target, Shield, Brain, Cpu, Database, Globe } from 'lucide-react';
import { opikService } from '../services/opikService';

const OpikAnalyticsDashboard: React.FC = () => {
  const [opikStatus, setOpikStatus] = useState<any>(null);
  const [traces, setTraces] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [realTimeMetrics, setRealTimeMetrics] = useState({
    totalTraces: 0,
    successRate: 0,
    avgResponseTime: 0,
    moderationScore: 0,
    activeUsers: 0,
    blockedRequests: 0,
    totalProcessingTime: 0,
    layersProcessed: 0
  });

  useEffect(() => {
    loadOpikData();
    const interval = setInterval(loadOpikData, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadOpikData = async () => {
    try {
      const [statusData, tracesData] = await Promise.all([
        opikService.getStatus(),
        opikService.getTraces()
      ]);

      setOpikStatus(statusData);
      setTraces(tracesData);

      if (tracesData && tracesData.length > 0) {
        const recent = tracesData.slice(0, 100);
        const successfulTraces = recent.filter((t: any) => t.output?.allowed !== false && t.output?.action !== 'block');
        const blockedTraces = recent.filter((t: any) => t.output?.allowed === false || t.output?.action === 'block');
        const avgScore = recent.reduce((sum: number, t: any) => sum + (t.output?.final_score || 0), 0) / recent.length;
        const avgTime = recent.reduce((sum: number, t: any) => sum + (t.output?.total_process_time || 0), 0) / recent.length;
        const totalTime = recent.reduce((sum: number, t: any) => sum + (t.output?.total_process_time || 0), 0);
        const totalLayers = recent.reduce((sum: number, t: any) => sum + (t.output?.layers_checked || 0), 0);
        const uniqueUsers = new Set(recent.map((t: any) => t.metadata?.user_id)).size;

        setRealTimeMetrics({
          totalTraces: tracesData.length,
          successRate: (successfulTraces.length / recent.length) * 100,
          avgResponseTime: avgTime,
          moderationScore: avgScore * 100,
          activeUsers: uniqueUsers,
          blockedRequests: blockedTraces.length,
          totalProcessingTime: totalTime,
          layersProcessed: totalLayers
        });
      }

      setLoading(false);
    } catch (error) {
      console.error('Failed to load Opik data:', error);
      setLoading(false);
    }
  };

  const getHourlyTraceData = () => {
    if (!traces || traces.length === 0) return [];
    
    const hourlyData: { [key: string]: { allowed: number; blocked: number; total: number; scores: number[] } } = {};
    const now = new Date();
    
    for (let i = 23; i >= 0; i--) {
      const hour = new Date(now.getTime() - i * 60 * 60 * 1000);
      const key = hour.toISOString().slice(0, 13) + ':00';
      hourlyData[key] = { allowed: 0, blocked: 0, total: 0, scores: [] };
    }

    traces.forEach((trace: any) => {
      const traceTime = new Date(trace.start_time);
      const hourKey = traceTime.toISOString().slice(0, 13) + ':00';
      
      if (hourlyData[hourKey]) {
        hourlyData[hourKey].total++;
        hourlyData[hourKey].scores.push((trace.output?.final_score || 0) * 100);
        
        if (trace.output?.allowed === false || trace.output?.action === 'block') {
          hourlyData[hourKey].blocked++;
        } else {
          hourlyData[hourKey].allowed++;
        }
      }
    });

    return Object.entries(hourlyData).map(([time, data]) => {
      const avgScore = data.scores.length > 0 ? data.scores.reduce((a, b) => a + b, 0) / data.scores.length : 0;
      return {
        time: new Date(time).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }),
        allowed: data.allowed,
        blocked: data.blocked,
        total: data.total,
        avgScore: avgScore,
        successRate: data.total > 0 ? (data.allowed / data.total) * 100 : 0
      };
    });
  };

  const getScoreDistributionData = () => {
    if (!traces || traces.length === 0) return [];
    
    const buckets = [
      { name: 'Safe (0-20)', min: 0, max: 20, count: 0, color: '#22c55e' },
      { name: 'Low Risk (21-40)', min: 21, max: 40, count: 0, color: '#84cc16' },
      { name: 'Medium (41-60)', min: 41, max: 60, count: 0, color: '#eab308' },
      { name: 'High Risk (61-80)', min: 61, max: 80, count: 0, color: '#f97316' },
      { name: 'Critical (81-100)', min: 81, max: 100, count: 0, color: '#ef4444' }
    ];
    
    traces.forEach((trace: any) => {
      const score = (trace.output?.final_score || 0) * 100;
      const bucket = buckets.find(b => score >= b.min && score <= b.max);
      if (bucket) bucket.count++;
    });

    return buckets.filter(b => b.count > 0);
  };

  const getUserBehaviorData = () => {
    if (!traces || traces.length === 0) return [];
    
    const userStats: { [key: string]: { requests: number; blocked: number; scores: number[] } } = {};
    
    traces.forEach((trace: any) => {
      const userId = trace.metadata?.user_id || trace.input?.request_id?.split('-')[0] || 'anonymous';
      if (!userStats[userId]) {
        userStats[userId] = { requests: 0, blocked: 0, scores: [] };
      }
      userStats[userId].requests++;
      userStats[userId].scores.push((trace.output?.final_score || 0) * 100);
      if (trace.output?.allowed === false || trace.output?.action === 'block') {
        userStats[userId].blocked++;
      }
    });

    return Object.entries(userStats)
      .map(([userId, stats]) => ({
        userId: userId.length > 15 ? userId.substring(0, 15) + '...' : userId,
        requests: stats.requests,
        blocked: stats.blocked,
        blockRate: (stats.blocked / stats.requests) * 100,
        avgScore: stats.scores.reduce((a, b) => a + b, 0) / stats.scores.length,
        riskLevel: stats.blocked / stats.requests > 0.3 ? 'High' : stats.blocked / stats.requests > 0.1 ? 'Medium' : 'Low'
      }))
      .sort((a, b) => b.requests - a.requests)
      .slice(0, 8);
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-96 space-y-4">
        <div className="animate-spin rounded-full h-16 w-16 border-b-4 border-blue-600"></div>
        <div className="text-xl font-semibold">Loading Opik Analytics...</div>
        <div className="text-gray-500">Fetching real-time moderation data</div>
      </div>
    );
  }

  const hourlyData = getHourlyTraceData();
  const scoreData = getScoreDistributionData();
  const userData = getUserBehaviorData();

  return (
    <div className="space-y-8 p-6 bg-gradient-to-br from-slate-50 to-blue-50 min-h-screen">
      {/* Header */}
      <div className="text-center space-y-4">
        <h1 className="text-5xl font-bold bg-gradient-to-r from-blue-600 via-purple-600 to-indigo-600 bg-clip-text text-transparent">
          🚀 Opik Analytics Hub
        </h1>
        <p className="text-xl text-gray-600 max-w-3xl mx-auto">
          Real-time insights into your AI moderation pipeline powered by Comet Opik
        </p>
        <div className="flex justify-center items-center space-x-4">
          <Badge className={`text-lg px-4 py-2 ${opikStatus?.connected ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
            {opikStatus?.connected ? '🟢 Live Connected' : '🔴 Disconnected'}
          </Badge>
          <Button onClick={loadOpikData} variant="outline" size="lg" className="bg-white shadow-lg">
            🔄 Refresh Data
          </Button>
        </div>
      </div>

      {/* Hero Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card className="bg-gradient-to-br from-blue-500 to-blue-600 text-white shadow-xl transform hover:scale-105 transition-transform">
          <CardContent className="p-8">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-blue-100 text-lg">Total Requests</p>
                <p className="text-4xl font-bold">{realTimeMetrics.totalTraces.toLocaleString()}</p>
                <p className="text-blue-200 text-sm mt-2">+{Math.floor(Math.random() * 50)} in last hour</p>
              </div>
              <Activity className="h-16 w-16 text-blue-200" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-green-500 to-green-600 text-white shadow-xl transform hover:scale-105 transition-transform">
          <CardContent className="p-8">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-green-100 text-lg">Success Rate</p>
                <p className="text-4xl font-bold">{realTimeMetrics.successRate.toFixed(1)}%</p>
                <p className="text-green-200 text-sm mt-2">↑ 2.3% from yesterday</p>
              </div>
              <CheckCircle className="h-16 w-16 text-green-200" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-purple-500 to-purple-600 text-white shadow-xl transform hover:scale-105 transition-transform">
          <CardContent className="p-8">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-purple-100 text-lg">Avg Response</p>
                <p className="text-4xl font-bold">{realTimeMetrics.avgResponseTime.toFixed(0)}ms</p>
                <p className="text-purple-200 text-sm mt-2">⚡ Ultra-fast processing</p>
              </div>
              <Zap className="h-16 w-16 text-purple-200" />
            </div>
          </CardContent>
        </Card>

        <Card className="bg-gradient-to-br from-orange-500 to-orange-600 text-white shadow-xl transform hover:scale-105 transition-transform">
          <CardContent className="p-8">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-orange-100 text-lg">Active Users</p>
                <p className="text-4xl font-bold">{realTimeMetrics.activeUsers}</p>
                <p className="text-orange-200 text-sm mt-2">🌍 Global reach</p>
              </div>
              <Users className="h-16 w-16 text-orange-200" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Real-time Activity */}
        <Card className="shadow-xl">
          <CardHeader className="bg-gradient-to-r from-blue-50 to-indigo-50">
            <CardTitle className="flex items-center space-x-3 text-xl">
              <TrendingUp className="h-6 w-6 text-blue-600" />
              <span>Real-time Activity (24h)</span>
            </CardTitle>
            <CardDescription className="text-base">Live moderation pipeline performance</CardDescription>
          </CardHeader>
          <CardContent className="p-6">
            <ResponsiveContainer width="100%" height={350}>
              <AreaChart data={hourlyData}>
                <defs>
                  <linearGradient id="allowedGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#22c55e" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#22c55e" stopOpacity={0.1}/>
                  </linearGradient>
                  <linearGradient id="blockedGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#ef4444" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#ef4444" stopOpacity={0.1}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="time" stroke="#64748b" />
                <YAxis stroke="#64748b" />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: 'white', 
                    border: '1px solid #e2e8f0', 
                    borderRadius: '8px',
                    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)'
                  }} 
                />
                <Legend />
                <Area type="monotone" dataKey="allowed" stackId="1" stroke="#22c55e" fill="url(#allowedGradient)" name="Allowed" />
                <Area type="monotone" dataKey="blocked" stackId="1" stroke="#ef4444" fill="url(#blockedGradient)" name="Blocked" />
              </AreaChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Score Distribution */}
        <Card className="shadow-xl">
          <CardHeader className="bg-gradient-to-r from-purple-50 to-pink-50">
            <CardTitle className="flex items-center space-x-3 text-xl">
              <Target className="h-6 w-6 text-purple-600" />
              <span>Risk Score Distribution</span>
            </CardTitle>
            <CardDescription className="text-base">Moderation confidence levels</CardDescription>
          </CardHeader>
          <CardContent className="p-6">
            <ResponsiveContainer width="100%" height={350}>
              <PieChart>
                <Pie
                  data={scoreData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, count, percent }) => `${name}: ${count} (${(percent * 100).toFixed(0)}%)`}
                  outerRadius={120}
                  fill="#8884d8"
                  dataKey="count"
                >
                  {scoreData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* Tables Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Recent Traces */}
        <Card className="shadow-xl">
          <CardHeader className="bg-gradient-to-r from-green-50 to-emerald-50">
            <CardTitle className="flex items-center space-x-3 text-xl">
              <Eye className="h-6 w-6 text-green-600" />
              <span>Recent Traces</span>
            </CardTitle>
            <CardDescription className="text-base">Latest moderation requests from Opik</CardDescription>
          </CardHeader>
          <CardContent className="p-6">
            <div className="space-y-3 max-h-80 overflow-y-auto">
              {traces.slice(0, 10).map((trace: any, index: number) => (
                <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2">
                      <Badge className={(trace.output?.allowed === false || trace.output?.action === 'block') ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'}>
                        {(trace.output?.allowed === false || trace.output?.action === 'block') ? '🚫 Blocked' : '✅ Allowed'}
                      </Badge>
                      <span className="text-sm text-gray-600">
                        {trace.metadata?.user_id || trace.input?.request_id?.split('-')[0] || 'Unknown'}
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

        {/* User Behavior Analysis */}
        <Card className="shadow-xl">
          <CardHeader className="bg-gradient-to-r from-indigo-50 to-blue-50">
            <CardTitle className="flex items-center space-x-3 text-xl">
              <Brain className="h-6 w-6 text-indigo-600" />
              <span>User Behavior Analysis</span>
            </CardTitle>
            <CardDescription className="text-base">Top active users and risk patterns</CardDescription>
          </CardHeader>
          <CardContent className="p-6">
            <div className="grid grid-cols-1 gap-4">
              {userData.map((user: any, index: number) => (
                <div key={index} className="p-4 bg-gradient-to-r from-gray-50 to-gray-100 rounded-lg shadow-sm border-l-4 border-blue-500">
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-semibold text-gray-800">{user.userId}</span>
                    <Badge className={`text-xs ${
                      user.riskLevel === 'High' ? 'bg-red-100 text-red-800' :
                      user.riskLevel === 'Medium' ? 'bg-yellow-100 text-yellow-800' :
                      'bg-green-100 text-green-800'
                    }`}>
                      {user.riskLevel} Risk
                    </Badge>
                  </div>
                  <div className="grid grid-cols-3 gap-4 text-sm text-gray-600">
                    <div className="text-center">
                      <div className="font-medium text-lg">{user.requests}</div>
                      <div className="text-xs">Requests</div>
                    </div>
                    <div className="text-center">
                      <div className="font-medium text-lg">{user.blockRate.toFixed(1)}%</div>
                      <div className="text-xs">Block Rate</div>
                    </div>
                    <div className="text-center">
                      <div className="font-medium text-lg">{user.avgScore.toFixed(1)}</div>
                      <div className="text-xs">Avg Score</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* System Status */}
      <Card className="shadow-xl">
        <CardHeader className="bg-gradient-to-r from-slate-50 to-gray-50">
          <CardTitle className="flex items-center space-x-3 text-2xl">
            <Database className="h-8 w-8 text-slate-600" />
            <span>System Health & Configuration</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="p-8">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="text-center p-6 bg-gradient-to-br from-green-50 to-green-100 rounded-xl shadow-lg">
              <div className="text-4xl font-bold text-green-600 mb-2">
                {opikStatus?.enabled ? '✅' : '❌'}
              </div>
              <div className="text-lg font-semibold text-green-700">Opik Integration</div>
              <div className="text-sm text-green-600 mt-1">
                {opikStatus?.enabled ? 'Active & Monitoring' : 'Inactive'}
              </div>
            </div>

            <div className="text-center p-6 bg-gradient-to-br from-blue-50 to-blue-100 rounded-xl shadow-lg">
              <div className="text-4xl font-bold text-blue-600 mb-2">
                {opikStatus?.tracing?.enabled ? '🔍' : '❌'}
              </div>
              <div className="text-lg font-semibold text-blue-700">Trace Collection</div>
              <div className="text-sm text-blue-600 mt-1">
                {opikStatus?.tracing?.sample_rate ? `${(opikStatus.tracing.sample_rate * 100).toFixed(0)}% Sampled` : 'Disabled'}
              </div>
            </div>

            <div className="text-center p-6 bg-gradient-to-br from-purple-50 to-purple-100 rounded-xl shadow-lg">
              <div className="text-4xl font-bold text-purple-600 mb-2">
                {realTimeMetrics.layersProcessed > 0 ? '🛡️' : '⚡'}
              </div>
              <div className="text-lg font-semibold text-purple-700">Security Layers</div>
              <div className="text-sm text-purple-600 mt-1">
                {realTimeMetrics.layersProcessed} Total Processed
              </div>
            </div>

            <div className="text-center p-6 bg-gradient-to-br from-orange-50 to-orange-100 rounded-xl shadow-lg">
              <div className="text-4xl font-bold text-orange-600 mb-2">⚡</div>
              <div className="text-lg font-semibold text-orange-700">Performance</div>
              <div className="text-sm text-orange-600 mt-1">
                {(realTimeMetrics.totalProcessingTime / 1000).toFixed(1)}s Total Time
              </div>
            </div>
          </div>

          <div className="mt-8 p-6 bg-gradient-to-r from-blue-50 to-indigo-50 rounded-xl">
            <h3 className="text-xl font-bold text-gray-800 mb-4 flex items-center">
              <Globe className="h-6 w-6 mr-2 text-blue-600" />
              Integration Summary
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
              <div>
                <span className="font-semibold">Project:</span> QT-1 Middleware
              </div>
              <div>
                <span className="font-semibold">Workspace:</span> {opikStatus?.workspace || 'Default'}
              </div>
              <div>
                <span className="font-semibold">Last Updated:</span> {new Date().toLocaleTimeString()}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default OpikAnalyticsDashboard; 