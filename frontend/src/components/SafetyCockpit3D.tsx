import React, { useState, useEffect, useMemo } from 'react';
import { SafetyEvent, EventType, EventFilter, getCockpitService, ConnectionState, AggregatedData } from '../services/cockpitService';
import { ThreeDVisualization, VisualizationConfig, Node3D } from './ThreeDVisualization';

// Control panel props
interface ControlPanelProps {
  config: VisualizationConfig;
  onConfigChange: (config: Partial<VisualizationConfig>) => void;
  filter: EventFilter;
  onFilterChange: (filter: EventFilter) => void;
  connectionState: ConnectionState;
  eventCount: number;
}

const ControlPanel: React.FC<ControlPanelProps> = ({
  config,
  onConfigChange,
  filter,
  onFilterChange,
  connectionState,
  eventCount
}) => {
  return (
    <div className="bg-gray-800 p-4 rounded-lg space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-white font-semibold">Safety Cockpit 3D</h3>
        <div className="flex items-center space-x-2">
          <div className={`w-2 h-2 rounded-full ${
            connectionState === ConnectionState.CONNECTED ? 'bg-green-500' :
            connectionState === ConnectionState.CONNECTING ? 'bg-yellow-500' : 'bg-red-500'
          }`} />
          <span className="text-xs text-gray-400">{connectionState}</span>
        </div>
      </div>

      {/* Event Statistics */}
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-gray-700 p-3 rounded">
          <div className="text-xs text-gray-400">Total Events</div>
          <div className="text-lg font-semibold text-white">{eventCount}</div>
        </div>
        <div className="bg-gray-700 p-3 rounded">
          <div className="text-xs text-gray-400">Connection</div>
          <div className="text-lg font-semibold text-white capitalize">{connectionState}</div>
        </div>
      </div>

      {/* Visualization Controls */}
      <div className="space-y-3">
        <div className="text-sm font-medium text-white">Visualization</div>
        
        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={config.showConnections}
            onChange={(e) => onConfigChange({ showConnections: e.target.checked })}
            className="rounded"
          />
          <span className="text-sm text-gray-300">Show Connections</span>
        </label>

        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={config.showThreatHeatmap}
            onChange={(e) => onConfigChange({ showThreatHeatmap: e.target.checked })}
            className="rounded"
          />
          <span className="text-sm text-gray-300">Threat Heatmap</span>
        </label>

        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={config.enableAnimations}
            onChange={(e) => onConfigChange({ enableAnimations: e.target.checked })}
            className="rounded"
          />
          <span className="text-sm text-gray-300">Animations</span>
        </label>

        <div className="space-y-2">
          <label className="text-sm text-gray-300">Node Scale</label>
          <input
            type="range"
            min="0.5"
            max="2.0"
            step="0.1"
            value={config.nodeScale}
            onChange={(e) => onConfigChange({ nodeScale: parseFloat(e.target.value) })}
            className="w-full"
          />
          <div className="text-xs text-gray-400">{config.nodeScale.toFixed(1)}x</div>
        </div>

        <div className="space-y-2">
          <label className="text-sm text-gray-300">Connection Opacity</label>
          <input
            type="range"
            min="0.1"
            max="1.0"
            step="0.1"
            value={config.connectionOpacity}
            onChange={(e) => onConfigChange({ connectionOpacity: parseFloat(e.target.value) })}
            className="w-full"
          />
          <div className="text-xs text-gray-400">{(config.connectionOpacity * 100).toFixed(0)}%</div>
        </div>
      </div>

      {/* Event Filters */}
      <div className="space-y-3">
        <div className="text-sm font-medium text-white">Event Filters</div>
        
        <div className="space-y-2">
          <label className="text-xs text-gray-400">Event Types</label>
          <div className="grid grid-cols-2 gap-1">
            {Object.values(EventType).map(type => (
              <label key={type} className="flex items-center space-x-1">
                <input
                  type="checkbox"
                  checked={!filter.types || filter.types.includes(type)}
                  onChange={(e) => {
                    const currentTypes = filter.types || Object.values(EventType);
                    const newTypes = e.target.checked
                      ? [...(filter.types || []), type].filter((t, i, arr) => arr.indexOf(t) === i)
                      : currentTypes.filter(t => t !== type);
                    onFilterChange({ ...filter, types: newTypes.length === Object.values(EventType).length ? undefined : newTypes });
                  }}
                  className="rounded text-xs"
                />
                <span className="text-xs text-gray-300 capitalize">{type}</span>
              </label>
            ))}
          </div>
        </div>

        <div className="space-y-2">
          <label className="text-xs text-gray-400">Severity Levels</label>
          <div className="space-y-1">
            {['high', 'medium', 'low'].map(severity => (
              <label key={severity} className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={!filter.severities || filter.severities.includes(severity)}
                  onChange={(e) => {
                    const currentSeverities = filter.severities || ['high', 'medium', 'low'];
                    const newSeverities = e.target.checked
                      ? [...(filter.severities || []), severity].filter((s, i, arr) => arr.indexOf(s) === i)
                      : currentSeverities.filter(s => s !== severity);
                    onFilterChange({ ...filter, severities: newSeverities.length === 3 ? undefined : newSeverities });
                  }}
                  className="rounded"
                />
                <span className="text-sm text-gray-300 capitalize">{severity}</span>
              </label>
            ))}
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="space-y-2">
        <button
          onClick={() => getCockpitService().requestAggregatedData()}
          className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded text-sm"
        >
          Refresh Data
        </button>
        
        <button
          onClick={() => getCockpitService().requestMetrics()}
          className="w-full bg-green-600 hover:bg-green-700 text-white py-2 px-4 rounded text-sm"
        >
          Get Metrics
        </button>
      </div>
    </div>
  );
};

// Event details panel
interface EventDetailsPanelProps {
  event: SafetyEvent | null;
  onClose: () => void;
}

const EventDetailsPanel: React.FC<EventDetailsPanelProps> = ({ event, onClose }) => {
  if (!event) return null;

  return (
    <div className="bg-gray-800 p-4 rounded-lg">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-white font-semibold">Event Details</h3>
        <button
          onClick={onClose}
          className="text-gray-400 hover:text-white"
        >
          ✕
        </button>
      </div>

      <div className="space-y-2 text-sm">
        <div className="grid grid-cols-2 gap-2">
          <span className="text-gray-400">ID:</span>
          <span className="text-white font-mono text-xs">{event.id}</span>
        </div>
        
        <div className="grid grid-cols-2 gap-2">
          <span className="text-gray-400">Type:</span>
          <span className={`text-white capitalize px-2 py-1 rounded text-xs ${getEventTypeColor(event.type)}`}>
            {event.type}
          </span>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <span className="text-gray-400">Timestamp:</span>
          <span className="text-white text-xs">{event.timestamp.toLocaleString()}</span>
        </div>

        {event.severity && (
          <div className="grid grid-cols-2 gap-2">
            <span className="text-gray-400">Severity:</span>
            <span className={`text-white capitalize px-2 py-1 rounded text-xs ${getSeverityColor(event.severity)}`}>
              {event.severity}
            </span>
          </div>
        )}

        {event.trace_id && (
          <div className="grid grid-cols-2 gap-2">
            <span className="text-gray-400">Trace ID:</span>
            <span className="text-white font-mono text-xs">{event.trace_id}</span>
          </div>
        )}

        {event.span_id && (
          <div className="grid grid-cols-2 gap-2">
            <span className="text-gray-400">Span ID:</span>
            <span className="text-white font-mono text-xs">{event.span_id}</span>
          </div>
        )}

        {event.tags && event.tags.length > 0 && (
          <div className="space-y-1">
            <span className="text-gray-400">Tags:</span>
            <div className="flex flex-wrap gap-1">
              {event.tags.map((tag, index) => (
                <span key={index} className="bg-gray-700 text-white px-2 py-1 rounded text-xs">
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}

        {event.data && Object.keys(event.data).length > 0 && (
          <div className="space-y-1">
            <span className="text-gray-400">Data:</span>
            <pre className="bg-gray-900 text-white p-2 rounded text-xs overflow-auto max-h-32">
              {JSON.stringify(event.data, null, 2)}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
};

// Statistics panel
interface StatisticsPanelProps {
  events: SafetyEvent[];
  aggregatedData: AggregatedData | null;
}

const StatisticsPanel: React.FC<StatisticsPanelProps> = ({ events, aggregatedData }) => {
  const stats = useMemo(() => {
    const typeCount = events.reduce((acc, event) => {
      acc[event.type] = (acc[event.type] || 0) + 1;
      return acc;
    }, {} as Record<EventType, number>);

    const severityCount = events.reduce((acc, event) => {
      if (event.severity) {
        acc[event.severity] = (acc[event.severity] || 0) + 1;
      }
      return acc;
    }, {} as Record<string, number>);

    const recentThreats = events
      .filter(e => e.type === EventType.THREAT)
      .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
      .slice(0, 5);

    return { typeCount, severityCount, recentThreats };
  }, [events]);

  return (
    <div className="bg-gray-800 p-4 rounded-lg space-y-4">
      <h3 className="text-white font-semibold">Statistics</h3>

      {/* Event Type Distribution */}
      <div className="space-y-2">
        <div className="text-sm text-gray-400">Event Types</div>
        {Object.entries(stats.typeCount).map(([type, count]) => (
          <div key={type} className="flex justify-between items-center">
            <span className={`text-xs px-2 py-1 rounded capitalize ${getEventTypeColor(type as EventType)}`}>
              {type}
            </span>
            <span className="text-white text-sm">{count}</span>
          </div>
        ))}
      </div>

      {/* Severity Distribution */}
      {Object.keys(stats.severityCount).length > 0 && (
        <div className="space-y-2">
          <div className="text-sm text-gray-400">Severity Levels</div>
          {Object.entries(stats.severityCount).map(([severity, count]) => (
            <div key={severity} className="flex justify-between items-center">
              <span className={`text-xs px-2 py-1 rounded capitalize ${getSeverityColor(severity)}`}>
                {severity}
              </span>
              <span className="text-white text-sm">{count}</span>
            </div>
          ))}
        </div>
      )}

      {/* Recent Threats */}
      {stats.recentThreats.length > 0 && (
        <div className="space-y-2">
          <div className="text-sm text-gray-400">Recent Threats</div>
          <div className="space-y-1 max-h-32 overflow-y-auto">
            {stats.recentThreats.map(threat => (
              <div key={threat.id} className="bg-gray-700 p-2 rounded text-xs">
                <div className="text-white">{threat.data?.threat_type || 'Unknown Threat'}</div>
                <div className="text-gray-400">{threat.timestamp.toLocaleTimeString()}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Aggregated Data */}
      {aggregatedData && (
        <div className="space-y-2">
          <div className="text-sm text-gray-400">Aggregated Data</div>
          <div className="text-xs text-gray-300">
            Window: {aggregatedData.window}
          </div>
          <div className="text-xs text-gray-300">
            Updated: {aggregatedData.timestamp.toLocaleTimeString()}
          </div>
        </div>
      )}
    </div>
  );
};

// Main Safety Cockpit 3D Component
export const SafetyCockpit3D: React.FC = () => {
  const [events, setEvents] = useState<SafetyEvent[]>([]);
  const [selectedEvent, setSelectedEvent] = useState<SafetyEvent | null>(null);
  const [connectionState, setConnectionState] = useState<ConnectionState>(ConnectionState.DISCONNECTED);
  const [aggregatedData, setAggregatedData] = useState<AggregatedData | null>(null);
  
  const [visualConfig, setVisualConfig] = useState<VisualizationConfig>({
    showConnections: true,
    showThreatHeatmap: true,
    enableAnimations: true,
    nodeScale: 1.0,
    connectionOpacity: 0.3,
    threatSensitivity: 0.8
  });

  const [filter, setFilter] = useState<EventFilter>({});

  // Initialize WebSocket connection
  useEffect(() => {
    const cockpitService = getCockpitService();

    // Event listeners
    const onStateChange = ({ currentState }: { currentState: ConnectionState }) => {
      setConnectionState(currentState);
    };

    const onEvent = (event: SafetyEvent) => {
      setEvents(prev => [...prev.slice(-999), event]); // Keep last 1000 events
    };

    const onHistorical = (historicalEvents: SafetyEvent[]) => {
      setEvents(historicalEvents);
    };

    const onAggregated = (data: any) => {
      setAggregatedData(data.data);
    };

    const onError = (error: Error) => {
      console.error('Cockpit service error:', error);
    };

    // Subscribe to events
    cockpitService.on('stateChange', onStateChange);
    cockpitService.on('event', onEvent);
    cockpitService.on('historical', onHistorical);
    cockpitService.on('aggregated', onAggregated);
    cockpitService.on('error', onError);

    // Connect
    cockpitService.connect();

    // Cleanup
    return () => {
      cockpitService.off('stateChange', onStateChange);
      cockpitService.off('event', onEvent);
      cockpitService.off('historical', onHistorical);
      cockpitService.off('aggregated', onAggregated);
      cockpitService.off('error', onError);
    };
  }, []);

  // Apply filter when it changes
  useEffect(() => {
    getCockpitService().setFilter(filter);
  }, [filter]);

  // Handle node selection
  const handleNodeClick = (node: Node3D) => {
    setSelectedEvent(node.data);
  };

  // Update visualization config
  const handleConfigChange = (newConfig: Partial<VisualizationConfig>) => {
    setVisualConfig(prev => ({ ...prev, ...newConfig }));
  };

  return (
    <div className="flex h-screen bg-gray-900">
      {/* Left sidebar - Controls */}
      <div className="w-80 p-4 space-y-4 overflow-y-auto">
        <ControlPanel
          config={visualConfig}
          onConfigChange={handleConfigChange}
          filter={filter}
          onFilterChange={setFilter}
          connectionState={connectionState}
          eventCount={events.length}
        />

        <StatisticsPanel events={events} aggregatedData={aggregatedData} />
      </div>

      {/* Main visualization area */}
      <div className="flex-1 p-4">
        <div className="h-full">
          <ThreeDVisualization
            width={800}
            height={600}
            events={events}
            config={visualConfig}
            onNodeClick={handleNodeClick}
          />
        </div>
      </div>

      {/* Right sidebar - Event details */}
      {selectedEvent && (
        <div className="w-80 p-4">
          <EventDetailsPanel
            event={selectedEvent}
            onClose={() => setSelectedEvent(null)}
          />
        </div>
      )}
    </div>
  );
};

// Helper functions
function getEventTypeColor(type: EventType): string {
  switch (type) {
    case EventType.THREAT:
    case EventType.ALERT:
      return 'bg-red-600';
    case EventType.MODERATION:
      return 'bg-orange-600';
    case EventType.PROVIDER:
      return 'bg-blue-600';
    case EventType.METRIC:
      return 'bg-green-600';
    default:
      return 'bg-gray-600';
  }
}

function getSeverityColor(severity: string): string {
  switch (severity.toLowerCase()) {
    case 'high':
      return 'bg-red-600';
    case 'medium':
      return 'bg-yellow-600';
    case 'low':
      return 'bg-green-600';
    default:
      return 'bg-gray-600';
  }
}

export default SafetyCockpit3D;