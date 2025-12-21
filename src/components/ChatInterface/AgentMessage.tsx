/**
 * AgentMessage Component
 * Requirements: 1.2, 1.3, 1.4
 * 
 * Displays agent messages with streaming support, Markdown rendering, and code highlighting.
 */

import React from 'react';
import type { Message } from '../../types/models';
import MarkdownRenderer from './MarkdownRenderer';
import './ChatInterface.css';

export interface AgentMessageProps {
  message: Message;
}

/**
 * AgentMessage Component
 * 
 * Requirement 1.2: Display agent messages with streaming support
 * Requirement 1.3: Render Markdown content (will integrate MarkdownRenderer)
 * Requirement 1.4: Syntax highlight code blocks (will integrate CodeBlock)
 */
const AgentMessage: React.FC<AgentMessageProps> = ({ message }) => {
  /**
   * Format timestamp to readable format
   */
  const formatTimestamp = (timestamp: number): string => {
    const date = new Date(timestamp);
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
  };

  return (
    <div 
      className={`message agent-message ${message.streaming ? 'streaming' : ''}`}
      data-message-id={message.id}
    >
      <div className="message-header">
        <span className="message-role">Agent</span>
        <span className="message-timestamp">{formatTimestamp(message.timestamp)}</span>
        {message.streaming && (
          <span className="streaming-indicator" aria-label="Streaming">
            <span className="dot"></span>
            <span className="dot"></span>
            <span className="dot"></span>
          </span>
        )}
      </div>
      <div className="message-content">
        {/* Render Markdown content */}
        <MarkdownRenderer content={message.content} />
      </div>
      
      {/* Display charts if available */}
      {message.charts && message.charts.length > 0 && (
        <div className="message-charts">
          {message.charts.map((chart, index) => (
            <div key={index} className="chart-placeholder">
              {/* TODO: Integrate ChartRenderer */}
              <p>Chart: {chart.type}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default AgentMessage;
