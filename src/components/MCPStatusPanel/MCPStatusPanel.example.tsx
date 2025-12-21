/**
 * MCPStatusPanel Usage Example
 * 
 * This file demonstrates how to integrate the MCPStatusPanel component
 * with the application store and use it in the Layout.
 */

import React, { useEffect } from 'react';
import MCPStatusPanel from './MCPStatusPanel';
import { useAppStore } from '../../stores/appStore';
import type { MCPServer } from '../../types/models';

/**
 * Example: Using MCPStatusPanel with mock data
 */
export function MCPStatusPanelExample() {
  const mcpServers = useAppStore((state) => state.mcpServers);
  const setMCPServers = useAppStore((state) => state.setMCPServers);

  // Initialize with mock data
  useEffect(() => {
    const mockServers: MCPServer[] = [
      {
        id: 'aws-mcp',
        name: 'AWS MCP Server',
        status: 'online',
        tools: [
          {
            name: 'ec2_list_instances',
            description: 'List all EC2 instances in the account',
            readonly: true,
          },
          {
            name: 'ec2_start_instance',
            description: 'Start an EC2 instance',
            readonly: false,
          },
          {
            name: 'ec2_stop_instance',
            description: 'Stop an EC2 instance',
            readonly: false,
          },
        ],
        lastHeartbeat: Date.now(),
      },
      {
        id: 'k8s-mcp',
        name: 'Kubernetes MCP Server',
        status: 'online',
        tools: [
          {
            name: 'kubectl_get_pods',
            description: 'Get all pods in a namespace',
            readonly: true,
          },
          {
            name: 'kubectl_describe_pod',
            description: 'Describe a specific pod',
            readonly: true,
          },
          {
            name: 'kubectl_delete_pod',
            description: 'Delete a pod',
            readonly: false,
          },
        ],
        lastHeartbeat: Date.now(),
      },
      {
        id: 'monitoring-mcp',
        name: 'Monitoring MCP Server',
        status: 'offline',
        tools: [
          {
            name: 'get_metrics',
            description: 'Retrieve system metrics',
            readonly: true,
          },
        ],
        lastHeartbeat: Date.now() - 120000,
      },
    ];

    setMCPServers(mockServers);
  }, [setMCPServers]);

  const handleServerClick = (serverId: string) => {
    console.log('Server clicked:', serverId);
  };

  return (
    <div style={{ width: '300px', height: '600px', border: '1px solid #ccc' }}>
      <MCPStatusPanel servers={mcpServers} onServerClick={handleServerClick} />
    </div>
  );
}

/**
 * Example: Using MCPStatusPanel in Layout
 */
export function LayoutWithMCPPanel() {
  const mcpServers = useAppStore((state) => state.mcpServers);

  return (
    <div style={{ display: 'flex', height: '100vh' }}>
      {/* Left Panel - MCP Status */}
      <aside style={{ width: '300px', borderRight: '1px solid #ccc' }}>
        <MCPStatusPanel servers={mcpServers} />
      </aside>

      {/* Main Content */}
      <main style={{ flex: 1, padding: '1rem' }}>
        <h1>Main Content Area</h1>
        <p>Chat interface would go here</p>
      </main>

      {/* Right Panel */}
      <aside style={{ width: '300px', borderLeft: '1px solid #ccc' }}>
        <p>Log viewer would go here</p>
      </aside>
    </div>
  );
}

/**
 * Example: Simulating status updates
 */
export function MCPStatusPanelWithUpdates() {
  const mcpServers = useAppStore((state) => state.mcpServers);
  const updateMCPStatus = useAppStore((state) => state.updateMCPStatus);
  const setMCPServers = useAppStore((state) => state.setMCPServers);

  useEffect(() => {
    // Initialize servers
    const mockServers: MCPServer[] = [
      {
        id: 'server-1',
        name: 'Test Server',
        status: 'online',
        tools: [
          {
            name: 'test_tool',
            description: 'A test tool',
            readonly: false,
          },
        ],
        lastHeartbeat: Date.now(),
      },
    ];
    setMCPServers(mockServers);

    // Simulate status changes
    const interval = setInterval(() => {
      const statuses: Array<'online' | 'offline' | 'error'> = ['online', 'offline', 'error'];
      const randomStatus = statuses[Math.floor(Math.random() * statuses.length)];
      updateMCPStatus('server-1', randomStatus);
    }, 3000);

    return () => clearInterval(interval);
  }, [setMCPServers, updateMCPStatus]);

  return (
    <div style={{ width: '300px', height: '400px', border: '1px solid #ccc' }}>
      <MCPStatusPanel servers={mcpServers} />
      <p style={{ padding: '1rem', fontSize: '0.875rem', color: '#666' }}>
        Status updates every 3 seconds
      </p>
    </div>
  );
}
