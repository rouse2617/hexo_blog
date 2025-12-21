/**
 * MCP Status Panel Unit Tests
 * Requirements: 2.1, 2.2, 2.3, 2.4, 2.5
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import MCPStatusPanel from './MCPStatusPanel';
import type { MCPServer } from '../../types/models';

describe('MCPStatusPanel', () => {
  const mockServers: MCPServer[] = [
    {
      id: 'server-1',
      name: 'Test Server 1',
      status: 'online',
      tools: [
        {
          name: 'tool1',
          description: 'Test tool 1',
          readonly: false,
        },
        {
          name: 'tool2',
          description: 'Test tool 2',
          readonly: true,
        },
      ],
      lastHeartbeat: Date.now(),
    },
    {
      id: 'server-2',
      name: 'Test Server 2',
      status: 'offline',
      tools: [],
      lastHeartbeat: Date.now() - 60000,
    },
    {
      id: 'server-3',
      name: 'Test Server 3',
      status: 'error',
      tools: [
        {
          name: 'tool3',
          description: 'Test tool 3',
          readonly: false,
        },
      ],
    },
  ];

  it('should render all configured servers', () => {
    render(<MCPStatusPanel servers={mockServers} />);
    
    expect(screen.getByText('Test Server 1')).toBeInTheDocument();
    expect(screen.getByText('Test Server 2')).toBeInTheDocument();
    expect(screen.getByText('Test Server 3')).toBeInTheDocument();
  });

  it('should display empty state when no servers are configured', () => {
    render(<MCPStatusPanel servers={[]} />);
    
    expect(screen.getByText('No MCP servers configured')).toBeInTheDocument();
  });

  it('should expand tool list when server is clicked', () => {
    render(<MCPStatusPanel servers={mockServers} />);
    
    const serverButton = screen.getByRole('button', { name: /Test Server 1/i });
    
    // Initially, tools should not be visible
    expect(screen.queryByText('tool1')).not.toBeInTheDocument();
    
    // Click to expand
    fireEvent.click(serverButton);
    
    // Tools should now be visible
    expect(screen.getByText('tool1')).toBeInTheDocument();
    expect(screen.getByText('tool2')).toBeInTheDocument();
  });

  it('should collapse tool list when expanded server is clicked again', () => {
    render(<MCPStatusPanel servers={mockServers} />);
    
    const serverButton = screen.getByRole('button', { name: /Test Server 1/i });
    
    // Expand
    fireEvent.click(serverButton);
    expect(screen.getByText('tool1')).toBeInTheDocument();
    
    // Collapse
    fireEvent.click(serverButton);
    expect(screen.queryByText('tool1')).not.toBeInTheDocument();
  });

  it('should display "No tools available" when server has no tools', () => {
    render(<MCPStatusPanel servers={mockServers} />);
    
    const serverButton = screen.getByRole('button', { name: /Test Server 2/i });
    fireEvent.click(serverButton);
    
    expect(screen.getByText('No tools available')).toBeInTheDocument();
  });

  it('should call onServerClick callback when server is clicked', () => {
    const onServerClick = vi.fn();
    render(<MCPStatusPanel servers={mockServers} onServerClick={onServerClick} />);
    
    const serverButton = screen.getByRole('button', { name: /Test Server 1/i });
    fireEvent.click(serverButton);
    
    expect(onServerClick).toHaveBeenCalledWith('server-1');
  });

  it('should display readonly badge for readonly tools', () => {
    render(<MCPStatusPanel servers={mockServers} />);
    
    const serverButton = screen.getByRole('button', { name: /Test Server 1/i });
    fireEvent.click(serverButton);
    
    const readonlyBadges = screen.getAllByText('Read-only');
    expect(readonlyBadges).toHaveLength(1);
  });

  it('should display correct status indicators for different server states', () => {
    const { container } = render(<MCPStatusPanel servers={mockServers} />);
    
    const onlineIndicator = container.querySelector('.status-online');
    const offlineIndicator = container.querySelector('.status-offline');
    const errorIndicator = container.querySelector('.status-error');
    
    expect(onlineIndicator).toBeInTheDocument();
    expect(offlineIndicator).toBeInTheDocument();
    expect(errorIndicator).toBeInTheDocument();
  });

  it('should have proper aria attributes for accessibility', () => {
    render(<MCPStatusPanel servers={mockServers} />);
    
    const serverButton = screen.getByRole('button', { name: /Test Server 1/i });
    
    // Initially collapsed
    expect(serverButton).toHaveAttribute('aria-expanded', 'false');
    
    // After click, expanded
    fireEvent.click(serverButton);
    expect(serverButton).toHaveAttribute('aria-expanded', 'true');
  });
});
