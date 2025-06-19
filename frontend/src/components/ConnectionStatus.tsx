import React from 'react';
import { useWebSocket } from '../contexts/WebSocketContext';
import { ConnectionStatus as Status } from '../services/websocket';

const ConnectionStatus: React.FC = () => {
  const { connectionStatus, connect } = useWebSocket();

  const getStatusInfo = () => {
    switch (connectionStatus) {
      case Status.CONNECTED:
        return {
          color: 'text-green-600',
          bgColor: 'bg-green-100',
          icon: '●',
          text: 'Connected',
          description: 'Real-time updates active'
        };
      case Status.CONNECTING:
        return {
          color: 'text-yellow-600',
          bgColor: 'bg-yellow-100',
          icon: '◐',
          text: 'Connecting',
          description: 'Establishing connection...'
        };
      case Status.RECONNECTING:
        return {
          color: 'text-orange-600',
          bgColor: 'bg-orange-100',
          icon: '◑',
          text: 'Reconnecting',
          description: 'Attempting to reconnect...'
        };
      case Status.ERROR:
        return {
          color: 'text-red-600',
          bgColor: 'bg-red-100',
          icon: '●',
          text: 'Error',
          description: 'Connection failed'
        };
      case Status.DISCONNECTED:
      default:
        return {
          color: 'text-gray-600',
          bgColor: 'bg-gray-100',
          icon: '○',
          text: 'Disconnected',
          description: 'No real-time updates'
        };
    }
  };

  const statusInfo = getStatusInfo();
  const canReconnect = connectionStatus === Status.DISCONNECTED || connectionStatus === Status.ERROR;

  return (
    <div className={`flex items-center space-x-2 px-3 py-2 rounded-lg ${statusInfo.bgColor}`}>
      <span className={`text-lg ${statusInfo.color}`} title={statusInfo.description}>
        {statusInfo.icon}
      </span>
      <div className="flex flex-col">
        <span className={`text-sm font-medium ${statusInfo.color}`}>
          {statusInfo.text}
        </span>
        <span className="text-xs text-gray-500">
          {statusInfo.description}
        </span>
      </div>
      {canReconnect && (
        <button
          onClick={() => connect().catch(console.error)}
          className="ml-2 px-2 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1"
          title="Reconnect"
        >
          Reconnect
        </button>
      )}
    </div>
  );
};

export default ConnectionStatus;