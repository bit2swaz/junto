/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import React, { createContext, useContext, useEffect, useRef, useState } from 'react';

type MessageHandler = (msg: any) => void;

interface WebSocketContextType {
  sendMessage: (msg: any) => void;
  isConnected: boolean;
  subscribe: (handler: MessageHandler) => () => void;
}

const WebSocketContext = createContext<WebSocketContextType | null>(null);

export const useWebSocket = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
};

export const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const subscribersRef = useRef<Set<MessageHandler>>(new Set());

  useEffect(() => {
    // In a real app, you'd get the token from auth context or local storage
    // For this demo, we'll assume it's in localStorage or just hardcode for testing if needed
    // But let's try to read from localStorage "token"
    const token = localStorage.getItem('token');
    if (!token) {
      console.log("No token found, skipping WS connection");
      return;
    }

    const connect = () => {
      const wsUrl = `ws://localhost:8080/ws?token=${token}`;
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket Connected');
        setIsConnected(true);
      };

      ws.onclose = () => {
        console.log('WebSocket Disconnected');
        setIsConnected(false);
        // Reconnect logic
        reconnectTimeoutRef.current = setTimeout(connect, 3000);
      };

      ws.onerror = (err) => {
        console.error('WebSocket Error:', err);
        ws.close();
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          // Notify subscribers
          subscribersRef.current.forEach(handler => handler(data));
        } catch (e) {
          console.error('Failed to parse WS message', e);
        }
      };

      wsRef.current = ws;
    };

    connect();

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, []);

  const sendMessage = (msg: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg));
    }
  };

  const subscribe = (handler: MessageHandler) => {
    subscribersRef.current.add(handler);
    return () => {
      subscribersRef.current.delete(handler);
    };
  };

  return (
    <WebSocketContext.Provider value={{ sendMessage, isConnected, subscribe }}>
      {children}
    </WebSocketContext.Provider>
  );
};
