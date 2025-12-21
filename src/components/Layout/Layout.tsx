import React, { useState } from 'react';
import type { ReactNode } from 'react';
import './Layout.css';

interface LayoutProps {
  leftPanel?: ReactNode;
  rightPanel?: ReactNode;
  children: ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ leftPanel, rightPanel, children }) => {
  const [leftDrawerOpen, setLeftDrawerOpen] = useState(false);
  const [rightDrawerOpen, setRightDrawerOpen] = useState(false);

  const toggleLeftDrawer = () => {
    setLeftDrawerOpen(!leftDrawerOpen);
    if (rightDrawerOpen) setRightDrawerOpen(false);
  };

  const toggleRightDrawer = () => {
    setRightDrawerOpen(!rightDrawerOpen);
    if (leftDrawerOpen) setLeftDrawerOpen(false);
  };

  const closeDrawers = () => {
    setLeftDrawerOpen(false);
    setRightDrawerOpen(false);
  };

  return (
    <div className="layout">
      {/* Mobile Header with Menu Buttons */}
      <header className="layout-header">
        <button
          className="drawer-toggle"
          onClick={toggleLeftDrawer}
          aria-label="Toggle left panel"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
        <h1 className="layout-title">OpsGenius</h1>
        <button
          className="drawer-toggle"
          onClick={toggleRightDrawer}
          aria-label="Toggle right panel"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
        </button>
      </header>

      {/* Overlay for mobile drawers */}
      {(leftDrawerOpen || rightDrawerOpen) && (
        <div className="drawer-overlay" onClick={closeDrawers} />
      )}

      {/* Three Column Layout */}
      <div className="layout-container">
        {/* Left Panel */}
        {leftPanel && (
          <aside className={`layout-left ${leftDrawerOpen ? 'drawer-open' : ''}`}>
            <div className="panel-content">
              {leftPanel}
            </div>
          </aside>
        )}

        {/* Main Content */}
        <main className="layout-main">
          {children}
        </main>

        {/* Right Panel */}
        {rightPanel && (
          <aside className={`layout-right ${rightDrawerOpen ? 'drawer-open' : ''}`}>
            <div className="panel-content">
              {rightPanel}
            </div>
          </aside>
        )}
      </div>
    </div>
  );
};

export default Layout;
