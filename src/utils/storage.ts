/**
 * Local Storage utilities for OpsGenius Frontend
 * Requirements: 8.1, 8.2, 8.5
 * 
 * Handles session saving, loading, and cleanup functionality.
 */

import type { StoredSession, LocalStorageSchema } from '../types/models';

const STORAGE_KEY = 'opsgenius_data';
const MAX_SESSIONS = 10;

/**
 * Get the local storage schema
 * Requirement 8.1: Session data persistence
 */
function getStorageData(): LocalStorageSchema {
  try {
    const data = localStorage.getItem(STORAGE_KEY);
    if (!data) {
      return {
        sessions: [],
        currentSessionId: '',
        mcpConfig: [],
      };
    }
    return JSON.parse(data);
  } catch (error) {
    console.error('Failed to read from localStorage:', error);
    return {
      sessions: [],
      currentSessionId: '',
      mcpConfig: [],
    };
  }
}

/**
 * Save the local storage schema
 * Requirement 8.1: Session data persistence
 */
function setStorageData(data: LocalStorageSchema): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch (error) {
    console.error('Failed to write to localStorage:', error);
    throw error;
  }
}

/**
 * Save a session to local storage
 * Requirement 8.1: Session saving
 */
export function saveSession(session: StoredSession): void {
  const data = getStorageData();
  
  // Find existing session index
  const existingIndex = data.sessions.findIndex(s => s.id === session.id);
  
  if (existingIndex >= 0) {
    // Update existing session
    data.sessions[existingIndex] = session;
  } else {
    // Add new session
    data.sessions.push(session);
    
    // Keep only the most recent MAX_SESSIONS sessions
    if (data.sessions.length > MAX_SESSIONS) {
      data.sessions.sort((a, b) => b.updatedAt - a.updatedAt);
      data.sessions = data.sessions.slice(0, MAX_SESSIONS);
    }
  }
  
  setStorageData(data);
}

/**
 * Load a session from local storage
 * Requirement 8.2: Session restoration
 */
export function loadSession(sessionId: string): StoredSession | null {
  const data = getStorageData();
  const session = data.sessions.find(s => s.id === sessionId);
  return session || null;
}

/**
 * Get all sessions from local storage
 * Requirement 8.5: Historical session listing
 */
export function getAllSessions(): StoredSession[] {
  const data = getStorageData();
  return data.sessions.sort((a, b) => b.updatedAt - a.updatedAt);
}

/**
 * Delete a session from local storage
 * Requirement 8.5: Session management
 */
export function deleteSession(sessionId: string): void {
  const data = getStorageData();
  data.sessions = data.sessions.filter(s => s.id !== sessionId);
  
  if (data.currentSessionId === sessionId) {
    data.currentSessionId = '';
  }
  
  setStorageData(data);
}

/**
 * Set the current session ID
 * Requirement 8.1: Current session tracking
 */
export function setCurrentSessionId(sessionId: string): void {
  const data = getStorageData();
  data.currentSessionId = sessionId;
  setStorageData(data);
}

/**
 * Get the current session ID
 * Requirement 8.2: Current session restoration
 */
export function getCurrentSessionId(): string {
  const data = getStorageData();
  return data.currentSessionId;
}

/**
 * Clear all sessions from local storage
 * Requirement 8.5: Session cleanup
 */
export function clearAllSessions(): void {
  const data = getStorageData();
  data.sessions = [];
  data.currentSessionId = '';
  setStorageData(data);
}
