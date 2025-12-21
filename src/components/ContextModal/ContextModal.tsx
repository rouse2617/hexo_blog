/**
 * ContextModal Component
 * Requirements: 4.1, 4.2, 4.3, 4.4, 4.5
 * 
 * Modal dialog that displays detailed context logs for a specific agent.
 * Includes background scroll locking and proper cleanup on close.
 */

import React, { useEffect } from 'react';
import type { ContextLog } from '../../types/models';
import './ContextModal.css';

interface ContextModalProps {
  visible: boolean;
  agentId: string | null;
  logs: ContextLog[];
  onClose: () => void;
}

/**
 * Format timestamp to readable string
 */
function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp);
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

/**
 * Get CSS class for log level
 */
function getLevelClass(level: string): string {
  const normalized = level.toLowerCase();
  switch (normalized) {
    case 'error':
      return 'context-log-level-error';
    case 'warning':
    case 'warn':
      return 'context-log-level-warning';
    case 'info':
    default:
      return 'context-log-level-info';
  }
}

/**
 * ContextModal Component
 * Requirement 4.1: Display context logs for specific agent
 * Requirement 4.2: Sort logs by timestamp in ascending order
 * Requirement 4.3: Handle modal close and state cleanup
 * Requirement 4.4: Handle empty log state
 * Requirement 4.5: Lock background scrolling when modal is open
 */
export const ContextModal: React.FC<ContextModalProps> = ({
  visible,
  agentId,
  logs,
  onClose,
}) => {
  // Requirement 4.5: Lock background scrolling when modal is open
  useEffect(() => {
    if (visible) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }

    // Cleanup on unmount
    return () => {
      document.body.style.overflow = '';
    };
  }, [visible]);

  // Handle ESC key to close modal
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && visible) {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('keydown', handleEscape);
    };
  }, [visible, onClose]);

  if (!visible) {
    return null;
  }

  // Requirement 4.2: Sort logs by timestamp in ascending order
  const sortedLogs = [...logs].sort((a, b) => a.timestamp - b.timestamp);

  // Handle backdrop click
  const handleBackdropClick = (event: React.MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  };

  return (
    <div className="context-modal-backdrop" onClick={handleBackdropClick}>
      <div className="context-modal" role="dialog" aria-modal="true">
        <div className="context-modal-header">
          <h2 className="context-modal-title">
            Agent Context: {agentId || 'Unknown'}
          </h2>
          <button
            className="context-modal-close"
            onClick={onClose}
            aria-label="Close modal"
          >
            ×
          </button>
        </div>

        <div className="context-modal-content">
          {sortedLogs.length === 0 ? (
            // Requirement 4.4: Handle empty log state
            <div className="context-modal-empty">
              <p>No context logs available for this agent.</p>
            </div>
          ) : (
            <div className="context-modal-logs">
              {sortedLogs.map((log) => (
                <div
                  key={log.id}
                  className="context-log-entry"
                  data-timestamp={log.timestamp}
                >
                  <div className="context-log-header">
                    <span className="context-log-timestamp">
                      {formatTimestamp(log.timestamp)}
                    </span>
                    <span className={`context-log-level ${getLevelClass(log.level)}`}>
                      {log.level.toUpperCase()}
                    </span>
                  </div>
                  <div className="context-log-message">{log.message}</div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="context-modal-footer">
          <button className="context-modal-close-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
