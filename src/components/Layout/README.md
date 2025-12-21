# Layout Component

A responsive three-column layout component for the OpsGenius Frontend application.

## Features

- **Three-column layout**: Left panel, main content area, and right panel
- **Responsive design**: 
  - Desktop (≥1024px): Full three-column layout with wider panels
  - Tablet (768px-1023px): Three-column layout with medium-sized panels
  - Mobile (<768px): Drawer-based navigation with overlay
- **Mobile drawer menu**: Slide-in panels from left and right on mobile devices
- **Smooth transitions**: Animated panel transitions and layout changes
- **Touch-friendly**: Optimized for mobile interactions
- **State preservation**: Maintains application state during layout changes

## Usage

```tsx
import Layout from './components/Layout';

function App() {
  return (
    <Layout
      leftPanel={<MCPStatusPanel />}
      rightPanel={<LogViewer />}
    >
      <ChatInterface />
    </Layout>
  );
}
```

## Props

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `leftPanel` | `ReactNode` | No | Content for the left sidebar panel |
| `rightPanel` | `ReactNode` | No | Content for the right sidebar panel |
| `children` | `ReactNode` | Yes | Main content area |

## Responsive Behavior

### Desktop (≥1024px)
- Left panel: 300px width
- Right panel: 360px width
- Main content: Flexible width (fills remaining space)
- All panels visible simultaneously

### Tablet (768px-1023px)
- Left panel: 240px width
- Right panel: 280px width
- Main content: Flexible width
- All panels visible simultaneously

### Mobile (<768px)
- Header with toggle buttons visible
- Panels slide in as drawers from left/right
- Overlay darkens background when drawer is open
- Only one drawer can be open at a time
- Clicking overlay closes the drawer

## Accessibility

- Toggle buttons have proper `aria-label` attributes
- Keyboard navigation supported
- Focus management for drawer interactions

## Styling

The component uses CSS modules with the following key classes:
- `.layout`: Main container
- `.layout-header`: Mobile header with toggle buttons
- `.layout-left`: Left panel/drawer
- `.layout-main`: Main content area
- `.layout-right`: Right panel/drawer
- `.drawer-overlay`: Mobile overlay background
- `.drawer-open`: Applied to open drawers

## Requirements Validation

This component satisfies the following requirements:
- **Requirement 7.1**: Mobile drawer menu for screens < 768px
- **Requirement 7.2**: Three-column layout for screens ≥ 768px
- **Requirement 7.4**: State preservation during layout changes

## Testing

Run tests with:
```bash
npm test -- src/components/Layout/Layout.test.tsx
```

The test suite covers:
- Rendering of all three sections
- Drawer toggle functionality
- Overlay interaction
- Mutual exclusion of drawers
- Mobile header display
