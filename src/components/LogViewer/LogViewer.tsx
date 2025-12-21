/**
 * LogViewer Component
 * Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
 * 
 * Main log viewer component that displays agent execution logs
 * with real-time updates and agent reference navigation.
 */

import React from 'react';
import { LogList } from './LogList';
import { useAppStore } from '../../stores/appStore';
import './LogViewer.css';

interface LogViewerProps {
  onAgentClick?: (agentId: string) => void;
}

/**
 * LogViewer Component
 * Requirement 3.1: Real-time agent execution log viewing
 * Requirement 3.2: Agent reference rendering
 * Requirement 3.3: Agent reference click handling
 * Requirement 3.4: Virtual scrolling for large log lists
 * Requirement 3.5: Automatic scroll control
 */
export const LogViewer: React.FC<LogViewerProps> = ({ onAgentClick }) => {
  const logs = useAppStore((state) => state.logs);
  const clearLogs = useAppStore((state) => state.clearLogs);

  const handleAgentClick = (agentId: string) => {
    if (onAgentClick) {
      onAgentClick(agentId);
    }
  };

  return (
    <div className="log-viewer">
      <div className="log-viewer-header">
        <h3 className="log-viewer-title">Agent Logs</h3>
        <button
          className="log-viewer-clear-button"
          onClick={clearLogs}
          disabled={logs.length === 0}
          title="Clear all logs"
        >
          Clear
        </button>
      </div>
      <div className="log-viewer-content">
        <LogList logs={logs} onAgentClick={handleAgentClick} autoScroll={true} />
      </div>
    </div>
  );
};
