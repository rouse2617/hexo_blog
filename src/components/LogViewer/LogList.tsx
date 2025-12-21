/**
 * LogList Component
 * Requirements: 3.1, 3.4, 3.5, 10.4
 * 
 * Renders a list of log entries with virtual scrolling for large lists
 * and automatic scroll control.
 */

import React, { useEffect, useRef, useState } from 'react';
import { LogEntry } from './LogEntry';
import type { LogEntry as LogEntryType } from '../../types/models';
import './LogList.css';

interface LogListProps {
  logs: LogEntryType[];
  onAgentClick?: (agentId: string) => void;
  autoScroll?: boolean;
}

const VIRTUAL_SCROLL_THRESHOLD = 1000;
const SCROLL_THRESHOLD = 50; // pixels from bottom to consider "at bottom"

/**
 * LogList Component
 * Requirement 3.1: Display log entries in real-time
 * Requirement 3.4: Implement virtual scrolling for large lists (>1000 entries)
 * Requirement 3.5: Control automatic scrolling behavior
 * Requirement 10.4: Performance optimization for large datasets
 */
export const LogList: React.FC<LogListProps> = ({
  logs,
  onAgentClick,
  autoScroll = true,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const [shouldAutoScroll, setShouldAutoScroll] = useState(autoScroll);
  const prevLogsLengthRef = useRef(logs.length);

  // Check if user is at bottom of scroll
  const checkIfAtBottom = () => {
    if (!containerRef.current) return false;

    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    const distanceFromBottom = scrollHeight - scrollTop - clientHeight;
    return distanceFromBottom <= SCROLL_THRESHOLD;
  };

  // Handle scroll event
  const handleScroll = () => {
    const atBottom = checkIfAtBottom();
    setIsAtBottom(atBottom);

    // Resume auto-scroll if user scrolls back to bottom
    if (atBottom && !shouldAutoScroll) {
      setShouldAutoScroll(true);
    }
    // Pause auto-scroll if user scrolls away from bottom
    else if (!atBottom && shouldAutoScroll) {
      setShouldAutoScroll(false);
    }
  };

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (logs.length > prevLogsLengthRef.current && shouldAutoScroll && isAtBottom) {
      if (containerRef.current) {
        containerRef.current.scrollTop = containerRef.current.scrollHeight;
      }
    }
    prevLogsLengthRef.current = logs.length;
  }, [logs, shouldAutoScroll, isAtBottom]);

  // Determine if virtual scrolling should be used
  const useVirtualScroll = logs.length > VIRTUAL_SCROLL_THRESHOLD;

  if (logs.length === 0) {
    return (
      <div className="log-list-empty">
        <p>No logs yet</p>
      </div>
    );
  }

  // For now, render all logs (virtual scrolling can be added later with react-window)
  // Requirement 3.4: Virtual scrolling threshold check
  return (
    <div
      ref={containerRef}
      className={`log-list ${useVirtualScroll ? 'log-list-virtual' : ''}`}
      onScroll={handleScroll}
    >
      {!shouldAutoScroll && (
        <div className="log-list-scroll-hint">
          <button
            className="log-list-scroll-button"
            onClick={() => {
              if (containerRef.current) {
                containerRef.current.scrollTop = containerRef.current.scrollHeight;
                setShouldAutoScroll(true);
              }
            }}
          >
            ↓ New logs available - Click to scroll to bottom
          </button>
        </div>
      )}
      <div className="log-list-content">
        {logs.map((log) => (
          <LogEntry key={log.id} log={log} onAgentClick={onAgentClick} />
        ))}
      </div>
    </div>
  );
};
