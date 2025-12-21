/**
 * Property-Based Tests for MCPStatusPanel Component
 * 
 * These tests verify universal properties that should hold for all inputs.
 */

import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/react';
import { fc } from '@fast-check/vitest';
import MCPStatusPanel from './MCPStatusPanel';
import type { MCPServer, MCPTool } from '../../types/models';

// Generator for MCPTool
const mcpToolArb = fc.record({
  name: fc.string({ minLength: 1, maxLength: 50 }).filter(s => s.trim().length > 0),
  description: fc.string({ maxLength: 200 }),
  readonly: fc.boolean(),
});

// Generator for MCPServer with unique IDs
const mcpServerArb = (index: number) => fc.record({
  id: fc.constant(`server-${index}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`),
  name: fc.string({ minLength: 1, maxLength: 50 }).filter(s => s.trim().length > 0),
  status: fc.constantFrom('online', 'offline', 'error') as fc.Arbitrary<'online' | 'offline' | 'error'>,
  tools: fc.array(mcpToolArb, { maxLength: 10 }),
  lastHeartbeat: fc.option(fc.integer({ min: 0 }), { nil: undefined }),
});

// Generator for array of servers with unique IDs
const mcpServersArb = (minLength: number, maxLength: number) => 
  fc.integer({ min: minLength, max: maxLength }).chain(length => 
    fc.tuple(...Array.from({ length }, (_, i) => mcpServerArb(i)))
  );

describe('MCPStatusPanel Property Tests', () => {
  // Feature: ops-genius-frontend, Property 6: MCP Server 列表完整性
  it('should render all servers from the configuration', () => {
    fc.assert(
      fc.property(
        mcpServersArb(0, 20),
        (servers) => {
          const { container } = render(<MCPStatusPanel servers={servers} />);
          
          // Count rendered server items
          const serverItems = container.querySelectorAll('.server-item');
          
          // Validates: Requirements 2.1
          expect(serverItems.length).toBe(servers.length);
        }
      ),
      { numRuns: 100 }
    );
  });

  // Feature: ops-genius-frontend, Property 9: 状态颜色映射正确性
  it('should correctly map server status to indicator colors', () => {
    fc.assert(
      fc.property(
        mcpServersArb(1, 10),
        (servers) => {
          const { container } = render(<MCPStatusPanel servers={servers} />);
          
          servers.forEach((server) => {
            // Find the status indicator for this server
            const serverItem = Array.from(container.querySelectorAll('.server-item')).find(
              (item) => item.querySelector('.server-name')?.textContent === server.name
            );
            
            expect(serverItem).toBeDefined();
            
            const statusIndicator = serverItem?.querySelector('.status-indicator');
            expect(statusIndicator).toBeDefined();
            
            // Verify correct status class
            // Validates: Requirements 2.4, 2.5
            switch (server.status) {
              case 'online':
                expect(statusIndicator?.classList.contains('status-online')).toBe(true);
                break;
              case 'offline':
                expect(statusIndicator?.classList.contains('status-offline')).toBe(true);
                break;
              case 'error':
                expect(statusIndicator?.classList.contains('status-error')).toBe(true);
                break;
            }
          });
        }
      ),
      { numRuns: 100 }
    );
  });

  // Additional property: Tool list expansion shows correct number of tools
  it('should display the correct number of tools when expanded', () => {
    fc.assert(
      fc.property(
        mcpServerArb(0),
        (server) => {
          const { container } = render(<MCPStatusPanel servers={[server]} />);
          
          // Find and click the server button
          const serverButton = container.querySelector('.server-header') as HTMLElement;
          expect(serverButton).toBeDefined();
          fireEvent.click(serverButton);
          
          // Count rendered tools
          const toolItems = container.querySelectorAll('.tool-item');
          
          // Validates: Requirements 2.3
          expect(toolItems.length).toBe(server.tools.length);
        }
      ),
      { numRuns: 100 }
    );
  });

  // Additional property: Server names are displayed correctly
  it('should display all server names correctly', () => {
    fc.assert(
      fc.property(
        mcpServersArb(1, 10),
        (servers) => {
          const { container } = render(<MCPStatusPanel servers={servers} />);
          
          servers.forEach((server) => {
            // Each server name should be visible in the DOM
            const serverNames = Array.from(container.querySelectorAll('.server-name'));
            const hasServerName = serverNames.some(el => el.textContent === server.name);
            expect(hasServerName).toBe(true);
          });
        }
      ),
      { numRuns: 100 }
    );
  });

  // Additional property: Empty server list shows empty state
  it('should show empty state when no servers are provided', () => {
    const { container } = render(<MCPStatusPanel servers={[]} />);
    
    const emptyState = container.querySelector('.empty-state');
    expect(emptyState).toBeDefined();
    expect(emptyState?.textContent).toContain('No MCP servers configured');
  });

  // Additional property: Readonly tools are marked correctly
  it('should mark readonly tools with a badge', () => {
    fc.assert(
      fc.property(
        fc.record({
          id: fc.constant(`server-readonly-${Date.now()}`),
          name: fc.string({ minLength: 1, maxLength: 50 }).filter(s => s.trim().length > 0),
          status: fc.constantFrom('online', 'offline', 'error') as fc.Arbitrary<'online' | 'offline' | 'error'>,
          tools: fc.array(
            fc.record({
              name: fc.string({ minLength: 1, maxLength: 50 }).filter(s => s.trim().length > 0),
              description: fc.string({ maxLength: 200 }),
              readonly: fc.constant(true),
            }),
            { minLength: 1, maxLength: 5 }
          ),
          lastHeartbeat: fc.option(fc.integer({ min: 0 }), { nil: undefined }),
        }),
        (server) => {
          const { container } = render(<MCPStatusPanel servers={[server]} />);
          
          // Find and click the server button
          const serverButton = container.querySelector('.server-header') as HTMLElement;
          expect(serverButton).toBeDefined();
          fireEvent.click(serverButton);
          
          // All tools should have readonly badge
          const readonlyBadges = container.querySelectorAll('.tool-badge.readonly');
          expect(readonlyBadges.length).toBe(server.tools.length);
        }
      ),
      { numRuns: 100 }
    );
  });
});
