/**
 * MCP Status Panel Component
 * Requirements: 2.1, 2.2, 2.3, 2.4, 2.5
 * 
 * Displays MCP Server list with status indicators and expandable tool lists.
 */

import React, { useState } from 'react';
import type { MCPServer } from '../../types/models';
import './MCPStatusPanel.css';

export interface MCPStatusPanelProps {
  servers: MCPServer[];
  onServerClick?: (serverId: string) => void;
}

/**
 * MCPStatusPanel Component
 * 
 * Requirement 2.1: Display all configured MCP Servers
 * Requirement 2.2: Update status indicators when server status changes
 * Requirement 2.3: Expand/collapse tool list on server click
 * Requirement 2.4: Display offline servers with red indicator
 * Requirement 2.5: Display online servers with green indicator
 */
const MCPStatusPanel: React.FC<MCPStatusPanelProps> = ({ servers, onServerClick }) => {
  const [expandedServerId, setExpandedServerId] = useState<string | null>(null);

  /**
   * Handle server click to expand/collapse tool list
   * Requirement 2.3: Expand tool list on click
   */
  const handleServerClick = (serverId: string) => {
    // Toggle expansion
    setExpandedServerId(expandedServerId === serverId ? null : serverId);
    
    // Call optional callback
    if (onServerClick) {
      onServerClick(serverId);
    }
  };

  /**
   * Get status indicator color based on server status
   * Requirement 2.4, 2.5: Status color mapping
   */
  const getStatusColor = (status: MCPServer['status']): string => {
    switch (status) {
      case 'online':
        return 'status-online'; // Green
      case 'offline':
        return 'status-offline'; // Red
      case 'error':
        return 'status-error'; // Yellow
      default:
        return 'status-offline';
    }
  };

  /**
   * Get status text for accessibility
   */
  const getStatusText = (status: MCPServer['status']): string => {
    switch (status) {
      case 'online':
        return 'Online';
      case 'offline':
        return 'Offline';
      case 'error':
        return 'Error';
      default:
        return 'Unknown';
    }
  };

  return (
    <div className="mcp-status-panel">
      <h2 className="panel-title">MCP Servers</h2>
      
      {servers.length === 0 ? (
        <div className="empty-state">
          <p>No MCP servers configured</p>
        </div>
      ) : (
        <ul className="server-list">
          {servers.map((server) => {
            const isExpanded = expandedServerId === server.id;
            
            return (
              <li key={server.id} className="server-item">
                <button
                  className="server-header"
                  onClick={() => handleServerClick(server.id)}
                  aria-expanded={isExpanded}
                  aria-label={`${server.name} - ${getStatusText(server.status)}`}
                >
                  <div className="server-info">
                    <span
                      className={`status-indicator ${getStatusColor(server.status)}`}
                      aria-label={getStatusText(server.status)}
                      title={getStatusText(server.status)}
                    />
                    <span className="server-name">{server.name}</span>
                  </div>
                  
                  <svg
                    className={`expand-icon ${isExpanded ? 'expanded' : ''}`}
                    width="16"
                    height="16"
                    viewBox="0 0 16 16"
                    fill="none"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 6l4 4 4-4"
                    />
                  </svg>
                </button>

                {isExpanded && (
                  <div className="tool-list-container">
                    {server.tools.length === 0 ? (
                      <p className="no-tools">No tools available</p>
                    ) : (
                      <ul className="tool-list">
                        {server.tools.map((tool, index) => (
                          <li key={`${server.id}-${tool.name}-${index}`} className="tool-item">
                            <div className="tool-header">
                              <span className="tool-name">{tool.name}</span>
                              {tool.readonly && (
                                <span className="tool-badge readonly">Read-only</span>
                              )}
                            </div>
                            {tool.description && (
                              <p className="tool-description">{tool.description}</p>
                            )}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
};

export default MCPStatusPanel;
