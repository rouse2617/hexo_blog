/**
 * ChatInterface Component
 * Requirements: 1.1, 1.2, 1.3, 1.4, 1.5
 * 
 * Main chat interface that integrates MessageList and MessageInput.
 * Connects to global state for session management.
 */

import React from 'react';
import { useAppStore } from '../../stores/appStore';
import MessageList from './MessageList';
import MessageInput from './MessageInput';
import './ChatInterface.css';

/**
 * ChatInterface Component
 * 
 * Requirement 1.1: Allow users to submit messages and display them
 * Requirement 1.2: Display agent responses with streaming support
 * Requirement 1.3: Render Markdown content in agent messages
 * Requirement 1.4: Syntax highlight code blocks
 * Requirement 1.5: Control submit button state based on input
 */
export const ChatInterface: React.FC = () => {
  const currentSessionId = useAppStore((state) => state.currentSessionId);
  const messages = useAppStore((state) => state.messages.get(currentSessionId) || []);
  const sendMessage = useAppStore((state) => state.sendMessage);
  const wsConnected = useAppStore((state) => state.wsConnected);

  const handleSendMessage = (content: string) => {
    sendMessage(content);
  };

  if (!currentSessionId) {
    return (
      <div className="chat-interface-empty">
        <p>No active session. Please create a new session.</p>
      </div>
    );
  }

  return (
    <div className="chat-interface" data-session-id={currentSessionId}>
      <MessageList messages={messages} autoScroll={true} />
      <MessageInput onSend={handleSendMessage} disabled={!wsConnected} />
    </div>
  );
};

export default ChatInterface;
