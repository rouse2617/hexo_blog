import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import Layout from './Layout';

describe('Layout Component', () => {
  it('should render children in main content area', () => {
    render(
      <Layout>
        <div data-testid="main-content">Main Content</div>
      </Layout>
    );

    expect(screen.getByTestId('main-content')).toBeInTheDocument();
    expect(screen.getByText('Main Content')).toBeInTheDocument();
  });

  it('should render left panel when provided', () => {
    render(
      <Layout leftPanel={<div data-testid="left-panel">Left Panel</div>}>
        <div>Main</div>
      </Layout>
    );

    expect(screen.getByTestId('left-panel')).toBeInTheDocument();
  });

  it('should render right panel when provided', () => {
    render(
      <Layout rightPanel={<div data-testid="right-panel">Right Panel</div>}>
        <div>Main</div>
      </Layout>
    );

    expect(screen.getByTestId('right-panel')).toBeInTheDocument();
  });

  it('should render all three sections when all props provided', () => {
    render(
      <Layout
        leftPanel={<div data-testid="left-panel">Left</div>}
        rightPanel={<div data-testid="right-panel">Right</div>}
      >
        <div data-testid="main-content">Main</div>
      </Layout>
    );

    expect(screen.getByTestId('left-panel')).toBeInTheDocument();
    expect(screen.getByTestId('main-content')).toBeInTheDocument();
    expect(screen.getByTestId('right-panel')).toBeInTheDocument();
  });

  it('should toggle left drawer when left toggle button clicked', () => {
    render(
      <Layout leftPanel={<div data-testid="left-panel">Left</div>}>
        <div>Main</div>
      </Layout>
    );

    const leftToggle = screen.getByLabelText('Toggle left panel');
    const leftPanel = screen.getByTestId('left-panel').closest('.layout-left');

    // Initially closed (no drawer-open class)
    expect(leftPanel).not.toHaveClass('drawer-open');

    // Click to open
    fireEvent.click(leftToggle);
    expect(leftPanel).toHaveClass('drawer-open');

    // Click to close
    fireEvent.click(leftToggle);
    expect(leftPanel).not.toHaveClass('drawer-open');
  });

  it('should toggle right drawer when right toggle button clicked', () => {
    render(
      <Layout rightPanel={<div data-testid="right-panel">Right</div>}>
        <div>Main</div>
      </Layout>
    );

    const rightToggle = screen.getByLabelText('Toggle right panel');
    const rightPanel = screen.getByTestId('right-panel').closest('.layout-right');

    // Initially closed
    expect(rightPanel).not.toHaveClass('drawer-open');

    // Click to open
    fireEvent.click(rightToggle);
    expect(rightPanel).toHaveClass('drawer-open');

    // Click to close
    fireEvent.click(rightToggle);
    expect(rightPanel).not.toHaveClass('drawer-open');
  });

  it('should close drawers when overlay is clicked', () => {
    render(
      <Layout
        leftPanel={<div data-testid="left-panel">Left</div>}
        rightPanel={<div data-testid="right-panel">Right</div>}
      >
        <div>Main</div>
      </Layout>
    );

    const leftToggle = screen.getByLabelText('Toggle left panel');
    const leftPanel = screen.getByTestId('left-panel').closest('.layout-left');

    // Open left drawer
    fireEvent.click(leftToggle);
    expect(leftPanel).toHaveClass('drawer-open');

    // Click overlay to close
    const overlay = document.querySelector('.drawer-overlay');
    if (overlay) {
      fireEvent.click(overlay);
      expect(leftPanel).not.toHaveClass('drawer-open');
    }
  });

  it('should close left drawer when right drawer is opened', () => {
    render(
      <Layout
        leftPanel={<div data-testid="left-panel">Left</div>}
        rightPanel={<div data-testid="right-panel">Right</div>}
      >
        <div>Main</div>
      </Layout>
    );

    const leftToggle = screen.getByLabelText('Toggle left panel');
    const rightToggle = screen.getByLabelText('Toggle right panel');
    const leftPanel = screen.getByTestId('left-panel').closest('.layout-left');
    const rightPanel = screen.getByTestId('right-panel').closest('.layout-right');

    // Open left drawer
    fireEvent.click(leftToggle);
    expect(leftPanel).toHaveClass('drawer-open');
    expect(rightPanel).not.toHaveClass('drawer-open');

    // Open right drawer (should close left)
    fireEvent.click(rightToggle);
    expect(leftPanel).not.toHaveClass('drawer-open');
    expect(rightPanel).toHaveClass('drawer-open');
  });

  it('should display mobile header with toggle buttons', () => {
    render(
      <Layout
        leftPanel={<div>Left</div>}
        rightPanel={<div>Right</div>}
      >
        <div>Main</div>
      </Layout>
    );

    expect(screen.getByText('OpsGenius')).toBeInTheDocument();
    expect(screen.getByLabelText('Toggle left panel')).toBeInTheDocument();
    expect(screen.getByLabelText('Toggle right panel')).toBeInTheDocument();
  });
});
