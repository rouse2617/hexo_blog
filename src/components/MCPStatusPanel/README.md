# MCPStatusPanel Component

## Overview

The MCPStatusPanel component displays a list of MCP (Model Context Protocol) servers with their current status and available tools. It provides an expandable/collapsible interface for viewing tool details.

## Requirements

- **2.1**: Display all configured MCP Servers
- **2.2**: Update status indicators when server status changes
- **2.3**: Expand/collapse tool list on server click
- **2.4**: Display offline servers with red indicator
- **2.5**: Display online servers with green indicator

## Features

- **Status Indicators**: Visual indicators showing server status (online/offline/error)
- **Expandable Tool Lists**: Click on a server to view its available tools
- **Tool Details**: Display tool names, descriptions, and readonly status
- **Responsive Design**: Works on desktop and mobile devices
- **Accessibility**: Proper ARIA attributes and keyboard navigation

## Usage

```tsx
import MCPStatusPanel from './components/MCPStatusPanel';
import { useAppStore } from './stores/appStore';

function App() {
  const mcpServers = useAppStore((state) => state.mcpServers);
  
  return (
    <MCPStatusPanel 
      servers={mcpServers}
      onServerClick={(serverId) => console.log('Clicked:', serverId)}
    />
  );
}
```

## Props

### MCPStatusPanelProps

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `servers` | `MCPServer[]` | Yes | Array of MCP servers to display |
| `onServerClick` | `(serverId: string) => void` | No | Callback when a server is clicked |

## Status Colors

- **Green** (online): Server is connected and operational
- **Red** (offline): Server is disconnected
- **Yellow** (error): Server encountered an error

## Component Structure

```
MCPStatusPanel
├── Panel Title
├── Server List
│   ├── Server Item
│   │   ├── Server Header (clickable)
│   │   │   ├── Status Indicator
│   │   │   ├── Server Name
│   │   │   └── Expand Icon
│   │   └── Tool List (when expanded)
│   │       └── Tool Items
│   │           ├── Tool Name
│   │           ├── Readonly Badge (if applicable)
│   │           └── Tool Description
```

## Styling

The component uses CSS modules for styling. Key classes:

- `.mcp-status-panel`: Main container
- `.server-list`: List of servers
- `.server-header`: Clickable server header
- `.status-indicator`: Status dot with color
- `.tool-list`: Expandable tool list
- `.tool-item`: Individual tool display

## Testing

The component includes comprehensive unit tests covering:

- Server list rendering
- Empty state display
- Expand/collapse functionality
- Status indicator colors
- Tool list display
- Readonly badge display
- Accessibility features

Run tests with:

```bash
npm test -- src/components/MCPStatusPanel/MCPStatusPanel.test.tsx --run
```

## Implementation Notes

1. **State Management**: Uses local state for expansion tracking
2. **Performance**: Only one server can be expanded at a time
3. **Accessibility**: Includes proper ARIA labels and keyboard support
4. **Responsive**: Adapts to mobile and desktop layouts
