/**
 * MessageInput Component
 * Requirements: 1.1, 1.5
 * 
 * Input field for user messages with submit button.
 */

import React, { useState, useRef, useEffect } from 'react';
import './ChatInterface.css';

export interface MessageInputProps {
  onSend: (message: string) => void;
  disabled?: boolean;
}

/**
 * MessageInput Component
 * 
 * Requirement 1.1: Allow users to submit messages
 * Requirement 1.5: Disable submit button when input is empty
 */
const MessageInput: React.FC<MessageInputProps> = ({ onSend, disabled = false }) => {
  const [message, setMessage] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  /**
   * Handle form submission
   */
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const trimmedMessage = message.trim();
    if (trimmedMessage && !disabled) {
      onSend(trimmedMessage);
      setMessage('');
      
      // Reset textarea height
      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto';
      }
    }
  };

  /**
   * Handle Enter key press (submit on Enter, new line on Shift+Enter)
   */
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  /**
   * Auto-resize textarea based on content
   */
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
    }
  }, [message]);

  /**
   * Check if submit button should be disabled
   * Requirement 1.5: Disable when input is empty or only whitespace
   */
  const isSubmitDisabled = disabled || message.trim().length === 0;

  return (
    <div className="message-input-container">
      <form className="message-input-form" onSubmit={handleSubmit}>
        <textarea
          ref={textareaRef}
          className="message-input"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type a message..."
          disabled={disabled}
          rows={1}
          aria-label="Message input"
        />
        <button
          type="submit"
          className="send-button"
          disabled={isSubmitDisabled}
          aria-label="Send message"
        >
          Send
        </button>
      </form>
    </div>
  );
};

export default MessageInput;
