import { useEffect } from 'react';
import Layout from './components/Layout/Layout';
import ChatInterface from './components/ChatInterface/ChatInterface';
import MCPStatusPanel from './components/MCPStatusPanel/MCPStatusPanel';
import { SessionManager } from './components/SessionManager/SessionManager';
import { useAppStore } from './stores/appStore';
import './App.css';

export default function App() {
  const createSession = useAppStore((state) => state.createSession);
  const currentSessionId = useAppStore((state) => state.currentSessionId);
  const mcpServers = useAppStore((state) => state.mcpServers);
  const connect = useAppStore((state) => state.connect);
  const wsConnected = useAppStore((state) => state.wsConnected);

  useEffect(() => {
    if (!currentSessionId) {
      createSession();
    }
  }, [currentSessionId, createSession]);

  useEffect(() => {
    if (!wsConnected) {
      connect();
    }
  }, [connect, wsConnected]);

  return (
    <Layout
      leftPanel={<SessionManager />}
      rightPanel={<MCPStatusPanel servers={mcpServers} />}
    >
      <ChatInterface />
    </Layout>
  );
}
