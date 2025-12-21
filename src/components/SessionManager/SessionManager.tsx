/**
 * SessionManager Component
 * Requirements: 8.3, 8.4, 8.5
 * 
 * Manages chat sessions with new session creation and historical session loading.
 * Displays the most recent 10 sessions.
 */

import React, { useEffect, useState } from 'react';
import { useAppStore } from '../../stores/appStore';
import { getAllSessions, saveSession, setCurrentSessionId } from '../../utils/storage';
import type { StoredSession } from '../../types/models';
import './SessionManager.css';

interface SessionManagerProps {
  onSessionChange?: (sessionId: string) => void;
}

/**
 * Format timestamp to readable string
 */
function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
  });
}

/**
 * SessionManager Component
 * Requirement 8.3: New session creation
 * Requirement 8.4: Historical session list (most recent 10)
 * Requirement 8.5: Session switching functionality
 */
export const SessionManager: React.FC<SessionManagerProps> = ({ onSessionChange }) => {
  const currentSessionId = useAppStore((state) => state.currentSessionId);
  const sessions = useAppStore((state) => state.sessions);
  const createSession = useAppStore((state) => state.createSession);
  const loadSession = useAppStore((state) => state.loadSession);
  
  const [historicalSessions, setHistoricalSessions] = useState<StoredSession[]>([]);
  const [isExpanded, setIsExpanded] = useState(false);

  // Load historical sessions from localStorage
  useEffect(() => {
    const loadHistoricalSessions = () => {
      const allSessions = getAllSessions();
      // Requirement 8.4: Limit to most recent 10 sessions
      setHistoricalSessions(allSessions.slice(0, 10));
    };

    loadHistoricalSessions();
  }, [currentSessionId]);

  // Save current session to localStorage when it changes
  useEffect(() => {
    if (currentSessionId) {
      const session = sessions.get(currentSessionId);
      if (session) {
        saveSession({
          id: session.id,
          title: session.title,
          createdAt: session.createdAt,
          updatedAt: session.updatedAt,
          messages: session.messages,
        });
        setCurrentSessionId(currentSessionId);
      }
    }
  }, [currentSessionId, sessions]);

  // Handle new session creation
  const handleNewSession = () => {
    const newSessionId = createSession();
    if (onSessionChange) {
      onSessionChange(newSessionId);
    }
    setIsExpanded(false);
  };

  // Handle session switching
  const handleLoadSession = (sessionId: string) => {
    if (sessionId === currentSessionId) {
      setIsExpanded(false);
      return;
    }

    loadSession(sessionId);
    if (onSessionChange) {
      onSessionChange(sessionId);
    }
    setIsExpanded(false);
  };

  return (
    <div className="session-manager">
      <button
        className="session-manager-new-button"
        onClick={handleNewSession}
        title="Create new session"
      >
        <span className="session-manager-icon">+</span>
        New Session
      </button>

      <div className="session-manager-history">
        <button
          className="session-manager-toggle"
          onClick={() => setIsExpanded(!isExpanded)}
          aria-expanded={isExpanded}
        >
          <span className="session-manager-icon">
            {isExpanded ? '▼' : '▶'}
          </span>
          History ({historicalSessions.length})
        </button>

        {isExpanded && (
          <div className="session-manager-list">
            {historicalSessions.length === 0 ? (
              <div className="session-manager-empty">
                No historical sessions
              </div>
            ) : (
              historicalSessions.map((session) => (
                <button
                  key={session.id}
                  className={`session-manager-item ${
                    session.id === currentSessionId ? 'active' : ''
                  }`}
                  onClick={() => handleLoadSession(session.id)}
                  title={session.title}
                >
                  <div className="session-manager-item-title">
                    {session.title}
                  </div>
                  <div className="session-manager-item-meta">
                    <span className="session-manager-item-time">
                      {formatTimestamp(session.updatedAt)}
                    </span>
                    <span className="session-manager-item-count">
                      {session.messages.length} messages
                    </span>
                  </div>
                </button>
              ))
            )}
          </div>
        )}
      </div>
    </div>
  );
};
