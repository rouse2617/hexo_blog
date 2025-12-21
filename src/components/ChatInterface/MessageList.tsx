/**
 * MessageList Component
 * Requirements: 1.1, 1.2
 * 
 * Displays list of messages with auto-scroll functionality.
 */

import React, { useEffect, useRef } from 'react';
import type { Message } from '../../types/models';
import UserMessage from './UserMessage';
import AgentMessage from './AgentMessage';
import './ChatInterface.css';

export interface MessageListProps {
  messages: Message[];
  autoScroll?: boolean;
}

/**
 * MessageList Component
 * 
 * Requirement 1.1: Display user messages
 * Requirement 1.2: Display agent messages with streaming support
 */
const MessageList: React.FC<MessageListProps> = ({ messages, autoScroll = true }) => {
  const listRef = useRef<HTMLDivElement>(null);
  const lastMessageRef = useRef<HTMLDivElement>(null);

  /**
   * Auto-scroll to bottom when new messages arrive
   */
  useEffect(() => {
    if (autoScroll && lastMessageRef.current) {
      lastMessageRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, autoScroll]);

  /**
   * Render empty state when no messages
   */
  if (messages.length === 0) {
    return (
      <div className="message-list">
        <div className="empty-state">
          <div className="empty-state-icon">💬</div>
          <p className="empty-state-text">No messages yet</p>
          <p className="empty-state-hint">Start a conversation by typing a message below</p>
        </div>
      </div>
    );
  }

  return (
    <div className="message-list" ref={listRef}>
      {messages.map((message, index) => {
        const isLast = index === messages.length - 1;
        
        return (
          <div
            key={message.id}
            ref={isLast ? lastMessageRef : null}
          >
            {message.role === 'user' ? (
              <UserMessage message={message} />
            ) : (
              <AgentMessage message={message} />
            )}
          </div>
        );
      })}
    </div>
  );
};

export default MessageList;
