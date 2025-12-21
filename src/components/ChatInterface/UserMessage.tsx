/**
 * UserMessage Component
 * Requirements: 1.1
 * 
 * Displays user messages in the chat interface.
 */

import React from 'react';
import type { Message } from '../../types/models';
import './ChatInterface.css';

export interface UserMessageProps {
  message: Message;
}

/**
 * UserMessage Component
 * 
 * Requirement 1.1: Display user messages with timestamp
 */
const UserMessage: React.FC<UserMessageProps> = ({ message }) => {
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
    <div className="message user-message" data-message-id={message.id}>
      <div className="message-header">
        <span className="message-role">You</span>
        <span className="message-timestamp">{formatTimestamp(message.timestamp)}</span>
      </div>
      <div className="message-content">
        <p>{message.content}</p>
      </div>
    </div>
  );
};

export default UserMessage;
