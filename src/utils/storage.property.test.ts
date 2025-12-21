/**
 * Property-based tests for storage utilities
 * Feature: ops-genius-frontend, Property 23: 会话数据往返一致性
 * Validates: Requirements 8.1, 8.2
 */

import { describe, it, beforeEach, expect } from 'vitest';
import fc from 'fast-check';
import {
  saveSession,
  loadSession,
  clearAllSessions,
} from './storage';
import type { StoredSession } from '../types/models';

describe('Storage Property Tests', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear();
    clearAllSessions();
  });

  // Feature: ops-genius-frontend, Property 23: 会话数据往返一致性
  it('session data should round-trip through localStorage', () => {
    fc.assert(
      fc.property(
        // Generate arbitrary StoredSession objects
        fc.record({
          id: fc.string({ minLength: 1 }),
          title: fc.string(),
          createdAt: fc.integer({ min: 0, max: Date.now() }),
          updatedAt: fc.integer({ min: 0, max: Date.now() }),
          messages: fc.array(
            fc.record({
              id: fc.string({ minLength: 1 }),
              role: fc.constantFrom('user' as const, 'agent' as const),
              content: fc.string(),
              timestamp: fc.integer({ min: 0, max: Date.now() }),
              streaming: fc.option(fc.boolean(), { nil: undefined }),
              charts: fc.option(
                fc.array(
                  fc.record({
                    type: fc.constantFrom('line' as const, 'bar' as const, 'pie' as const),
                    title: fc.option(fc.string(), { nil: undefined }),
                    data: fc.array(
                      fc.record({
                        x: fc.oneof(fc.string(), fc.integer()),
                        y: fc.integer(),
                        label: fc.option(fc.string(), { nil: undefined }),
                      })
                    ),
                    xAxisLabel: fc.option(fc.string(), { nil: undefined }),
                    yAxisLabel: fc.option(fc.string(), { nil: undefined }),
                  })
                ),
                { nil: undefined }
              ),
            })
          ),
        }),
        (session: StoredSession) => {
          // Clear localStorage before each iteration to ensure isolation
          localStorage.clear();
          clearAllSessions();

          // Save the session
          saveSession(session);

          // Load the session back
          const loaded = loadSession(session.id);

          // Verify round-trip consistency
          expect(loaded).not.toBeNull();
          expect(loaded).toEqual(session);
        }
      ),
      { numRuns: 100 }
    );
  });
});
