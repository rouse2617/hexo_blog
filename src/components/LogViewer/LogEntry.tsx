/**
 * LogEntry Component
 * Requirements: 3.1, 3.2, 3.3
 * 
 * Renders a single log entry with timestamp, level indicator, message,
 * and clickable agent references.
 */

import React from 'react';
import type { LogEntry as LogEntryType } from '../../types/models';
import './LogEntry.css';

interface LogEntryProps {
  log: LogEntryType;
  onAgentClick?: (agentId: string) => void;
}

/**
 * Format timestamp to readable string
 */
function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString('en-US', {
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
  switch (level) {
    case 'error':
      return 'log-entry-level-error';
    case 'warning':
      return 'log-entry-level-warning';
    case 'info':
    default:
      return 'log-entry-level-info';
  }
}

/**
 * Parse message and extract agent references
 * Requirement 3.2: Render agent references as clickable links
 */
function parseMessageWithRefs(
  message: string,
  agentRefs: LogEntryType['agentRefs'],
  onAgentClick?: (agentId: string) => void
): React.ReactNode {
  if (!agentRefs || agentRefs.length === 0) {
    return message;
  }

  // Split message by agent reference patterns
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;

  agentRefs.forEach((ref, index) => {
    // Find reference in message (e.g., "@AgentName" or "Agent:AgentName")
    const pattern = new RegExp(`@${ref.displayName}|Agent:${ref.displayName}`, 'g');
    const match = pattern.exec(message.substring(lastIndex));

    if (match) {
      const matchIndex = lastIndex + match.index;

      // Add text before reference
      if (matchIndex > lastIndex) {
        parts.push(message.substring(lastIndex, matchIndex));
      }

      // Add clickable reference
      parts.push(
        <button
          key={`ref-${index}`}
          className="log-entry-agent-ref"
          onClick={() => onAgentClick?.(ref.id)}
          title={`View ${ref.displayName} details`}
        >
          {match[0]}
        </button>
      );

      lastIndex = matchIndex + match[0].length;
    }
  });

  // Add remaining text
  if (lastIndex < message.length) {
    parts.push(message.substring(lastIndex));
  }

  return parts.length > 0 ? parts : message;
}

/**
 * LogEntry Component
 * Requirement 3.1: Display log entry with timestamp and level
 * Requirement 3.2: Render agent references as clickable elements
 * Requirement 3.3: Trigger callback on agent reference click
 */
export const LogEntry: React.FC<LogEntryProps> = ({ log, onAgentClick }) => {
  return (
    <div className="log-entry" data-log-id={log.id}>
      <div className="log-entry-header">
        <span className="log-entry-timestamp">{formatTimestamp(log.timestamp)}</span>
        <span className={`log-entry-level ${getLevelClass(log.level)}`}>
          {log.level.toUpperCase()}
        </span>
      </div>
      <div className="log-entry-message">
        {parseMessageWithRefs(log.message, log.agentRefs, onAgentClick)}
      </div>
    </div>
  );
};
