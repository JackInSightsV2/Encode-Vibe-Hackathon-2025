import React, { createContext, useContext, useEffect, useState, ReactNode, useRef } from 'react';
import { ConnectionStatus, WSMessage, webSocketService } from '../services/websocket';

interface WebSocketContextType {
  connectionStatus: ConnectionStatus;
  isConnected: boolean;
  sendMessage: (message: Omit<WSMessage, 'timestamp'>) => void;
  sendClientMessage: (data: Record<string, any>, id?: string) => void;
  lastMessage: WSMessage | null;
  connect: () => Promise<void>;
  disconnect: () => void;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

interface WebSocketProviderProps {
  children: ReactNode;
  autoConnect?: boolean;
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ 
  children, 
  autoConnect = true 
}) => {
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(
    webSocketService.getStatus()
  );
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);

  useEffect(() => {
    // Set up WebSocket event handlers
    webSocketService.setHandlers({
      onStatusChange: (status) => {
        setConnectionStatus(status);
      },
      onMessage: (message) => {
        setLastMessage(message);
      },
      onError: (error) => {
        console.error('WebSocket error in context:', error);
      },
    });

    // Auto-connect if enabled
    if (autoConnect) {
      webSocketService.connect().catch((error) => {
        console.error('Failed to auto-connect WebSocket:', error);
      });
    }

    // Cleanup on unmount
    return () => {
      webSocketService.disconnect();
    };
  }, [autoConnect]);

  const isConnected = connectionStatus === ConnectionStatus.CONNECTED;

  const contextValue: WebSocketContextType = {
    connectionStatus,
    isConnected,
    sendMessage: webSocketService.sendMessage.bind(webSocketService),
    sendClientMessage: webSocketService.sendClientMessage.bind(webSocketService),
    lastMessage,
    connect: webSocketService.connect.bind(webSocketService),
    disconnect: webSocketService.disconnect.bind(webSocketService),
  };

  return (
    <WebSocketContext.Provider value={contextValue}>
      {children}
    </WebSocketContext.Provider>
  );
};

// Custom hook to use WebSocket context
export const useWebSocket = (): WebSocketContextType => {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
};

// Hook for specific message types
export const useWebSocketMessage = (
  messageType: string,
  handler: (data: any, message?: WSMessage) => void
) => {
  const { lastMessage } = useWebSocket();
  const handlerRef = useRef(handler);
  const lastProcessedRef = useRef<string | null>(null);
  
  // Keep handler ref up to date
  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);

  useEffect(() => {
    if (lastMessage && lastMessage.type === messageType) {
      // Create a unique identifier for this message to prevent duplicate processing
      const messageId = `${lastMessage.timestamp}-${lastMessage.type}-${JSON.stringify(lastMessage.data)}`;
      
      if (lastProcessedRef.current !== messageId) {
        console.log(`🔄 Processing new ${messageType} message:`, messageId);
        lastProcessedRef.current = messageId;
        handlerRef.current(lastMessage.data, lastMessage);
      } else {
        console.log(`⏭️ Skipping duplicate ${messageType} message:`, messageId);
      }
    }
  }, [lastMessage, messageType]);
};